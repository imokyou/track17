package track17

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestServer creates a test server and client for unit testing.
func newTestServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	client := New("test-api-key", WithBaseURL(server.URL))
	return client, server
}

// TestNew verifies client initialization and default values.
func TestNew(t *testing.T) {
	client := New("my-key")
	if client.apiKey != "my-key" {
		t.Errorf("expected apiKey 'my-key', got '%s'", client.apiKey)
	}
	if client.baseURL != DefaultBaseURL {
		t.Errorf("expected baseURL '%s', got '%s'", DefaultBaseURL, client.baseURL)
	}
	if client.Tracking == nil {
		t.Error("expected Tracking service to be initialized")
	}
	if client.Query == nil {
		t.Error("expected Query service to be initialized")
	}
	if client.Push == nil {
		t.Error("expected Push service to be initialized")
	}
	if client.RealTime == nil {
		t.Error("expected RealTime service to be initialized")
	}
}

// TestNewWithOptions verifies option application.
func TestNewWithOptions(t *testing.T) {
	client := New("key",
		WithBaseURL("http://custom.api"),
		WithRetry(3, 0),
		WithDebug(true),
	)
	if client.baseURL != "http://custom.api" {
		t.Errorf("expected baseURL 'http://custom.api', got '%s'", client.baseURL)
	}
	if client.maxRetries != 3 {
		t.Errorf("expected maxRetries 3, got %d", client.maxRetries)
	}
	if !client.debug {
		t.Error("expected debug to be true")
	}
}

