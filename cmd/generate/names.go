package main

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/MarkRosemaker/openapi"
	edit "github.com/MarkRosemaker/openapi-edit"
	"github.com/ettle/strcase"
)

// operationNames overrides the name derived from an operation's method and path, by "METHOD /path" as the original
// writes it, where the derived one reads badly. An entry that names no operation fails the run: upstream changed.
var operationNames = map[string]string{
	"GET /llms.txt":                              "GetAPIDocs",
	"POST /vfx":                                  "CreateVfx",
	"POST /uploads/{ticket}":                     "UploadWithTicket",
	"DELETE /background-jobs/{job_id}":           "CancelBackgroundJob",
	"POST /vocal-animation":                      "CreateVocalAnimation",
	"POST /portrait-character-pro":               "CreatePortraitCharacterPro",
	"POST /create-tileset-sidescroller":          "CreateSidescrollerTileset",
	"GET /tilesets-sidescroller":                 "ListSidescrollerTilesets",
	"GET /tilesets-sidescroller/{tileset_id}":    "GetSidescrollerTileset",
	"DELETE /tilesets-sidescroller/{tileset_id}": "DeleteSidescrollerTileset",
}

// reAlias matches the summary of an operation that only repeats another, as in "Create a tileset · alias of
// /create-tileset".
var reAlias = regexp.MustCompile(`alias of (/\S+)`)

// dropAliases removes the operations that repeat another one, which a client needs only once.
func dropAliases(doc *openapi.Document) error {
	for path, item := range doc.Paths.ByIndex() {
		for method, op := range item.Operations {
			m := reAlias.FindStringSubmatch(op.Summary)
			if m == nil {
				continue
			}

			if target, ok := doc.Paths[openapi.Path(m[1])]; !ok || operation(target, method) == nil {
				return fmt.Errorf("%s %s is an alias of %s %s, which upstream does not have", method, path, method, m[1])
			}

			item.SetOperation(method, nil)
		}

		if operation(item, "") == nil {
			delete(doc.Paths, path)
		}
	}

	return nil
}

// operation is the path item's operation of the method, or any of its operations if method is empty.
func operation(item *openapi.PathItem, method string) *openapi.Operation {
	for m, op := range item.Operations {
		if method == "" || m == method {
			return op
		}
	}

	return nil
}

// reVersion matches a version suffix, as in generate-image-v2.
var reVersion = regexp.MustCompile(`-v\d+$`)

// nameOperations gives every operation of the original its name, from operationNames or else from its method and
// path, and fails where an override names no operation or two operations get the same name.
func nameOperations(doc *openapi.Document) error {
	unused := maps.Clone(operationNames)
	actions, resources := actionSegments(doc), resourceSegments(doc)
	byName := map[string]string{}

	var errs []error

	for path, item := range doc.Paths.ByIndex() {
		for method, op := range item.Operations {
			key := method + " " + string(path)

			name, ok := operationNames[key]
			if ok {
				delete(unused, key)
			}

			if name == "" {
				name = deriveName(strings.ToLower(method), string(path), actions, resources)
			}

			if other, ok := byName[name]; ok {
				errs = append(errs, fmt.Errorf("%s and %s are both named %s", other, key, name))
			}

			name = strcase.ToGoPascal(name) // as openapi-codegen names the method, and so the bodies after it
			byName[name] = key
			op.OperationID = name
		}
	}

	if len(unused) > 0 {
		errs = append(errs, fmt.Errorf("operationNames names operations upstream no longer has: %s",
			strings.Join(slices.Sorted(maps.Keys(unused)), ", ")))
	}

	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("naming operations, fix operationNames:\n%w", err)
	}

	return nil
}

// actionSegments are the first path segments that act, such as create-image-pixflux, without their version suffix
// where the action has no other version: generate-image-v2 is generate-image, but animate-with-text-v2 and
// generate-8-rotations-v2 stay, since animate-with-text and generate-8-rotations-v3 exist too.
func actionSegments(doc *openapi.Document) map[string]string {
	segs := map[string]bool{}
	for path := range doc.Paths {
		segs[firstSegment(string(path))] = true
	}

	versions := map[string]int{}
	for seg := range segs {
		versions[reVersion.ReplaceAllString(seg, "")]++
	}

	actions := map[string]string{}

	for seg := range segs {
		if short := reVersion.ReplaceAllString(seg, ""); versions[short] == 1 {
			actions[seg] = short
		} else {
			actions[seg] = seg
		}
	}

	return actions
}

