package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultAuthURL is the Etsy OAuth authorization endpoint.
	DefaultAuthURL = "https://www.etsy.com/oauth/connect"
	// DefaultTokenURL is the Etsy OAuth token endpoint.
	DefaultTokenURL = "https://api.etsy.com/v3/public/oauth/token"
)

// Client performs Etsy's OAuth 2.0 authorization-code and refresh flows.
type Client struct {
	// ClientID is the app keystring from the Etsy developer portal.
	ClientID string
	// RedirectURI must exactly match a callback URL registered for the app.
	RedirectURI string
	// AuthURL overrides DefaultAuthURL.
	AuthURL string
	// TokenURL overrides DefaultTokenURL.
	TokenURL string
	// HTTPClient overrides the default HTTP client.
	HTTPClient *http.Client
}

// New creates an OAuth client with Etsy's default endpoints.
func New(clientID, redirectURI string) *Client {
	return &Client{
		ClientID:    clientID,
		RedirectURI: redirectURI,
		AuthURL:     DefaultAuthURL,
		TokenURL:    DefaultTokenURL,
	}
}

// AuthCodeURL builds the URL that asks the user to grant the requested scopes.
// The same codeVerifier must be passed to Exchange.
func (c *Client) AuthCodeURL(state, codeVerifier string, scopes ...Scope) (string, error) {
	if c == nil {
		return "", errors.New("etsy: oauth client is nil")
	}
	if c.ClientID == "" {
		return "", errors.New("etsy: oauth client ID is empty")
	}
	if c.RedirectURI == "" {
		return "", errors.New("etsy: oauth redirect URI is empty")
	}
	if err := validateCodeVerifier(codeVerifier); err != nil {
		return "", err
	}
	if len(scopes) == 0 {
		return "", errors.New("etsy: at least one OAuth scope is required")
	}

	scopeNames := make([]string, len(scopes))
	for i, scope := range scopes {
		if scope == "" {
			return "", errors.New("etsy: OAuth scope is empty")
		}
		scopeNames[i] = string(scope)
	}

	authURL := c.AuthURL
	if authURL == "" {
		authURL = DefaultAuthURL
	}
	u, err := url.Parse(authURL)
	if err != nil {
		return "", fmt.Errorf("etsy: parse OAuth URL: %w", err)
	}

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", c.ClientID)
	q.Set("redirect_uri", c.RedirectURI)
	q.Set("scope", strings.Join(scopeNames, " "))
	if state != "" {
		q.Set("state", state)
	}
	q.Set("code_challenge", CodeChallengeS256(codeVerifier))
	q.Set("code_challenge_method", "S256")
	u.RawQuery = strings.ReplaceAll(q.Encode(), "+", "%20")

	return u.String(), nil
}

// Exchange trades an authorization code for an access token and refresh token.
func (c *Client) Exchange(ctx context.Context, code, codeVerifier string) (*Token, error) {
	if c == nil {
		return nil, errors.New("etsy: oauth client is nil")
	}
	if c.ClientID == "" {
		return nil, errors.New("etsy: oauth client ID is empty")
	}
	if code == "" {
		return nil, errors.New("etsy: authorization code is empty")
	}
	if err := validateCodeVerifier(codeVerifier); err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", c.ClientID)
	form.Set("code", code)
	form.Set("code_verifier", codeVerifier)
	if c.RedirectURI != "" {
		form.Set("redirect_uri", c.RedirectURI)
	}
	return c.requestToken(ctx, form)
}

// Refresh trades a refresh token for a new access token and refresh token.
func (c *Client) Refresh(ctx context.Context, refreshToken string) (*Token, error) {
	if c == nil {
		return nil, errors.New("etsy: oauth client is nil")
	}
	if c.ClientID == "" {
		return nil, errors.New("etsy: oauth client ID is empty")
	}
	if refreshToken == "" {
		return nil, errors.New("etsy: refresh token is empty")
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", c.ClientID)
	form.Set("refresh_token", refreshToken)
	token, err := c.requestToken(ctx, form)
	if err != nil {
		return nil, err
	}
	// Etsy normally rotates the refresh token, but keep the old one if the
	// response omits it.
	if token.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}
	return token, nil
}

func (c *Client) requestToken(ctx context.Context, form url.Values) (*Token, error) {
	tokenURL := c.TokenURL
	if tokenURL == "" {
		tokenURL = DefaultTokenURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("etsy: create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("etsy: request OAuth token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("etsy: read OAuth token response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, newTokenError(resp.StatusCode, body)
	}

	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("etsy: decode OAuth token: %w", err)
	}
	if token.AccessToken == "" {
		return nil, errors.New("etsy: OAuth token response has no access_token")
	}
	if token.TokenType == "" {
		token.TokenType = "Bearer"
	}
	if token.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().UTC().Add(time.Duration(token.ExpiresIn) * time.Second)
	}
	return &token, nil
}
