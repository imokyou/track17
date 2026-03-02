package track17

import (
	"context"
	"fmt"
)

// PushService handles manual push notification operations.
type PushService struct {
	client *Client
}

// PushRequest represents a request to manually trigger a webhook push.
type PushRequest struct {
	// Number is the tracking number (required).
	Number string `json:"number"`

	// CarrierCode is the carrier code (required).
	CarrierCode int `json:"carrier"`
}

// PushResponse contains the result of a batch push request.
type PushResponse struct {
	Accepted []PushAccepted `json:"accepted,omitempty"`
	Rejected []RejectedItem `json:"rejected,omitempty"`
}

// PushAccepted represents a successfully submitted push request.
type PushAccepted struct {
	Number  string `json:"number"`
	Carrier int    `json:"carrier"`
}

// Push manually triggers a webhook push for the specified tracking numbers.
// This forces an immediate webhook callback with the latest tracking data.
//
// Example:
//
//	resp, err := client.Push.Push(ctx, []track17.PushRequest{
//	    {Number: "RR123456789CN", CarrierCode: 3011},
//	})
func (s *PushService) Push(ctx context.Context, items []PushRequest) (*PushResponse, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("track17: items cannot be empty")
	}
	if len(items) > 40 {
		return nil, fmt.Errorf("track17: too many items, max 40 per request, got %d", len(items))
	}
	var result PushResponse
	if err := s.client.doRequest(ctx, "/push", items, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
