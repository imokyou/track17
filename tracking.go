package track17

import (
	"context"
	"fmt"
)

// TrackingService handles shipment registration and management operations.
type TrackingService struct {
	client *Client
}

// RegisterRequest represents a single tracking number registration request.
type RegisterRequest struct {
	// Number is the tracking number (required, 5-50 characters).
	Number string `json:"number"`

	// CarrierCode is the carrier code. If 0, auto-detection is used.
	CarrierCode int `json:"carrier,omitempty"`

	// Param is an optional secondary carrier code.
	Param int `json:"param,omitempty"`

	// Tag is a custom tag/label for the tracking number.
	Tag string `json:"tag,omitempty"`

	// Remark is a user remark.
	Remark string `json:"remark,omitempty"`

	// OrderNo is the associated order number.
	OrderNo string `json:"order_no,omitempty"`

	// OrderTime is the order creation time.
	OrderTime string `json:"order_time,omitempty"`

	// Lang is the language for result translation (e.g., "en", "cn").
	Lang string `json:"lang,omitempty"`

	// AutoDetect indicates whether to auto-detect the carrier.
	AutoDetect bool `json:"auto_detection,omitempty"`

	// FulfillmentID is the associated fulfillment ID.
	FulfillmentID string `json:"fulfillment_id,omitempty"`
}

// RegisterResponse contains the result of a batch registration request.
type RegisterResponse struct {
	// Accepted contains successfully registered tracking numbers.
	Accepted []RegisterAccepted `json:"accepted,omitempty"`

	// Rejected contains tracking numbers that failed to register.
	Rejected []RejectedItem `json:"rejected,omitempty"`
}

// RegisterAccepted represents a successfully registered tracking number.
type RegisterAccepted struct {
	// Number is the tracking number.
	Number string `json:"number"`

	// Carrier is the resolved carrier code.
	Carrier int `json:"carrier"`

	// Param is the secondary carrier code.
	Param int `json:"param,omitempty"`

	// Tag is the custom tag.
	Tag string `json:"tag,omitempty"`
}

