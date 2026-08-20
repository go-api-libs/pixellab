module github.com/go-api-libs/pixellab

go 1.26.5

tool (
	github.com/MarkRosemaker/openapi-codegen/cmd/openapi-codegen
	github.com/MarkRosemaker/openapi-compress/cmd/openapi-compress
	github.com/MarkRosemaker/openapi-enrich/cmd/openapi-enrich
	github.com/MarkRosemaker/openapi-flatten/cmd/openapi-flatten
)

require (
	github.com/MarkRosemaker/openapi v0.0.0-20260820160151-de7a41b136a1
	github.com/MarkRosemaker/openapi-codegen v0.0.0-20260819142118-96bce8b45793
	github.com/MarkRosemaker/openapi-enrich v0.0.0-20260820161439-d0d5f1c7112d
	github.com/ettle/strcase v0.2.0
	github.com/go-api-libs/api v0.0.0-20260705004954-dad48fbb4ab2
	github.com/google/uuid v1.6.0
)

require (
	cloud.google.com/go v0.123.0 // indirect
	github.com/MarkRosemaker/errpath v0.0.0-20260425165607-bbd4959d04d9 // indirect
	github.com/MarkRosemaker/json2yaml v0.0.0-20260507220148-d6cc0d01bff0 // indirect
	github.com/MarkRosemaker/jsonutil v0.0.0-20260718153618-78b5039427a4 // indirect
	github.com/MarkRosemaker/openapi-compare v0.0.0-20260820160232-bb2bacb1ba6e // indirect
	github.com/MarkRosemaker/openapi-compress v0.0.0-20260819161509-4f330f849f38 // indirect
	github.com/MarkRosemaker/openapi-flatten v0.0.0-20260820160336-c4c88aa56e6a // indirect
	github.com/MarkRosemaker/openapi-merge v0.0.0-20260820160232-909cf279cd8b // indirect
	github.com/MarkRosemaker/ordmap v0.0.0-20260820160128-6fdc6d1395b1 // indirect
	github.com/MarkRosemaker/yaml v0.0.0-20260508005758-fe21a538b084 // indirect
	github.com/MarkRosemaker/yaml2json v0.0.0-20260507220136-7748efc522b2 // indirect
	github.com/go-api-libs/types v0.0.0-20251210072721-82754f56609d // indirect
	github.com/spf13/afero v1.15.0 // indirect
	golang.org/x/exp v0.0.0-20260820142414-ca536658362e // indirect
	golang.org/x/mod v0.40.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/tools v0.49.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	mvdan.cc/gofumpt v0.11.0 // indirect
)
