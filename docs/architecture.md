# Architecture

`go generate` runs `cmd/generate`, which builds the library from PixelLab's own
specification in five steps:

1. **Fetch** (`fetch.go`): replace `api/openapi-v2_original.json`, indented so a
   diff shows what changed upstream, and `api/llms.txt` with their newest. In a
   Claude Code session (`CLAUDECODE`), the copies are used as they are.
2. **Transform** (`transform.go`, `names.go`, `bodies.go`): put the version in
   every path, move the bounds upstream declares beside a reference, as an
   image size's, into a copy of what it refers to, tell in each property of
   an image size what its width and height say beyond their bounds, before
   compress merges the sizes alike but for that, drop the JSON of no
   schema FastAPI declares beside a download's own media type, and the operations
   that are only aliases of another, let a 422's detail be a message as well as
   validation errors, and name every operation after its method
   and path, and every body after its operation.
   Type every string the API holds to one format (`formats.go`): an ID name
   upstream types anywhere is typed everywhere, and `formats` gives what
   upstream leaves bare, so the library has a `uuid.UUID` or `url.URL`, not a
   `string`. This runs again after the passes, for what the interactions show.
3. **Passes**: `openapi-enrich` with `api/interactions.json`, then
   `openapi-flatten`, `openapi-compress` and `openapi-flatten` again, as the
   commands of the same names would.
4. **Consolidate** (`consolidate.go`): name the bodies compress merged, and merge
   what upstream repeats per endpoint: the enums of `enumFamilies`, every
   width-and-height object into `ImageSize`, and every object of one URL per
   direction into `RotationURLs`. An enum joins its family only where it
   shares a value with another, and a plain string of a family's property, as
   a response's `direction`, refers to the merged enum. A job's images, which
   upstream gives no schema, are named by their shape: `Image`,
   `DirectionalImages`, `Images` and `JobImages`, either of the two.
5. **Generate**: write `api/openapi.json`, the library with `openapi-codegen`,
   and `pkg/pixellab/image_size.go`, a checked constructor per request size:
   its bounds and step, and what `sizeRules` gives of upstream's words, as
   Pixen's maximum area or the sides a tile may have.
   The library's tests replay `api/interactions.json` through the client, so a
   response the API sent that the library cannot read fails them.

Every rule finds what it changes by what upstream cannot rename without
changing the API, a method and path or a schema's shape, never by a name a pass
made up. A table entry that no longer matches anything fails the run, as does a
name two operations or bodies would get.

`pkg/pixellab/errors.go` and `types.go` are written by hand.
