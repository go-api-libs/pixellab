package main

import (
	"strings"

	"github.com/MarkRosemaker/openapi"
	edit "github.com/MarkRosemaker/openapi-edit"
)

// The recorded interactions call the API at its host, with the version in the path.
const (
	serverURL  = "https://api.pixellab.ai"
	pathPrefix = "/v2"
)

// transform makes the original fit for generating a library: every path carries the version the original leaves to
// its server, and an optional value is one that may be left out rather than one that may also be null.
func transform(doc *openapi.Document) {
	doc.Info.Description = strings.TrimSpace(doc.Info.Description)
	doc.Servers = openapi.Servers{{URL: serverURL}}

	paths := make(openapi.Paths, len(doc.Paths))
	for path, item := range doc.Paths.ByIndex() {
		paths.Set(pathPrefix+path, item)
	}

	doc.Paths = paths

	edit.WalkSchemas(doc, dropNull)
}

// dropNull removes null from an anyOf, as FastAPI writes every optional value, and the anyOf with it where one
// alternative is left.
func dropNull(s *openapi.Schema) {
	alts := s.AnyOf[:0]
	for _, alt := range s.AnyOf {
		if alt.Type != openapi.TypeNull {
			alts = append(alts, alt)
		}
	}

	if len(alts) == len(s.AnyOf) {
		return
	}

	if s.AnyOf = alts; len(alts) != 1 {
		return
	}

	only := alts[0]
	if only.Ref != nil {
		if only.Ref.Value.Example == nil {
			only.Ref.Value.Example = s.Example
		}

		s.Replace(&openapi.Schema{Ref: only.Ref, Title: s.Title, Description: s.Description, Default: s.Default})

		return
	}

	title, description, def := s.Title, s.Description, s.Default
	s.Replace(only)
	s.Title, s.Description = title, description

	if len(def) > 0 {
		s.Default = def
	}
}
