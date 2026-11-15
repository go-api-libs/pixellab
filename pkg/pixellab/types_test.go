package pixellab_test

import (
	"encoding/json/v2"
	"iter"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/MarkRosemaker/cassette"
	"github.com/go-api-libs/pixellab/pkg/pixellab"
)

// rotations is the rotation_urls of a recorded character, read through the client: a 4-directional one, whose
// diagonals the API sends as null, or an 8-directional one.
func rotations(t *testing.T, eight bool) pixellab.RotationURLs {
	t.Helper()

	ias, err := cassette.InteractionsReadFile(filepath.Join("..", "..", "api", "interactions.json"))
	if err != nil {
		t.Fatal(err)
	}

	c, err := pixellab.NewClient(pixellab.WithBearer("**************************************************"), pixellab.WithHTTPClient(&http.Client{Transport: cassette.Replay(ias)}))
	if err != nil {
		t.Fatal(err)
	}

	id := uuid.MustParse("11480c99-c7dc-4f63-aa67-7d1a09435cea")
	if eight {
		id = uuid.MustParse("a34046fa-fd42-43d5-828c-fa4a6d062f70")
	}

	ch, err := c.GetCharacter(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}

	if ch.RotationURLs == nil {
		t.Fatal("no rotation_urls")
	}

	return *ch.RotationURLs
}

func TestRotationURLs(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		eight bool
		want  []pixellab.Direction
	}{
		{false, []pixellab.Direction{
			pixellab.DirectionSouth, pixellab.DirectionEast, pixellab.DirectionNorth, pixellab.DirectionWest,
		}},
		{true, []pixellab.Direction{
			pixellab.DirectionSouth, pixellab.DirectionSouthEast, pixellab.DirectionEast, pixellab.DirectionNorthEast,
			pixellab.DirectionNorth, pixellab.DirectionNorthWest, pixellab.DirectionWest, pixellab.DirectionSouthWest,
		}},
	} {
		u := rotations(t, tc.eight)

		if got := slices.Collect(u.Keys()); !slices.Equal(got, tc.want) {
			t.Errorf("eight %v: Keys = %v, want %v", tc.eight, got, tc.want)
		}

		if got := u.Length(); got != len(tc.want) {
			t.Errorf("eight %v: Length = %d, want %d", tc.eight, got, len(tc.want))
		}

		if got := u.IsEightDirectional(); got != tc.eight {
			t.Errorf("IsEightDirectional = %v, want %v", got, tc.eight)
		}

		// each URL is the one of its direction, in the order of its key
		values := slices.Collect(u.Values())
		for i, dir := range tc.want {
			got, ok := u.Get(dir)
			if !ok || got != values[i] || !strings.HasSuffix(got.Path, "/rotations/"+string(dir)+".png") {
				t.Errorf("eight %v: Get(%s) = %v, %v; value %v", tc.eight, dir, got.String(), ok, values[i].String())
			}
		}

		// breaking off after n, as a caller looking for one does, yields the first n
		for n := 1; n < len(tc.want); n++ {
			if got := first(u.Keys(), n); !slices.Equal(got, tc.want[:n]) {
				t.Errorf("eight %v: first %d keys = %v", tc.eight, n, got)
			}

			if got := first(u.Values(), n); !slices.Equal(got, values[:n]) {
				t.Errorf("eight %v: first %d values = %v", tc.eight, n, got)
			}
		}
	}

	four := rotations(t, false)
	for _, dir := range []pixellab.Direction{pixellab.DirectionNorthEast, pixellab.DirectionNone, "up"} {
		if got, ok := four.Get(dir); ok || got != (url.URL{}) {
			t.Errorf("Get(%q) on 4 directions = %v, %v", dir, got.String(), ok)
		}
	}
}

// first is the first n of seq, breaking off after them.
func first[T any](seq iter.Seq[T], n int) []T {
	var got []T
	for v := range seq {
		if got = append(got, v); len(got) == n {
			break
		}
	}

	return got
}

func TestHTTPValidationError(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ body, want string }{
		{
			`{"detail": "Sorry, we are experiencing high demand and need to temporarily limit free tier."}`,
			"Sorry, we are experiencing high demand and need to temporarily limit free tier.",
		},
		{`{"detail": [
			{"loc": ["body", "description"], "msg": "Field required", "type": "missing"},
			{"loc": ["body", "image_size", "width"], "msg": "Input should be less than or equal to 256", "type": "less_than_equal"}
		]}`, "Input should be less than or equal to 256"},
		{`{}`, "validation error"},
	} {
		var e pixellab.HTTPValidationError
		if err := json.Unmarshal([]byte(tc.body), &e); err != nil {
			t.Fatal(err)
		}

		var err error = &e
		if got := err.Error(); got != tc.want {
			t.Errorf("Error() = %q, want %q", got, tc.want)
		}
	}
}
