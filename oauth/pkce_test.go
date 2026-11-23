package oauth_test

import (
	"regexp"
	"testing"

	"github.com/rizquuula/etsy-openapi-client/oauth"
)

func TestCodeChallengeS256RFC7636(t *testing.T) {
	t.Parallel()

	const (
		verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		want     = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	)
	if got := oauth.CodeChallengeS256(verifier); got != want {
		t.Fatalf("challenge = %q, want %q", got, want)
	}
}

func TestGenerateCodeVerifier(t *testing.T) {
	t.Parallel()

	first, err := oauth.GenerateCodeVerifier()
	if err != nil {
		t.Fatalf("GenerateCodeVerifier: %v", err)
	}
	second, err := oauth.GenerateCodeVerifier()
	if err != nil {
		t.Fatalf("GenerateCodeVerifier: %v", err)
	}
	if first == second {
		t.Fatal("two generated verifiers are identical")
	}
	if len(first) != 43 {
		t.Errorf("verifier length = %d, want 43", len(first))
	}
	if !regexp.MustCompile(`^[A-Za-z0-9._~-]+$`).MatchString(first) {
		t.Errorf("verifier %q contains invalid characters", first)
	}
}
