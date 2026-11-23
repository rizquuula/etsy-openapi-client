# Etsy Open API v3 client for Go

A typed Go client for the [Etsy Open API v3](https://developers.etsy.com/documentation/),
generated from the official OpenAPI 3.0.2 specification.

- All **107** operations and **83** schemas from the official spec.
- Typed `*WithResponse` methods with `JSON200`, `JSON400`, and other status fields.
- API-key authentication (`x-api-key: keystring:shared_secret`).
- OAuth 2.0 authorization-code flow with PKCE (S256).
- Automatic access-token refresh with a persistence callback.
- Typed `APIError` with Etsy rate-limit and `Retry-After` headers.

The vendored spec is [`api/openapi.json`](api/openapi.json), downloaded from
`https://www.etsy.com/openapi/generated/oas/3.0.0.json`.

## What it can do

**Etsy API coverage — 107 operations:**

- **Shops** — find shops, fetch a shop or a shop by owner, update shop details,
  and manage sections, production partners, return policies, holiday
  preferences, economic operators, and processing/readiness profiles.
- **Listings** — create draft listings; read, update, and delete listings;
  find active and featured listings; batch-fetch listings; filter by shop,
  section, receipt, or return policy; and manage properties, personalization,
  translations, images, videos, files, inventory, products, and variation
  images.
- **Orders and payments** — read shop receipts, update receipts, create
  shipments, read receipt transactions, payments, and payment-account ledger
  entries.
- **Users** — get the authenticated user, fetch user profiles, and read user
  addresses.
- **Taxonomy** — browse seller and buyer taxonomy nodes and their properties.
- **Reviews** — read reviews by listing and by shop.
- **Utility** — ping the API and retrieve token scopes.

**Client capabilities:**

- Every operation is a typed `*WithResponse` method with `JSON200`, `JSON400`,
  and other status fields.
- API-key authentication for every request, using
  `x-api-key: keystring:shared_secret`.
- OAuth 2.0 authorization-code flow with PKCE (S256), in the independent
  `oauth` package.
- Automatic access-token refresh one minute before expiry, with a persistence
  callback.
- Consumer-defined token access: `etsy.TokenSource` needs only
  `AccessToken(ctx) (string, error)`.
- Typed `APIError` with status code, Etsy error message, rate-limit headers,
  `Retry-After`, raw body, and headers.
- 83 generated request/response models, regenerable from the vendored spec.

## Requirements

- Go 1.26 or newer.

## Install

```sh
go get github.com/rizquuula/etsy-openapi-client
```

## Architecture

```
etsy (root)  API client, generated contract, APIError, TokenSource
oauth/       OAuth flow, PKCE, token refresh, TokenError
```

The root package and `oauth` do not import each other in production code.
`etsy.TokenSource` is consumer-defined, and `*oauth.RefreshingTokenSource`
satisfies it structurally through its `AccessToken` method.

The generated client is [`api.gen.go`](api.gen.go); do not edit it by hand.

## Public endpoints

Every request needs the app keystring and shared secret, even public endpoints.
An OAuth token is optional for public read endpoints.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	etsy "github.com/rizquuula/etsy-openapi-client"
)

func main() {
	client, err := etsy.New(etsy.Config{
		Keystring:    os.Getenv("ETSY_KEYSTRING"),
		SharedSecret: os.Getenv("ETSY_SHARED_SECRET"),
	})
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.GetShopWithResponse(context.Background(), 12345678)
	if err != nil {
		log.Fatal(err) // transport or response-decoding error
	}
	if err := etsy.ErrorFromResponse(resp.HTTPResponse, resp.Body); err != nil {
		log.Fatal(err) // HTTP 4xx/5xx
	}
	if resp.JSON200 == nil || resp.JSON200.ShopName == nil {
		log.Fatal("shop returned no name")
	}
	fmt.Println(*resp.JSON200.ShopName)
}
```

## OAuth 2.0

Etsy requires PKCE on every authorization-code flow. Keep the verifier and state
with the user's session between the redirect and the callback.

```go
import (
	"context"

	etsy "github.com/rizquuula/etsy-openapi-client"
	"github.com/rizquuula/etsy-openapi-client/oauth"
)

