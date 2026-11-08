# Architecture

`go generate` runs `cmd/generate`, which builds the library from PixelLab's own
specification in five steps:

1. **Fetch** (`fetch.go`): replace `api/openapi-v2_original.json`, indented so a
   diff shows what changed upstream, and `api/llms.txt` with their newest. In a
   Claude Code session (`CLAUDECODE`), the copies are used as they are.
2. **Transform** (`transform.go`, `names.go`, `bodies.go`): put the version in
   every path, drop the null of FastAPI's optional values and the operations
   that are only aliases of another, and name every operation after its method
   and path, and every body after its operation.
3. **Passes**: `openapi-enrich` with `api/interactions.json`, then
   `openapi-flatten`, `openapi-compress` and `openapi-flatten` again, as the
   commands of the same names would.
4. **Consolidate** (`consolidate.go`): name the bodies compress merged, and merge
   what upstream repeats per endpoint: the enums of `enumFamilies`, every
   width-and-height object into `ImageSize`, and every object of one URL per
   direction into `RotationURLs`.
5. **Generate**: write `api/openapi.json`, the library with `openapi-codegen`,
   and `pkg/pixellab/image_size.go`, a checked constructor per request size.

Every rule finds what it changes by what upstream cannot rename without
changing the API, a method and path or a schema's shape, never by a name a pass
made up. A table entry that no longer matches anything fails the run, as does a
name two operations or bodies would get.

`pkg/pixellab/errors.go` and `types.go` are written by hand.
