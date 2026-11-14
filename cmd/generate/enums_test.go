package main

import (
	"encoding/json/jsontext"
	"slices"
	"testing"

	"github.com/MarkRosemaker/openapi"
)

func TestDeclareEnum(t *testing.T) {
	t.Parallel()

	mark := jsontext.Value(`{"x-pixellab-enum":"BodyTemplate"}`)

	s := &openapi.Schema{Type: openapi.TypeString, Enum: values("lion", "custom")}
	if err := declareEnum(s, []string{"mannequin", "lion"}, mark); err != nil {
		t.Fatal(err)
	}

	if got, want := s.Enum, values("lion", "custom", "mannequin"); !slices.EqualFunc(got, want,
		func(a, b jsontext.Value) bool { return string(a) == string(b) }) {
		t.Errorf("enum = %s, want %s", got, want)
	}

	if got := enumOf(s); got != "BodyTemplate" {
		t.Errorf("enumOf = %q, want BodyTemplate", got)
	}

	if len(s.Extensions) != 0 {
		t.Errorf("enumOf left the mark: %s", s.Extensions)
	}

	for _, bad := range []*openapi.Schema{
		{Type: openapi.TypeInteger},
		{Type: openapi.TypeString, Format: openapi.FormatUUID},
		{Type: openapi.TypeString, Extensions: mark},
	} {
		if err := declareEnum(bad, []string{"a"}, mark); err == nil {
			t.Errorf("declared an enum of %+v", bad)
		}
	}
}

func TestEnumValues(t *testing.T) {
	t.Parallel()

	s := &openapi.Schema{Type: openapi.TypeString, Enum: values("side", "top-down"), Default: jsontext.Value(`"side"`)}

	if got, want := enumValues(s, propertyRef{owner: "CreateRoomRequest", schema: &openapi.Schema{}}),
		`One of: side, top-down. Defaults to "side".`; got != want {
		t.Errorf("request: got %q, want %q", got, want)
	}

	if got, want := enumValues(s, propertyRef{owner: "GetRoomResponse", schema: &openapi.Schema{}}),
		"One of: side, top-down."; got != want {
		t.Errorf("response: got %q, want %q", got, want)
	}
}

// values is the strings as JSON.
func values(ss ...string) []jsontext.Value {
	vs := make([]jsontext.Value, len(ss))
	for i, s := range ss {
		vs[i] = jsontext.Value(`"` + s + `"`)
	}

	return vs
}

func TestIsJobStatus(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		property string
		s        *openapi.Schema
		want     bool
	}{
		{"status", &openapi.Schema{Type: openapi.TypeString, Default: jsontext.Value(`"processing"`)}, true},
		{"status", &openapi.Schema{Type: openapi.TypeString, Description: "Job status: processing, failed, completed"}, true},
		{"status", &openapi.Schema{Type: openapi.TypeString, Description: "Current job status (processing, completed, failed)"}, true},
		{"status", &openapi.Schema{Type: openapi.TypeString, Const: jsontext.Value(`"processing"`), Default: jsontext.Value(`"processing"`)}, false},
		{"status", &openapi.Schema{Type: openapi.TypeString, Description: "Generation status: 'completed', 'failed', or 'pending'."}, false},
		{"state", &openapi.Schema{Type: openapi.TypeString, Default: jsontext.Value(`"processing"`)}, false},
		{"status", &openapi.Schema{Type: openapi.TypeString, Enum: values("queued", "rate_limited"), Default: jsontext.Value(`"processing"`)}, false},
	} {
		if got := isJobStatus(&openapi.Schema{}, tc.property, tc.s); got != tc.want {
			t.Errorf("isJobStatus(%q, %+v) = %v, want %v", tc.property, tc.s, got, tc.want)
		}
	}
}

func TestMatchesPath(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		path string
		want bool
	}{
		{"/background-jobs/0a54f3a8-4d32-415d-8d34-2a1a222bca19", true},
		{"/background-jobs", false},
		{"/characters/0a54f3a8-4d32-415d-8d34-2a1a222bca19", false},
	} {
		if got := matchesPath(tc.path, "/background-jobs/{job_id}"); got != tc.want {
			t.Errorf("matchesPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}
