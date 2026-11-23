package oauth_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rizquuula/etsy-openapi-client/oauth"
)

func TestRefreshingTokenSourceUsesValidToken(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"unexpected","expires_in":3600}`)
	}))
	defer server.Close()

	auth := oauth.New("key", "https://example.com/callback")
	auth.TokenURL = server.URL
	auth.HTTPClient = server.Client()
	source, err := oauth.NewRefreshingTokenSource(auth, &oauth.Token{
		AccessToken:  "valid-access",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("NewRefreshingTokenSource: %v", err)
	}

	token, err := source.Token(context.Background())
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if token.AccessToken != "valid-access" {
		t.Errorf("access token = %q", token.AccessToken)
	}
	if got := requests.Load(); got != 0 {
		t.Errorf("refresh requests = %d, want 0", got)
	}
}

func TestRefreshingTokenSourceRefreshesExpiredToken(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"access_token": "new-access",
			"token_type": "Bearer",
			"expires_in": 3600,
			"refresh_token": "new-refresh"
		}`)
	}))
	defer server.Close()

	auth := oauth.New("key", "https://example.com/callback")
	auth.TokenURL = server.URL
	auth.HTTPClient = server.Client()
	source, err := oauth.NewRefreshingTokenSource(auth, &oauth.Token{
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("NewRefreshingTokenSource: %v", err)
	}

	var persisted atomic.Int32
	source.OnRefresh = func(ctx context.Context, token *oauth.Token) error {
		persisted.Add(1)
		if token.AccessToken != "new-access" || token.RefreshToken != "new-refresh" {
			return fmt.Errorf("unexpected token: %+v", token)
		}
		return nil
	}

	token, err := source.Token(context.Background())
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if token.AccessToken != "new-access" {
		t.Errorf("access token = %q", token.AccessToken)
	}
	if got := requests.Load(); got != 1 {
		t.Errorf("refresh requests = %d, want 1", got)
	}
	if got := persisted.Load(); got != 1 {
		t.Errorf("persist callbacks = %d, want 1", got)
	}

	// The refreshed token is valid, so the second call uses the cache.
	if _, err := source.Token(context.Background()); err != nil {
		t.Fatalf("second Token: %v", err)
	}
	if got := requests.Load(); got != 1 {
		t.Errorf("refresh requests after cache hit = %d, want 1", got)
	}
}

func TestRefreshingTokenSourceAccessToken(t *testing.T) {
	t.Parallel()

	auth := oauth.New("key", "https://example.com/callback")
	source, err := oauth.NewRefreshingTokenSource(auth, &oauth.Token{AccessToken: "access"})
	if err != nil {
		t.Fatalf("NewRefreshingTokenSource: %v", err)
	}
	token, err := source.AccessToken(context.Background())
	if err != nil {
		t.Fatalf("AccessToken: %v", err)
	}
	if token != "access" {
		t.Errorf("access token = %q", token)
	}
}

func TestRefreshingTokenSourceConcurrentRefresh(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		time.Sleep(20 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"access_token": "new-access",
			"token_type": "Bearer",
			"expires_in": 3600,
			"refresh_token": "new-refresh"
		}`)
	}))
	defer server.Close()

	auth := oauth.New("key", "https://example.com/callback")
	auth.TokenURL = server.URL
	auth.HTTPClient = server.Client()
	source, err := oauth.NewRefreshingTokenSource(auth, &oauth.Token{
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("NewRefreshingTokenSource: %v", err)
	}

	const goroutines = 20
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := source.Token(context.Background())
			if err != nil {
				errs <- err
				return
			}
			if token.AccessToken != "new-access" {
				errs <- fmt.Errorf("access token = %q", token.AccessToken)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	if got := requests.Load(); got != 1 {
		t.Errorf("refresh requests = %d, want 1", got)
	}
}

func TestRefreshingTokenSourceWithoutRefreshToken(t *testing.T) {
	t.Parallel()

	auth := oauth.New("key", "https://example.com/callback")
	source, err := oauth.NewRefreshingTokenSource(auth, &oauth.Token{
		AccessToken: "expired",
		ExpiresAt:   time.Now().Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("NewRefreshingTokenSource: %v", err)
	}
	if _, err := source.Token(context.Background()); !errors.Is(err, oauth.ErrNoRefreshToken) {
		t.Errorf("error = %v, want ErrNoRefreshToken", err)
	}
}

func TestNewRefreshingTokenSourceValidation(t *testing.T) {
	t.Parallel()

	if _, err := oauth.NewRefreshingTokenSource(nil, &oauth.Token{AccessToken: "a"}); err == nil {
		t.Error("nil auth: expected error")
	}
	if _, err := oauth.NewRefreshingTokenSource(oauth.New("key", "uri"), nil); !errors.Is(err, oauth.ErrNoToken) {
		t.Errorf("nil token error = %v, want ErrNoToken", err)
	}
	if _, err := oauth.NewRefreshingTokenSource(oauth.New("key", "uri"), &oauth.Token{}); !errors.Is(err, oauth.ErrNoToken) {
		t.Errorf("empty token error = %v, want ErrNoToken", err)
	}
}

func TestRefreshingTokenSourceSetToken(t *testing.T) {
	t.Parallel()

	auth := oauth.New("key", "https://example.com/callback")
	source, err := oauth.NewRefreshingTokenSource(auth, &oauth.Token{AccessToken: "first"})
	if err != nil {
		t.Fatalf("NewRefreshingTokenSource: %v", err)
	}
	source.SetToken(&oauth.Token{AccessToken: "second"})
	token, err := source.Token(context.Background())
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if token.AccessToken != "second" {
		t.Errorf("access token = %q, want second", token.AccessToken)
	}
}
