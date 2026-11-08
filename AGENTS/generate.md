# When upstream outgrows cmd/generate

`cmd/generate` names and merges what PixelLab's specification holds by rules,
with three tables for what no rule can know. When upstream changes in a way a
rule no longer fits, `go generate` fails and names the one to update:

- `operationNames` in `names.go`: an operation's name, by method and path.
- `bodyNames` in `bodies.go`: a body several operations share.
- `enumFamilies` in `consolidate.go`: the properties whose enums are merged.
