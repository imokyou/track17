package track17

import (
	"errors"
	"fmt"
)

// API error codes returned by the 17Track API.
const (
	ErrSuccess            = 0
	ErrInternalError      = -18010001 // 内部错误
	ErrInvalidJSON        = -18010002 // JSON 格式错误
	ErrInvalidAPIKey      = -18010003 // API Key 无效
	ErrIPNotAllowed       = -18010004 // IP 不在白名单
	ErrRateLimited        = -18010005 // 请求频率超限
	ErrInsufficientQuota  = -18010006 // 余额不足
	ErrInvalidParam       = -18010007 // 参数错误
	ErrTooManyItems       = -18010008 // 批量数量超限
	ErrAlreadyRegistered  = -18019901 // 单号已注册
	ErrNotRegistered      = -18019902 // 单号未注册
	ErrInvalidNumber      = -18019903 // 单号格式无效
	ErrInvalidCarrier     = -18019904 // 运输商代码无效
	ErrCarrierRequired    = -18019905 // 需要指定运输商
	ErrChangeCarrierLimit = -18019906 // 修改运输商次数已达上限
	ErrRetrackLimit       = -18019907 // 重启跟踪次数已达上限
	ErrNumberTooShort     = -18019908 // 单号长度不足
	ErrNumberTooLong      = -18019909 // 单号长度过长
	ErrTrackStopped       = -18019910 // 单号已停止跟踪
	ErrTrackNotStopped    = -18019911 // 单号未停止跟踪
	ErrDuplicateInBatch   = -18019912 // 批量请求中存在重复单号
)

// errorMessages maps error codes to human-readable messages.
var errorMessages = map[int]string{
	ErrSuccess:            "success",
	ErrInternalError:      "internal server error",
	ErrInvalidJSON:        "invalid JSON format",
	ErrInvalidAPIKey:      "invalid API key",
	ErrIPNotAllowed:       "IP address not in whitelist",
	ErrRateLimited:        "rate limit exceeded",
	ErrInsufficientQuota:  "insufficient quota",
	ErrInvalidParam:       "invalid parameter",
	ErrTooManyItems:       "too many items in batch request",
	ErrAlreadyRegistered:  "tracking number already registered",
	ErrNotRegistered:      "tracking number not registered",
	ErrInvalidNumber:      "invalid tracking number format",
	ErrInvalidCarrier:     "invalid carrier code",
	ErrCarrierRequired:    "carrier code is required",
	ErrChangeCarrierLimit: "change carrier limit reached (max 5)",
	ErrRetrackLimit:       "retrack limit reached (max 1)",
	ErrNumberTooShort:     "tracking number too short (min 5 chars)",
	ErrNumberTooLong:      "tracking number too long (max 50 chars)",
	ErrTrackStopped:       "tracking already stopped",
	ErrTrackNotStopped:    "tracking not stopped",
	ErrDuplicateInBatch:   "duplicate tracking number in batch",
}

// getErrorMessage returns a human-readable message for the given error code.
func getErrorMessage(code int) string {
	if msg, ok := errorMessages[code]; ok {
		return msg
	}
	return fmt.Sprintf("unknown error (code: %d)", code)
}

// APIError represents an error returned by the 17Track API.
type APIError struct {
	// Code is the API error code.
	Code int `json:"code"`

	// Message is a human-readable description of the error.
	Message string `json:"message"`

	// StatusCode is the HTTP status code (if applicable).
	StatusCode int `json:"-"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.StatusCode > 0 && e.StatusCode != 200 {
		return fmt.Sprintf("track17: API error %d (HTTP %d): %s", e.Code, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("track17: API error %d: %s", e.Code, e.Message)
}

// Unwrap returns nil since APIError is a root error.
// This method enables errors.Is and errors.As support.
func (e *APIError) Unwrap() error { return nil }

// IsInternalError returns true if the error is an internal server error.
func (e *APIError) IsInternalError() bool { return e.Code == ErrInternalError }

// IsInvalidAPIKey returns true if the API key is invalid.
func (e *APIError) IsInvalidAPIKey() bool { return e.Code == ErrInvalidAPIKey }

// IsIPNotAllowed returns true if the IP is not in the whitelist.
func (e *APIError) IsIPNotAllowed() bool { return e.Code == ErrIPNotAllowed }

// IsRateLimited returns true if the rate limit was exceeded.
func (e *APIError) IsRateLimited() bool { return e.Code == ErrRateLimited }

// IsInsufficientQuota returns true if the quota is insufficient.
func (e *APIError) IsInsufficientQuota() bool { return e.Code == ErrInsufficientQuota }

// IsAlreadyRegistered returns true if the tracking number is already registered.
func (e *APIError) IsAlreadyRegistered() bool { return e.Code == ErrAlreadyRegistered }

// IsNotRegistered returns true if the tracking number is not registered.
func (e *APIError) IsNotRegistered() bool { return e.Code == ErrNotRegistered }

// RejectedItem represents an item that was rejected by the API in a batch operation.
type RejectedItem struct {
	Number  string        `json:"number"`
	Error   RejectedError `json:"error"`
	Carrier int           `json:"carrier,omitempty"`
}

// RejectedError contains error details for a rejected item.
type RejectedError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// IsAPIError returns true and the underlying *APIError if err is an *APIError.
// This function supports wrapped errors via errors.As.
func IsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}
