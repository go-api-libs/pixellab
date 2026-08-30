package main

import (
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/MarkRosemaker/openapi"
	codegen "github.com/MarkRosemaker/openapi-codegen"
	enrich "github.com/MarkRosemaker/openapi-enrich"
	"github.com/ettle/strcase"
)

const (
	rawPath  = "/Users/mark/Downloads/openapi.json"
	specPath = "api/openapi.json"
)

var reDirection = regexp.MustCompile(`","default":"(east|high top-down)"`)

var opIDMapping = map[string]string{
	// "GenerateImageV2GenerateImageV2Post": "GenerateImage",
}

func main() {
	data, err := os.ReadFile(rawPath)
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

	for _, p := range doc.Paths {
		for _, p := range p.Parameters {
			s := p.Value.Schema
			if s == nil {
				continue
			}

			improveSchema(s)
		}

		for _, op := range p.Operations {
			op.OperationID = strcase.ToGoPascal(op.OperationID)
			if opID, ok := opIDMapping[op.OperationID]; ok {
				op.OperationID = strcase.ToGoPascal(opID)
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

	for name, s := range doc.Components.Schemas {
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

		// s.Title = ""
		for _, p := range s.Properties {
			if p.Value.Type != "" {
				continue
			}
		}
	}

	if err := doc.Validate(); err != nil {
		log.Fatalf("Validation ERROR: %v", err)
	}

	for _, path := range doc.Paths {
		for _, op := range path.Operations {
			op.Responses.Sort()
		}
	}
	doc.Components.SortMaps()

	if err := enrich.Enrich(doc, nil); err != nil {
		log.Fatalf("enrich: %v", err)
	}

	// edit.RenameSchema(doc, "")
	// TODO: rename via openapi-edit

	// if err := flatten.Document(doc); err != nil {
	// 	log.Fatalf("flatten: %v", err)
	// }

	// if err := compress.Document(doc, compress.Config{}); err != nil {
	// 	log.Fatalf("compress: %v", err)
	// }

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
	// s.Title = ""

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
