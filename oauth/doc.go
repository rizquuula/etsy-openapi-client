// Package oauth implements Etsy's OAuth 2.0 authorization-code flow with PKCE
// and automatic token refresh.
//
// The package is independent of the API client package. A
// *RefreshingTokenSource satisfies the API client's TokenSource interface
// structurally through its AccessToken method.
package oauth
