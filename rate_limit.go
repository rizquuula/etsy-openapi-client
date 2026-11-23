package etsy

import (
	"net/http"
	"strconv"
	"strings"
)

// RateLimit is the rate-limit state reported by Etsy response headers.
type RateLimit struct {
	LimitPerSecond      int
	RemainingThisSecond int
	LimitPerDay         int
	RemainingToday      int
}

// ParseRateLimit reads Etsy's rate-limit headers. Missing headers are zero.
func ParseRateLimit(header http.Header) RateLimit {
	remainingThisSecond := intHeader(header, "x-remaining-this-second")
	if header.Get("x-remaining-this-second") == "" {
		// Etsy's documentation uses this spelling in one place.
		remainingThisSecond = intHeader(header, "x-remaining-this-secon")
	}
	return RateLimit{
		LimitPerSecond:      intHeader(header, "x-limit-per-second"),
		RemainingThisSecond: remainingThisSecond,
		LimitPerDay:         intHeader(header, "x-limit-per-day"),
		RemainingToday:      intHeader(header, "x-remaining-today"),
	}
}

func intHeader(header http.Header, name string) int {
	value, err := strconv.Atoi(strings.TrimSpace(header.Get(name)))
	if err != nil || value < 0 {
		return 0
	}
	return value
}
