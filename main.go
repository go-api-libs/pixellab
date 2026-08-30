package main

import (
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	"fmt"
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

	i := 0
	for name, s := range doc.Components.Schemas.ByIndex() {
		switch s.Title {
		case "ImageSize":
			i++
			newName := s.Title
			if i > 1 {
				newName = fmt.Sprintf("%s%d", s.Title, i)
			}
			if err := edit.RenameSchema(doc, name, newName); err != nil {
				log.Fatal(err)
			}
		}

	}

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
	// app__endpoints__external__v2__image_to_pixelart__ImageSize

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
