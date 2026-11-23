package oauth

import (
	"encoding/json"
	"fmt"
)

// TokenError is a non-2xx response from the OAuth token endpoint.
type TokenError struct {
	StatusCode  int
	Code        string `json:"error"`
	Description string `json:"error_description"`
	URI         string `json:"error_uri"`
	Body        []byte `json:"-"`
}

func newTokenError(statusCode int, body []byte) *TokenError {
	err := &TokenError{StatusCode: statusCode, Body: body}
	_ = json.Unmarshal(body, err)
	return err
}

// Error implements error.
func (e *TokenError) Error() string {
	code := e.Code
	if code == "" {
		code = fmt.Sprintf("HTTP %d", e.StatusCode)
	}
	if e.Description != "" {
		return fmt.Sprintf("etsy oauth: %s: %s", code, e.Description)
	}
	return "etsy oauth: " + code
}
