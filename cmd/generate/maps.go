package main

import (
	"fmt"
	"maps"
	"slices"

	"github.com/MarkRosemaker/openapi"
)

// mapValues gives the objects upstream declares free-form (additionalProperties: true) the values the API holds them
// to, by "Schema.property" of the original, so that the library has a map where it would otherwise have an empty
// struct that drops every key. An entry needs evidence of the values, which its comment gives. An entry that matches
// nothing, or no free-form object, fails the run.
var mapValues = map[string]*openapi.Schema{
	// "Download URLs for each tile (e.g. {"tile_0": "https://...", "tile_1": "https://..."})"
	"GetTilesProResponse.storage_urls": {Type: openapi.TypeString, Format: openapi.FormatURI},
}

// typeMaps gives each object mapValues names the values it holds.
func typeMaps(doc *openapi.Document) error {
	unused := maps.Clone(mapValues)

	for name, s := range doc.Components.Schemas.ByIndex() {
		for prop, p := range s.Properties.ByIndex() {
			key := name + "." + prop

			v, ok := mapValues[key]
			if !ok {
				continue
			}

			delete(unused, key)

			o := nonNull(p)
			if o.Type != openapi.TypeObject || len(o.Properties) > 0 || o.AdditionalProperties == nil ||
				o.AdditionalProperties.Schema != nil || !o.AdditionalProperties.Allowed {
				return fmt.Errorf("%s: not a free-form object any more", key)
			}

			o.AdditionalProperties = &openapi.AdditionalProperties{Schema: v}
		}
	}

	if len(unused) > 0 {
		return fmt.Errorf("mapValues names what is not there any more: %v", slices.Sorted(maps.Keys(unused)))
	}

	return nil
}
