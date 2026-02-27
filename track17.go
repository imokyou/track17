// Package track17 provides an enterprise-grade Go SDK for the 17Track API v2.4.
//
// The SDK supports all 17Track API endpoints including tracking registration,
// carrier management, tracking queries, real-time lookups, push notifications,
// and webhook handling.
//
// Usage:
//
//	client := track17.New("your-api-key")
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
	"net/http"
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

// Client is the 17Track API client. It manages communication with the 17Track
// API v2.4 and provides access to the various API services.
type Client struct {
	// httpClient is the underlying HTTP client used for API requests.
	httpClient *http.Client

	// apiKey is the 17Track API key used for authentication.
	apiKey string

	// baseURL is the base URL for the 17Track API.
	baseURL string

	// rateLimiter controls request rate limiting.
	rateLimiter *rateLimiter

	// retry configuration
	maxRetries int
	retryWait  time.Duration

	// debug enables verbose logging.
	debug bool

	// Services
	Tracking *TrackingService
	Query    *QueryService
	Push     *PushService
	RealTime *RealTimeService
}

// New creates a new 17Track API client with the given API key and options.
//
// The client is safe for concurrent use by multiple goroutines.
//
// Example:
//
//	client := track17.New("your-api-key",
//	    track17.WithTimeout(10*time.Second),
//	    track17.WithRetry(3, time.Second),
//	)
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		maxRetries: 0,
		retryWait:  time.Second,
	}

	for _, opt := range opts {
		opt(c)
	}

	// Initialize rate limiter
	c.rateLimiter = newRateLimiter(DefaultRateLimit)

	// Initialize services
	c.Tracking = &TrackingService{client: c}
	c.Query = &QueryService{client: c}
	c.Push = &PushService{client: c}
	c.RealTime = &RealTimeService{client: c}

	return c
}

// apiResponse is the common response wrapper from the 17Track API.
type apiResponse struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

// doRequest performs an authenticated API request to the given path.
func (c *Client) doRequest(ctx context.Context, path string, body interface{}, result interface{}) error {
	// Wait for rate limiter
	c.rateLimiter.wait()

	// Marshal request body
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("track17: failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)

		if c.debug {
			fmt.Printf("[track17] POST %s%s\n  Body: %s\n", c.baseURL, path, string(jsonBody))
		}
	}

	url := c.baseURL + path

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Re-create body reader for retry
			if body != nil {
				jsonBody, _ := json.Marshal(body)
				bodyReader = bytes.NewReader(jsonBody)
			}
			wait := c.retryWait * time.Duration(1<<uint(attempt-1)) // Exponential backoff
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
			c.rateLimiter.wait()
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bodyReader)
		if err != nil {
			return fmt.Errorf("track17: failed to create request: %w", err)
		}

		req.Header.Set("17token", c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "track17-go-sdk/"+Version)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("track17: request failed: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("track17: failed to read response body: %w", err)
			continue
		}

		if c.debug {
			fmt.Printf("[track17] Response %d: %s\n", resp.StatusCode, string(respBody))
		}

		// Handle HTTP-level errors
		if resp.StatusCode != http.StatusOK {
			lastErr = &APIError{
				Code:       resp.StatusCode,
				Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
				StatusCode: resp.StatusCode,
			}
			// Only retry on 5xx errors or 429
			if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
				continue
			}
			return lastErr
		}

		// Parse API response
		var apiResp apiResponse
		if err := json.Unmarshal(respBody, &apiResp); err != nil {
			return fmt.Errorf("track17: failed to parse response: %w", err)
		}

		// Check API-level error code
		if apiResp.Code != 0 {
			return &APIError{
				Code:       apiResp.Code,
				Message:    getErrorMessage(apiResp.Code),
				StatusCode: resp.StatusCode,
			}
		}

		// Parse result
		if result != nil && apiResp.Data != nil {
			if err := json.Unmarshal(apiResp.Data, result); err != nil {
				return fmt.Errorf("track17: failed to parse response data: %w", err)
			}
		}

		return nil
	}

	return lastErr
}

// rateLimiter implements a simple token bucket rate limiter.
type rateLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	lastTime time.Time
}

// newRateLimiter creates a rate limiter that allows rps requests per second.
func newRateLimiter(rps int) *rateLimiter {
	return &rateLimiter{
		interval: time.Second / time.Duration(rps),
	}
}

// wait blocks until a request can be made without exceeding the rate limit.
func (rl *rateLimiter) wait() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if elapsed := now.Sub(rl.lastTime); elapsed < rl.interval {
		time.Sleep(rl.interval - elapsed)
	}
	rl.lastTime = time.Now()
}
