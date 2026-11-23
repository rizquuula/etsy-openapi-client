// Package etsy is a typed Go client for the Etsy Open API v3.
//
// The client is generated from the official OpenAPI 3.0.2 specification and
// covers every documented operation. Create a client with New, then call the
// generated *WithResponse methods:
//
//	client, err := etsy.New(etsy.Config{
//		Keystring:    "keystring",
//		SharedSecret: "shared-secret",
//		AccessToken:  "user-id.access-token",
//	})
//	if err != nil {
//		return err
//	}
//	resp, err := client.GetMeWithResponse(ctx)
//
// HTTP error statuses are returned in the typed response rather than as a Go
// error. Use ErrorFromResponse to turn them into an *APIError.
//
// OAuth 2.0 authorization-code (with PKCE), refresh, and automatic refresh are
// available in the separate oauth subpackage. An *oauth.RefreshingTokenSource
// satisfies TokenSource without importing this package.
package etsy
