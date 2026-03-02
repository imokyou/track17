// Package track17 provides an enterprise-grade Go SDK for the 17Track API v2.4.
//
// The SDK supports all 17Track API endpoints including tracking registration,
// carrier management, tracking queries, real-time lookups, push notifications,
// and webhook handling.
//
// Usage:
//
//	client, err := track17.New("your-api-key")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Register tracking numbers
//	resp, err := client.Tracking.Register(ctx, []track17.RegisterRequest{
//	    {Number: "RR123456789CN", CarrierCode: 3011},
//	})
//
//	// Get tracking info
//	info, err := client.Query.GetTrackInfo(ctx, []track17.GetTrackInfoRequest{
//	    {Number: "RR123456789CN", CarrierCode: 3011},
//	})
package track17

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	// DefaultBaseURL is the default base URL for the 17Track API v2.4.
	DefaultBaseURL = "https://api.17track.net/track/v2.4"

	// DefaultRateLimit is the default rate limit (requests per second).
	DefaultRateLimit = 3

	// DefaultTimeout is the default request timeout.
	DefaultTimeout = 30 * time.Second

	// Version is the SDK version.
	Version = "1.0.0"
)

// Logger defines a minimal logging interface for the SDK.
// Implementations must be safe for concurrent use.
type Logger interface {
	Printf(format string, v ...interface{})
}

// slogAdapter wraps *slog.Logger to satisfy the Logger interface.
type slogAdapter struct {
	l *slog.Logger
}

func (a *slogAdapter) Printf(format string, v ...interface{}) {
	a.l.Debug(fmt.Sprintf(format, v...))
}

// maskAPIKey masks an API key for safe logging.
// It shows the last 4 characters: "****xxxx".
func maskAPIKey(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}

// Client is the 17Track API client. It manages communication with the 17Track
// API v2.4 and provides access to the various API services.
//
// Client is safe for concurrent use by multiple goroutines.
type Client struct {
	// httpClient is the underlying HTTP client used for API requests.
	httpClient *http.Client

	// apiKey is the 17Track API key used for authentication.
	apiKey string

	// baseURL is the base URL for the 17Track API.
	baseURL string

	// rateLimiter controls request rate limiting.
	rateLimiter *rateLimiter

	// breaker is the circuit breaker for fault tolerance.
	breaker *circuitBreaker

	// retry configuration
	maxRetries int
	retryWait  time.Duration

	// debug enables verbose logging.
	debug bool

	// logger is the logger used for debug output.
	logger Logger

	// Services
	Tracking *TrackingService
	Query    *QueryService
	Push     *PushService
	RealTime *RealTimeService
}

// New creates a new 17Track API client with the given API key and options.
//
// Returns an error if apiKey is empty, instead of panicking.
// The client is safe for concurrent use by multiple goroutines.
//
// Example:
//
//	client, err := track17.New("your-api-key",
//	    track17.WithTimeout(10*time.Second),
//	    track17.WithRetry(3, time.Second),
//	    track17.WithCircuitBreaker(5, 30*time.Second),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
func New(apiKey string, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("track17: API key must not be empty")
	}

	slogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	c := &Client{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		maxRetries: 0,
		retryWait:  time.Second,
		breaker:    newCircuitBreaker(5, 30*time.Second),
		logger:     &slogAdapter{l: slogger},
	}

	for _, opt := range opts {
		opt(c)
	}

	// Initialize rate limiter after options are applied (rps may be overridden)
	if c.rateLimiter == nil {
		c.rateLimiter = newRateLimiter(DefaultRateLimit)
	}

	// Initialize services
	c.Tracking = &TrackingService{client: c}
	c.Query = &QueryService{client: c}
	c.Push = &PushService{client: c}
	c.RealTime = &RealTimeService{client: c}

	return c, nil
}

// Close releases any resources held by the client.
// It is safe to call Close multiple times.
func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}

// apiResponse is the common response wrapper from the 17Track API.
type apiResponse struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

