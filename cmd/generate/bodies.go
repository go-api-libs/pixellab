package main

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/MarkRosemaker/openapi"
	edit "github.com/MarkRosemaker/openapi-edit"
	"github.com/ettle/strcase"
)

// bodyNames names the bodies several operations share after what they hold, by what one of the operations would
// call it, as the body GenerateImage returns, which 18 other operations return too. An entry that no operation's body
// matches fails the run.
var bodyNames = map[string]string{
	"GenerateImageResponse":          "JobResponse",
	"ResizeResponse":                 "ImageResponse",
	"AnimateWithTextResponse":        "ImagesResponse",
	"CreateCharacterProResponse":     "CharacterJobResponse",
	"CreateMapObjectResponse":        "ObjectJobResponse",
	"AnimatePixminimaxResponse":      "AnimationJobResponse",
	"EnhancePixenPromptResponse":     "EnhancedPromptResponse",
	"CreateTilesProResponse":         "TileJobResponse",
	"CreateTilesetResponse":          "TilesetJobResponse",
	"DeleteTilesProResponse":         "DeleteTileResponse",
	"DeleteObjectAnimationsResponse": "DeleteAnimationsResponse",
	"CreateUploadRequest":            "UploadRequest",
}

// nameBodies names each request and response body after the one operation it belongs to, as CreateCharacterRequest.
// A body operations share is named from bodyNames, or else after the words their names for it share, as
// UpdateTagsRequest for UpdateCharacterTagsRequest and UpdateObjectTagsRequest, or else after its title.
//
// It runs before the passes, so that flatten names what it moves out of a body after the body, and after them, for
// the bodies compress merged; only then do all of bodyNames have to be used.
func nameBodies(doc *openapi.Document, final bool) error {
	names := map[string][]string{} // by component schema, what each operation would call it

	for _, item := range doc.Paths.ByIndex() {
		for _, op := range item.Operations {
			if op.RequestBody != nil && op.RequestBody.Value != nil {
				for _, mt := range op.RequestBody.Value.Content.ByIndex() {
					addBodyName(names, mt.Schema, op.OperationID+"Request")
				}
			}

			for code, rsp := range op.Responses.ByIndex() {
				if rsp.Value == nil || !strings.HasPrefix(string(code), "2") {
					continue
				}

				for _, mt := range rsp.Value.Content.ByIndex() {
					addBodyName(names, mt.Schema, op.OperationID+"Response")
				}
			}
		}
	}

	unused := maps.Clone(bodyNames)
	to := map[string]string{}

	for schema, candidates := range names {
		candidates = slices.Compact(slices.Sorted(slices.Values(candidates)))

		name := ""

		for _, c := range candidates {
			if n, ok := bodyNames[c]; ok {
				name = n

				delete(unused, c)
			}
		}

		switch {
		case name != "":
		case len(candidates) == 1:
			name = candidates[0]
		default:
			name = cmp.Or(commonWords(candidates), strcase.ToGoPascal(doc.Components.Schemas[schema].Title))
		}

		if name != schema {
			to[schema] = name
		}
	}

	if final && len(unused) > 0 {
		return fmt.Errorf("bodyNames names bodies upstream no longer has: %s",
			strings.Join(slices.Sorted(maps.Keys(unused)), ", "))
	}

	if err := edit.RenameSchemas(doc, to); err != nil {
		if _, ok := errors.AsType[*edit.ErrSchemaExists](err); ok {
			return fmt.Errorf("naming bodies, add an entry to bodyNames: %w", err)
		}

		return err
	}

	return nil
}

// commonWords is the name made of the words the names begin and end with in common, if they share at least one at
// either end and three in all, as UpdateTagsRequest for UpdateCharacterTagsRequest and UpdateObjectTagsRequest.
func commonWords(names []string) string {
	split := make([][]string, len(names))
	for i, n := range names {
		split[i] = words(n)
	}

	prefix, suffix, shortest := split[0], split[0], len(split[0])

	for _, words := range split[1:] {
		shortest = min(shortest, len(words))

		n := 0
		for n < len(prefix) && n < len(words) && prefix[n] == words[n] {
			n++
		}

		prefix = prefix[:n]

		m := 0
		for m < len(suffix) && m < len(words) && suffix[len(suffix)-1-m] == words[len(words)-1-m] {
			m++
		}

		suffix = suffix[len(suffix)-m:]
	}

	if n := len(prefix) + len(suffix); len(prefix) == 0 || len(suffix) == 0 || n < 3 || n > shortest {
		return ""
	}

	return strings.Join(prefix, "") + strings.Join(suffix, "")
}

// words splits a Go name into its words, as Create, UI, Asset and Request in CreateUIAssetRequest.
func words(name string) []string {
	var out []string

	start := 0

	for i := 1; i < len(name); i++ {
		prev, cur := rune(name[i-1]), rune(name[i])
		next := rune(0)

		if i+1 < len(name) {
			next = rune(name[i+1])
		}

		if unicode.IsUpper(cur) && (unicode.IsLower(prev) || unicode.IsDigit(prev) || unicode.IsUpper(prev) && unicode.IsLower(next)) ||
			unicode.IsDigit(cur) && !unicode.IsDigit(prev) {
			out = append(out, name[start:i])
			start = i
		}
	}

	return append(out, name[start:])
}

func addBodyName(names map[string][]string, s *openapi.Schema, name string) {
	if s != nil {
		s = nonNull(s) // an optional body, which upstream writes as one that may be null
	}

	if s != nil && s.Ref != nil {
		schema := strings.TrimPrefix(s.Ref.Identifier, schemaRefPrefix)
		names[schema] = append(names[schema], name)
	}
}

// moduleName matches the name FastAPI gives a model it found in two modules, such as
// app__endpoints__external__v2__generate_image_v2__ReferenceImage.
var reModuleName = regexp.MustCompile(`^app__endpoints__external__v2?__(\w+?)__(\w+)$`)

// nameModels gives the models FastAPI named after their module, or with an -Input or -Output suffix, Go names: their
// own, or, where two share it, prefixed with their endpoint, as Generate8RotationsV2ReferenceImage.
func nameModels(doc *openapi.Document) error {
	byTitle := map[string][]string{}

	for name := range doc.Components.Schemas {
		if m := reModuleName.FindStringSubmatch(name); m != nil {
			byTitle[m[2]] = append(byTitle[m[2]], name)
		}
	}

	to := map[string]string{}

	for title, names := range byTitle {
		for _, name := range names {
			if _, taken := doc.Components.Schemas[title]; len(names) == 1 && !taken {
				to[name] = title
			} else {
				to[name] = strcase.ToGoPascal(reModuleName.FindStringSubmatch(name)[1]) + title
			}
		}
	}

	for name := range doc.Components.Schemas {
		if base, ok := strings.CutSuffix(name, "-Input"); ok {
			to[name] = base
		} else if base, ok := strings.CutSuffix(name, "-Output"); ok {
			to[name] = base + "Output"
		}
	}

	return edit.RenameSchemas(doc, to)
}
