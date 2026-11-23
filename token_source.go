package etsy

import "context"

// TokenSource supplies a bearer token for API requests. It is defined by the
// consumer, so implementations do not need to import this package.
type TokenSource interface {
	AccessToken(ctx context.Context) (string, error)
}

type staticTokenSource string

func (s staticTokenSource) AccessToken(context.Context) (string, error) {
	if s == "" {
		return "", ErrNoToken
	}
	return string(s), nil
}
