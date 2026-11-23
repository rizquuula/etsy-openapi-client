SPEC_URL := https://www.etsy.com/openapi/generated/oas/3.0.0.json
GENERATOR := github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.9.0
GOLANGCI_LINT := github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0

.PHONY: generate update-spec build test vet lint fmt check

generate:
	go run $(GENERATOR) -config oapi-codegen.yaml api/openapi.json
	gofmt -w api.gen.go

update-spec:
	curl -fsSL $(SPEC_URL) -o api/openapi.json
	jq -S . api/openapi.json > api/openapi.json.tmp
	mv api/openapi.json.tmp api/openapi.json

build:
	go build ./...

test:
	go test -race ./...

vet:
	go vet ./...

lint:
	go run $(GOLANGCI_LINT) run ./...

fmt:
	gofmt -l -w .

check:
	test -z "$$(gofmt -l .)" || (echo "gofmt needs to run:"; gofmt -l .; exit 1)
	go vet ./...
	go test -race ./...
	$(MAKE) lint
