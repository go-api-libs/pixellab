package main

import (
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	"fmt"
	"go/format"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/MarkRosemaker/openapi"
	codegen "github.com/MarkRosemaker/openapi-codegen"
	compress "github.com/MarkRosemaker/openapi-compress"
	edit "github.com/MarkRosemaker/openapi-edit"
	enrich "github.com/MarkRosemaker/openapi-enrich"
	flatten "github.com/MarkRosemaker/openapi-flatten"
	"github.com/ettle/strcase"
)

const (
	originalPath = "api/openapi-v2_original.json"
	specPath     = "api/openapi.json"
)

var reDirection = regexp.MustCompile(`","default":"(east|high top-down)"`)

var repl = strings.NewReplacer(
	" (Pro)", "",
	"(", "", ")", "",
	",", "", "↔", " ",
	" + result", "",
	"'", "",
	" a ", " ",
	"+", "",
)

var opIDMapping = map[string]string{
	"image_to_pixelart_pro_image_to_pixelart_pro_post": "ConvertImageToPixelArtPro",
	"inpaint_v3_inpaint_v3_post":                       "InpaintImageV3",
}

func main() {
	data, err := os.ReadFile(originalPath)
	if err != nil {
		log.Fatal(err)
	}

	data = bytes.ReplaceAll(data, []byte(`"additionalProperties":false,`), nil)
	data = bytes.ReplaceAll(data, []byte(`"additionalProperties":true`), []byte(`"additionalProperties": {"type": "object"}`))
	data = bytes.ReplaceAll(data, []byte(`view angle","default":"side"`), []byte(`view angle"`)) // (default: side)
	data = bytes.ReplaceAll(data, []byte(`,"default":{"width":128,"height":128}`), nil)
	data = bytes.ReplaceAll(data, []byte(`,"default":{"width":16,"height":16}`), nil)

	data = reDirection.ReplaceAll(data, []byte(` (default: \"$1\")"`))

	doc, err := openapi.LoadFromDataJSON(data)
	if err != nil {
		log.Fatal(err)
	}

	doc.Info.Description = strings.TrimSpace(doc.Info.Description)
	doc.Servers = openapi.Servers{{URL: "https://api.pixellab.ai/v2"}}

	for _, p := range doc.Paths {
		for _, p := range p.Parameters {
			s := p.Value.Schema
			if s == nil {
				continue
			}

			improveSchema(s)
		}

		for _, op := range p.Operations {
			if opID, ok := opIDMapping[op.OperationID]; ok {
				op.OperationID = opID
			} else {
				op.OperationID = strcase.ToGoPascal(repl.Replace(op.Summary))
			}

			for _, p := range op.Parameters {
				s := p.Value.Schema
				if s == nil {
					continue
				}

				improveSchema(s)
			}

			for _, r := range op.Responses {
				for _, c := range r.Value.Content {
					s := c.Schema
					if s == nil || s.Value.Type != "" {
						continue
					}

					// fill empty schemas
					s.Value.Type = openapi.TypeObject
				}
			}
		}
	}

	for name, s := range doc.Components.Schemas.ByIndex() {
		improveSchema(&openapi.SchemaRef{Value: s})

		switch name {
		case "CameraView":
			s.Default = mustEncode(string("side"))
		case "app__endpoints__external__v2__create_map_object__ImageSize":
			s.Default = mustEncode(dim{
				Width:  128,
				Height: 128,
			})
		case "TileSize", "SidescrollerTileSize":
			s.Default = mustEncode(dim{
				Width:  16,
				Height: 16,
			})
		}

		for _, p := range s.Properties {
			if p.Value.Type != "" {
				continue
			}
		}
	}

	if err := doc.Validate(); err != nil {
		log.Fatalf("Validation ERROR: %v", err)
	}

	if err := enrich.Enrich(doc, nil); err != nil {
		log.Fatalf("enrich: %v", err)
	}

	if err := flatten.Document(doc); err != nil {
		log.Fatalf("flatten: %v", err)
	}

	if err := compress.Document(doc, compress.Config{}); err != nil {
		log.Fatalf("compress: %v", err)
	}

	imageSizeSpecs := consolidateImageSizes(doc)

	for _, path := range doc.Paths {
		for _, op := range path.Operations {
			op.Responses.Sort()
		}
	}
	doc.Components.SortMaps()

	for _, v := range []struct{ old, new string }{
		// {"app__endpoints__external__v2__animate_with_skeleton__ImageSize", "ImageSize"},
		// {"app__endpoints__external__v2__animate_with_text_v2__ReferenceImageSize", "ReferenceImageSize"},
		// {"app__endpoints__external__v2__edit_animation_v2__FrameImage", "FrameImage"},
	} {
		if err := edit.RenameSchema(doc, v.old, v.new); err != nil {
			log.Fatal(err)
		}
	}

	if err := doc.WriteToFile(specPath); err != nil {
		log.Fatal(err)
	}

	if err := codegen.Generate(codegen.Config{
		Debug:       true,
		Spec:        doc,
		PackageName: "pixellab",
		OutputDir:   "pkg/pixellab",
	}); err != nil {
		log.Fatalf("codegen: %v", err)
	}

	if err := writeImageSizeConstructors("pkg/pixellab/image_size.go", imageSizeSpecs); err != nil {
		log.Fatalf("image size constructors: %v", err)
	}
}

