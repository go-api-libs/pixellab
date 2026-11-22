package main

import (
	"cmp"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/MarkRosemaker/cassette"
	"github.com/MarkRosemaker/openapi"
	edit "github.com/MarkRosemaker/openapi-edit"
)

// declaredEnums are, by the name of the enum, the strings upstream holds to a set of values without declaring it, or
// declares in pieces: the values it documents, and where it holds them to them, as "Schema.property" of the original,
// "path?parameter", or "METHOD path property.property" of what an operation answers, where only a recorded interaction
// shows it, as what a background job last answered, or every property match picks of the original's schemas. What a
// recorded interaction shows besides is added to them. An entry, or a match, that names nothing fails the run.
var declaredEnums = map[string]declaredEnum{
	// "Animation template ID. Required for template mode. Available: `angry`, ...", the rest cut off: each body
	// template has animations of its own, which the API names when turned down one it does not have, and the
	// recordings hold that answer for every body template; a character's animations name the template they were made
	// from, which their GET filters by
	"AnimationTemplate": {
		// a body template does not have every animation, and the API names those it has
		rejected: &rejection{
			by: "BodyTemplate",
			// "`custom` skips the skeleton, for bodies no template fits; such a character can't use skeleton-based
			// animations."
			none: []string{"custom"},
			re:   regexp.MustCompile(`^Invalid template_animation_id '[^']*' for template '(?P<for>[^']+)'\. Available: (?P<values>.+)$`),
		},
		at: []string{
			"CreateCharacterAnimationRequest.template_animation_id",
			"AnimationGroup.animation_type",
			"/characters/{character_id}/animations?animation_type",
			"GET /background-jobs/{job_id} last_response.template_animation_id",
		},
	},

	// "'mannequin' (the only humanoid template), 'bear', 'cat', 'dog', 'horse', 'lion' (quadrupeds)", the body a
	// character's skeleton is fitted to, which some requests declare an enum of and others, and the character, do not;
	// some requests take "custom" too, which "skips the skeleton, for bodies no template fits"
	"BodyTemplate": {
		values: []string{"mannequin", "bear", "cat", "dog", "horse", "lion"},
		at: []string{
			"AnimateWithSkeletonV3Request.template_id",
			"CreateCharacterProFlashRequest.template_id",
			"CreateCharacterProRequest.template_id",
			"CreateCharacterV3Request.template_id",
			"CreateCharacterWith4DirectionsRequest.template_id",
			"CreateCharacterWith8DirectionsRequest.template_id",
			"CharacterDetail.template_id",
			"CharacterSummary.template_id",
			"GET /background-jobs/{job_id} last_response.template_id",
		},
	},

	// "Generation status: 'completed', 'failed', or 'pending'."
	"CharacterStatus": {
		values: []string{"completed", "failed", "pending"},
		at:     []string{"CharacterDetail.status", "CharacterSummary.status"},
	},

	// "Current job status (processing, completed, failed)", besides "pending" and "running" of some, "stalled" and
	// "finalizing" of a font's and a portrait's, and the "queued" a state or an object starts out as: what each response
	// that starts a job, or reads one, says of it
	"JobStatus": {
		values: []string{"queued", "processing", "completed", "failed", "pending", "running", "stalled", "finalizing"},
		at:     []string{"GetVocalAnimationResponse.status"}, // "processing, completed or failed"
		match:  isJobStatus,
	},

	// what a job is doing, as its last answer says: upstream writes nothing of it, so the recorded interactions give
	// every value
	"JobStage": {
		at: []string{"GET /background-jobs/{job_id} last_response.status"},
	},

	// "Object status (pending, processing, completed, review, failed; expanding / review-expansion while an
	// 8-direction expansion runs or awaits review)"
	"ObjectStatus": {
		values: []string{"pending", "processing", "completed", "review", "failed", "expanding", "review-expansion"},
		at:     []string{"ObjectDetail.status", "ObjectSummary.status"},
	},

	// "Generation status: completed, processing, or failed" of a tile or tileset, "processing | completed | failed" of
	// a UI asset, and the enum of a room; the tiles pro and sidescroller summaries, which say nothing of theirs, are
	// listed beside the others
	"AssetStatus": {
		values: []string{"processing", "completed", "failed"},
		at: []string{
			"IsometricTileSummary.status", "SidescrollerTilesetSummary.status", "TilesetSummary.status",
			"TilesProSummary.status", "UIAssetDetail.status", "UIAssetSummary.status", "Room.status",
		},
	},

	// "pending, running, completed or failed" of a song and a sound effect
	"SoundStatus": {
		values: []string{"pending", "running", "completed", "failed"},
		at:     []string{"Song.status", "SoundEffect.status"},
	},

	// "What this group is: tiles (independent variations), paths (road autotiles), tileset (corner wang terrain
	// transition), coastline (hex edge set), building (construction kit)."
	"TilesProKind": {
		values: []string{"tiles", "paths", "tileset", "coastline", "building"},
		at:     []string{"GetTilesProResponse.kind"},
	},

	// "Subscription status: active, trial, expired, or none. Team (enterprise) members get enterprise, ..."
	"SubscriptionStatus": {
		values: []string{"active", "trial", "expired", "none", "enterprise"},
		at:     []string{"Subscription.status"},
	},
}

