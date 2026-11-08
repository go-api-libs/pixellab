package main

import (
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/MarkRosemaker/openapi"
	edit "github.com/MarkRosemaker/openapi-edit"
	"github.com/ettle/strcase"
)

const schemaRefPrefix = "#/components/schemas/"

// enumFamilies are, by the name of the enum they are merged into, the properties whose string enums, or arrays of
// them, mean the same in every request. Upstream writes one per endpoint, each with the values that endpoint accepts.
var enumFamilies = map[string][]string{
	"CameraView": {"view"},
	"Direction":  {"direction", "directions", "deleted_directions"},
	"Outline":    {"outline"},
	"Detail":     {"detail"},
	"Shading":    {"shading"},
}

// propertyRef is a property that refers to a component schema.
type propertyRef struct {
	owner, property string
	schema          *openapi.Schema // the property's own schema, holding the reference
}

// propertyRefs lists, by the component schema they refer to, the properties of component schemas that refer to one,
// directly, as the items of an array or as the keys of a map.
func propertyRefs(doc *openapi.Document) map[string][]propertyRef {
	refs := map[string][]propertyRef{}

	for owner, s := range doc.Components.Schemas.ByIndex() {
		for prop, p := range s.Properties.ByIndex() {
			for _, ref := range []*openapi.Schema{p, p.Items, p.PropertyNames} {
				if ref != nil && ref.Ref != nil {
					name := strings.TrimPrefix(ref.Ref.Identifier, schemaRefPrefix)
					refs[name] = append(refs[name], propertyRef{owner: owner, property: prop, schema: p})
				}
			}
		}
	}

	return refs
}

// consolidateEnums merges, for each of enumFamilies, the string enums that properties of its name refer to into one
// enum of every value, and tells in each property's description which of them it accepts.
func consolidateEnums(doc *openapi.Document) error {
	for _, name := range slices.Sorted(maps.Keys(enumFamilies)) {
		props, refs := enumFamilies[name], propertyRefs(doc)

		var members []string

		for target, rs := range refs {
			s := doc.Components.Schemas[target]
			if s.Type == openapi.TypeString && len(s.Enum) > 0 && slices.ContainsFunc(rs, func(r propertyRef) bool { return slices.Contains(props, r.property) }) {
				members = append(members, target)
			}
		}

		members = sharingValues(doc, members)
		if len(members) == 0 {
			return fmt.Errorf("enumFamilies: no property %q refers to a string enum any more", props)
		}

		slices.Sort(members)

		merged := &openapi.Schema{Title: name, Type: openapi.TypeString}

		for _, m := range members {
			s := doc.Components.Schemas[m]
			for _, v := range s.Enum {
				if !slices.ContainsFunc(merged.Enum, func(e jsontext.Value) bool { return string(e) == string(v) }) {
					merged.Enum = append(merged.Enum, v)
				}
			}

			if len(members) > 1 {
				for _, r := range refs[m] {
					r.schema.Description = joinLines(r.schema.Description, enumValues(s))
				}
			}
		}

		if err := replaceSchemas(doc, name, merged, members); err != nil {
			return err
		}

		referPlainStrings(doc, name, props)
	}

	return nil
}

// sharingValues keeps the enums that share a value with another, where there are several: an enum upstream names after
// the same property, but for something else, as the portrait_to_character direction, stays an enum of its own.
func sharingValues(doc *openapi.Document, members []string) []string {
	if len(members) < 2 {
		return members
	}

	count := map[string]int{}

	for _, m := range members {
		for _, v := range doc.Components.Schemas[m].Enum {
			count[string(v)]++
		}
	}

	return slices.DeleteFunc(members, func(m string) bool {
		return !slices.ContainsFunc(doc.Components.Schemas[m].Enum, func(v jsontext.Value) bool { return count[string(v)] > 1 })
	})
}