type dim struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func improveSchema(ss *openapi.SchemaRef) {
	s := ss.Value

	for _, p := range s.Properties {
		improveSchema(p)
	}

	if s.Items != nil {
		improveSchema(s.Items)
	}

	if s.Type == "" && len(s.AnyOf) == 0 {
		s.Type = openapi.TypeObject
	}

	if len(s.AnyOf) < 2 {
		return
	}

	if n := s.AnyOf[len(s.AnyOf)-1]; n.Value.Type != openapi.TypeNull {
		return
	}

	s.AnyOf = s.AnyOf[:len(s.AnyOf)-1]
	if len(s.AnyOf) > 1 {
		return
	}

	prime := ss.Value.AnyOf[0]
	if prime.Ref != nil {
		ss.Ref = prime.Ref
		ss.Value.AnyOf = nil
		ss.Ref.Description = s.Description
		if prime.Value.Example == nil {
			prime.Value.Example = s.Example
		}
	} else {
		title, descr := s.Title, s.Description
		*s = *prime.Value
		s.Title = title
		s.Description = descr
	}

	improveSchema(ss)
}

func mustEncode(v any) jsontext.Value {
	b := &bytes.Buffer{}
	if err := json.NewEncoder(b).Encode(v); err != nil {
		panic(err)
	}

	return jsontext.Value(bytes.TrimSpace(b.Bytes()))
}

func ptr[T any](v T) *T { return &v }

// imageSizeUsage identifies one field, on one generated Go request type,
// that carries an image size.
type imageSizeUsage struct {
	// typeName is the generated Go request type, e.g. "CreateImagePixfluxRequest".
	typeName string
	// field is the generated Go field name on that type, e.g. "ImageSize".
	field string
}

// funcName is the name of the constructor generated for this usage.
func (u imageSizeUsage) funcName() string {
	return "New" + u.varName()
}

// varName is the name of the fixed-size global variable generated for this
// usage when its bounds don't leave room for a constructor to check anything.
func (u imageSizeUsage) varName() string {
	base := strings.TrimSuffix(u.typeName, "Request")
	if u.field == "ImageSize" {
		return "ImageSizeFor" + base
	}
	return u.field + "For" + base
}

