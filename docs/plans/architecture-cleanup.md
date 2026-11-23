# Plan: Etsy Go client architecture cleanup

## 1. Goal and success criteria

**Goal.** Restructure the working client into an idiomatic Go library: a small root API
package with a hand-written `Client` facade, an independent `oauth` subpackage, the
generated contract at the root, consumer-defined token access, and enforceable lint
budgets. Authentication behavior and all 107 operations stay the same.

**Success criteria**

- `etsy.New` returns `*etsy.Client`; `etsy.Client` promotes every generated
  `*WithResponse` method. The coverage test reflects on `*etsy.Client` and logs
  **107 operations**.
- Root package has **≤ 8 production `.go` files**; `oauth` has **≤ 8**. Both
  are within the skill's ≤ 10 target. Every hand-written exported type lives in
  a file named after it.
- `etsy` does **not** import `oauth`; `oauth` does **not** import `etsy`.
  Verified by `go list -deps ./...`.
- `go vet`, `go test -race ./...`, and `golangci-lint run` pass. `api.gen.go` is
  excluded from lint. Verified by `make check`.
- `make generate` reproduces `api.gen.go` byte-for-byte from `api/openapi.json`.
- `go build ./...` compiles the example and README snippets use the new API.
- No live Etsy calls; no credentials used.

## 2. Non-goals

- No behavior change: same endpoints, headers, OAuth grants, refresh leeway, and
  error parsing.
- No generated code behind `internal/gen`: it would force 190+ aliases or leak
  internal types through the public API.
- No hexagonal/domain layers. This is a library, not a service.
- No retry/backoff, multipart upload helpers, webhook verification, or CLI.
- No live smoke test.

## 3. Context and assumptions

- Root today: `etsy.gen.go` (36,712 lines), `client.go` (Config + New),
  `oauth.go` (418 lines, 7 exported types), `errors.go` (2 exported types),
  `doc.go`.
- `New` returns the generated `*ClientWithResponses`; the generated low-level
  `Client` name blocks a clean facade.
- `oapi-codegen` supports `output-options.client-type-name`. A spike with
  `RequestClient` renamed the raw client and left `NewClient`,
  `ClientOption`, and `ClientWithResponses` intact.
- The spec currently lives in `spec/openapi.json`; the skill recommends `api/`
  for contract artifacts.
- 29 tests currently live in `package etsy`; the skill recommends external
  `package etsy_test` for API-surface tests.
- No `.golangci.yml` or `make lint` exists. `golangci-lint` is not installed;
  pin `v2.15.0` and run it through `go run` (assumption: current Go 1.26 support).
- Go 1.26.6, `oapi-codegen/runtime v1.9.1`, `oapi-codegen v2.9.0`.
- The `.env` file in the workspace was not read and is not part of this plan.

## 4. Unknowns

- **Blocking:** none.
- **Non-blocking:** exact `.golangci.yml` v2 exclusion syntax. Resolve with
  `golangci-lint config verify` and an explicit `api.gen.go` path.
- **Decision flagged:** OAuth flow type naming. Chosen: `oauth.Client` +
  `oauth.New`, not `oauth.OAuth` (avoids stutter). Revise before step 3 if needed.
- **Decision flagged:** public API break is accepted because the module is
  pre-1.0 and has no release.

## 5. Approach and alternatives

**Chosen approach.** Keep a flat root library. Rename the generated low-level
client to `RequestClient`, add a hand-written `etsy.Client` facade that embeds
`*ClientWithResponses`, and move OAuth into its own `oauth` package. Root defines
the consumer-side `TokenSource` interface it needs; `oauth.RefreshingTokenSource`
satisfies it structurally, so neither package imports the other. Split the
remaining hand-written types one per file, move the spec to `api/`, and add lint
budgets.

**Rejected alternatives**

- *Keep everything in root, only split files*: simpler, but `oauth.go` and
  `errors.go` stay multi-type, and the public constructor keeps returning a
  generated type.
- *Generated code in `internal/gen`*: cannot expose internal types safely; would
  require aliases for 83 models and 107 response types.
- *Generated code in a public subpackage*: leaks `subpackage.Response` types
  through the facade and duplicates the model surface.
- *Hexagonal architecture*: a library has no domain/use-case/adapter split.

## 6. Files to change

```
api/openapi.json                         [MOVE]  from spec/openapi.json
api/README.md                            [MOVE]  from spec/README.md
api.gen.go                               [RENAME] generated from etsy.gen.go
oapi-codegen.yaml                        [EDIT]   output api.gen.go, client-type-name
generate.go                              [EDIT]   api/openapi.json path
Makefile                                 [EDIT]   generate, lint, check, build
.golangci.yml                            [NEW]    baseline + size/complexity budgets
doc.go                                   [EDIT]   new package architecture
client.go                                [EDIT]   Client facade, New, request editor
config.go                                [NEW]    Config + HTTPDoer validation
token_source.go                          [NEW]    consumer TokenSource + static adapter
errors.go                                [EDIT]   ErrNoToken only
api_error.go                             [NEW]    APIError + ErrorFromResponse
rate_limit.go                            [NEW]    RateLimit + ParseRateLimit
client_test.go, config_test.go           [EDIT/NEW] external API tests
api_error_test.go, rate_limit_test.go    [NEW]    external error tests
coverage_test.go                         [EDIT]   reflect on *etsy.Client; api/ path
oauth/                                   [NEW]    independent auth package
  doc.go, client.go, pkce.go, scope.go,
  token.go, refreshing_token_source.go,
  token_error.go, errors.go
  client_test.go, pkce_test.go,
  refreshing_token_source_test.go,
  token_error_test.go
oauth.go                                 [DELETE] replaced by oauth/
examples/quickstart/main.go              [EDIT]   new API
README.md                                [EDIT]   architecture, imports, regeneration
```

