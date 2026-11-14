package main

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/MarkRosemaker/openapi"
)

// The recorded interactions call the API at its host, with the version in the path.
const (
	serverURL  = "https://api.pixellab.ai"
	pathPrefix = "/v2"
)

// transform makes the original fit for generating a library: every path carries the version the original leaves to
// its server.
func transform(doc *openapi.Document) error {
	doc.Info.Description = strings.TrimSpace(doc.Info.Description)
	doc.Servers = openapi.Servers{{URL: serverURL}}

	paths := make(openapi.Paths, len(doc.Paths))
	for path, item := range doc.Paths.ByIndex() {
		paths.Set(pathPrefix+path, item)
	}

	doc.Paths = paths

	if err := narrowRefs(doc); err != nil {
		return err
	}

	describeSizes(doc)

	dropEmptyJSON(doc)

	return detailEitherWay(doc)
}

// narrowRefs moves the properties upstream declares beside a reference, which may be null, into a copy of what it
// refers to, as an image size's bounds beside {"$ref": ".../ProFlashImageSize"} or beside {"anyOf": [{"$ref": ...},
// {"type": "null"}]}: they narrow what the reference allows, which a reference cannot say itself.
func narrowRefs(doc *openapi.Document) error {
	for name, s := range doc.Components.Schemas.ByIndex() {
		for prop, p := range s.Properties.ByIndex() {
			if len(p.Properties) == 0 {
				continue
			}

			ref := nonNull(p)
			if ref.Ref == nil {
				return fmt.Errorf("%s.%s: properties beside something other than a reference", name, prop)
			}

			narrowed := *ref.Ref.Value
			narrowed.Title, narrowed.Description = "", ""
			narrowed.Properties = maps.Clone(narrowed.Properties)

			for key, bounds := range p.Properties.ByIndex() {
				if narrowed.Properties[key] == nil {
					return fmt.Errorf("%s.%s: %s has no property %q", name, prop, ref.Ref.Identifier, key)
				}

				narrowed.Properties.Set(key, bounds)
			}

			if ref == p {
				narrowed.Description = p.Description
			} else {
				p.Properties, p.Required = nil, nil
			}

			ref.Replace(&narrowed)
		}
	}

	return nil
}

// detailEitherWay lets the detail of a 422 be a message as well as the validation errors upstream declares: the API
// answers a generation it turns down with {"detail": "Generation failed"}.
func detailEitherWay(doc *openapi.Document) error {
	s, ok := doc.Components.Schemas["HTTPValidationError"]
	if !ok || s.Properties["detail"] == nil || s.Properties["detail"].Type != openapi.TypeArray {
		return errors.New("HTTPValidationError has no array of details any more")
	}

	errs := *s.Properties["detail"]
	errs.Title = ""
	s.Properties["detail"].Replace(&openapi.Schema{
		Title: "Detail",
		OneOf: openapi.SchemaList{{Type: openapi.TypeString}, &errs},
	})

	return nil
}

// dropEmptyJSON removes the JSON of no schema FastAPI declares beside a response's own media type, as a ZIP download's,
// which would leave the client unable to read the download.
func dropEmptyJSON(doc *openapi.Document) {
	for _, item := range doc.Paths.ByIndex() {
		for _, op := range item.Operations {
			for _, rsp := range op.Responses.ByIndex() {
				if rsp.Value == nil || len(rsp.Value.Content) < 2 {
					continue
				}

				if mt, ok := rsp.Value.Content["application/json"]; ok && isEmptySchema(mt.Schema) {
					delete(rsp.Value.Content, "application/json")
				}
			}
		}
	}
}

// isEmptySchema reports whether s says nothing of a value: a schema of no keywords.
func isEmptySchema(s *openapi.Schema) bool {
	if s == nil {
		return true
	}

	b, err := json.Marshal(s)

	return err == nil && string(b) == "{}"
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