// imageSizeGroups maps the schema name each distinct ImageSize bound set has
// right after openapi-compress runs, to the Go request types/fields that end
// up using it.
//
// The upstream API generates one "ImageSize" schema per endpoint (FastAPI
// names anonymous models after the enclosing function), so schemas that
// happen to share the exact same width/height bounds are already merged by
// openapi-compress by the time consolidateImageSizes runs. What compress
// can't tell us is which Go request types a surviving schema came from -
// that mapping has to be maintained here.
//
// If the upstream API adds or renames an endpoint that uses ImageSize,
// consolidateImageSizes fails loudly asking for this table to be updated.
var imageSizeGroups = map[string][]imageSizeUsage{
	"app__endpoints__external__v__animate_with_skeleton__ImageSize": {
		{"AnimateWithSkeletonRequest", "ImageSize"},
		{"EditAnimationV2Request", "ImageSize"},
	},
	"app__endpoints__external__v2__animate_with_text__ImageSize": {
		{"AnimateWithTextRequest", "ImageSize"},
	},
	"app__endpoints__external__v__create_character_with__directions__ImageSize": {
		{"CreateCharacterWith4DirectionsRequest", "ImageSize"},
		{"CreateCharacterWith8DirectionsRequest", "ImageSize"},
		{"InterpolationV2Request", "ImageSize"},
	},
	"app__endpoints__external__v__create_image_bitforge__ImageSize": {
		{"CreateImageBitforgeRequest", "ImageSize"},
		{"InpaintRequest", "ImageSize"},
		{"ResizeRequest", "ReferenceImageSize"},
		{"ResizeRequest", "TargetSize"},
		{"RotateRequest", "ImageSize"},
	},
	"app__endpoints__external__v2__create_image_pixen__ImageSize": {
		{"CreateImagePixenRequest", "ImageSize"},
		{"EnhancePixenPromptRequest", "ImageSize"},
	},
	"app__endpoints__external__v__create_image_pixflux__ImageSize": {
		{"CreateImagePixfluxRequest", "ImageSize"},
		{"EditImageRequest", "ImageSize"},
	},
	"app__endpoints__external__v2__create_isometric_tile__ImageSize": {
		{"CreateIsometricTileRequest", "ImageSize"},
	},
	"app__endpoints__external__v2__create_map_object__ImageSize": {
		{"CreateMapObjectRequest", "ImageSize"},
	},
	"app__endpoints__external__v2__create_ui_asset__ImageSize": {
		{"CreateUIAssetRequest", "ImageSize"},
	},
	"app__endpoints__external__v2__edit_images_v2__ImageSize": {
		{"EditImagesV2Request", "ImageSize"},
	},
	"app__endpoints__external__v2__generate_image_v2__ImageSize": {
		{"GenerateImageV2Request", "ImageSize"},
	},
	"app__endpoints__external__v2__generate_ui_v2__ImageSize": {
		{"GenerateUIV2Request", "ImageSize"},
	},
	"app__endpoints__external__v2__image_to_pixelart__ImageSize": {
		{"ImageToPixelartRequest", "ImageSize"},
	},
	"app__endpoints__external__v2__remove_background__ImageSize": {
		{"RemoveBackgroundRequest", "ImageSize"},
	},
}

// imageSizeSpec is the width/height bound set for one merged group of
// ImageSize usages, plus the Go request types/fields that use it.
type imageSizeSpec struct {
	usages []imageSizeUsage

	widthMin, heightMin int
	widthMax, heightMax int
}