// declaredEnum is the values an enum of declaredEnums documents, and where it holds strings to them: the places at
// names, and the properties match picks, by the schema that holds them, their name and their schema in the original.
type declaredEnum struct {
	values, at []string
	match      func(owner *openapi.Schema, property string, s *openapi.Schema) bool
	rejected   *rejection
}

// rejection is how the API turns down a value of an enum, as a body template's animation it does not have: with the
// values it takes instead, for a value of another enum. Every value a recorded 422 names is one of the enum, and the
// enum tells which the other's values take.
type rejection struct {
	by   string   // the enum the value it answers for is of
	none []string // the values of by that take none, which no 422 can say
	// what the 422's detail says, its group "for" that value, its group "values" those it takes, comma-separated
	re *regexp.Regexp
}

// accepted is, by the value of r.by each answers for, the values the 422s recorded in ias say it takes, in the order
// they say them.
func (r *rejection) accepted(ias cassette.Interactions) map[string][]string {
	byFor := map[string][]string{}

	for _, ia := range ias {
		if ia.Response.StatusCode != 422 {
			continue
		}

		var body struct {
			Detail any `json:"detail"`
		}
		if json.Unmarshal(ia.Response.Body, &body) != nil {
			continue
		}

		detail, _ := body.Detail.(string)

		m := r.re.FindStringSubmatch(detail)
		if m == nil {
			continue
		}

		values := strings.Split(m[r.re.SubexpIndex("values")], ", ")
		for i := range values {
			values[i] = strings.TrimSpace(values[i])
		}

		byFor[m[r.re.SubexpIndex("for")]] = values
	}

	return byFor
}

// allValues is the values e documents and those every recorded 422 names, each once.
func (e declaredEnum) allValues(ias cassette.Interactions) ([]string, error) {
	values := slices.Clone(e.values)
	if e.rejected == nil {
		return values, nil
	}

	accepted := e.rejected.accepted(ias)
	if len(accepted) == 0 {
		return nil, fmt.Errorf("no recorded 422 turns a value down as %s", e.rejected.re)
	}

	for _, k := range slices.Sorted(maps.Keys(accepted)) {
		for _, v := range accepted[k] {
			if !slices.Contains(values, v) {
				values = append(values, v)
			}
		}
	}

	return values, nil
}

// describeRejections tells, in each enum of declaredEnums the API turns values down of, which values of the other
// enum take which of its own, as the recorded 422s say: for every value of that enum, and for no other.
func describeRejections(doc *openapi.Document, ias cassette.Interactions) error {
	for _, name := range slices.Sorted(maps.Keys(declaredEnums)) {
		r := declaredEnums[name].rejected
		if r == nil {
			continue
		}

		enum, by := doc.Components.Schemas[name], doc.Components.Schemas[r.by]
		if enum == nil || by == nil {
			return fmt.Errorf("%s or %s is no schema", name, r.by)
		}

		var b strings.Builder
		fmt.Fprintf(&b, "Not every %s takes every %s. Each takes these, as the API answers one it does not take:\n", r.by, name)

		accepted := r.accepted(ias)

		// in the order of the other enum's values, which is upstream's, as mannequin, "the only humanoid template",
		// before the quadrupeds
		var missing []string
		for _, v := range by.Enum {
			var k string
			if json.Unmarshal(v, &k) != nil {
				continue
			}

			switch values, none := accepted[k], slices.Contains(r.none, k); {
			case values != nil && none:
				return fmt.Errorf("a recorded 422 names what %s takes, which takes no %s", k, name)
			case none:
				fmt.Fprintf(&b, "\n  - %s: none", k)
			case values != nil:
				fmt.Fprintf(&b, "\n  - %s: %s", k, strings.Join(values, ", "))
			default:
				missing = append(missing, k)
			}

			delete(accepted, k)
		}

		if len(missing) > 0 {
			return fmt.Errorf("no recorded 422 tells which %s %s takes: record one turning down a value for each",
				name, strings.Join(missing, ", "))
		}

		if len(accepted) > 0 {
			return fmt.Errorf("a recorded 422 names %s, which is no %s", strings.Join(slices.Sorted(maps.Keys(accepted)), ", "), r.by)
		}

		for _, k := range r.none {
			if !slices.ContainsFunc(by.Enum, func(v jsontext.Value) bool { return string(v) == `"`+k+`"` }) {
				return fmt.Errorf("%q, which takes no %s, is no %s", k, name, r.by)
			}
		}

		enum.Description = strings.TrimSpace(enum.Description + "\n\n" + b.String())
	}

	return nil
}

