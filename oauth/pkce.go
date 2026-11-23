package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// GenerateCodeVerifier returns a PKCE code verifier from 32 bytes of entropy.
func GenerateCodeVerifier() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("etsy: generate PKCE verifier: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// CodeChallengeS256 returns the S256 PKCE challenge for a code verifier.
func CodeChallengeS256(codeVerifier string) string {
	sum := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func validateCodeVerifier(codeVerifier string) error {
	if n := len(codeVerifier); n < 43 || n > 128 {
		return fmt.Errorf("etsy: PKCE verifier length is %d; want 43 to 128", n)
	}
	for _, r := range codeVerifier {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-', r == '.', r == '_', r == '~':
		default:
			return fmt.Errorf("etsy: PKCE verifier contains invalid character %q", r)
		}
	}
	return nil
}