// referPlainStrings makes the plain strings of the properties and parameters, or of their arrays, as a response's
// direction, refer to the enum: what upstream writes as a string there is a value of it.
func referPlainStrings(doc *openapi.Document, name string, props []string) {
	enum := doc.Components.Schemas[name]

	refer := func(p *openapi.Schema) {
		if p.Items != nil {
			p = p.Items
		}

		if p.Type == openapi.TypeString && p.Ref == nil && p.Format == "" && len(p.Enum) == 0 && len(p.Const) == 0 {
			p.Replace(&openapi.Schema{
				Description: p.Description,
				Ref:         &openapi.SchemaRef{Identifier: schemaRefPrefix + name, Value: enum},
			})
		}
	}

	for _, s := range doc.Components.Schemas.ByIndex() {
		for prop, p := range s.Properties.ByIndex() {
			if slices.Contains(props, prop) {
				refer(p)
			}
		}
	}

	for _, p := range doc.Components.Parameters.ByIndex() {
		if p.Value != nil && p.Value.Schema != nil && slices.Contains(props, p.Value.Name) {
			refer(p.Value.Schema)
		}
	}

	for _, item := range doc.Paths.ByIndex() {
		for _, op := range item.Operations {
			for _, p := range slices.Concat(item.Parameters, op.Parameters) {
				if p.Value != nil && p.Value.Schema != nil && slices.Contains(props, p.Value.Name) {
					refer(p.Value.Schema)
				}
			}
		}
	}
}

// enumValues tells the values an enum accepts, and the one it defaults to.
func enumValues(s *openapi.Schema) string {
	vals := make([]string, len(s.Enum))
	for i, e := range s.Enum {
		vals[i] = strings.Trim(string(e), `"`)
	}

	text := "One of: " + strings.Join(vals, ", ") + "."
	if len(s.Default) > 0 {
		text += fmt.Sprintf(" Defaults to %s.", s.Default)
	}

	return text
}

// imageSize is the bounds of one property's image size, which a constructor checks.
type imageSize struct {
	owner, field                             string
	widthMin, widthMax, heightMin, heightMax int
}

// consolidateImageSizes merges every object of nothing but an integer width and height into one ImageSize, and tells
// in each property's description which sizes it accepts. It returns the bounds of the properties of requests, for
// constructors that check them.
func consolidateImageSizes(doc *openapi.Document) ([]imageSize, error) {
	refs := propertyRefs(doc)

	var (
		members []string
		sizes   []imageSize
	)

	for name, s := range doc.Components.Schemas.ByIndex() {
		if !isSize(s) {
			continue
		}

		members = append(members, name)

		w, h := s.Properties["width"], s.Properties["height"]
		for _, r := range refs[name] {
			r.schema.Description = joinLines(r.schema.Description, sizeBounds(w, h))

			if strings.HasSuffix(r.owner, "Request") {
				sizes = append(sizes, imageSize{
					owner: r.owner, field: strcase.ToGoPascal(r.property),
					widthMin: intOf(w.Min), widthMax: intOf(w.Max), heightMin: intOf(h.Min), heightMax: intOf(h.Max),
				})
			}
		}
	}

	if len(members) == 0 {
		return nil, errors.New("no image size schemas any more")
	}

	merged := &openapi.Schema{
		Title: "ImageSize",
		Type:  openapi.TypeObject,
		Description: "Pixel dimensions of an image. Which sizes a request accepts is told where it is used; " +
			"the New*ImageSize constructors check them.",
		Required: []string{"width", "height"},
	}
	merged.Properties.Set("width", &openapi.Schema{Type: openapi.TypeInteger, Description: "Width in pixels.", Min: new(1.0)})
	merged.Properties.Set("height", &openapi.Schema{Type: openapi.TypeInteger, Description: "Height in pixels.", Min: new(1.0)})

	slices.SortFunc(sizes, func(a, b imageSize) int { return strings.Compare(a.owner+a.field, b.owner+b.field) })

	return sizes, replaceSchemas(doc, "ImageSize", merged, members)
}

// isSize reports whether s is an object of an integer width and height and nothing else.
func isSize(s *openapi.Schema) bool {
	if s.Type != openapi.TypeObject || len(s.Properties) != 2 {
		return false
	}

	w, h := s.Properties["width"], s.Properties["height"]

	return w != nil && h != nil && w.Type == openapi.TypeInteger && h.Type == openapi.TypeInteger
}

func sizeBounds(w, h *openapi.Schema) string {
	return fmt.Sprintf("Width: %s. Height: %s.", bounds(w), bounds(h))
}

func bounds(p *openapi.Schema) string {
	text := ""

	switch lo, hi := p.Min, p.Max; {
	case lo != nil && hi != nil && *lo == *hi:
		text = fmt.Sprintf("%d px", int(*lo))
	case lo != nil && hi != nil:
		text = fmt.Sprintf("%d-%d px", int(*lo), int(*hi))
	case lo != nil:
		text = fmt.Sprintf("at least %d px", int(*lo))
	case hi != nil:
		text = fmt.Sprintf("up to %d px", int(*hi))
	default:
		text = "any"
	}

	if len(p.Default) > 0 {
		text += fmt.Sprintf(", %s by default", p.Default)
	}

	return text
}