// isJobStatus reports whether a property is a job's status: a status beside the job's background_job_id, or one that
// starts out "processing" or that upstream calls a job's. One it fixes with a const, as "Always 'processing'", or
// gives an enum of its own, as a direction's submission's "queued" or "rate_limited", stays as it is.
func isJobStatus(owner *openapi.Schema, property string, s *openapi.Schema) bool {
	return property == "status" && len(s.Const) == 0 && len(s.Enum) == 0 && (owner.Properties["background_job_id"] != nil ||
		string(s.Default) == `"processing"` || strings.Contains(strings.ToLower(s.Description), "job status"))
}

// enumMark marks a string declareEnums made an enum of with the name of its enum, which consolidateEnums merges them
// into, so that compress, which keeps schemas of different extensions apart, merges no other enum into one.
const enumMark = "x-pixellab-enum"

// declareEnums makes the strings declaredEnums names enums of the values it gives, marked with the enum's name, before
// the recorded interactions add to them. It returns the places it found.
func declareEnums(doc *openapi.Document, ias cassette.Interactions) (map[string]bool, error) {
	declared := map[string]bool{}

	for _, name := range slices.Sorted(maps.Keys(declaredEnums)) {
		e := declaredEnums[name]

		values, err := e.allValues(ias)
		if err != nil {
			return nil, fmt.Errorf("declaredEnums: %s: %w", name, err)
		}

		mark, err := json.Marshal(map[string]string{enumMark: name})
		if err != nil {
			return nil, err
		}

		for _, at := range e.at {
			for _, s := range enumTargets(doc, at) {
				if err := declareEnum(stringSchema(s), values, mark); err != nil {
					return nil, fmt.Errorf("%s: %w", at, err)
				}

				declared[at] = true
			}
		}

		if e.match == nil {
			continue
		}

		matched := false

		for schema, s := range doc.Components.Schemas.ByIndex() {
			for prop, p := range s.Properties.ByIndex() {
				if !e.match(s, prop, p) {
					continue
				}

				if err := declareEnum(stringSchema(p), values, mark); err != nil {
					return nil, fmt.Errorf("%s.%s: %w", schema, prop, err)
				}

				matched = true
			}
		}

		if !matched {
			return nil, fmt.Errorf("declaredEnums: %s matches no property any more", name)
		}
	}

	return declared, nil
}

// referDeclaredEnums makes the places of declaredEnums that declareEnums did not find, which the passes made, refer to
// their enum, which takes every value an interaction recorded there, and is made where nothing else holds it.
func referDeclaredEnums(doc *openapi.Document, declared map[string]bool, ias cassette.Interactions) error {
	var unused []string

	for _, name := range slices.Sorted(maps.Keys(declaredEnums)) {
		values, err := declaredEnums[name].allValues(ias)
		if err != nil {
			return fmt.Errorf("declaredEnums: %s: %w", name, err)
		}

		enum := doc.Components.Schemas[name]
		if enum == nil {
			enum = &openapi.Schema{Title: name, Type: openapi.TypeString}
			for _, v := range values {
				enum.Enum = append(enum.Enum, jsontext.Value(`"`+v+`"`))
			}
		}

		for _, at := range declaredEnums[name].at {
			if declared[at] {
				continue
			}

			found := enumTargets(doc, at)
			if len(found) == 0 {
				unused = append(unused, at)
			}

			for _, p := range found {
				s := stringSchema(p)
				if s.Type != openapi.TypeString || s.Ref != nil || s.Format != "" || len(s.Enum) > 0 || len(s.Const) > 0 {
					return fmt.Errorf("%s: not a plain string", at)
				}

				for _, v := range append(recordedValues(ias, at), s.Example) {
					if v != nil && !slices.ContainsFunc(enum.Enum, func(e jsontext.Value) bool { return string(e) == string(v) }) {
						enum.Enum = append(enum.Enum, v)
					}
				}

				doc.Components.Schemas.Set(name, enum)

				ref := edit.SchemaRef(doc, name)
				ref.Description = s.Description
				s.Replace(ref)
			}
		}
	}

	if len(unused) > 0 {
		slices.Sort(unused)
		return fmt.Errorf("declaredEnums names strings neither upstream nor a recorded interaction has: %s",
			strings.Join(unused, ", "))
	}

	return describeRejections(doc, ias)
}

