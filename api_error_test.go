package etsy_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	etsy "github.com/rizquuula/etsy-openapi-client"
)

func TestErrorFromResponse(t *testing.T) {
	if err := etsy.ErrorFromResponse(nil, nil); err != nil {
		t.Errorf("nil response error = %v, want nil", err)
	}
	if err := etsy.ErrorFromResponse(&http.Response{StatusCode: http.StatusOK}, nil); err != nil {
		t.Errorf("200 error = %v, want nil", err)
	}

	header := http.Header{
		"X-Limit-Per-Second":      {"150"},
		"X-Remaining-This-Second": {"149"},
		"X-Limit-Per-Day":         {"100000"},
		"X-Remaining-Today":       {"99998"},
		"Retry-After":             {"7"},
	}
	body := []byte(`{"error":"Slow down"}`)
	resp := &http.Response{StatusCode: http.StatusTooManyRequests, Header: header}
	err := etsy.ErrorFromResponse(resp, body)

	var apiErr *etsy.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusTooManyRequests {
		t.Errorf("status = %d", apiErr.StatusCode)
	}
	if apiErr.Message != "Slow down" {
		t.Errorf("message = %q", apiErr.Message)
	}
	if apiErr.RetryAfter != 7*time.Second {
		t.Errorf("RetryAfter = %v, want 7s", apiErr.RetryAfter)
	}
	wantLimit := etsy.RateLimit{
		LimitPerSecond:      150,
		RemainingThisSecond: 149,
		LimitPerDay:         100000,
		RemainingToday:      99998,
	}
	if apiErr.RateLimit != wantLimit {
		t.Errorf("RateLimit = %+v, want %+v", apiErr.RateLimit, wantLimit)
	}
	if !apiErr.IsRateLimited() {
		t.Error("IsRateLimited = false")
	}
	if !strings.Contains(apiErr.Error(), "429 Too Many Requests") {
		t.Errorf("Error() = %q", apiErr.Error())
	}
	if !strings.Contains(apiErr.Error(), "Slow down") {
		t.Errorf("Error() = %q", apiErr.Error())
	}
}

func TestAPIErrorNonJSONBody(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusInternalServerError, Header: http.Header{}}
	err := etsy.ErrorFromResponse(resp, []byte("<html>oops</html>"))
	var apiErr *etsy.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *APIError", err)
	}
	if apiErr.Message != "" {
		t.Errorf("message = %q, want empty", apiErr.Message)
	}
	if !strings.Contains(apiErr.Error(), "500 Internal Server Error") {
		t.Errorf("Error() = %q", apiErr.Error())
	}
	if string(apiErr.Body) != "<html>oops</html>" {
		t.Errorf("body = %q", apiErr.Body)
	}
}

func TestAPIErrorHeaderIsCopied(t *testing.T) {
	original := http.Header{"X-Limit-Per-Day": {"10"}}
	resp := &http.Response{StatusCode: http.StatusBadRequest, Header: original}
	apiErr := etsy.NewAPIError(resp, nil)
	original.Set("X-Limit-Per-Day", "20")
	if got := apiErr.Header.Get("X-Limit-Per-Day"); got != "10" {
		t.Errorf("copied header = %q, want 10", got)
	}
}

func TestNilAPIError(t *testing.T) {
	var apiErr *etsy.APIError
	if got := apiErr.Error(); got != "<nil>" {
		t.Errorf("Error() = %q", got)
	}
	if apiErr.IsRateLimited() {
		t.Error("nil APIError is rate limited")
	}
}

func TestNewAPIErrorNilResponse(t *testing.T) {
	apiErr := etsy.NewAPIError(nil, []byte("body"))
	if apiErr == nil {
		t.Fatal("NewAPIError returned nil")
	}
	if string(apiErr.Body) != "body" {
		t.Errorf("body = %q", apiErr.Body)
	}
}

func TestRetryAfterHTTPDate(t *testing.T) {
	when := time.Now().Add(time.Hour).UTC().Format(http.TimeFormat)
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Retry-After": {when}},
	}
	apiErr := etsy.NewAPIError(resp, nil)
	if apiErr.RetryAfter < 59*time.Minute || apiErr.RetryAfter > time.Hour {
		t.Errorf("RetryAfter = %v, want about 1 hour", apiErr.RetryAfter)
	}
}