auth := oauth.New(keystring, "https://example.com/oauth/callback")

verifier, err := oauth.GenerateCodeVerifier()
if err != nil {
	return err
}
state := "a-single-use-random-value"

authURL, err := auth.AuthCodeURL(
	state,
	verifier,
	oauth.ScopeShopsRead,
	oauth.ScopeTransactionsRead,
)
if err != nil {
	return err
}
// Redirect the user to authURL. On the callback, verify state and then:
token, err := auth.Exchange(ctx, callbackCode, verifier)
if err != nil {
	return err
}
```

Use the token for authenticated calls:

```go
client, err := etsy.New(etsy.Config{
	Keystring:    keystring,
	SharedSecret: sharedSecret,
	AccessToken:  token.AccessToken,
})
```

### Automatic refresh

Etsy access tokens live for one hour and refresh tokens for 90 days. A
`RefreshingTokenSource` refreshes one minute before expiry and rotates the
refresh token when Etsy returns a new one.

```go
source, err := oauth.NewRefreshingTokenSource(auth, token)
if err != nil {
	return err
}
source.OnRefresh = func(ctx context.Context, token *oauth.Token) error {
	return saveToken(ctx, token) // persist access_token, refresh_token, expires_at
}

client, err := etsy.New(etsy.Config{
	Keystring:    keystring,
	SharedSecret: sharedSecret,
	TokenSource:  source, // satisfies etsy.TokenSource
})
```

`oauth.Token` includes `ExpiresAt` so a stored token keeps its expiry across
restarts. If a token is expired and has no refresh token, the source returns
`oauth.ErrNoRefreshToken`.

## Errors and rate limits

Generated methods return a Go error only for transport or decoding failures.
HTTP error statuses are exposed through the typed response and
`ErrorFromResponse`:

```go
resp, err := client.GetListingWithResponse(ctx, listingID, nil)
if err != nil {
	return err
}
if err := etsy.ErrorFromResponse(resp.HTTPResponse, resp.Body); err != nil {
	var apiErr *etsy.APIError
	if errors.As(err, &apiErr) {
		log.Printf("status=%d message=%q retry_after=%s",
			apiErr.StatusCode, apiErr.Message, apiErr.RetryAfter)
		log.Printf("limit/day=%d remaining=%d",
			apiErr.RateLimit.LimitPerDay, apiErr.RateLimit.RemainingToday)
	}
	return err
}
```

`APIError.IsRateLimited()` reports HTTP 429. Etsy recommends exponential
backoff on 429; this client does not retry automatically. Token endpoint
failures are `*oauth.TokenError`.

## Uploads

The spec declares multipart uploads as raw bodies, so use the generated
`*WithBodyWithResponse` methods:

```go
var body bytes.Buffer
writer := multipart.NewWriter(&body)
file, err := writer.CreateFormFile("image", "photo.jpg")
// copy the image into file, then close the writer

resp, err := client.UploadListingImageWithBodyWithResponse(
	ctx, shopID, listingID, writer.FormDataContentType(), &body,
)
```

## Regenerating and linting

```sh
make update-spec   # download the latest official spec into api/
make generate      # run the pinned oapi-codegen version
make check         # gofmt, go vet, go test -race, golangci-lint
```

Generation uses `oapi-codegen v2.9.0`. `jq` is required by `make update-spec`.
`make lint` runs `golangci-lint v2.15.0` through `go run`; generated code is
excluded from size and complexity linters.

## Verification

```sh
go test -race ./...
```

The tests cover PKCE against the RFC 7636 vector, authorization URLs, token
exchange and refresh against `httptest` servers, concurrent refresh, API-key
and bearer headers, typed response decoding, operation coverage, and API
errors. No live Etsy API calls are made, and no live credentials are required.

## Stability

The module is pre-1.0. The architecture and public API may change between
minor versions.

## License

No license has been selected for this repository yet.
