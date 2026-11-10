// Command generate builds api/openapi.json from PixelLab's own specification, fetched anew on every run, and the
// library in pkg/pixellab from it.
package main

import (
	"context"
	"fmt"
	"log"
	"slices"

	"github.com/MarkRosemaker/cassette"
	"github.com/MarkRosemaker/openapi"
	codegen "github.com/MarkRosemaker/openapi-codegen"
	compress "github.com/MarkRosemaker/openapi-compress"
	enrich "github.com/MarkRosemaker/openapi-enrich"
	flatten "github.com/MarkRosemaker/openapi-flatten"
)

const (
	originalPath     = "api/openapi-v2_original.json"
	specPath         = "api/openapi.json"
	interactionsPath = "api/interactions.json"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	if err := fetchUpstream(ctx); err != nil {
		return fmt.Errorf("fetching upstream: %w", err)
	}

	doc, err := openapi.LoadFromFile(originalPath)
	if err != nil {
		return err
	}

	if err := dropAliases(doc); err != nil {
		return err
	}

	if err := nameOperations(doc); err != nil {
		return err
	}

	if err := transform(doc); err != nil {
		return err
	}

	if err := typeStrings(doc, false); err != nil {
		return err
	}

	if err := nameModels(doc); err != nil {
		return err
	}

	if err := nameBodies(doc, false); err != nil {
		return err
	}

	if err := doc.Validate(); err != nil {
		return fmt.Errorf("transformed original: %w", err)
	}

	ias, err := cassette.InteractionsReadFile(interactionsPath)
	if err != nil {
		return err
	}

	// the placeholder for the next interaction to record has no response
	ias = slices.DeleteFunc(ias, func(ia *cassette.Interaction) bool { return ia.Response.StatusCode == 0 })

	if err := applyPasses(doc, ias); err != nil {
		return err
	}

	if err := typeStrings(doc, true); err != nil {
		return err
	}

	if err := nameBodies(doc, true); err != nil {
		return err
	}

	if err := consolidateEnums(doc); err != nil {
		return err
	}

	if err := nameImages(doc); err != nil {
		return err
	}

	if err := consolidateRotations(doc); err != nil {
		return err
	}

	sizes, err := consolidateImageSizes(doc)
	if err != nil {
		return err
	}

	if err := goNames(doc); err != nil {
		return err
	}

	doc.Components.SortMaps()

	if err := doc.Validate(); err != nil {
		return fmt.Errorf("generated specification: %w", err)
	}

	if err := doc.WriteToFile(specPath); err != nil {
		return err
	}

	// the recorded interactions are replayed through the client, as a test that it reads what the API answers
	if err := codegen.Generate(codegen.Config{
		Spec:         doc,
		PackageName:  "pixellab",
		OutputDir:    "pkg/pixellab",
		Types:        true,
		Client:       true,
		ClientTest:   true,
		Debug:        true,
		Interactions: ias,
	}); err != nil {
		return fmt.Errorf("generating the library: %w", err)
	}

	return writeImageSizes(sizes)
}

// applyPasses runs the passes the openapi-* commands would, in the order a spec goes through them, then sorts as they
// do.
func applyPasses(doc *openapi.Document, ias cassette.Interactions) error {
	flattenDoc := func(d *openapi.Document) error { return flatten.Document(d, flatten.Config{MarkOrigin: true}) }

	for _, pass := range []struct {
		name string
		run  func(*openapi.Document) error
	}{
		{"enrich", func(d *openapi.Document) error { return enrich.Enrich(d, ias) }},
		{"flatten", flattenDoc},
		{"compress", func(d *openapi.Document) error { return compress.Document(d, compress.Config{}) }},
		{"flatten again", flattenDoc},
	} {
		if err := pass.run(doc); err != nil {
			return fmt.Errorf("%s: %w", pass.name, err)
		}
	}

	for _, path := range doc.Paths {
		for _, op := range path.Operations {
			op.Responses.Sort()
		}
	}

	doc.Components.SortMaps()

	return nil
}
