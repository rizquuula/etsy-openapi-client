package oauth_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rizquuula/etsy-openapi-client/oauth"
)

func TestNewDefaults(t *testing.T) {
	t.Parallel()

	client := oauth.New("client-id", "https://example.com/callback")
	if client.AuthURL != oauth.DefaultAuthURL {
		t.Errorf("AuthURL = %q, want %q", client.AuthURL, oauth.DefaultAuthURL)
	}
	if client.TokenURL != oauth.DefaultTokenURL {
		t.Errorf("TokenURL = %q, want %q", client.TokenURL, oauth.DefaultTokenURL)
	}
}

func TestAuthCodeURL(t *testing.T) {
	t.Parallel()

	client := oauth.New("keystring-123", "https://example.com/callback")
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

	got, err := client.AuthCodeURL("state-abc", verifier, oauth.ScopeShopsRead, oauth.ScopeTransactionsRead)
	if err != nil {
		t.Fatalf("AuthCodeURL: %v", err)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	if u.Scheme != "https" || u.Host != "www.etsy.com" || u.Path != "/oauth/connect" {
		t.Errorf("URL = %s, want https://www.etsy.com/oauth/connect", got)
	}
	q := u.Query()
	for key, want := range map[string]string{
		"response_type":         "code",
		"client_id":             "keystring-123",
		"redirect_uri":          "https://example.com/callback",
		"scope":                 "shops_r transactions_r",
		"state":                 "state-abc",
		"code_challenge":        oauth.CodeChallengeS256(verifier),
		"code_challenge_method": "S256",
	} {
		if q.Get(key) != want {
			t.Errorf("%s = %q, want %q", key, q.Get(key), want)
		}
	}
	if !strings.Contains(u.RawQuery, "shops_r%20transactions_r") {
		t.Errorf("raw query does not percent-encode the scope separator: %s", u.RawQuery)
	}
}

func TestAuthCodeURLValidation(t *testing.T) {
	t.Parallel()

	validVerifier := strings.Repeat("a", 43)
	tests := []struct {
		name     string
		client   *oauth.Client
		state    string
		verifier string
		scopes   []oauth.Scope
	}{
		{name: "nil client", client: nil, verifier: validVerifier, scopes: []oauth.Scope{oauth.ScopeShopsRead}},
		{name: "empty client", client: &oauth.Client{RedirectURI: "https://example.com"}, verifier: validVerifier, scopes: []oauth.Scope{oauth.ScopeShopsRead}},
		{name: "empty redirect", client: &oauth.Client{ClientID: "key"}, verifier: validVerifier, scopes: []oauth.Scope{oauth.ScopeShopsRead}},
		{name: "no scopes", client: oauth.New("key", "https://example.com"), verifier: validVerifier},
		{name: "empty scope", client: oauth.New("key", "https://example.com"), verifier: validVerifier, scopes: []oauth.Scope{""}},
		{name: "short verifier", client: oauth.New("key", "https://example.com"), verifier: "short", scopes: []oauth.Scope{oauth.ScopeShopsRead}},
		{name: "invalid verifier", client: oauth.New("key", "https://example.com"), verifier: strings.Repeat("a", 42) + "!", scopes: []oauth.Scope{oauth.ScopeShopsRead}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := tt.client.AuthCodeURL(tt.state, tt.verifier, tt.scopes...); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestExchange(t *testing.T) {
	t.Parallel()

	const verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"access_token": "1234.access",
			"token_type": "Bearer",
			"expires_in": 3600,
			"refresh_token": "1234.refresh",
			"scope": "shops_r transactions_r"
		}`)
	}))
	defer server.Close()

	client := oauth.New("keystring-123", "https://example.com/callback")
	client.TokenURL = server.URL
	client.HTTPClient = server.Client()

	token, err := client.Exchange(context.Background(), "auth-code", verifier)
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if token.AccessToken != "1234.access" || token.RefreshToken != "1234.refresh" {
		t.Errorf("token = %+v", token)
	}
	if token.TokenType != "Bearer" || token.ExpiresIn != 3600 || token.Scope != "shops_r transactions_r" {
		t.Errorf("token = %+v", token)
	}
	if until := time.Until(token.ExpiresAt); until < 59*time.Minute || until > 61*time.Minute {
		t.Errorf("ExpiresAt is %v from now, want about 1 hour", until)
	}
	for key, want := range map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     "keystring-123",
		"redirect_uri":  "https://example.com/callback",
		"code":          "auth-code",
		"code_verifier": verifier,
	} {
		if gotForm.Get(key) != want {
			t.Errorf("form %s = %q, want %q", key, gotForm.Get(key), want)
		}
	}
}

func TestRefresh(t *testing.T) {
	t.Parallel()

	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"access_token": "new-access",
			"token_type": "Bearer",
			"expires_in": 3600,
			"refresh_token": "new-refresh"
		}`)
	}))
	defer server.Close()

	client := oauth.New("keystring-123", "https://example.com/callback")
	client.TokenURL = server.URL
	client.HTTPClient = server.Client()

	token, err := client.Refresh(context.Background(), "old-refresh")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if token.AccessToken != "new-access" || token.RefreshToken != "new-refresh" {
		t.Errorf("token = %+v", token)
	}
	if gotForm.Get("grant_type") != "refresh_token" {
		t.Errorf("grant_type = %q", gotForm.Get("grant_type"))
	}
	if gotForm.Get("client_id") != "keystring-123" {
		t.Errorf("client_id = %q", gotForm.Get("client_id"))
	}
	if gotForm.Get("refresh_token") != "old-refresh" {
		t.Errorf("refresh_token = %q", gotForm.Get("refresh_token"))
	}
}

func TestRefreshKeepsRefreshTokenWhenOmitted(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"access_token": "new-access",
			"token_type": "Bearer",
			"expires_in": 3600
		}`)
	}))
	defer server.Close()

	client := oauth.New("keystring-123", "https://example.com/callback")
	client.TokenURL = server.URL
	client.HTTPClient = server.Client()

	token, err := client.Refresh(context.Background(), "old-refresh")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if token.RefreshToken != "old-refresh" {
		t.Errorf("refresh token = %q, want old-refresh", token.RefreshToken)
	}
}

func TestAuthCodeURLUsesCustomAuthURL(t *testing.T) {
	t.Parallel()

	client := oauth.New("keystring-123", "https://example.com/callback")
	client.AuthURL = "https://auth.example.com/connect"

	got, err := client.AuthCodeURL("", strings.Repeat("a", 43), oauth.ScopeShopsRead)
	if err != nil {
		t.Fatalf("AuthCodeURL: %v", err)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	if u.Host != "auth.example.com" || u.Path != "/connect" {
		t.Errorf("URL = %s", got)
	}
	if u.Query().Get("state") != "" {
		t.Errorf("state = %q, want omitted", u.Query().Get("state"))
	}
}
