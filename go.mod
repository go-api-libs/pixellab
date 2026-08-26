module github.com/go-api-libs/pixellab

go 1.27

tool (
	github.com/MarkRosemaker/openapi-codegen/cmd/openapi-codegen
	github.com/MarkRosemaker/openapi-compress/cmd/openapi-compress
	github.com/MarkRosemaker/openapi-enrich/cmd/openapi-enrich
	github.com/MarkRosemaker/openapi-flatten/cmd/openapi-flatten
)

require (
	github.com/MarkRosemaker/openapi v0.0.0-20260824220141-39020beab076
	github.com/MarkRosemaker/openapi-codegen v0.0.0-20260826041226-a9963753fadf
	github.com/MarkRosemaker/openapi-enrich v0.0.0-20260825160830-c495352ed663
	github.com/ettle/strcase v0.2.0
	github.com/go-api-libs/api v0.0.0-20260822224219-ad37a7b68775
	github.com/google/uuid v1.6.0
)

require (
	cloud.google.com/go v0.123.0 // indirect
	github.com/MarkRosemaker/errpath v0.0.0-20260425165607-bbd4959d04d9 // indirect
	github.com/MarkRosemaker/json2yaml v0.0.0-20260820194645-20aa3a7082f4 // indirect
	github.com/MarkRosemaker/jsonutil v0.0.0-20260822121424-820b30d4cb47 // indirect
	github.com/MarkRosemaker/openapi-compare v0.0.0-20260824220227-96aa2a51f115 // indirect
	github.com/MarkRosemaker/openapi-compress v0.0.0-20260825222001-c4cd92f5cea6 // indirect
	github.com/MarkRosemaker/openapi-edit v0.0.0-20260824220228-fbdc202814fd // indirect
	github.com/MarkRosemaker/openapi-flatten v0.0.0-20260825130739-b5b6ef5e16d7 // indirect
	github.com/MarkRosemaker/openapi-merge v0.0.0-20260825130414-17e28863a04c // indirect
	github.com/MarkRosemaker/ordmap v0.0.0-20260824220120-9c8900dd7193 // indirect
	github.com/MarkRosemaker/yaml v0.0.0-20260820194724-a126111ba94f // indirect
	github.com/MarkRosemaker/yaml2json v0.0.0-20260820194543-4c959435803e // indirect
	github.com/go-api-libs/types v0.0.0-20260821232109-0cf45378823e // indirect
	github.com/spf13/afero v1.15.0 // indirect
	golang.org/x/exp v0.0.0-20260824195058-e88cd73687aa // indirect
	golang.org/x/mod v0.40.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/tools v0.49.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	mvdan.cc/gofumpt v0.11.0 // indirect
)
