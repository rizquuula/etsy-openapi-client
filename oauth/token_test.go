package oauth_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rizquuula/etsy-openapi-client/oauth"
)

func TestTokenJSONOmitsZeroExpiry(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(oauth.Token{AccessToken: "access"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(data), "expires_at") {
		t.Errorf("JSON contains zero expires_at: %s", data)
	}

	data, err = json.Marshal(oauth.Token{AccessToken: "access", ExpiresAt: time.Now()})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(data), "expires_at") {
		t.Errorf("JSON does not contain expires_at: %s", data)
	}
}