// TestRegister verifies the tracking registration API.
func TestRegister(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("17token") != "test-api-key" {
			t.Errorf("expected 17token header 'test-api-key', got '%s'", r.Header.Get("17token"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		// Verify request body
		var items []RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if len(items) != 1 || items[0].Number != "RR123456789CN" {
			t.Errorf("unexpected request body: %+v", items)
		}

		// Return response
		resp := apiResponse{
			Code: 0,
			Data: mustJSON(t, RegisterResponse{
				Accepted: []RegisterAccepted{
					{Number: "RR123456789CN", Carrier: 3011},
				},
			}),
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	resp, err := client.Tracking.Register(context.Background(), []RegisterRequest{
		{Number: "RR123456789CN", CarrierCode: 3011},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Accepted) != 1 {
		t.Fatalf("expected 1 accepted, got %d", len(resp.Accepted))
	}
	if resp.Accepted[0].Number != "RR123456789CN" {
		t.Errorf("expected number 'RR123456789CN', got '%s'", resp.Accepted[0].Number)
	}
	if resp.Accepted[0].Carrier != 3011 {
		t.Errorf("expected carrier 3011, got %d", resp.Accepted[0].Carrier)
	}
}

// TestGetTrackInfo verifies the tracking info query API.
func TestGetTrackInfo(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{
			Code: 0,
			Data: mustJSON(t, GetTrackInfoResponse{
				Accepted: []TrackInfo{
					{
						Number:  "RR123456789CN",
						Carrier: 3011,
						Track: &TrackDetail{
							Status:          "Y",
							LatestStatus:    StatusDelivered,
							LatestSubStatus: SubStatusDelivered_Signed,
							LatestEvent:     "Delivered to recipient",
							Events: []TrackEvent{
								{
									Time:        "2024-01-15T10:30:00",
									Status:      StatusDelivered,
									SubStatus:   SubStatusDelivered_Signed,
									Description: "Delivered to recipient",
									Location:    "New York, NY",
								},
							},
						},
					},
				},
			}),
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	resp, err := client.Query.GetTrackInfo(context.Background(), []GetTrackInfoRequest{
		{Number: "RR123456789CN", CarrierCode: 3011},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Accepted) != 1 {
		t.Fatalf("expected 1 accepted, got %d", len(resp.Accepted))
	}
	info := resp.Accepted[0]
	if info.Track.LatestStatus != StatusDelivered {
		t.Errorf("expected status %d, got %d", StatusDelivered, info.Track.LatestStatus)
	}
	if len(info.Track.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(info.Track.Events))
	}
}

// TestGetTrackList verifies the tracking list query API.
func TestGetTrackList(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{
			Code: 0,
			Data: mustJSON(t, GetTrackListResponse{
				Accepted: []TrackInfo{
					{Number: "TK001", Carrier: 3011},
					{Number: "TK002", Carrier: 3012},
				},
				PageNo:  1,
				HasNext: true,
			}),
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	resp, err := client.Query.GetTrackList(context.Background(), GetTrackListRequest{
		TrackingStatus: StatusInTransit,
		PageNo:         1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Accepted) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Accepted))
	}
	if !resp.HasNext {
		t.Error("expected HasNext to be true")
	}
}

// TestGetQuota verifies the quota query API.
func TestGetQuota(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{
			Code: 0,
			Data: mustJSON(t, QuotaInfo{
				Total:     10000,
				Used:      2500,
				Remaining: 7500,
			}),
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	quota, err := client.Query.GetQuota(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quota.Remaining != 7500 {
		t.Errorf("expected remaining 7500, got %d", quota.Remaining)
	}
}

// TestRealTimeTrackInfo verifies the real-time tracking query API.
func TestRealTimeTrackInfo(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{
			Code: 0,
			Data: mustJSON(t, RealTimeResponse{
				Accepted: []RealTimeAccepted{
					{
						Number:  "RM123456789US",
						Carrier: 21051,
						Track: &TrackDetail{
							LatestStatus: StatusInTransit,
						},
					},
				},
			}),
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	resp, err := client.RealTime.GetRealTimeTrackInfo(context.Background(), []RealTimeRequest{
		{
			Number:      "RM123456789US",
			CarrierCode: 21051,
			Mode:        RealTimeModeStandard,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Accepted) != 1 {
		t.Fatalf("expected 1 accepted, got %d", len(resp.Accepted))
	}
	if resp.Accepted[0].Track.LatestStatus != StatusInTransit {
		t.Errorf("expected status InTransit, got %d", resp.Accepted[0].Track.LatestStatus)
	}
}

// TestPush verifies the manual push API.
func TestPush(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{
			Code: 0,
			Data: mustJSON(t, PushResponse{
				Accepted: []PushAccepted{
					{Number: "RR123456789CN", Carrier: 3011},
				},
			}),
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	resp, err := client.Push.Push(context.Background(), []PushRequest{
		{Number: "RR123456789CN", CarrierCode: 3011},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Accepted) != 1 {
		t.Fatalf("expected 1 accepted, got %d", len(resp.Accepted))
	}
}

// TestAPIError verifies API-level error handling.
func TestAPIError(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{Code: ErrInvalidAPIKey}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	_, err := client.Query.GetQuota(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := IsAPIError(err)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if !apiErr.IsInvalidAPIKey() {
		t.Errorf("expected invalid API key error, got code %d", apiErr.Code)
	}
}

// TestHTTPError verifies HTTP-level error handling.
func TestHTTPError(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	})
	defer server.Close()

	_, err := client.Query.GetQuota(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := IsAPIError(err)
	if !ok {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("expected HTTP 500, got %d", apiErr.StatusCode)
	}
}

// TestRejectedItems verifies handling of partially rejected batch requests.
func TestRejectedItems(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{
			Code: 0,
			Data: mustJSON(t, RegisterResponse{
				Accepted: []RegisterAccepted{
					{Number: "TK001", Carrier: 3011},
				},
				Rejected: []RejectedItem{
					{
						Number: "TK002",
						Error: RejectedError{
							Code:    ErrAlreadyRegistered,
							Message: "tracking number already registered",
						},
					},
				},
			}),
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	resp, err := client.Tracking.Register(context.Background(), []RegisterRequest{
		{Number: "TK001"},
		{Number: "TK002"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Accepted) != 1 {
		t.Errorf("expected 1 accepted, got %d", len(resp.Accepted))
	}
	if len(resp.Rejected) != 1 {
		t.Errorf("expected 1 rejected, got %d", len(resp.Rejected))
	}
	if resp.Rejected[0].Error.Code != ErrAlreadyRegistered {
		t.Errorf("expected error code %d, got %d", ErrAlreadyRegistered, resp.Rejected[0].Error.Code)
	}
}

// TestChangeCarrier verifies the change carrier API.
func TestChangeCarrier(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{
			Code: 0,
			Data: mustJSON(t, ChangeCarrierResponse{
				Accepted: []ChangeCarrierAccepted{
					{Number: "RR123456789CN", CarrierOld: 3011, CarrierNew: 3012},
				},
			}),
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	resp, err := client.Tracking.ChangeCarrier(context.Background(), []ChangeCarrierRequest{
		{Number: "RR123456789CN", CarrierOld: 3011, CarrierNew: 3012},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Accepted) != 1 {
		t.Fatalf("expected 1 accepted, got %d", len(resp.Accepted))
	}
}

// TestStopTrack verifies the stop tracking API.
func TestStopTrack(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{
			Code: 0,
			Data: mustJSON(t, StopTrackResponse{
				Accepted: []StopTrackAccepted{
					{Number: "RR123456789CN", Carrier: 3011},
				},
			}),
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	resp, err := client.Tracking.StopTrack(context.Background(), []StopTrackRequest{
		{Number: "RR123456789CN", CarrierCode: 3011},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Accepted) != 1 {
		t.Fatalf("expected 1 accepted, got %d", len(resp.Accepted))
	}
}

// mustJSON marshals v to json.RawMessage for test responses.
func mustJSON(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	return data
}
