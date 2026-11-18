package main

import (
	"encoding/json/v2"
	"testing"

	"github.com/MarkRosemaker/openapi"
)

// TestTypeFreeForms gives a free-form object freeForms names its shape, a map or a struct, and fails where that object
// is typed upstream.
func TestTypeFreeForms(t *testing.T) {
	t.Parallel()

	load := func(storageURLs string) *openapi.Document {
		var doc openapi.Document
		if err := json.Unmarshal([]byte(`{
			"openapi": "3.1.0", "info": {"title": "t", "version": "1"},
			"components": {"schemas": {
				"GetTilesProResponse": {"type": "object", "properties": {
					"storage_urls": `+storageURLs+`,
					"tile_rules": {"anyOf": [{"type": "object", "additionalProperties": true}, {"type": "null"}]}
				}}
			}}
		}`), &doc); err != nil {
			t.Fatal(err)
		}

		return &doc
	}

	doc := load(`{"type": "object", "additionalProperties": true}`)
	if err := typeFreeForms(doc); err != nil {
		t.Fatal(err)
	}

	props := doc.Components.Schemas["GetTilesProResponse"].Properties

	if v := props["storage_urls"].AdditionalProperties.Schema; v == nil || v.Type != openapi.TypeString || v.Format != openapi.FormatURI {
		t.Errorf("storage_urls holds %+v, want URI strings", v)
	}

	rules := nonNull(props["tile_rules"])
	if rules.Properties["tiles"] == nil || rules.Properties["terrains"] == nil || rules.AdditionalProperties != nil {
		t.Errorf("tile_rules is %+v, want its documented properties", rules)
	}

	if len(props["tile_rules"].AnyOf) != 2 {
		t.Error("tile_rules may still be null")
	}

	if err := typeFreeForms(load(`{"type": "object", "additionalProperties": {"type": "string"}}`)); err == nil {
		t.Error("an object typed upstream is no longer free-form, and the entry should fail the run")
	}
}