// doRequest performs an authenticated API request to the given path.
// It enforces rate limiting, circuit breaking, and retry with exponential
// back-off + jitter.
func (c *Client) doRequest(ctx context.Context, path string, body interface{}, result interface{}) error {
	// Check circuit breaker before acquiring the rate-limiter slot.
	if err := c.breaker.Allow(); err != nil {
		return err
	}

	// Wait for rate limiter.
	if err := c.rateLimiter.wait(ctx); err != nil {
		return err
	}

	// Serialize body once; re-use for retries.
	jsonBody, err := c.serializeBody(body)
	if err != nil {
		return err
	}

	if c.debug && body != nil {
		c.logger.Printf("[track17] POST %s%s\n  Body: %s", c.baseURL, path, string(jsonBody))
	}

	url := c.baseURL + path

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			wait := c.jitteredWait(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
			if err := c.rateLimiter.wait(ctx); err != nil {
				return err
			}
		}

		respBody, statusCode, err := c.executeRequest(ctx, url, jsonBody)
		if err != nil {
			lastErr = err
			c.breaker.RecordFailure()
			continue
		}

		if c.debug {
			c.logger.Printf("[track17] Response %d: %s", statusCode, string(respBody))
		}

		// Handle HTTP-level errors.
		if statusCode != http.StatusOK {
			lastErr = &APIError{
				Code:       statusCode,
				Message:    fmt.Sprintf("HTTP %d: %s", statusCode, string(respBody)),
				StatusCode: statusCode,
			}
			if statusCode >= 500 || statusCode == http.StatusTooManyRequests {
				c.breaker.RecordFailure()
				continue
			}
			c.breaker.RecordFailure()
			return lastErr
		}

		// Parse and validate API-level response.
		if err := c.parseAPIResponse(respBody, result); err != nil {
			c.breaker.RecordFailure()
			return err
		}

		c.breaker.RecordSuccess()
		return nil
	}

	return lastErr
}

// serializeBody marshals body to JSON. Returns nil bytes when body is nil.
func (c *Client) serializeBody(body interface{}) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("track17: failed to marshal request body: %w", err)
	}
	return b, nil
}

// buildRequest constructs an authenticated HTTP POST request.
func (c *Client) buildRequest(ctx context.Context, url string, body []byte) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("track17: failed to create request: %w", err)
	}

	req.Header.Set("17token", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "track17-go-sdk/"+Version)

	if c.debug {
		c.logger.Printf("[track17] 17token: %s", maskAPIKey(c.apiKey))
	}

	return req, nil
}

// executeRequest sends a single HTTP request and returns the response body and status code.
func (c *Client) executeRequest(ctx context.Context, url string, body []byte) ([]byte, int, error) {
	req, err := c.buildRequest(ctx, url, body)
	if err != nil {
		return nil, 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("track17: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("track17: failed to read response body: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// parseAPIResponse unmarshals the 17Track API envelope and then the data payload.
func (c *Client) parseAPIResponse(respBody []byte, result interface{}) error {
	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return fmt.Errorf("track17: failed to parse response: %w", err)
	}

	if apiResp.Code != 0 {
		return &APIError{
			Code:    apiResp.Code,
			Message: getErrorMessage(apiResp.Code),
		}
	}

	if result != nil && apiResp.Data != nil {
		if err := json.Unmarshal(apiResp.Data, result); err != nil {
			return fmt.Errorf("track17: failed to parse response data: %w", err)
		}
	}

	return nil
}

// jitteredWait returns an exponential back-off duration with ±25% random jitter
// to avoid the "thundering herd" problem when many goroutines retry at once.
func (c *Client) jitteredWait(attempt int) time.Duration {
	base := c.retryWait * time.Duration(1<<uint(attempt-1))
	// Add ±25% jitter: jitter ∈ [0, base/2)
	jitter := time.Duration(rand.Int63n(int64(base)/2 + 1))
	return base + jitter
}

// rateLimiter implements a simple token bucket rate limiter.
type rateLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	lastTime time.Time
}

// newRateLimiter creates a rate limiter that allows rps requests per second.
func newRateLimiter(rps int) *rateLimiter {
	if rps <= 0 {
		rps = DefaultRateLimit
	}
	return &rateLimiter{
		interval: time.Second / time.Duration(rps),
	}
}

// wait blocks until a request can be made without exceeding the rate limit.
// It respects context cancellation.
func (rl *rateLimiter) wait(ctx context.Context) error {
	rl.mu.Lock()
	now := time.Now()
	var sleepDur time.Duration
	if elapsed := now.Sub(rl.lastTime); elapsed < rl.interval {
		sleepDur = rl.interval - elapsed
	}
	rl.lastTime = now.Add(sleepDur)
	rl.mu.Unlock()

	if sleepDur > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleepDur):
		}
	}
	return nil
}
