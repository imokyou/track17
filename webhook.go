package track17

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Webhook event type constants.
const (
	// EventTrackingUpdated is sent when tracking information is updated.
	EventTrackingUpdated = "TRACKING_UPDATED"

	// EventTrackingStopped is sent when tracking is stopped (e.g., expired after delivery).
	EventTrackingStopped = "TRACKING_STOPPED"
)

// WebhookEvent represents a webhook callback event from 17Track.
type WebhookEvent struct {
	// Event is the event type (e.g., "TRACKING_UPDATED", "TRACKING_STOPPED").
	Event string `json:"event"`

	// Data contains the tracking information associated with the event.
	Data *WebhookData `json:"data,omitempty"`
}

// WebhookData contains the tracking data in a webhook event.
type WebhookData struct {
	// Number is the tracking number.
	Number string `json:"number"`

	// Carrier is the carrier code.
	Carrier int `json:"carrier"`

	// Param is the secondary carrier code.
	Param int `json:"param,omitempty"`

	// Tag is the custom tag.
	Tag string `json:"tag,omitempty"`

	// Track contains the detailed tracking information.
	Track *TrackDetail `json:"track,omitempty"`
}

// VerifySignature verifies the webhook signature using SHA256.
//
// The signature is computed as: SHA256(rawPayload + "/" + apiKey)
//
// Example:
//
//	valid := track17.VerifySignature(payload, signature, "your-api-key")
func VerifySignature(payload []byte, signature string, apiKey string) bool {
	h := sha256.New()
	h.Write(payload)
	h.Write([]byte("/"))
	h.Write([]byte(apiKey))
	expected := hex.EncodeToString(h.Sum(nil))
	return subtle.ConstantTimeCompare([]byte(expected), []byte(signature)) == 1
}

// ParseWebhook reads and parses a webhook HTTP request.
// It verifies the signature and returns the parsed event.
//
// Returns an error if the signature is invalid or the payload cannot be parsed.
//
// Example:
//
//	func webhookHandler(w http.ResponseWriter, r *http.Request) {
//	    event, err := track17.ParseWebhook(r, "your-api-key")
//	    if err != nil {
//	        http.Error(w, "invalid webhook", http.StatusBadRequest)
//	        return
//	    }
//	    fmt.Printf("Event: %s, Number: %s\n", event.Event, event.Data.Number)
//	    w.WriteHeader(http.StatusOK)
//	}
func ParseWebhook(r *http.Request, apiKey string) (*WebhookEvent, error) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("track17: failed to read webhook body: %w", err)
	}

	// Verify signature
	signature := r.Header.Get("sign")
	if signature == "" {
		return nil, fmt.Errorf("track17: missing webhook signature")
	}

	if !VerifySignature(body, signature, apiKey) {
		return nil, fmt.Errorf("track17: invalid webhook signature")
	}

	// Parse event
	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("track17: failed to parse webhook payload: %w", err)
	}

	return &event, nil
}

// WebhookHandler creates an http.Handler that verifies and processes webhook events.
// The provided function is called for each valid webhook event.
//
// Example:
//
//	handler := track17.WebhookHandler("your-api-key", func(event track17.WebhookEvent) {
//	    switch event.Event {
//	    case track17.EventTrackingUpdated:
//	        fmt.Printf("Updated: %s\n", event.Data.Number)
//	    case track17.EventTrackingStopped:
//	        fmt.Printf("Stopped: %s\n", event.Data.Number)
//	    }
//	})
//	http.Handle("/webhook", handler)
func WebhookHandler(apiKey string, fn func(WebhookEvent)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		event, err := ParseWebhook(r, apiKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		fn(*event)
		w.WriteHeader(http.StatusOK)
	})
}
