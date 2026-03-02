package track17

import (
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
//	client := track17.New("key", track17.WithHTTPClient(&http.Client{
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
//	client := track17.New("key", track17.WithBaseURL("http://localhost:8080"))
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
//	client := track17.New("key", track17.WithTimeout(10*time.Second))
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// WithRetry configures the retry strategy for failed requests.
// Retries use exponential backoff starting from the given wait duration.
// Only 5xx errors and 429 (Too Many Requests) are retried.
//
// Example:
//
//	client := track17.New("key", track17.WithRetry(3, time.Second))
func WithRetry(maxRetries int, wait time.Duration) Option {
	return func(c *Client) {
		c.maxRetries = maxRetries
		c.retryWait = wait
	}
}

// WithDebug enables debug logging to stderr.
// When enabled, all request/response bodies are logged.
//
// Example:
//
//	client := track17.New("key", track17.WithDebug(true))
func WithDebug(debug bool) Option {
	return func(c *Client) {
		c.debug = debug
	}
}

// WithLogger sets a custom logger for debug output.
// The logger must implement the Logger interface (Printf method).
// Default is log.New(os.Stderr, "", log.LstdFlags).
//
// Example:
//
//	client := track17.New("key", track17.WithLogger(myLogger))
func WithLogger(logger Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}
