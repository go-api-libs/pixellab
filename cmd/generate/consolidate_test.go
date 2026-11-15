package main

import (
	"encoding/json/v2"
	"testing"

	"github.com/MarkRosemaker/openapi"
)

// TestReferPlainStrings refers a plain string of a family's property to its enum, but not one whose recorded example
// is no value of it: a failed job's detail is a message, not a style's detail.
func TestReferPlainStrings(t *testing.T) {
	t.Parallel()

	var doc openapi.Document
	if err := json.Unmarshal([]byte(`{
		"openapi": "3.1.0", "info": {"title": "t", "version": "1"},
		"components": {"schemas": {
			"Detail": {"type": "string", "enum": ["low detail", "high detail"]},
			"Style": {"type": "object", "properties": {"detail": {"type": "string"}}},
			"Recorded": {"type": "object", "properties": {"detail": {"type": "string", "example": "high detail"}}},
			"LastResponse": {"type": "object", "properties": {"detail": {"type": "string", "example": "Connection timed out."}}}
		}}
	}`), &doc); err != nil {
		t.Fatal(err)
	}

	referPlainStrings(&doc, "Detail", []string{"detail"})

	for name, want := range map[string]bool{"Style": true, "Recorded": true, "LastResponse": false} {
		p := doc.Components.Schemas[name].Properties["detail"]
		if got := p.Ref != nil; got != want {
			t.Errorf("%s.detail refers to Detail: %v, want %v", name, got, want)
		}
	}
}
