# api/openapi.json is generated

`go generate` rebuilds `api/openapi.json` from scratch on every run, from
PixelLab's specification in `api/openapi-v2_original.json`, so an edit to it
is lost on the next run. Change `cmd/generate` instead.

Outside Claude Code, each run first replaces `api/openapi-v2_original.json`
and `api/llms.txt` with PixelLab's newest. Where upstream changed in a way a
rule no longer fits, the run fails and names the table to update:
`operationNames`, `bodyNames` or `enumFamilies`.
