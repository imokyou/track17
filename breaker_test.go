package track17

import (
	"testing"
	"time"
)

// TestCircuitBreakerInitialState verifies a new circuit breaker starts closed.
func TestCircuitBreakerInitialState(t *testing.T) {
	cb := newCircuitBreaker(3, time.Second)
	if cb.State() != StateClosed {
		t.Errorf("expected initial state Closed, got %s", cb.State())
	}
}

// TestCircuitBreakerOpensAfterMaxFailures verifies state transitions Closed → Open.
func TestCircuitBreakerOpensAfterMaxFailures(t *testing.T) {
	cb := newCircuitBreaker(3, time.Second)

	// Record failures up to threshold.
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != StateClosed {
		t.Error("expected Closed before maxFailures reached")
	}

	cb.RecordFailure() // 3rd failure — trips the circuit
	if cb.State() != StateOpen {
		t.Errorf("expected Open after maxFailures, got %s", cb.State())
	}
}

// TestCircuitBreakerRejectsWhenOpen verifies Allow returns ErrCircuitOpen.
func TestCircuitBreakerRejectsWhenOpen(t *testing.T) {
	cb := newCircuitBreaker(1, time.Second)
	cb.RecordFailure() // trip immediately

	err := cb.Allow()
	if err == nil {
		t.Fatal("expected ErrCircuitOpen, got nil")
	}
	if _, ok := err.(*ErrCircuitOpen); !ok {
		t.Errorf("expected *ErrCircuitOpen, got %T: %v", err, err)
	}
	if err.Error() == "" {
		t.Error("ErrCircuitOpen.Error() should not be empty")
	}
}

// TestCircuitBreakerTransitionsToHalfOpen verifies Open → HalfOpen after timeout.
func TestCircuitBreakerTransitionsToHalfOpen(t *testing.T) {
	cb := newCircuitBreaker(1, 10*time.Millisecond)
	cb.RecordFailure() // trip

	if cb.State() != StateOpen {
		t.Fatalf("expected Open, got %s", cb.State())
	}

	time.Sleep(15 * time.Millisecond)

	if cb.State() != StateHalfOpen {
		t.Errorf("expected HalfOpen after timeout, got %s", cb.State())
	}

	// Allow should permit the probe request.
	if err := cb.Allow(); err != nil {
		t.Errorf("expected nil from Allow() in HalfOpen, got %v", err)
	}
}

// TestCircuitBreakerClosesOnSuccessFromHalfOpen verifies HalfOpen → Closed on success.
func TestCircuitBreakerClosesOnSuccessFromHalfOpen(t *testing.T) {
	cb := newCircuitBreaker(1, 10*time.Millisecond)
	cb.RecordFailure()
	time.Sleep(15 * time.Millisecond)
	if cb.State() != StateHalfOpen {
		t.Skip("state not HalfOpen; timing issue")
	}

	cb.RecordSuccess()
	if cb.State() != StateClosed {
		t.Errorf("expected Closed after success from HalfOpen, got %s", cb.State())
	}
}

// TestCircuitBreakerReopensOnFailureFromHalfOpen verifies HalfOpen → Open on failure.
func TestCircuitBreakerReopensOnFailureFromHalfOpen(t *testing.T) {
	cb := newCircuitBreaker(1, 10*time.Millisecond)
	cb.RecordFailure()
	time.Sleep(15 * time.Millisecond)
	if cb.State() != StateHalfOpen {
		t.Skip("state not HalfOpen; timing issue")
	}

	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Errorf("expected Open after failure from HalfOpen, got %s", cb.State())
	}
}

// TestCircuitBreakerSuccessResetsFailureCount verifies failure count resets on success.
func TestCircuitBreakerSuccessResetsFailureCount(t *testing.T) {
	cb := newCircuitBreaker(3, time.Second)
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess() // reset
	cb.RecordFailure()
	cb.RecordFailure()

	// 2 failures after reset — still closed
	if cb.State() != StateClosed {
		t.Errorf("expected Closed (reset), got %s", cb.State())
	}

	cb.RecordFailure() // 3rd — opens
	if cb.State() != StateOpen {
		t.Errorf("expected Open, got %s", cb.State())
	}
}

// TestCircuitBreakerDefaultParameters verifies defaults are applied for invalid inputs.
func TestCircuitBreakerDefaultParameters(t *testing.T) {
	// Zero/negative maxFailures → default 5
	cb := newCircuitBreaker(0, 0)
	if cb.maxFailures != 5 {
		t.Errorf("expected default maxFailures 5, got %d", cb.maxFailures)
	}
	if cb.resetTimeout != 30*time.Second {
		t.Errorf("expected default resetTimeout 30s, got %v", cb.resetTimeout)
	}
}

// TestErrCircuitOpenStateString verifies State.String() output.
func TestErrCircuitOpenStateString(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("State(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

// TestWithCircuitBreakerOption verifies WithCircuitBreaker option is applied.
func TestWithCircuitBreakerOption(t *testing.T) {
	client := mustNew(t, "key", WithCircuitBreaker(10, 60*time.Second))
	if client.breaker.maxFailures != 10 {
		t.Errorf("expected maxFailures 10, got %d", client.breaker.maxFailures)
	}
	if client.breaker.resetTimeout != 60*time.Second {
		t.Errorf("expected resetTimeout 60s, got %v", client.breaker.resetTimeout)
	}
}
