package oauth

import "errors"

var (
	// ErrNoToken is returned when a token is missing or empty.
	ErrNoToken = errors.New("etsy: access token is empty")
	// ErrNoRefreshToken is returned when a token is expired and cannot be refreshed.
	ErrNoRefreshToken = errors.New("etsy: access token expired and no refresh token is available")
)
