package track17

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
	if client.logger == nil {
		t.Error("expected logger to be initialized")
	}
}

// TestNewEmptyAPIKey verifies New panics with empty API key.
func TestNewEmptyAPIKey(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for empty API key")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "API key must not be empty") {
			t.Errorf("unexpected panic message: %v", r)
		}
	}()
	New("")
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

// TestWithHTTPClient verifies custom HTTP client option.
func TestWithHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 5 * time.Second}
	client := New("key", WithHTTPClient(customClient))
	if client.httpClient != customClient {
		t.Error("expected custom HTTP client to be set")
	}
}

// TestWithTimeout verifies timeout option.
func TestWithTimeout(t *testing.T) {
	client := New("key", WithTimeout(5*time.Second))
	if client.httpClient.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", client.httpClient.Timeout)
	}
}

// TestWithLogger verifies custom logger option.
func TestWithLogger(t *testing.T) {
	var logged bool
	logger := &testLogger{fn: func(format string, v ...interface{}) {
		logged = true
	}}
	client := New("key", WithLogger(logger), WithDebug(true))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(apiResponse{Code: 0})
	}))
	defer server.Close()
	client.baseURL = server.URL

	client.Query.GetQuota(context.Background())
	if !logged {
		t.Error("expected custom logger to be called")
	}
}

type testLogger struct {
	fn func(format string, v ...interface{})
}

func (l *testLogger) Printf(format string, v ...interface{}) {
	l.fn(format, v...)
}

