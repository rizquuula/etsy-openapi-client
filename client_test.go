package etsy_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	etsy "github.com/rizquuula/etsy-openapi-client"
	"github.com/rizquuula/etsy-openapi-client/oauth"
)

// Compile-time proof of the public constructor and token source contracts.
var (
	_ func(etsy.Config) (*etsy.Client, error) = etsy.New
	_ etsy.TokenSource                        = (*oauth.RefreshingTokenSource)(nil)
)

type tokenSourceFunc func(context.Context) (string, error)

func (f tokenSourceFunc) AccessToken(ctx context.Context) (string, error) {
	return f(ctx)
}

func TestNewAddsCredentialsAndDecodesResponse(t *testing.T) {
	var gotAPIKey, gotAuthorization, gotUserAgent, gotPath, gotLanguage string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("x-api-key")
		gotAuthorization = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		gotPath = r.URL.Path
		gotLanguage = r.URL.Query().Get("language")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"listing_id":123,"title":"Handmade"}`)
	}))
	defer server.Close()

	client, err := etsy.New(etsy.Config{
		Keystring:    "key-123",
		SharedSecret: "secret-456",
		AccessToken:  "9876.user-token",
		BaseURL:      server.URL,
		HTTPClient:   server.Client(),
		UserAgent:    "custom-agent/1.0",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	language := "de"
	resp, err := client.GetListingWithResponse(context.Background(), 123, &etsy.GetListingParams{Language: &language})
	if err != nil {
		t.Fatalf("GetListingWithResponse: %v", err)
	}
	if gotAPIKey != "key-123:secret-456" {
		t.Errorf("x-api-key = %q", gotAPIKey)
	}
	if gotAuthorization != "Bearer 9876.user-token" {
		t.Errorf("Authorization = %q", gotAuthorization)
	}
	if gotUserAgent != "custom-agent/1.0" {
		t.Errorf("User-Agent = %q", gotUserAgent)
	}
	if gotPath != "/v3/application/listings/123" {
		t.Errorf("path = %q", gotPath)
	}
	if gotLanguage != "de" {
		t.Errorf("language = %q", gotLanguage)
	}
	if resp.JSON200 == nil {
		t.Fatal("JSON200 is nil")
	}
	if resp.JSON200.ListingId == nil || *resp.JSON200.ListingId != 123 {
		t.Errorf("listing ID = %v", resp.JSON200.ListingId)
	}
	if resp.JSON200.Title == nil || *resp.JSON200.Title != "Handmade" {
		t.Errorf("title = %v", resp.JSON200.Title)
	}
	if err := etsy.ErrorFromResponse(resp.HTTPResponse, resp.Body); err != nil {
		t.Errorf("ErrorFromResponse: %v", err)
	}
}

func TestNewWithoutToken(t *testing.T) {
	var gotAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"shop_id":7,"shop_name":"Test Shop"}`)
	}))
	defer server.Close()

	client, err := etsy.New(etsy.Config{
		Keystring:    "key",
		SharedSecret: "secret",
		BaseURL:      server.URL,
		HTTPClient:   server.Client(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	resp, err := client.GetShopWithResponse(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetShopWithResponse: %v", err)
	}
	if gotAuthorization != "" {
		t.Errorf("Authorization = %q, want empty", gotAuthorization)
	}
	if resp.JSON200 == nil || resp.JSON200.ShopName == nil || *resp.JSON200.ShopName != "Test Shop" {
		t.Errorf("shop = %+v", resp.JSON200)
	}
}

func TestNewUsesTokenSource(t *testing.T) {
	var gotAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"shop_id":7,"shop_name":"Test Shop"}`)
	}))
	defer server.Close()

	source := tokenSourceFunc(func(context.Context) (string, error) {
		return "source-token", nil
	})
	client, err := etsy.New(etsy.Config{
		Keystring:    "key",
		SharedSecret: "secret",
		TokenSource:  source,
		BaseURL:      server.URL,
		HTTPClient:   server.Client(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := client.GetShopWithResponse(context.Background(), 7); err != nil {
		t.Fatalf("GetShopWithResponse: %v", err)
	}
	if gotAuthorization != "Bearer source-token" {
		t.Errorf("Authorization = %q", gotAuthorization)
	}
}

func TestNewTokenSourceError(t *testing.T) {
	var called atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Store(true)
	}))
	defer server.Close()

	wantErr := errors.New("token unavailable")
	source := tokenSourceFunc(func(context.Context) (string, error) {
		return "", wantErr
	})
	client, err := etsy.New(etsy.Config{
		Keystring:    "key",
		SharedSecret: "secret",
		TokenSource:  source,
		BaseURL:      server.URL,
		HTTPClient:   server.Client(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := client.GetShopWithResponse(context.Background(), 7); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if called.Load() {
		t.Error("HTTP request was sent despite token source failure")
	}
}

func TestNewEmptyTokenSource(t *testing.T) {
	source := tokenSourceFunc(func(context.Context) (string, error) {
		return "", nil
	})
	client, err := etsy.New(etsy.Config{
		Keystring:    "key",
		SharedSecret: "secret",
		TokenSource:  source,
		BaseURL:      "http://127.0.0.1:1",
		HTTPClient:   http.DefaultClient,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := client.GetShopWithResponse(context.Background(), 7); !errors.Is(err, etsy.ErrNoToken) {
		t.Fatalf("error = %v, want ErrNoToken", err)
	}
}

func TestNewReturnsTypedAPIErrorHelper(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"error":"Listing not found"}`)
	}))
	defer server.Close()

	client, err := etsy.New(etsy.Config{
		Keystring:    "key",
		SharedSecret: "secret",
		BaseURL:      server.URL,
		HTTPClient:   server.Client(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	resp, err := client.GetListingWithResponse(context.Background(), 999, nil)
	if err != nil {
		t.Fatalf("GetListingWithResponse: %v", err)
	}
	apiErr := etsy.ErrorFromResponse(resp.HTTPResponse, resp.Body)
	var typed *etsy.APIError
	if !errors.As(apiErr, &typed) {
		t.Fatalf("error = %v, want *APIError", apiErr)
	}
	if typed.StatusCode != http.StatusNotFound || typed.Message != "Listing not found" {
		t.Errorf("APIError = %+v", typed)
	}
}