## 7. Flow after the change

```
  app
   |
   | etsy.New(Config{..., TokenSource: source})          *
   v
  etsy.Client *                                          [hand-written facade]
   |  embeds
   v
  *ClientWithResponses  ->  ClientInterface  ->  RequestClient *  [generated]
   |                                               |
   | request editor *                              | HTTPS
   v                                               v
  etsy.TokenSource.AccessToken(ctx) *            Etsy API
   ^
   | structural satisfaction (no import)
  *oauth.RefreshingTokenSource *  ->  *oauth.Client *  ->  Etsy token endpoint
```

Dependency direction:

```
examples --> etsy
examples --> oauth
etsy     -X-> oauth     consumer defines TokenSource
oauth    -X-> etsy      no dependency
```

`*` = new or materially changed by this plan.

## 8. Steps

| # | Red: failing test | Green: implementation | Done when |
|---|---|---|---|
| 1 | `client_test.go` references `*etsy.Client` and a promoted `GetShopWithResponse`; compile fails | `client.go` facade; `config.go`; `oapi-codegen.yaml` `client-type-name: RequestClient`; regenerate `api.gen.go`; `New` returns `*Client` | `go test ./...` passes; coverage logs 107 on `*etsy.Client` |
| 2 | `config_test.go` + custom `etsy.TokenSource` using `AccessToken` fail | `token_source.go`; `errors.go`; move Config validation to `config.go`; update editor | config/token tests pass; root has no `oauth` import |
| 3 | `oauth/*_test.go` external tests fail to compile | create `oauth/` package (`Client`, `Token`, `RefreshingTokenSource`, `TokenError`, PKCE, scopes); delete `oauth.go`; adapt root tests | `go test ./...`; no import cycle; `var _ etsy.TokenSource = (*oauth.RefreshingTokenSource)(nil)` compiles in a test |
| 4 | adapted `api_error_test.go` and `rate_limit_test.go` fail | split `errors.go` into `api_error.go`, `rate_limit.go`, `errors.go` | error tests pass; file budgets hold |
| 5 | `[NO-TEST]` tooling only | `.golangci.yml`; `Makefile` lint/check targets; pin linter | `make lint` clean with `api.gen.go` excluded |
| 6 | `[NO-TEST]` docs/example only | `doc.go`, `README.md`, `examples/quickstart/main.go` | `go build ./...` passes |
| 7 | final verification | regenerate, run all gates, count files/ops | checksums, test output, lint output recorded |

## 9. Parallel slices

| Slice | Owns | Runs after | Can run with |
|---|---|---|---|
| A: facade + config + generator | `api/`, `api.gen.go`, `client.go`, `config.go`, `token_source.go`, `oapi-codegen.yaml`, `generate.go` | — | — |
| B: OAuth extraction | `oauth/`, delete `oauth.go`, OAuth test adaptation | A | C |
| C: error split | `api_error.go`, `rate_limit.go`, `errors.go`, error tests | A | B |
| D: lint + Makefile | `.golangci.yml`, `Makefile` | B, C | — |
| E: docs + example | `doc.go`, `README.md`, `examples/` | A, B, C | — |

Serial fallback: if slices B and C collide on `go.mod`/test helpers, run A → B → C → D → E.

## 10. Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Public API break (`New` return, OAuth move) | High, intentional | Pre-1.0; README migration section; examples/tests compile against the new API |
| Generated rename changes more than expected | Low | Spike verified `RequestClient`; regenerate and run 107-op coverage test |
| OAuth import cycle | Low | Consumer-defined `etsy.TokenSource`; `oauth` imports stdlib only; `go list -deps` check |
| Lint config v2 syntax mismatch | Medium | Pin `golangci-lint v2.15.0`; `config verify`; iterate |
| Generated file trips size linters | High | Auto-generated header + explicit path exclusion; verify lint output |
| External tests cannot reach unexported helpers | Low | Test through public API; keep internal tests only where truly needed |
| `.env` secrets exposed | Low | Do not read, do not run live calls |

## 11. Validation

- **11a Unit:** adapted 29 tests + new config/token-source tests, external package.
- **11b Integration:** `httptest` OAuth exchange/refresh and API header/decode flows.
- **11c Visual:** not applicable (library). Inspect `go doc` for `etsy.Client`,
  `etsy.Config`, `etsy.TokenSource`, `oauth.Client`, `oauth.RefreshingTokenSource`.
- **11d Metrics/logs:** operation coverage log (107), lint output, file counts,
  dependency graph, generation checksum.

## 12. Rollout and reversibility

No release and no migration. The module is pre-1.0; the API break is documented in
the README. Revert by restoring the previous tree. No persisted data, no store, no
external state.

## 13. Follow-ups

- Typed multipart upload helpers.
- Optional retry/backoff for 429.
- Webhook signature verification.
- Live smoke test once credentials are available.
- CI workflow running `make check` and `govulncheck`.

## 14. Outcome

- Root and `oauth` each have 8 production `.go` files, within the ≤10 target.
- `etsy.New` returns `*etsy.Client`; the reflection coverage test sees all
  **107** operations through the facade.
- Root does not import `oauth`; `oauth` does not import root.
- 38 tests pass with `go test -count=1 -race ./...`.
- `go vet ./...` and `golangci-lint v2.15.0` report 0 issues.
- `make generate` reproduces `api.gen.go` byte-for-byte.
- No live Etsy API calls were made.
