//go:build ignore

// Package main holds code generation directives for this module.
package main

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.9.0 -config oapi-codegen.yaml api/openapi.json

func main() {}
