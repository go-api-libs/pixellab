package main

import (
	"fmt"
	"maps"
	"slices"

	"github.com/MarkRosemaker/openapi"
)

// freeForms gives the objects upstream declares free-form (additionalProperties: true) the shape the API holds them
// to, by "Schema.property" of the original, so that the library has a map or a struct where it would otherwise have
// an empty struct that drops every key. An entry needs evidence of the shape, which its comment gives. An entry that
// matches nothing, or no free-form object, fails the run.
var freeForms = map[string]func() *openapi.Schema{
	// "Download URLs for each tile (e.g. {"tile_0": "https://...", "tile_1": "https://..."})"
	"GetTilesProResponse.storage_urls": func() *openapi.Schema {
		return mapOf(&openapi.Schema{Type: openapi.TypeString, Format: openapi.FormatURI})
	},
	// "rule_type edge|corner|outline, arity 4 (square/iso edges or corners) or 6 (hex edges), connectivity
	// same|other, terrains [feature, background], and tiles keyed like storage_urls with each tile's bitmask", which a
	// recorded tile set holds in an object of its own, as {"tile_2": {"mask": 5}}
	"GetTilesProResponse.tile_rules": func() *openapi.Schema {
		s := &openapi.Schema{Type: openapi.TypeObject}
		s.Properties.Set("rule_type", &openapi.Schema{
			Type: openapi.TypeString, Description: "How tiles connect: by their edges, their corners or an outline.",
		})
		s.Properties.Set("arity", &openapi.Schema{
			Type: openapi.TypeInteger, Description: "How many edges or corners a tile connects by: 4, or 6 for hex edges.",
		})
		s.Properties.Set("connectivity", &openapi.Schema{
			Type: openapi.TypeString, Description: "What a set bit connects to: the same terrain, or the other.",
		})
		s.Properties.Set("terrains", &openapi.Schema{
			Type: openapi.TypeArray, Items: &openapi.Schema{Type: openapi.TypeString},
			Description: "The feature terrain, then the background.",
		})
		tile := &openapi.Schema{Type: openapi.TypeObject, Required: []string{"mask"}}
		tile.Properties.Set("mask", &openapi.Schema{
			Type: openapi.TypeInteger,
			Description: "The tile's bitmask. Edge/4: bit0=N bit1=E bit2=S bit3=W; edge/6: hex edges CCW from vertex 0; " +
				"corner/4: NW<<3|NE<<2|SW<<1|SE.",
		})

		tiles := mapOf(tile)
		tiles.Description = "Each tile's placement, keyed like storage_urls. A tile absent is stamp-only."
		s.Properties.Set("tiles", tiles)

		return s
	},
}

// mapOf is an object of any keys, each holding a value of a schema.
func mapOf(values *openapi.Schema) *openapi.Schema {
	return &openapi.Schema{Type: openapi.TypeObject, AdditionalProperties: &openapi.AdditionalProperties{Schema: values}}
}

// typeFreeForms gives each object freeForms names its shape, keeping what upstream says of it besides: its title,
// its description, and that it may be null.
func typeFreeForms(doc *openapi.Document) error {
	unused := maps.Clone(freeForms)

	for name, s := range doc.Components.Schemas.ByIndex() {
		for prop, p := range s.Properties.ByIndex() {
			key := name + "." + prop

			shape, ok := freeForms[key]
			if !ok {
				continue
			}

			delete(unused, key)

			o := nonNull(p)
			if o.Type != openapi.TypeObject || len(o.Properties) > 0 || o.AdditionalProperties == nil ||
				o.AdditionalProperties.Schema != nil || !o.AdditionalProperties.Allowed {
				return fmt.Errorf("%s: not a free-form object any more", key)
			}

			typed := shape()
			o.Properties, o.AdditionalProperties = typed.Properties, typed.AdditionalProperties
		}
	}

	if len(unused) > 0 {
		return fmt.Errorf("freeForms names what is not there any more: %v", slices.Sorted(maps.Keys(unused)))
	}

	return nil
}
