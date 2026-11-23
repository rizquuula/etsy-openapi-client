package etsy

import (
	"errors"
	"net/http"
	"time"
)

// DefaultBaseURL is the Etsy Open API v3 base URL.
const DefaultBaseURL = "https://openapi.etsy.com"

// HTTPDoer performs HTTP requests. *http.Client implements it.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Config configures an Etsy API client.
type Config struct {
	// Keystring is the app API key from the Etsy developer portal.
	Keystring string
	// SharedSecret is the app's shared secret. Etsy expects both values in the
	// x-api-key header, separated by a colon.
	SharedSecret string

	// AccessToken is a static OAuth access token. Do not set it together with
	// TokenSource.
	AccessToken string
	// TokenSource supplies access tokens for requests. Use an
	// *oauth.RefreshingTokenSource to refresh OAuth tokens automatically.
	TokenSource TokenSource

	// BaseURL overrides DefaultBaseURL.
	BaseURL string
	// HTTPClient overrides the default HTTP client.
	HTTPClient HTTPDoer
	// UserAgent overrides the default User-Agent.
	UserAgent string
}

func (c Config) validate() error {
	if c.Keystring == "" {
		return errors.New("etsy: keystring is required")
	}
	if c.SharedSecret == "" {
		return errors.New("etsy: shared secret is required")
	}
	if c.AccessToken != "" && c.TokenSource != nil {
		return errors.New("etsy: set AccessToken or TokenSource, not both")
	}
	return nil
}

func (c Config) tokenSource() TokenSource {
	if c.TokenSource != nil {
		return c.TokenSource
	}
	if c.AccessToken != "" {
		return staticTokenSource(c.AccessToken)
	}
	return nil
}

func (c Config) httpClient() HTTPDoer {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (c Config) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return DefaultBaseURL
}

func (c Config) userAgent() string {
	if c.UserAgent != "" {
		return c.UserAgent
	}
	return defaultUserAgent
}
