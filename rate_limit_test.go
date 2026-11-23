package etsy_test

import (
	"net/http"
	"testing"

	etsy "github.com/rizquuula/etsy-openapi-client"
)

func TestParseRateLimit(t *testing.T) {
	t.Parallel()

	header := http.Header{
		"X-Limit-Per-Second":      {"150"},
		"X-Remaining-This-Second": {"149"},
		"X-Limit-Per-Day":         {"100000"},
		"X-Remaining-Today":       {"99998"},
	}
	want := etsy.RateLimit{
		LimitPerSecond:      150,
		RemainingThisSecond: 149,
		LimitPerDay:         100000,
		RemainingToday:      99998,
	}
	if got := etsy.ParseRateLimit(header); got != want {
		t.Errorf("ParseRateLimit() = %+v, want %+v", got, want)
	}
}

func TestParseRateLimitTypoCompatibility(t *testing.T) {
	t.Parallel()

	header := http.Header{"X-Remaining-This-Secon": {"42"}}
	if got := etsy.ParseRateLimit(header).RemainingThisSecond; got != 42 {
		t.Errorf("RemainingThisSecond = %d, want 42", got)
	}
}

func TestParseRateLimitZeroValueWins(t *testing.T) {
	t.Parallel()

	header := http.Header{
		"X-Remaining-This-Second": {"0"},
		"X-Remaining-This-Secon":  {"42"},
	}
	if got := etsy.ParseRateLimit(header).RemainingThisSecond; got != 0 {
		t.Errorf("RemainingThisSecond = %d, want 0", got)
	}
}

func TestParseRateLimitMissingHeaders(t *testing.T) {
	t.Parallel()

	if got := etsy.ParseRateLimit(nil); got != (etsy.RateLimit{}) {
		t.Errorf("ParseRateLimit(nil) = %+v, want zero", got)
	}
}
