// Package generate holds only go:generate directives - there's no library
// code at module root, so this isn't package main.
package generate

//TODO legacy, re-enable later: go:generate go run ./cmd/generate
//go:generate go tool openapi-enrich
//go:generate go tool openapi-flatten
//go:generate go tool openapi-compress
//go:generate go tool openapi-codegen -client -debug
