package main

import (
	"encoding/json/v2"
	"testing"

	"github.com/MarkRosemaker/openapi"
)

// TestTypeMaps gives a free-form object mapValues names its values, and fails where that object is typed upstream.
func TestTypeMaps(t *testing.T) {
	t.Parallel()

	load := func(storageURLs string) *openapi.Document {
		var doc openapi.Document
		if err := json.Unmarshal([]byte(`{
			"openapi": "3.1.0", "info": {"title": "t", "version": "1"},
			"components": {"schemas": {
				"GetTilesProResponse": {"type": "object", "properties": {"storage_urls": `+storageURLs+`}}
			}}
		}`), &doc); err != nil {
			t.Fatal(err)
		}

		return &doc
	}

	doc := load(`{"type": "object", "additionalProperties": true}`)
	if err := typeMaps(doc); err != nil {
		t.Fatal(err)
	}

	v := doc.Components.Schemas["GetTilesProResponse"].Properties["storage_urls"].AdditionalProperties.Schema
	if v == nil || v.Type != openapi.TypeString || v.Format != openapi.FormatURI {
		t.Errorf("storage_urls holds %+v, want URI strings", v)
	}

	if err := typeMaps(load(`{"type": "object", "additionalProperties": {"type": "string"}}`)); err == nil {
		t.Error("an object typed upstream is no longer free-form, and the entry should fail the run")
	}
}