// consolidateImageSizes merges every per-endpoint "ImageSize" schema left
// over from openapi-compress into a single shared "ImageSize" component,
// moving each variant's specific width/height bounds into the description of
// the field that used it. It returns the bound sets so
// writeImageSizeConstructors can generate validating Go constructors for
// them - one per request type/field - which is how that per-request
// information stays usable after the schemas are merged into one type.
func consolidateImageSizes(doc *openapi.Document) []imageSizeSpec {
	canonical := &openapi.Schema{
		Title: "ImageSize",
		Type:  openapi.TypeObject,
		Description: "Pixel dimensions of an image. Valid width/height bounds are " +
			"specific to the request this is used in - see the field description " +
			"where it's used, or build one with the matching New*ImageSize " +
			"constructor in pkg/pixellab, which validates against the exact " +
			"bounds for that request.",
		Required: []string{"width", "height"},
		Properties: openapi.SchemaRefs{
			// Every real request bounds width/height to at least 1px; keeping
			// that floor on the shared schema (rather than leaving it
			// unbounded) also keeps this schema's shape from accidentally
			// matching some unrelated, truly-unconstrained width/height
			// schema (e.g. CharacterSize) and getting merged into it by
			// openapi-compress.
			"width":  {Value: &openapi.Schema{Title: "Width", Type: openapi.TypeInteger, Description: "Width in pixels.", Min: ptr(1.0)}},
			"height": {Value: &openapi.Schema{Title: "Height", Type: openapi.TypeInteger, Description: "Height in pixels.", Min: ptr(1.0)}},
		},
	}
	doc.Components.Schemas.Set("ImageSize", canonical)

	var specs []imageSizeSpec

	for name, s := range doc.Components.Schemas.ByIndex() {
		if s.Title != "ImageSize" || name == "ImageSize" {
			continue
		}

		usages, ok := imageSizeGroups[name]
		if !ok {
			log.Fatalf(
				"consolidateImageSizes: no imageSizeGroups entry for %q; "+
					"a new ImageSize variant appeared in the upstream API and "+
					"needs mapping to its Go request type(s) in main.go", name)
		}

		spec := imageSizeSpec{usages: usages}
		spec.widthMin, spec.widthMax = intBounds(s.Properties["width"])
		spec.heightMin, spec.heightMax = intBounds(s.Properties["height"])
		specs = append(specs, spec)

		if err := edit.MergeSchema(doc, name, "ImageSize", imageSizeUsageDescription(s)); err != nil {
			log.Fatal(err)
		}
	}

	return specs
}

// intBounds reads the integer minimum/maximum of a width or height property.
func intBounds(p *openapi.SchemaRef) (min, max int) {
	if p.Value.Min != nil {
		min = int(*p.Value.Min)
	}
	if p.Value.Max != nil {
		max = int(*p.Value.Max)
	}
	return min, max
}

// imageSizeUsageDescription composes the bounds information carried by an
// about-to-be-merged ImageSize variant into text, so it survives on the
// field that references it even after the schema itself is gone.
func imageSizeUsageDescription(s *openapi.Schema) string {
	var parts []string
	if d := strings.TrimSpace(s.Description); d != "" {
		parts = append(parts, d)
	}

	if wt := imageSizeAxisText(s.Properties["width"]); wt != "" {
		parts = append(parts, "Width: "+wt+".")
	}
	if ht := imageSizeAxisText(s.Properties["height"]); ht != "" {
		parts = append(parts, "Height: "+ht+".")
	}

	if len(s.Default) > 0 {
		parts = append(parts, fmt.Sprintf("Defaults to %s if omitted.", s.Default))
	}

	return strings.Join(parts, "\n")
}

// imageSizeGenericAxisDescriptions are per-field descriptions that add no
// information beyond the numeric bound already computed from min/max, and
// are dropped rather than repeated verbatim.
var imageSizeGenericAxisDescriptions = map[string]bool{
	"Image width in pixels":  true,
	"Width in pixels":        true,
	"Image height in pixels": true,
	"Height in pixels":       true,
}

// imageSizeAxisText formats one width or height property as "<bound> -
// <description>", keeping whichever half carries information.
func imageSizeAxisText(p *openapi.SchemaRef) string {
	bound := imageSizeFormatBound(p)

	d := strings.TrimSpace(p.Value.Description)
	if imageSizeGenericAxisDescriptions[d] {
		d = ""
	}
	if def := p.Value.Default; len(def) > 0 {
		d = strings.TrimSpace(fmt.Sprintf("%s (defaults to %s if omitted)", d, def))
	}

	switch {
	case bound != "" && d != "":
		return bound + " - " + d
	case bound != "":
		return bound
	default:
		return d
	}
}