// TestClose verifies the Close method doesn't panic.
func TestClose(t *testing.T) {
	client := New("key")
	client.Close() // should not panic
	client.Close() // safe to call multiple times
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

// TestRegisterValidation verifies input validation for Register.
func TestRegisterValidation(t *testing.T) {
	client := New("key")

	// Empty items
	_, err := client.Tracking.Register(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil items")
	}

	_, err = client.Tracking.Register(context.Background(), []RegisterRequest{})
	if err == nil {
		t.Fatal("expected error for empty items")
	}

	// Too many items
	items := make([]RegisterRequest, 41)
	_, err = client.Tracking.Register(context.Background(), items)
	if err == nil {
		t.Fatal("expected error for too many items")
	}
	if !strings.Contains(err.Error(), "max 40") {
		t.Errorf("expected max 40 error, got: %v", err)
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

// TestGetTrackInfoValidation verifies input validation for GetTrackInfo.
func TestGetTrackInfoValidation(t *testing.T) {
	client := New("key")

	_, err := client.Query.GetTrackInfo(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil items")
	}

	items := make([]GetTrackInfoRequest, 41)
	_, err = client.Query.GetTrackInfo(context.Background(), items)
	if err == nil {
		t.Fatal("expected error for too many items")
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

// TestRealTimeValidation verifies input validation for RealTime.
func TestRealTimeValidation(t *testing.T) {
	client := New("key")

	_, err := client.RealTime.GetRealTimeTrackInfo(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil items")
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

// TestPushValidation verifies input validation for Push.
func TestPushValidation(t *testing.T) {
	client := New("key")

	_, err := client.Push.Push(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil items")
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

// TestAPIErrorMethodsComprehensive tests all APIError helper methods.
func TestAPIErrorMethodsComprehensive(t *testing.T) {
	tests := []struct {
		code   int
		method string
		check  func(*APIError) bool
	}{
		{ErrInternalError, "IsInternalError", func(e *APIError) bool { return e.IsInternalError() }},
		{ErrInvalidAPIKey, "IsInvalidAPIKey", func(e *APIError) bool { return e.IsInvalidAPIKey() }},
		{ErrIPNotAllowed, "IsIPNotAllowed", func(e *APIError) bool { return e.IsIPNotAllowed() }},
		{ErrRateLimited, "IsRateLimited", func(e *APIError) bool { return e.IsRateLimited() }},
		{ErrInsufficientQuota, "IsInsufficientQuota", func(e *APIError) bool { return e.IsInsufficientQuota() }},
		{ErrAlreadyRegistered, "IsAlreadyRegistered", func(e *APIError) bool { return e.IsAlreadyRegistered() }},
		{ErrNotRegistered, "IsNotRegistered", func(e *APIError) bool { return e.IsNotRegistered() }},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			e := &APIError{Code: tt.code, Message: "test"}
			if !tt.check(e) {
				t.Errorf("%s() returned false for code %d", tt.method, tt.code)
			}
		})
	}
}

// TestAPIErrorString tests the Error() method with and without HTTP status.
func TestAPIErrorString(t *testing.T) {
	// Without HTTP status
	e := &APIError{Code: ErrInvalidAPIKey, Message: "invalid API key"}
	expected := "track17: API error -18010003: invalid API key"
	if e.Error() != expected {
		t.Errorf("expected %q, got %q", expected, e.Error())
	}

	// With HTTP status
	e = &APIError{Code: 500, Message: "HTTP 500: server error", StatusCode: 500}
	result := e.Error()
	if !strings.Contains(result, "HTTP 500") {
		t.Errorf("expected HTTP status in error string, got %q", result)
	}
}

// TestAPIErrorUnwrap verifies Unwrap returns nil for root errors.
func TestAPIErrorUnwrap(t *testing.T) {
	e := &APIError{Code: ErrInvalidAPIKey, Message: "test"}
	if e.Unwrap() != nil {
		t.Error("expected Unwrap to return nil")
	}
}

// TestGetErrorMessage tests error message lookup.
func TestGetErrorMessage(t *testing.T) {
	msg := getErrorMessage(ErrInvalidAPIKey)
	if msg != "invalid API key" {
		t.Errorf("expected 'invalid API key', got %q", msg)
	}

	// Unknown code
	msg = getErrorMessage(99999)
	if !strings.Contains(msg, "unknown error") {
		t.Errorf("expected 'unknown error', got %q", msg)
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

// TestHTTPError4xx verifies non-retryable HTTP errors.
func TestHTTPError4xx(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
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
	if apiErr.StatusCode != 403 {
		t.Errorf("expected HTTP 403, got %d", apiErr.StatusCode)
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

// TestChangeCarrierValidation verifies input validation for ChangeCarrier.
func TestChangeCarrierValidation(t *testing.T) {
	client := New("key")

	_, err := client.Tracking.ChangeCarrier(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil items")
	}
}

// TestChangeInfo verifies the change info API.
func TestChangeInfo(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{
			Code: 0,
			Data: mustJSON(t, ChangeInfoResponse{
				Accepted: []ChangeInfoAccepted{
					{Number: "RR123456789CN", Carrier: 3011},
				},
			}),
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	tag := "VIP"
	resp, err := client.Tracking.ChangeInfo(context.Background(), []ChangeInfoRequest{
		{Number: "RR123456789CN", CarrierCode: 3011, Tag: &tag},
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

// TestStopTrackValidation verifies input validation for StopTrack.
func TestStopTrackValidation(t *testing.T) {
	client := New("key")

	_, err := client.Tracking.StopTrack(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil items")
	}
}

// TestReTrack verifies the retrack API.
func TestReTrack(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{
			Code: 0,
			Data: mustJSON(t, ReTrackResponse{
				Accepted: []ReTrackAccepted{
					{Number: "RR123456789CN", Carrier: 3011},
				},
			}),
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	resp, err := client.Tracking.ReTrack(context.Background(), []ReTrackRequest{
		{Number: "RR123456789CN", CarrierCode: 3011},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Accepted) != 1 {
		t.Fatalf("expected 1 accepted, got %d", len(resp.Accepted))
	}
}

// TestReTrackValidation verifies input validation for ReTrack.
func TestReTrackValidation(t *testing.T) {
	client := New("key")

	_, err := client.Tracking.ReTrack(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil items")
	}
}

// TestDeleteTrack verifies the delete track API.
func TestDeleteTrack(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{
			Code: 0,
			Data: mustJSON(t, DeleteTrackResponse{
				Accepted: []DeleteTrackAccepted{
					{Number: "RR123456789CN", Carrier: 3011},
				},
			}),
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	resp, err := client.Tracking.DeleteTrack(context.Background(), []DeleteTrackRequest{
		{Number: "RR123456789CN", CarrierCode: 3011},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Accepted) != 1 {
		t.Fatalf("expected 1 accepted, got %d", len(resp.Accepted))
	}
}

// TestDeleteTrackValidation verifies input validation for DeleteTrack.
func TestDeleteTrackValidation(t *testing.T) {
	client := New("key")

	_, err := client.Tracking.DeleteTrack(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil items")
	}
}

// TestContextCancellation verifies that context cancellation is respected.
func TestContextCancellation(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Simulate slow server
		json.NewEncoder(w).Encode(apiResponse{Code: 0})
	})
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.Query.GetQuota(ctx)
	if err == nil {
		t.Fatal("expected error due to context cancellation")
	}
}

// TestRetryOn5xx verifies retry behavior on 5xx HTTP errors.
func TestRetryOn5xx(t *testing.T) {
	attempts := 0
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error"))
			return
		}
		json.NewEncoder(w).Encode(apiResponse{
			Code: 0,
			Data: mustJSON(t, QuotaInfo{Remaining: 100}),
		})
	})
	defer server.Close()
	client.maxRetries = 3
	client.retryWait = time.Millisecond

	quota, err := client.Query.GetQuota(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
	if quota.Remaining != 100 {
		t.Errorf("expected remaining 100, got %d", quota.Remaining)
	}
}

// TestUserAgent verifies the User-Agent header is set correctly.
func TestUserAgent(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua != "track17-go-sdk/"+Version {
			t.Errorf("expected User-Agent 'track17-go-sdk/%s', got '%s'", Version, ua)
		}
		json.NewEncoder(w).Encode(apiResponse{Code: 0})
	})
	defer server.Close()

	client.Query.GetQuota(context.Background())
}

// TestDebugOutput verifies debug output doesn't panic.
func TestDebugOutput(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(apiResponse{
			Code: 0,
			Data: mustJSON(t, QuotaInfo{Remaining: 100}),
		})
	})
	defer server.Close()
	client.debug = true

	// Should not panic
	_, err := client.Query.GetQuota(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestIsAPIErrorWithNonAPIError verifies IsAPIError returns false for non-API errors.
func TestIsAPIErrorWithNonAPIError(t *testing.T) {
	_, ok := IsAPIError(context.DeadlineExceeded)
	if ok {
		t.Error("expected IsAPIError to return false for non-API error")
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
