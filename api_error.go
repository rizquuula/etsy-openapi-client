package etsy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// APIError is a non-2xx response from the Etsy API.
type APIError struct {
	StatusCode int
	// Message is Etsy's error message, when the body contains one.
	Message string
	// RateLimit is parsed from the response headers.
	RateLimit RateLimit
	// RetryAfter is parsed from the Retry-After header. It is zero when absent.
	RetryAfter time.Duration
	// Body is the raw response body.
	Body []byte
	// Header is a copy of the response headers.
	Header http.Header
}

// ErrorFromResponse returns an *APIError for non-2xx responses and nil for
// successful ones. Pass the HTTPResponse and Body fields of a generated
// *WithResponse result:
//
//	resp, err := client.GetShopWithResponse(ctx, shopID)
//	if err != nil { ... }
//	if err := etsy.ErrorFromResponse(resp.HTTPResponse, resp.Body); err != nil { ... }
func ErrorFromResponse(resp *http.Response, body []byte) error {
	if resp == nil {
		return nil
	}
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	return NewAPIError(resp, body)
}

// NewAPIError builds an *APIError from a response and its body.
func NewAPIError(resp *http.Response, body []byte) *APIError {
	if resp == nil {
		return &APIError{Body: body}
	}
	err := &APIError{
		StatusCode: resp.StatusCode,
		Body:       body,
		RateLimit:  ParseRateLimit(resp.Header),
		RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
		Header:     cloneHeader(resp.Header),
	}
	var payload struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &payload) == nil {
		err.Message = payload.Error
	}
	return err
}

// IsRateLimited reports whether the request was rejected with HTTP 429.
func (e *APIError) IsRateLimited() bool {
	return e != nil && e.StatusCode == http.StatusTooManyRequests
}

// Error implements error.
func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}
	statusText := http.StatusText(e.StatusCode)
	var message string
	switch {
	case e.StatusCode > 0 && statusText != "":
		message = fmt.Sprintf("etsy: %d %s", e.StatusCode, statusText)
	case e.StatusCode > 0:
		message = fmt.Sprintf("etsy: HTTP %d", e.StatusCode)
	default:
		message = "etsy: API error"
	}
	if e.Message != "" && e.Message != statusText {
		message += ": " + e.Message
	}
	return message
}

func parseRetryAfter(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil {
		if wait := time.Until(when); wait > 0 {
			return wait
		}
	}
	return 0
}

func cloneHeader(header http.Header) http.Header {
	if header == nil {
		return nil
	}
	clone := make(http.Header, len(header))
	for key, values := range header {
		clone[key] = append([]string(nil), values...)
	}
	return clone
}
