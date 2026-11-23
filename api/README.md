# OpenAPI specification

`openapi.json` is the official Etsy Open API v3 specification, downloaded from:

<https://www.etsy.com/openapi/generated/oas/3.0.0.json>

It is checked in so generation is deterministic and does not depend on the network.
Run `make update-spec` to refresh it, then `make generate` to regenerate
`api.gen.go`.
