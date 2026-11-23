package oauth_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rizquuula/etsy-openapi-client/oauth"
)

func TestTokenError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"invalid_grant","error_description":"The code has expired.","error_uri":"https://example.com/error"}`)
	}))
	defer server.Close()

	client := oauth.New("keystring-123", "https://example.com/callback")
	client.TokenURL = server.URL
	client.HTTPClient = server.Client()

	_, err := client.Exchange(context.Background(), "expired", strings.Repeat("a", 43))
	var tokenErr *oauth.TokenError
	if !errors.As(err, &tokenErr) {
		t.Fatalf("error = %v, want *TokenError", err)
	}
	if tokenErr.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d", tokenErr.StatusCode)
	}
	if tokenErr.Code != "invalid_grant" || tokenErr.Description != "The code has expired." {
		t.Errorf("error = %+v", tokenErr)
	}
	if tokenErr.Error() != "etsy oauth: invalid_grant: The code has expired." {
		t.Errorf("Error() = %q", tokenErr.Error())
	}
}