// Register registers tracking numbers for automatic tracking.
// A maximum of 40 tracking numbers can be registered per request.
//
// Example:
//
//	resp, err := client.Tracking.Register(ctx, []track17.RegisterRequest{
//	    {Number: "RR123456789CN", CarrierCode: 3011, Lang: "en"},
//	    {Number: "EE987654321US"},  // auto-detect carrier
//	})
func (s *TrackingService) Register(ctx context.Context, items []RegisterRequest) (*RegisterResponse, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("track17: items cannot be empty")
	}
	if len(items) > 40 {
		return nil, fmt.Errorf("track17: too many items, max 40 per request, got %d", len(items))
	}
	var result RegisterResponse
	if err := s.client.doRequest(ctx, "/register", items, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ChangeCarrierRequest represents a request to change the carrier code.
type ChangeCarrierRequest struct {
	// Number is the tracking number (required).
	Number string `json:"number"`

	// CarrierOld is the current carrier code (required).
	CarrierOld int `json:"carrier_old"`

	// CarrierNew is the new carrier code (required).
	CarrierNew int `json:"carrier_new"`

	// ParamNew is the new secondary carrier code.
	ParamNew int `json:"param_new,omitempty"`
}

// ChangeCarrierResponse contains the result of a batch change carrier request.
type ChangeCarrierResponse struct {
	Accepted []ChangeCarrierAccepted `json:"accepted,omitempty"`
	Rejected []RejectedItem          `json:"rejected,omitempty"`
}

// ChangeCarrierAccepted represents a successfully changed carrier.
type ChangeCarrierAccepted struct {
	Number     string `json:"number"`
	CarrierOld int    `json:"carrier_old"`
	CarrierNew int    `json:"carrier_new"`
}

// ChangeCarrier changes the carrier code for registered tracking numbers.
// Each tracking number can be changed a maximum of 5 times.
//
// Example:
//
//	resp, err := client.Tracking.ChangeCarrier(ctx, []track17.ChangeCarrierRequest{
//	    {Number: "RR123456789CN", CarrierOld: 3011, CarrierNew: 3012},
//	})
func (s *TrackingService) ChangeCarrier(ctx context.Context, items []ChangeCarrierRequest) (*ChangeCarrierResponse, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("track17: items cannot be empty")
	}
	if len(items) > 40 {
		return nil, fmt.Errorf("track17: too many items, max 40 per request, got %d", len(items))
	}
	var result ChangeCarrierResponse
	if err := s.client.doRequest(ctx, "/changecarrier", items, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ChangeInfoRequest represents a request to change additional info for a tracking number.
type ChangeInfoRequest struct {
	// Number is the tracking number (required).
	Number string `json:"number"`

	// CarrierCode is the carrier code (required).
	CarrierCode int `json:"carrier"`

	// Tag is the new custom tag.
	Tag *string `json:"tag,omitempty"`

	// Remark is the new remark.
	Remark *string `json:"remark,omitempty"`

	// OrderNo is the new order number.
	OrderNo *string `json:"order_no,omitempty"`

	// FulfillmentID is the new fulfillment ID.
	FulfillmentID *string `json:"fulfillment_id,omitempty"`
}

// ChangeInfoResponse contains the result of a batch change info request.
type ChangeInfoResponse struct {
	Accepted []ChangeInfoAccepted `json:"accepted,omitempty"`
	Rejected []RejectedItem       `json:"rejected,omitempty"`
}

// ChangeInfoAccepted represents successfully changed info.
type ChangeInfoAccepted struct {
	Number  string `json:"number"`
	Carrier int    `json:"carrier"`
}

// ChangeInfo modifies additional information for registered tracking numbers.
//
// Example:
//
//	tag := "VIP-Order"
//	resp, err := client.Tracking.ChangeInfo(ctx, []track17.ChangeInfoRequest{
//	    {Number: "RR123456789CN", CarrierCode: 3011, Tag: &tag},
//	})
func (s *TrackingService) ChangeInfo(ctx context.Context, items []ChangeInfoRequest) (*ChangeInfoResponse, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("track17: items cannot be empty")
	}
	if len(items) > 40 {
		return nil, fmt.Errorf("track17: too many items, max 40 per request, got %d", len(items))
	}
	var result ChangeInfoResponse
	if err := s.client.doRequest(ctx, "/changeinfo", items, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// StopTrackRequest represents a request to stop tracking a number.
type StopTrackRequest struct {
	// Number is the tracking number (required).
	Number string `json:"number"`

	// CarrierCode is the carrier code (required).
	CarrierCode int `json:"carrier"`
}

// StopTrackResponse contains the result of a batch stop track request.
type StopTrackResponse struct {
	Accepted []StopTrackAccepted `json:"accepted,omitempty"`
	Rejected []RejectedItem      `json:"rejected,omitempty"`
}

// StopTrackAccepted represents a successfully stopped tracking number.
type StopTrackAccepted struct {
	Number  string `json:"number"`
	Carrier int    `json:"carrier"`
}

// StopTrack stops automatic tracking for the specified tracking numbers.
//
// Example:
//
//	resp, err := client.Tracking.StopTrack(ctx, []track17.StopTrackRequest{
//	    {Number: "RR123456789CN", CarrierCode: 3011},
//	})
func (s *TrackingService) StopTrack(ctx context.Context, items []StopTrackRequest) (*StopTrackResponse, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("track17: items cannot be empty")
	}
	if len(items) > 40 {
		return nil, fmt.Errorf("track17: too many items, max 40 per request, got %d", len(items))
	}
	var result StopTrackResponse
	if err := s.client.doRequest(ctx, "/stoptrack", items, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReTrackRequest represents a request to restart tracking.
type ReTrackRequest struct {
	// Number is the tracking number (required).
	Number string `json:"number"`

	// CarrierCode is the carrier code (required).
	CarrierCode int `json:"carrier"`
}

// ReTrackResponse contains the result of a batch retrack request.
type ReTrackResponse struct {
	Accepted []ReTrackAccepted `json:"accepted,omitempty"`
	Rejected []RejectedItem    `json:"rejected,omitempty"`
}

// ReTrackAccepted represents a successfully restarted tracking number.
type ReTrackAccepted struct {
	Number  string `json:"number"`
	Carrier int    `json:"carrier"`
}

// ReTrack restarts tracking for stopped tracking numbers.
// Each tracking number can only be restarted once.
//
// Example:
//
//	resp, err := client.Tracking.ReTrack(ctx, []track17.ReTrackRequest{
//	    {Number: "RR123456789CN", CarrierCode: 3011},
//	})
func (s *TrackingService) ReTrack(ctx context.Context, items []ReTrackRequest) (*ReTrackResponse, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("track17: items cannot be empty")
	}
	if len(items) > 40 {
		return nil, fmt.Errorf("track17: too many items, max 40 per request, got %d", len(items))
	}
	var result ReTrackResponse
	if err := s.client.doRequest(ctx, "/retrack", items, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteTrackRequest represents a request to delete a tracking number.
type DeleteTrackRequest struct {
	// Number is the tracking number (required).
	Number string `json:"number"`

	// CarrierCode is the carrier code (required).
	CarrierCode int `json:"carrier"`
}

// DeleteTrackResponse contains the result of a batch delete request.
type DeleteTrackResponse struct {
	Accepted []DeleteTrackAccepted `json:"accepted,omitempty"`
	Rejected []RejectedItem        `json:"rejected,omitempty"`
}

// DeleteTrackAccepted represents a successfully deleted tracking number.
type DeleteTrackAccepted struct {
	Number  string `json:"number"`
	Carrier int    `json:"carrier"`
}

// DeleteTrack permanently deletes tracking numbers and their data from the system.
//
// Example:
//
//	resp, err := client.Tracking.DeleteTrack(ctx, []track17.DeleteTrackRequest{
//	    {Number: "RR123456789CN", CarrierCode: 3011},
//	})
func (s *TrackingService) DeleteTrack(ctx context.Context, items []DeleteTrackRequest) (*DeleteTrackResponse, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("track17: items cannot be empty")
	}
	if len(items) > 40 {
		return nil, fmt.Errorf("track17: too many items, max 40 per request, got %d", len(items))
	}
	var result DeleteTrackResponse
	if err := s.client.doRequest(ctx, "/deletetrack", items, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