// enumTargets is what at names: a property, as "Schema.property", the parameter of each operation of a path, as
// "path?parameter", or the property of what an operation answers, as "METHOD path property.property".
func enumTargets(doc *openapi.Document, at string) []*openapi.Schema {
	if f := strings.Fields(at); len(f) == 3 {
		return answerTargets(doc, f[0], f[1], strings.Split(f[2], "."))
	}

	if path, param, ok := strings.Cut(at, "?"); ok {
		item := pathItem(doc, path)
		if item == nil {
			return nil
		}

		var found []*openapi.Schema

		for _, op := range item.Operations {
			for _, p := range slices.Concat(item.Parameters, op.Parameters) {
				if p.Value != nil && p.Value.Name == param && p.Value.Schema != nil {
					found = append(found, p.Value.Schema)
				}
			}
		}

		return found
	}

	schema, prop, _ := strings.Cut(at, ".")
	if s := doc.Components.Schemas[schema]; s != nil && s.Properties[prop] != nil {
		return []*openapi.Schema{s.Properties[prop]}
	}

	return nil
}

// declareEnum makes s an enum of the values, besides those it has, marked as mark says.
func declareEnum(s *openapi.Schema, values []string, mark jsontext.Value) error {
	switch {
	case s.Type != openapi.TypeString:
		return fmt.Errorf("not a string, but %q", s.Type)
	case s.Ref != nil || s.Format != "" || len(s.Const) > 0:
		return fmt.Errorf("not a plain string")
	case len(s.Extensions) > 0:
		return fmt.Errorf("already has extensions: %s", s.Extensions)
	}

	for _, v := range values {
		if !slices.ContainsFunc(s.Enum, func(e jsontext.Value) bool { return string(e) == `"`+v+`"` }) {
			s.Enum = append(s.Enum, jsontext.Value(`"`+v+`"`))
		}
	}

	s.Extensions = mark

	return nil
}

// enumOf is the name of the enum of declaredEnums s was made, and removes its mark.
func enumOf(s *openapi.Schema) string {
	if len(s.Extensions) == 0 {
		return ""
	}

	var ext map[string]string
	if err := json.Unmarshal(s.Extensions, &ext); err != nil || ext[enumMark] == "" {
		return ""
	}

	s.Extensions = nil

	return ext[enumMark]
}

// pathItem is the path upstream names, which the passes may have moved the prefix of into the server URL.
func pathItem(doc *openapi.Document, path string) *openapi.PathItem {
	return cmp.Or(doc.Paths[openapi.Path(pathPrefix+path)], doc.Paths[openapi.Path(path)])
}

// answerTargets is the property at the end of props of what each success of the operation answers.
func answerTargets(doc *openapi.Document, method, path string, props []string) []*openapi.Schema {
	item := pathItem(doc, path)
	if item == nil {
		return nil
	}

	var found []*openapi.Schema

	for m, op := range item.Operations {
		if m != method {
			continue
		}

		for code, rsp := range op.Responses.ByIndex() {
			if rsp.Value == nil || !strings.HasPrefix(string(code), "2") {
				continue
			}

			for _, mt := range rsp.Value.Content.ByIndex() {
				s := mt.Schema
				for _, prop := range props {
					if s = resolved(s); s != nil {
						s = s.Properties[prop]
					}
				}

				if s != nil {
					found = append(found, s)
				}
			}
		}
	}

	return found
}

// resolved is what s, or its alternative to null, refers to, or s itself.
func resolved(s *openapi.Schema) *openapi.Schema {
	if s == nil {
		return nil
	}

	if s = nonNull(s); s.Ref != nil {
		return s.Ref.Value
	}

	return s
}

// recordedValues is every string an interaction recorded at the place at, as "METHOD path property.property" of what
// an operation answers, in the order recorded.
func recordedValues(ias cassette.Interactions, at string) []jsontext.Value {
	f := strings.Fields(at)
	if len(f) != 3 {
		return nil
	}

	var vals []jsontext.Value

	for _, ia := range ias {
		if ia.Request.Method != f[0] || !matchesPath(strings.TrimPrefix(ia.Request.URL.Path, pathPrefix), f[1]) {
			continue
		}

		var v any
		if json.Unmarshal(ia.Response.Body, &v) != nil {
			continue
		}

		for prop := range strings.SplitSeq(f[2], ".") {
			m, _ := v.(map[string]any)
			v = m[prop]
		}

		if str, ok := v.(string); ok {
			if val, err := json.Marshal(str); err == nil && !slices.ContainsFunc(vals, func(e jsontext.Value) bool { return string(e) == string(val) }) {
				vals = append(vals, val)
			}
		}
	}

	return vals
}

// matchesPath reports whether path is one of template's, as /background-jobs/{job_id}.
func matchesPath(path, template string) bool {
	got, want := strings.Split(path, "/"), strings.Split(template, "/")
	if len(got) != len(want) {
		return false
	}

	for i, w := range want {
		if !strings.HasPrefix(w, "{") && got[i] != w {
			return false
		}
	}

	return true
}
