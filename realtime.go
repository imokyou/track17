package track17

import (
	"context"
	"fmt"
)

// RealTimeService handles real-time tracking queries.
type RealTimeService struct {
	client *Client
}

// RealTimeMode defines the real-time query mode.
type RealTimeMode string

const (
	// RealTimeModeStandard uses standard mode (costs 1 quota).
	RealTimeModeStandard RealTimeMode = "standard"

	// RealTimeModeInstant uses instant mode for faster results (costs 10 quota).
	RealTimeModeInstant RealTimeMode = "instant"
)

// RealTimeRequest represents a request to query real-time tracking info.
type RealTimeRequest struct {
	// Number is the tracking number (required).
	Number string `json:"number"`

	// CarrierCode is the carrier code (required for most carriers).
	CarrierCode int `json:"carrier,omitempty"`

	// Param is an optional secondary carrier code.
	Param int `json:"param,omitempty"`

	// FinalCarrierCode is the final-mile carrier code (optional).
	FinalCarrierCode int `json:"final_carrier,omitempty"`

	// Lang is the language for translation (optional).
	Lang string `json:"lang,omitempty"`

	// Mode is the query mode: "standard" (1 quota) or "instant" (10 quota).
	// Default is "standard".
	Mode RealTimeMode `json:"track_mode,omitempty"`

	// AutoDetect indicates whether to auto-detect the carrier.
	AutoDetect bool `json:"auto_detection,omitempty"`
}

// RealTimeResponse contains the result of a real-time tracking query.
type RealTimeResponse struct {
	Accepted []RealTimeAccepted `json:"accepted,omitempty"`
	Rejected []RejectedItem     `json:"rejected,omitempty"`
}

// RealTimeAccepted represents a successful real-time tracking result.
type RealTimeAccepted struct {
	// Number is the tracking number.
	Number string `json:"number"`

	// Carrier is the carrier code.
	Carrier int `json:"carrier"`

	// Param is the secondary carrier code.
	Param int `json:"param,omitempty"`

	// Track contains the tracking details.
	Track *TrackDetail `json:"track,omitempty"`
}

// GetRealTimeTrackInfo queries real-time tracking information directly from carriers.
//
// Standard mode costs 1 quota per request and is suitable for periodic updates.
// Instant mode costs 10 quota per request but provides faster results.
//
// Example:
//
//	resp, err := client.RealTime.GetRealTimeTrackInfo(ctx, []track17.RealTimeRequest{
//	    {
//	        Number:      "RR123456789CN",
//	        CarrierCode: 3011,
//	        Mode:        track17.RealTimeModeStandard,
//	        Lang:        "en",
//	    },
//	})
func (s *RealTimeService) GetRealTimeTrackInfo(ctx context.Context, items []RealTimeRequest) (*RealTimeResponse, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("track17: items cannot be empty")
	}
	if len(items) > 40 {
		return nil, fmt.Errorf("track17: too many items, max 40 per request, got %d", len(items))
	}
	var result RealTimeResponse
	if err := s.client.doRequest(ctx, "/getRealTimeTrackInfo", items, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
