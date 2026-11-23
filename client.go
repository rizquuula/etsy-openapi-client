package etsy

import (
	"context"
	"fmt"
	"net/http"
)

// Version is the client version sent in the default User-Agent.
const Version = "0.1.0"

const defaultUserAgent = "etsy-openapi-client/" + Version

// Client is an Etsy Open API v3 client. It embeds the generated typed client,
// so methods such as GetShopWithResponse and CreateDraftListingWithFormdataBodyWithResponse
// are promoted. Use New to create one.
type Client struct {
	*ClientWithResponses
}

// New creates a client that adds the Etsy API key and OAuth token to requests.
func New(cfg Config) (*Client, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	api, err := NewClientWithResponses(
		cfg.baseURL(),
		WithHTTPClient(cfg.httpClient()),
		WithRequestEditorFn(newAuthEditor(cfg, cfg.tokenSource())),
	)
	if err != nil {
		return nil, fmt.Errorf("etsy: create client: %w", err)
	}
	return &Client{ClientWithResponses: api}, nil
}

func newAuthEditor(cfg Config, source TokenSource) RequestEditorFn {
	apiKey := cfg.Keystring + ":" + cfg.SharedSecret
	userAgent := cfg.userAgent()
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "application/json")
		if source == nil {
			return nil
		}
		token, err := source.AccessToken(ctx)
		if err != nil {
			return fmt.Errorf("etsy: get access token: %w", err)
		}
		if token == "" {
			return ErrNoToken
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
}