// resourceSegments are the first path segments that name resources: those an ID follows in some path, as in
// /characters/{character_id}. An action a job ID follows, as in /generate-font-pro/{job_id}, is none.
func resourceSegments(doc *openapi.Document) map[string]bool {
	resources := map[string]bool{}

	for path := range doc.Paths {
		segs := strings.Split(strings.TrimPrefix(string(path), "/"), "/")
		if len(segs) > 1 && strings.HasPrefix(segs[1], "{") && !isJob(segs) {
			resources[segs[0]] = true
		}
	}

	return resources
}

// isJob reports whether the path segments are those of an action's job, as in /generate-font-pro/{job_id}.
func isJob(segs []string) bool {
	return len(segs) == 2 && segs[1] == "{job_id}" && !isPlural(segs[0])
}

func firstSegment(path string) string {
	seg, _, _ := strings.Cut(strings.TrimPrefix(path, "/"), "/")
	return seg
}

// deriveName names an operation after its method and path:
//
//   - GET /characters lists them, GET /characters/{id} gets one, DELETE deletes and PATCH updates. A resource is
//     singular where an ID follows it: DELETE /characters/{id}/animations is DeleteCharacterAnimations.
//   - POST to a resource creates one, POST /characters/animations creates a character animation, and POST to an
//     action, such as /create-image-pixflux, is named after it. GET of an action's job, /generate-font-pro/{job_id},
//     gets the job.
//   - POST below an item does to it what it says: /objects/{id}/states creates a state, /objects/{id}/portrait sets
//     it, and /objects/{id}/select-frames selects its frames.
func deriveName(method, path string, actions map[string]string, resources map[string]bool) string {
	segs := strings.Split(strings.TrimPrefix(path, "/"), "/")
	first := actions[segs[0]]
	pascal := func(words ...string) string { return strcase.ToGoPascal(strings.Join(words, " ")) }

	if isJob(segs) {
		if method == "get" {
			return "Get" + pascal(first, "job")
		}

		return pascal(first)
	}

	var words []string

	for i, seg := range segs {
		if strings.HasPrefix(seg, "{") {
			continue
		}

		if i == 0 {
			seg = first
		}

		if i+1 < len(segs) && strings.HasPrefix(segs[i+1], "{") {
			seg = singular(seg)
		}

		words = append(words, seg)
	}

	endsWithID := strings.HasPrefix(segs[len(segs)-1], "{")

	switch {
	case method == "get" && !endsWithID && len(segs) == 1 && resources[segs[0]]:
		return "List" + pascal(words...)
	case method == "get":
		return "Get" + pascal(words...)
	case method == "delete":
		return "Delete" + pascal(words...)
	case method == "patch" || method == "put":
		return "Update" + pascal(words...)
	case len(segs) == 1 && resources[segs[0]]:
		return "Create" + pascal(singular(first))
	case !resources[segs[0]] || endsWithID:
		return pascal(words...)
	}

	// POST below a resource, or one of its items
	owner, sub := words[:len(words)-1], words[len(words)-1]
	owner[0] = singular(owner[0])

	switch verb, rest, _ := strings.Cut(sub, "-"); {
	case rest == "" && isPlural(sub):
		return "Create" + pascal(append(owner, singular(sub))...)
	case rest == "":
		return "Set" + pascal(append(owner, sub)...)
	default:
		return pascal(verb) + pascal(append(owner, rest)...)
	}
}

func isPlural(seg string) bool {
	return strings.HasSuffix(seg, "s") && !strings.HasSuffix(seg, "ss")
}

func singular(seg string) string {
	switch {
	case strings.HasSuffix(seg, "ies"):
		return strings.TrimSuffix(seg, "ies") + "y"
	case isPlural(seg):
		return strings.TrimSuffix(seg, "s")
	default:
		return seg
	}
}

// initialisms are the words Go writes in capitals, in the case a name from upstream may have them in.
var initialisms = map[string]string{"Url": "URL", "Urls": "URLs", "Ui": "UI", "Id": "ID", "Ids": "IDs"}

// goName writes the initialisms of a name as Go does, as CharacterRotationURLs for CharacterRotationUrls.
func goName(name string) string {
	ws := words(name)
	for i, w := range ws {
		if v, ok := initialisms[w]; ok {
			ws[i] = v
		}
	}

	return strings.Join(ws, "")
}

// goNames writes the initialisms of every component schema's name as Go does.
func goNames(doc *openapi.Document) error {
	to := map[string]string{}

	for name := range doc.Components.Schemas {
		if g := goName(name); g != name {
			to[name] = g
		}
	}

	return edit.RenameSchemas(doc, to)
}
