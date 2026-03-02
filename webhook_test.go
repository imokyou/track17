package track17

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestVerifySignature(t *testing.T) {
	payload := []byte(`{"event":"TRACKING_UPDATED","data":{"number":"RR123456789CN"}}`)
	apiKey := "test-key"

	// Compute expected signature
	hash := sha256.Sum256([]byte(string(payload) + "/" + apiKey))
	validSig := hex.EncodeToString(hash[:])

	tests := []struct {
		name      string
		payload   []byte
		signature string
		apiKey    string
		valid     bool
	}{
		{
			name:      "valid signature",
			payload:   payload,
			signature: validSig,
			apiKey:    apiKey,
			valid:     true,
		},
		{
			name:      "invalid signature",
			payload:   payload,
			signature: "invalid-sig",
			apiKey:    apiKey,
			valid:     false,
		},
		{
			name:      "wrong api key",
			payload:   payload,
			signature: validSig,
			apiKey:    "wrong-key",
			valid:     false,
		},
		{
			name:      "tampered payload",
			payload:   []byte(`{"event":"TRACKING_UPDATED","data":{"number":"TAMPERED"}}`),
			signature: validSig,
			apiKey:    apiKey,
			valid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VerifySignature(tt.payload, tt.signature, tt.apiKey)
			if result != tt.valid {
				t.Errorf("VerifySignature() = %v, want %v", result, tt.valid)
			}
		})
	}
}

func TestParseWebhook(t *testing.T) {
	apiKey := "test-key"
	event := WebhookEvent{
		Event: EventTrackingUpdated,
		Data: &WebhookData{
			Number:  "RR123456789CN",
			Carrier: 3011,
			Track: &TrackDetail{
				LatestStatus: StatusDelivered,
			},
		},
	}

	payload, _ := json.Marshal(event)
	hash := sha256.Sum256([]byte(string(payload) + "/" + apiKey))
	signature := hex.EncodeToString(hash[:])

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("sign", signature)

	parsed, err := ParseWebhook(req, apiKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.Event != EventTrackingUpdated {
		t.Errorf("expected event %s, got %s", EventTrackingUpdated, parsed.Event)
	}
	if parsed.Data.Number != "RR123456789CN" {
		t.Errorf("expected number RR123456789CN, got %s", parsed.Data.Number)
	}
	if parsed.Data.Track.LatestStatus != StatusDelivered {
		t.Errorf("expected status %d, got %d", StatusDelivered, parsed.Data.Track.LatestStatus)
	}
}

func TestParseWebhookInvalidSignature(t *testing.T) {
	payload := []byte(`{"event":"TRACKING_UPDATED"}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("sign", "invalid")

	_, err := ParseWebhook(req, "test-key")
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestParseWebhookMissingSignature(t *testing.T) {
	payload := []byte(`{"event":"TRACKING_UPDATED"}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))

	_, err := ParseWebhook(req, "test-key")
	if err == nil {
		t.Fatal("expected error for missing signature")
	}
}

func TestWebhookHandler(t *testing.T) {
	apiKey := "test-key"
	var received WebhookEvent

	handler := WebhookHandler(apiKey, func(event WebhookEvent) {
		received = event
	})

	event := WebhookEvent{
		Event: EventTrackingStopped,
		Data: &WebhookData{
			Number:  "TK999",
			Carrier: 100,
		},
	}
	payload, _ := json.Marshal(event)
	hash := sha256.Sum256([]byte(string(payload) + "/" + apiKey))
	signature := hex.EncodeToString(hash[:])

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("sign", signature)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
	if received.Event != EventTrackingStopped {
		t.Errorf("expected event %s, got %s", EventTrackingStopped, received.Event)
	}
	if received.Data.Number != "TK999" {
		t.Errorf("expected number TK999, got %s", received.Data.Number)
	}
}

func TestWebhookHandlerWrongMethod(t *testing.T) {
	handler := WebhookHandler("key", func(event WebhookEvent) {})
	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

// TestParseWebhookReplayAttack verifies stale timestamps are rejected.
func TestParseWebhookReplayAttack(t *testing.T) {
	apiKey := "test-key"
	// Timestamp from 10 minutes ago — beyond the 5-minute window.
	staleTime := time.Now().Add(-10 * time.Minute).Unix()

	event := WebhookEvent{
		Event:     EventTrackingUpdated,
		Timestamp: staleTime,
		Data:      &WebhookData{Number: "RR123456789CN", Carrier: 3011},
	}
	payload, _ := json.Marshal(event)
	hash := sha256.Sum256([]byte(string(payload) + "/" + apiKey))
	signature := hex.EncodeToString(hash[:])

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("sign", signature)

	_, err := ParseWebhook(req, apiKey)
	if err == nil {
		t.Fatal("expected error for stale timestamp (replay attack)")
	}
	if !stringContains(err.Error(), "too old") {
		t.Errorf("expected 'too old' in error, got: %v", err)
	}
}

// TestParseWebhookFutureTimestamp verifies far-future timestamps are rejected.
func TestParseWebhookFutureTimestamp(t *testing.T) {
	apiKey := "test-key"
	futureTime := time.Now().Add(10 * time.Minute).Unix()

	event := WebhookEvent{
		Event:     EventTrackingUpdated,
		Timestamp: futureTime,
		Data:      &WebhookData{Number: "RR123456789CN", Carrier: 3011},
	}
	payload, _ := json.Marshal(event)
	hash := sha256.Sum256([]byte(string(payload) + "/" + apiKey))
	signature := hex.EncodeToString(hash[:])

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("sign", signature)

	_, err := ParseWebhook(req, apiKey)
	if err == nil {
		t.Fatal("expected error for future timestamp (replay attack)")
	}
	if !stringContains(err.Error(), "future") {
		t.Errorf("expected 'future' in error, got: %v", err)
	}
}

// TestParseWebhookFreshTimestamp verifies a current timestamp is accepted.
func TestParseWebhookFreshTimestamp(t *testing.T) {
	apiKey := "test-key"
	freshTime := time.Now().Unix()

	event := WebhookEvent{
		Event:     EventTrackingUpdated,
		Timestamp: freshTime,
		Data:      &WebhookData{Number: "RR123456789CN", Carrier: 3011},
	}
	payload, _ := json.Marshal(event)
	hash := sha256.Sum256([]byte(string(payload) + "/" + apiKey))
	signature := hex.EncodeToString(hash[:])

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("sign", signature)

	parsed, err := ParseWebhook(req, apiKey)
	if err != nil {
		t.Fatalf("unexpected error for fresh timestamp: %v", err)
	}
	if parsed.Timestamp != freshTime {
		t.Errorf("expected Timestamp %d, got %d", freshTime, parsed.Timestamp)
	}
}

// TestParseWebhookZeroTimestamp verifies events without timestamp field pass through.
func TestParseWebhookZeroTimestamp(t *testing.T) {
	apiKey := "test-key"
	event := WebhookEvent{
		Event: EventTrackingUpdated,
		// Timestamp: 0 — absent / legacy event
		Data: &WebhookData{Number: "RR123456789CN", Carrier: 3011},
	}
	payload, _ := json.Marshal(event)
	hash := sha256.Sum256([]byte(string(payload) + "/" + apiKey))
	signature := hex.EncodeToString(hash[:])

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("sign", signature)

	_, err := ParseWebhook(req, apiKey)
	if err != nil {
		t.Fatalf("unexpected error for zero timestamp: %v", err)
	}
}

// stringContains is a helper because strings package is not imported in this file.
func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
