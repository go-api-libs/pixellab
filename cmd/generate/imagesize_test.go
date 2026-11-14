package main

import (
	"strings"
	"testing"

	"github.com/MarkRosemaker/openapi"
)

func TestSizeNotes(t *testing.T) {
	t.Parallel()

	size := func(w, h string) *openapi.Schema {
		s := &openapi.Schema{Type: openapi.TypeObject}
		s.Properties.Set("width", &openapi.Schema{Type: openapi.TypeInteger, Description: w})
		s.Properties.Set("height", &openapi.Schema{Type: openapi.TypeInteger, Description: h})

		return s
	}

	for _, tc := range []struct {
		w, h, want string
	}{
		{"Image width in pixels", "Image height in pixels", ""},
		{"Frame width in pixels (32-256).", "Frame height in pixels (32-256).", ""},
		{"Canvas width", "Canvas height", ""},
		{"Image width in pixels. Sizes above 24px often give better results.", "Image height in pixels. Sizes above 24px often give better results.", "Width and height: Sizes above 24px often give better results."},
		{"Individual tile width in pixels (16 or 32)", "Individual tile height in pixels (16 or 32)", "Width and height: 16 or 32"},
		{"Character size in pixels.", "Character size in pixels.", "Character size in pixels."},
		{"Output width in pixels (192-688; square 512, 16:9 688)", "Output height in pixels (192-688; square 512, 9:16 688)", "Width: 192-688; square 512, 16:9 688\nHeight: 192-688; square 512, 9:16 688"},
		{"Image width in pixels", "Image height in pixels, at most the width", "Height: at most the width"},
	} {
		if got := sizeNotes(size(tc.w, tc.h)); got != tc.want {
			t.Errorf("%q, %q: got %q, want %q", tc.w, tc.h, got, tc.want)
		}
	}
}

func TestWriteRule(t *testing.T) {
	t.Parallel()

	var b strings.Builder

	writeRule(&b, sizeRule{sides: []int{16, 24, 32}, areaSide: 512, squareBelow: 32})

	for _, want := range []string{
		"switch width {\n\tcase 16, 24, 32:",
		`"height must be 16, 24 or 32 pixels, got %d"`,
		"if width*height > 512*512 {",
		"if (width < 32 || height < 32) && width != height {",
	} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("no %q in:\n%s", want, b.String())
		}
	}

	b.Reset()
	writeRule(&b, sizeRule{})

	if b.Len() != 0 {
		t.Errorf("a zero rule checks something:\n%s", b.String())
	}
}
