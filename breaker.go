package track17

import (
	"fmt"
	"sync"
	"time"
)

// State represents the state of a circuit breaker.
type State int

const (
	// StateClosed is the normal operating state; requests flow through.
	StateClosed State = iota

	// StateOpen means the circuit is tripped; requests are rejected immediately.
	StateOpen

	// StateHalfOpen allows a single probe request to test if the upstream has recovered.
	StateHalfOpen
)

// String returns a human-readable string for the State.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when the circuit breaker is open and rejects a request.
type ErrCircuitOpen struct {
	// ResetAt is the earliest time the circuit may transition to half-open.
	ResetAt time.Time
}

func (e *ErrCircuitOpen) Error() string {
	return fmt.Sprintf("track17: circuit breaker is open, retry after %s", e.ResetAt.Format(time.RFC3339))
}

// circuitBreaker implements a three-state (Closed → Open → HalfOpen → Closed)
// circuit breaker with zero external dependencies.
//
// State transitions:
//   - Closed  → Open      : consecutive failures exceed maxFailures
//   - Open    → HalfOpen  : resetTimeout has elapsed since last failure
//   - HalfOpen → Closed   : the probe request succeeds
//   - HalfOpen → Open     : the probe request fails
type circuitBreaker struct {
	mu           sync.Mutex
	state        State
	failures     int
	maxFailures  int
	resetTimeout time.Duration
	openedAt     time.Time
}

// newCircuitBreaker creates a circuit breaker.
//
//   - maxFailures: consecutive failures required to trip the circuit (default 5)
//   - resetTimeout: how long to wait in Open state before probing (default 30s)
func newCircuitBreaker(maxFailures int, resetTimeout time.Duration) *circuitBreaker {
	if maxFailures <= 0 {
		maxFailures = 5
	}
	if resetTimeout <= 0 {
		resetTimeout = 30 * time.Second
	}
	return &circuitBreaker{
		state:        StateClosed,
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
	}
}

// State returns the current circuit state. Safe for concurrent access.
func (cb *circuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.currentState()
}

// currentState returns the effective state, promoting Open → HalfOpen when the
// reset timeout has elapsed. Must be called with cb.mu held.
func (cb *circuitBreaker) currentState() State {
	if cb.state == StateOpen && time.Since(cb.openedAt) >= cb.resetTimeout {
		cb.state = StateHalfOpen
	}
	return cb.state
}

// Allow returns nil if the request is permitted to proceed, or *ErrCircuitOpen if
// the circuit is open. It also transitions Open → HalfOpen when appropriate.
func (cb *circuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.currentState() {
	case StateClosed:
		return nil
	case StateHalfOpen:
		// Allow exactly one probe request through.
		return nil
	default: // StateOpen
		return &ErrCircuitOpen{ResetAt: cb.openedAt.Add(cb.resetTimeout)}
	}
}

// RecordSuccess records a successful request, resetting the breaker to Closed.
func (cb *circuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.state = StateClosed
}

// RecordFailure records a failed request. When the failure count reaches
// maxFailures the circuit transitions to Open.
func (cb *circuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// A failure in HalfOpen immediately re-opens the circuit.
	if cb.state == StateHalfOpen {
		cb.state = StateOpen
		cb.openedAt = time.Now()
		return
	}

	cb.failures++
	if cb.failures >= cb.maxFailures {
		cb.state = StateOpen
		cb.openedAt = time.Now()
	}
}
