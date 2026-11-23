package oauth

import "time"

// Token is an OAuth 2.0 token returned by Etsy.
type Token struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	// ExpiresAt is set by this client. It is persisted so a stored token keeps
	// its expiry across process restarts.
	ExpiresAt time.Time `json:"expires_at,omitzero"`
}