// imageSizeFormatBound renders a property's minimum/maximum as text.
func imageSizeFormatBound(p *openapi.SchemaRef) string {
	min, max := p.Value.Min, p.Value.Max
	switch {
	case min != nil && max != nil && *min == *max:
		return fmt.Sprintf("%d px, fixed", int(*min))
	case min != nil && max != nil:
		return fmt.Sprintf("%d-%d px", int(*min), int(*max))
	case min != nil:
		return fmt.Sprintf("%d+ px", int(*min))
	case max != nil:
		return fmt.Sprintf("up to %d px", int(*max))
	default:
		return ""
	}
}

// writeImageSizeConstructors generates one validating constructor per
// request type/field that used to have its own ImageSize schema, so the
// per-request bounds openapi-compress used to enforce through distinct
// schemas keep being enforced now that they all share the ImageSize type.
func writeImageSizeConstructors(path string, specs []imageSizeSpec) error {
	var b strings.Builder

	b.WriteString("// Code generated by `go run ./main.go`. DO NOT EDIT.\n\n")
	b.WriteString("package pixellab\n\nimport \"fmt\"\n\n")

	for _, spec := range specs {
		for _, u := range spec.usages {
			if spec.widthMin == spec.widthMax && spec.heightMin == spec.heightMax {
				writeImageSizeVar(&b, u, spec)
			} else {
				writeImageSizeConstructor(&b, u, spec)
			}
		}
	}

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return fmt.Errorf("formatting %s: %w", path, err)
	}

	return os.WriteFile(path, formatted, 0o644)
}

// writeImageSizeVar emits a fixed-value global instead of a constructor when
// a usage's bounds pin width and height to a single size, since there's
// nothing left for a constructor to validate.
func writeImageSizeVar(b *strings.Builder, u imageSizeUsage, spec imageSizeSpec) {
	name := u.varName()

	fmt.Fprintf(b, "// %s is the fixed ImageSize required by %s.%s (%dx%d px).\n",
		name, u.typeName, u.field, spec.widthMin, spec.heightMin)
	fmt.Fprintf(b, "var %s = ImageSize{Width: %d, Height: %d}\n\n", name, spec.widthMin, spec.heightMin)
}

func writeImageSizeConstructor(b *strings.Builder, u imageSizeUsage, spec imageSizeSpec) {
	name := u.funcName()

	fmt.Fprintf(b, "// %s builds the ImageSize for %s.%s, checking width and height\n", name, u.typeName, u.field)
	fmt.Fprintf(b, "// against the bounds that request requires:\n")
	fmt.Fprintf(b, "//   - width: %s\n", imageSizeBoundComment(spec.widthMin, spec.widthMax))
	fmt.Fprintf(b, "//   - height: %s\n", imageSizeBoundComment(spec.heightMin, spec.heightMax))
	fmt.Fprintf(b, "func %s(width, height int) (ImageSize, error) {\n", name)

	writeImageSizeCheck(b, "width", spec.widthMin, spec.widthMax)
	writeImageSizeCheck(b, "height", spec.heightMin, spec.heightMax)

	b.WriteString("\treturn ImageSize{Width: width, Height: height}, nil\n")
	b.WriteString("}\n\n")
}

func imageSizeBoundComment(min, max int) string {
	if min == max {
		return fmt.Sprintf("exactly %d", min)
	}
	return fmt.Sprintf("%d-%d", min, max)
}

func writeImageSizeCheck(b *strings.Builder, axis string, min, max int) {
	if min == max {
		fmt.Fprintf(b, "\tif %s != %d {\n", axis, min)
		fmt.Fprintf(b, "\t\treturn ImageSize{}, fmt.Errorf(\"%s must be exactly %d pixels, got %%d\", %s)\n", axis, min, axis)
		b.WriteString("\t}\n")
		return
	}

	fmt.Fprintf(b, "\tif %s < %d || %s > %d {\n", axis, min, axis, max)
	fmt.Fprintf(b, "\t\treturn ImageSize{}, fmt.Errorf(\"%s must be between %d and %d pixels, got %%d\", %s)\n", axis, min, max, axis)
	b.WriteString("\t}\n")
}