func intOf(f *float64) int {
	if f == nil {
		return 0
	}

	return int(*f)
}

// replaceSchemas puts merged in place of the schemas named members, under name, which may be one of them.
func replaceSchemas(doc *openapi.Document, name string, merged *openapi.Schema, members []string) error {
	if slices.Contains(members, name) {
		tmp := name + "Merged"
		if err := edit.RenameSchema(doc, name, tmp); err != nil {
			return err
		}

		members[slices.Index(members, name)] = tmp
	}

	doc.Components.Schemas.Set(name, merged)

	to := make(map[string]string, len(members))
	for _, m := range members {
		to[m] = name
	}

	return edit.RedirectSchemas(doc, to)
}

func joinLines(a, b string) string {
	if a = strings.TrimSpace(a); a == "" {
		return b
	}

	return a + "\n" + b
}

// directions are the properties of an object of one URL per direction.
var directions = []string{"south", "west", "east", "north", "south-east", "north-east", "north-west", "south-west"}

// consolidateRotations merges the objects of one property per direction, which characters and objects each have,
// into RotationURLs, of URLs, which requires what all of them do.
func consolidateRotations(doc *openapi.Document) error {
	var members []string

	for name, s := range doc.Components.Schemas.ByIndex() {
		if s.Type == openapi.TypeObject && len(s.Properties) == len(directions) &&
			!slices.ContainsFunc(directions, func(d string) bool { return s.Properties[d] == nil }) {
			members = append(members, name)
		}
	}

	if len(members) == 0 {
		return errors.New("no rotation URL schemas any more")
	}

	first := doc.Components.Schemas[members[0]]
	merged := &openapi.Schema{
		Title:       "RotationURLs",
		Type:        openapi.TypeObject,
		Description: "One image URL per direction; the diagonals only for 8 directions.",
		Required:    first.Required,
	}

	for _, d := range directions {
		p := *first.Properties[d]
		p.Format = openapi.FormatURI
		merged.Properties.Set(d, &p)
	}

	for _, m := range members[1:] {
		merged.Required = slices.DeleteFunc(merged.Required, func(r string) bool {
			return !slices.Contains(doc.Components.Schemas[m].Required, r)
		})
	}

	return replaceSchemas(doc, "RotationURLs", merged, members)
}

// nameImages names a job's images by their shape, as upstream gives them no schema: an image of its own encoding and
// size is an Image, one per compass point DirectionalImages, a list of them Images, and either of those JobImages.
func nameImages(doc *openapi.Document) error {
	to := map[string]string{}

	// each shape refers to the one before it, so each takes a pass of its own
	for _, shape := range []struct {
		name  string
		match func(*openapi.Schema) bool
	}{
		{"Image", func(s *openapi.Schema) bool { return hasProperties(s, "type", "width", "height", "base64") }},
		{"DirectionalImages", func(s *openapi.Schema) bool {
			return hasProperties(s, "east", "west", "north", "south") && refersTo(s.Properties["east"], "Image", to)
		}},
		{"Images", func(s *openapi.Schema) bool {
			return s.Type == openapi.TypeArray && refersTo(s.Items, "Image", to)
		}},
		{"JobImages", func(s *openapi.Schema) bool {
			return len(s.OneOf) == 2 && refersTo(s.OneOf[0], "DirectionalImages", to) && refersTo(s.OneOf[1], "Images", to)
		}},
	} {
		for name, s := range doc.Components.Schemas.ByIndex() {
			if _, named := to[name]; !named && shape.match(s) {
				to[name] = shape.name
			}
		}
	}

	if !slices.Contains(slices.Collect(maps.Values(to)), "Image") {
		return errors.New("no image schema any more")
	}

	return edit.RenameSchemas(doc, to)
}

// hasProperties reports whether s is an object of these properties and no other.
func hasProperties(s *openapi.Schema, props ...string) bool {
	return s.Type == openapi.TypeObject && len(s.Properties) == len(props) &&
		!slices.ContainsFunc(props, func(p string) bool { return s.Properties[p] == nil })
}

// refersTo reports whether s refers to the schema that to renames to name.
func refersTo(s *openapi.Schema, name string, to map[string]string) bool {
	return s != nil && s.Ref != nil && to[strings.TrimPrefix(s.Ref.Identifier, schemaRefPrefix)] == name
}
