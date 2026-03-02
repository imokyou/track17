package track17

import (
	"context"
	"fmt"
)

// QueryService handles tracking information queries.
type QueryService struct {
	client *Client
}

// GetTrackInfoRequest represents a request to get tracking information.
type GetTrackInfoRequest struct {
	// Number is the tracking number (required).
	Number string `json:"number"`

	// CarrierCode is the carrier code.
	CarrierCode int `json:"carrier,omitempty"`
}

// GetTrackInfoResponse contains the result of a get track info request.
type GetTrackInfoResponse struct {
	Accepted []TrackInfo    `json:"accepted,omitempty"`
	Rejected []RejectedItem `json:"rejected,omitempty"`
}

// GetTrackInfo retrieves detailed tracking information for the specified tracking numbers.
//
// Example:
//
//	resp, err := client.Query.GetTrackInfo(ctx, []track17.GetTrackInfoRequest{
//	    {Number: "RR123456789CN", CarrierCode: 3011},
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, info := range resp.Accepted {
//	    fmt.Printf("Number: %s, Status: %d\n", info.Number, info.Track.LatestStatus)
//	}
func (s *QueryService) GetTrackInfo(ctx context.Context, items []GetTrackInfoRequest) (*GetTrackInfoResponse, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("track17: items cannot be empty")
	}
	if len(items) > 40 {
		return nil, fmt.Errorf("track17: too many items, max 40 per request, got %d", len(items))
	}
	var result GetTrackInfoResponse
	if err := s.client.doRequest(ctx, "/gettrackinfo", items, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetTrackListRequest represents a request to list registered tracking numbers.
type GetTrackListRequest struct {
	// TimeRange is the time range for filtering (optional).
	TimeRange *TimeRange `json:"time_range,omitempty"`

	// TrackingStatus filters by main tracking status (optional).
	TrackingStatus int `json:"tracking_status,omitempty"`

	// PushStatus filters by push result status (optional).
	// 0: not pushed, 1: pushed successfully, 2: push failed
	PushStatus int `json:"push_status,omitempty"`

	// PageNo is the page number, starting from 1.
	PageNo int `json:"page_no,omitempty"`

	// CarrierCode filters by carrier code (optional).
	CarrierCode int `json:"carrier,omitempty"`

	// Tag filters by custom tag (optional).
	Tag string `json:"tag,omitempty"`

	// OrderNo filters by order number (optional).
	OrderNo string `json:"order_no,omitempty"`

	// Number filters by tracking number (optional, partial match).
	Number string `json:"number,omitempty"`

	// Lang is the language for translation (optional).
	Lang string `json:"lang,omitempty"`
}

// TimeRange defines a time range for list queries.
type TimeRange struct {
	// From is the start time (ISO 8601 format).
	From string `json:"from,omitempty"`

	// To is the end time (ISO 8601 format).
	To string `json:"to,omitempty"`
}

// GetTrackListResponse contains the result of a get track list request.
type GetTrackListResponse struct {
	// Accepted contains the tracking information list.
	Accepted []TrackInfo `json:"accepted,omitempty"`

	// PageNo is the current page number.
	PageNo int `json:"page_no,omitempty"`

	// HasNext indicates whether there are more pages.
	HasNext bool `json:"has_next,omitempty"`
}

// GetTrackList retrieves a paginated list of registered tracking numbers.
// Each page contains up to 40 items.
//
// Example:
//
//	resp, err := client.Query.GetTrackList(ctx, track17.GetTrackListRequest{
//	    TrackingStatus: track17.StatusInTransit,
//	    PageNo:         1,
//	})
//	for _, info := range resp.Accepted {
//	    fmt.Printf("Number: %s\n", info.Number)
//	}
//	if resp.HasNext {
//	    // fetch next page...
//	}
func (s *QueryService) GetTrackList(ctx context.Context, req GetTrackListRequest) (*GetTrackListResponse, error) {
	var result GetTrackListResponse
	if err := s.client.doRequest(ctx, "/gettracklist", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetQuota retrieves the current account quota information.
//
// Example:
//
//	quota, err := client.Query.GetQuota(ctx)
//	fmt.Printf("Remaining: %d\n", quota.Remaining)
func (s *QueryService) GetQuota(ctx context.Context) (*QuotaInfo, error) {
	var result QuotaInfo
	if err := s.client.doRequest(ctx, "/getquota", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
