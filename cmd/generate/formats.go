package main

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/MarkRosemaker/openapi"
)

// formats gives the strings upstream declares without a format the one the API holds them to, by property name for
// every schema, or by "Schema.property" of the original. An entry that matches nothing fails the run.
var formats = map[string]openapi.Format{
	// a background job is fetched at /background-jobs/{job_id}, which returns its id as a UUID
	"background_job_id":              openapi.FormatUUID,
	"background_job_ids":             openapi.FormatUUID,
	"job_id":                         openapi.FormatUUID,
	"CancelBackgroundJobResponse.id": openapi.FormatUUID,

	// a resource's id is the UUID its own path takes, as /characters/{character_id}
	"CharacterSummary.id":           openapi.FormatUUID,
	"ObjectSummary.id":              openapi.FormatUUID,
	"ObjectDetail.id":               openapi.FormatUUID,
	"UIAssetDetail.id":              openapi.FormatUUID,
	"UIAssetSummary.id":             openapi.FormatUUID,
	"TilesetSummary.id":             openapi.FormatUUID,
	"SidescrollerTilesetSummary.id": openapi.FormatUUID,
	"IsometricTileSummary.id":       openapi.FormatUUID,
	"TilesProSummary.id":            openapi.FormatUUID,

	// "Public URLs for each frame in order"
	"AnimationDirection.frames": openapi.FormatURI,
}

// typeStrings holds every string the API holds to one format to it, so that the library has a uuid.UUID or url.URL
// where it would otherwise have a string:
//
//   - formats gives what upstream leaves bare.
//   - A property name ending in _id, or _ids for an array, that has a format anywhere has it everywhere: upstream
//     types character_id in one schema and leaves it bare in eleven.
//
// It runs before the passes, for formats, which names schemas of the original, and after them, when the recorded
// interactions have given formats of their own, as to group_id. A recorded interaction can still take a format away,
// where a value does not fit it.
func typeStrings(doc *openapi.Document, final bool) error {
	declared, err := declaredFormats(doc)
	if err != nil {
		return err
	}

	unused := maps.Clone(formats)

	// format is the format of what keys name, the first of them formats has, or else the one declared for name.
	format := func(name string, keys ...string) (openapi.Format, bool) {
		for _, key := range keys {
			if f, ok := formats[key]; ok && !final {
				delete(unused, key)
				return f, true
			}
		}

		f, ok := declared[idName(name)]

		return f, ok
	}

	for name, s := range doc.Components.Schemas.ByIndex() {
		for prop, p := range s.Properties.ByIndex() {
			if f, ok := format(prop, name+"."+prop, prop); ok && !(final && disproved(p)) {
				if err := setFormat(p, f); err != nil {
					return fmt.Errorf("%s.%s: %w", name, prop, err)
				}
			}
		}
	}

	for _, item := range doc.Paths.ByIndex() {
		for _, op := range item.Operations {
			for _, p := range slices.Concat(item.Parameters, op.Parameters) {
				if p.Value == nil || p.Value.Schema == nil {
					continue
				}

				if f, ok := format(p.Value.Name, p.Value.Name); ok && !(final && disproved(p.Value.Schema)) {
					if err := setFormat(p.Value.Schema, f); err != nil {
						return fmt.Errorf("parameter %s: %w", p.Value.Name, err)
					}
				}
			}
		}
	}

	if !final && len(unused) > 0 {
		return fmt.Errorf("formats names strings upstream no longer has: %s",
			strings.Join(slices.Sorted(maps.Keys(unused)), ", "))
	}

	return nil
}

// declaredFormats is, by ID name, the format upstream gives it where it gives one.
func declaredFormats(doc *openapi.Document) (map[string]openapi.Format, error) {
	declared := map[string]openapi.Format{}

	var errs []error

	add := func(name string, s *openapi.Schema) {
		if s == nil || idName(name) == "" {
			return
		}

		if s.Items != nil {
			s = s.Items
		}

		if s.Type != openapi.TypeString || s.Format == "" {
			return
		}

		name = idName(name)
		if f, ok := declared[name]; ok && f != s.Format {
			errs = append(errs, fmt.Errorf("upstream gives %s both %s and %s", name, f, s.Format))
		}

		declared[name] = s.Format
	}

	for _, s := range doc.Components.Schemas.ByIndex() {
		for prop, p := range s.Properties.ByIndex() {
			add(prop, p)
		}
	}

	for _, item := range doc.Paths.ByIndex() {
		for _, op := range item.Operations {
			for _, p := range slices.Concat(item.Parameters, op.Parameters) {
				if p.Value != nil {
					add(p.Value.Name, p.Value.Schema)
				}
			}
		}
	}

	return declared, errors.Join(errs...)
}

// idName is the ID a property name holds, as character_id for character_id and character_ids, or "" for any other.
func idName(name string) string {
	switch {
	case strings.HasSuffix(name, "_id"):
		return name
	case strings.HasSuffix(name, "_ids"):
		return strings.TrimSuffix(name, "s")
	default:
		return ""
	}
}

// disproved reports whether a recorded value took the format of a string, or of the strings of an array, away: it
// has an example, which enrich gives it, but no format.
func disproved(s *openapi.Schema) bool {
	if s.Items != nil {
		s = s.Items
	}

	return s.Format == "" && s.Example != nil
}

// setFormat gives a string, or the strings of an array, the format, unless it has another.
func setFormat(s *openapi.Schema, f openapi.Format) error {
	if s.Items != nil {
		s = s.Items
	}

	switch {
	case s.Type != openapi.TypeString:
		return fmt.Errorf("not a string, but %q", s.Type)
	case s.Format == "" || s.Format == f:
		s.Format = f
		return nil
	default:
		return fmt.Errorf("format %s, not %s", s.Format, f)
	}
}
