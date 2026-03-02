package track17

import (
	"log/slog"
	"net/http"
	"time"
)

// Option configures the Client.
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client for the API client.
// This allows users to configure custom transport, proxy, TLS settings, etc.
//
// Example:
//
//	client, _ := track17.New("key", track17.WithHTTPClient(&http.Client{
//	    Transport: &http.Transport{
//	        MaxIdleConns: 100,
//	    },
//	}))
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithBaseURL sets a custom base URL for the API.
// This is useful for testing against a mock server.
//
// Example:
//
//	client, _ := track17.New("key", track17.WithBaseURL("http://localhost:8080"))
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithTimeout sets the request timeout for the HTTP client.
// Default is 30 seconds.
//
// Example:
//
//	client, _ := track17.New("key", track17.WithTimeout(10*time.Second))
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// WithRetry configures the retry strategy for failed requests.
// Retries use exponential backoff with jitter starting from the given wait duration.
// Only 5xx errors and 429 (Too Many Requests) are retried.
//
// Example:
//
//	client, _ := track17.New("key", track17.WithRetry(3, time.Second))
func WithRetry(maxRetries int, wait time.Duration) Option {
	return func(c *Client) {
		c.maxRetries = maxRetries
		c.retryWait = wait
	}
}

// WithDebug enables debug logging.
// When enabled, all request/response details are logged using the configured logger.
//
// Note: API keys are automatically masked ("****xxxx") in debug output.
//
// Example:
//
//	client, _ := track17.New("key", track17.WithDebug(true))
func WithDebug(debug bool) Option {
	return func(c *Client) {
		c.debug = debug
	}
}

// WithLogger sets a custom logger for debug output.
// The logger must implement the Logger interface (Printf method).
// Default is a slog-based logger writing to stderr.
//
// Example:
//
//	client, _ := track17.New("key", track17.WithLogger(myLogger))
func WithLogger(logger Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithSlogLogger sets a *slog.Logger as the debug output backend.
// This is the recommended way to integrate with structured logging pipelines.
//
// Example:
//
//	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//	client, _ := track17.New("key", track17.WithSlogLogger(logger))
func WithSlogLogger(l *slog.Logger) Option {
	return func(c *Client) {
		c.logger = &slogAdapter{l: l}
	}
}

// WithRateLimit configures the maximum number of API requests per second.
// Default is 3 requests/second (DefaultRateLimit).
//
// Set this according to your 17Track API plan's rate limit.
//
// Example:
//
//	client, _ := track17.New("key", track17.WithRateLimit(10))
func WithRateLimit(rps int) Option {
	return func(c *Client) {
		c.rateLimiter = newRateLimiter(rps)
	}
}

// WithCircuitBreaker configures the circuit breaker for fault tolerance.
//
//   - maxFailures: number of consecutive failures before the circuit opens (default 5)
//   - resetTimeout: duration to wait in Open state before probing (default 30s)
//
// When the circuit is open, requests immediately return *ErrCircuitOpen without
// hitting the network, protecting downstream resources.
//
// Example:
//
//	client, _ := track17.New("key", track17.WithCircuitBreaker(5, 30*time.Second))
func WithCircuitBreaker(maxFailures int, resetTimeout time.Duration) Option {
	return func(c *Client) {
		c.breaker = newCircuitBreaker(maxFailures, resetTimeout)
	}
}
