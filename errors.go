package etsy

import "errors"

// ErrNoToken is returned when a token source yields an empty access token.
var ErrNoToken = errors.New("etsy: token source returned an empty access token")
