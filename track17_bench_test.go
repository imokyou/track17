package track17

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// benchCtx is a shared background context for benchmarks.
var benchCtx = context.Background()

// benchTestServer creates a test server that returns an empty successful response.
// Suitable for benchmarking the SDK client pathway (not network).
func benchTestServer(b *testing.B) (*Client, *httptest.Server) {
	b.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(apiResponse{
			Code: 0,
			Data: json.RawMessage(`{"accepted":[],"rejected":[]}`),
		})
	}))
	c, err := New("bench-api-key", WithBaseURL(server.URL), WithRateLimit(10000))
	if err != nil {
		b.Fatalf("New() error: %v", err)
	}
	return c, server
}

// BenchmarkRegister measures throughput of the Register API pathway (mock server).
func BenchmarkRegister(b *testing.B) {
	client, server := benchTestServer(b)
	defer server.Close()

	items := []RegisterRequest{
		{Number: "RR123456789CN", CarrierCode: 3011},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Tracking.Register(benchCtx, items)
	}
}

// BenchmarkGetTrackInfo measures throughput of the GetTrackInfo query pathway.
func BenchmarkGetTrackInfo(b *testing.B) {
	client, server := benchTestServer(b)
	defer server.Close()

	items := []GetTrackInfoRequest{
		{Number: "RR123456789CN", CarrierCode: 3011},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Query.GetTrackInfo(benchCtx, items)
	}
}

// BenchmarkRateLimiter measures the overhead of the rate limiter under high load.
func BenchmarkRateLimiter(b *testing.B) {
	// High rps so we don't actually sleep during the benchmark.
	rl := newRateLimiter(100000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = rl.wait(benchCtx)
		}
	})
}

// BenchmarkVerifySignature measures the cost of webhook SHA-256 signature verification.
func BenchmarkVerifySignature(b *testing.B) {
	payload := []byte(`{"event":"TRACKING_UPDATED","data":{"number":"RR123456789CN"}}`)
	apiKey := "benchmark-api-key-12345"
	// Pre-compute a valid signature.
	_ = VerifySignature(payload, "warmup", apiKey) // warm up

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifySignature(payload, "deadbeef", apiKey)
	}
}

// BenchmarkCircuitBreakerAllow measures the overhead of the circuit breaker Allow().
func BenchmarkCircuitBreakerAllow(b *testing.B) {
	cb := newCircuitBreaker(100, time.Second)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cb.Allow()
		}
	})
}
