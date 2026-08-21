module github.com/go-api-libs/pixellab

go 1.27

tool (
	github.com/MarkRosemaker/openapi-codegen/cmd/openapi-codegen
	github.com/MarkRosemaker/openapi-compress/cmd/openapi-compress
	github.com/MarkRosemaker/openapi-enrich/cmd/openapi-enrich
	github.com/MarkRosemaker/openapi-flatten/cmd/openapi-flatten
)

require (
	github.com/MarkRosemaker/openapi v0.0.0-20260821232459-82751aa58c4c
	github.com/MarkRosemaker/openapi-codegen v0.0.0-20260820215922-0dc036f27e70
	github.com/MarkRosemaker/openapi-enrich v0.0.0-20260821233606-5fb5e77e0883
	github.com/ettle/strcase v0.2.0
	github.com/go-api-libs/api v0.0.0-20260821155530-ebc29700b6ea
	github.com/google/uuid v1.6.0
)

require (
	cloud.google.com/go v0.123.0 // indirect
	github.com/MarkRosemaker/errpath v0.0.0-20260425165607-bbd4959d04d9 // indirect
	github.com/MarkRosemaker/json2yaml v0.0.0-20260820194645-20aa3a7082f4 // indirect
	github.com/MarkRosemaker/jsonutil v0.0.0-20260820212410-12ba6685df41 // indirect
	github.com/MarkRosemaker/openapi-compare v0.0.0-20260821232534-24b42373a000 // indirect
	github.com/MarkRosemaker/openapi-compress v0.0.0-20260821234159-79c61039f7a4 // indirect
	github.com/MarkRosemaker/openapi-flatten v0.0.0-20260821232550-ac155e73eb12 // indirect
	github.com/MarkRosemaker/openapi-merge v0.0.0-20260821232535-ef5698a4147f // indirect
	github.com/MarkRosemaker/ordmap v0.0.0-20260821225345-9c948bb0ea43 // indirect
	github.com/MarkRosemaker/yaml v0.0.0-20260820194724-a126111ba94f // indirect
	github.com/MarkRosemaker/yaml2json v0.0.0-20260820194543-4c959435803e // indirect
	github.com/go-api-libs/types v0.0.0-20260821232109-0cf45378823e // indirect
	github.com/spf13/afero v1.15.0 // indirect
	golang.org/x/exp v0.0.0-20260820142414-ca536658362e // indirect
	golang.org/x/mod v0.40.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/tools v0.49.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	mvdan.cc/gofumpt v0.11.0 // indirect
)
