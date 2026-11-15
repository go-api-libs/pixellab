package pixellab_test

import (
	"testing"

	"github.com/go-api-libs/pixellab/pkg/pixellab"
)

// TestImageSize checks a size each kind of rule accepts and one it rejects, by the message the caller gets.
func TestImageSize(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name          string
		new           func(width, height int) (pixellab.ImageSize, error)
		width, height int
		err           string
	}{
		// bounds
		{"skeleton", pixellab.NewImageSizeForAnimateWithSkeleton, 64, 128, ""},
		{"skeleton", pixellab.NewImageSizeForAnimateWithSkeleton, 8, 64, "width must be 16 to 256 pixels, got 8"},
		{"skeleton", pixellab.NewImageSizeForAnimateWithSkeleton, 64, 512, "height must be 16 to 256 pixels, got 512"},
		// steps
		{"canva", pixellab.NewImageSizeForCreateCharacterCanva, 68, 68, ""},
		{"canva", pixellab.NewImageSizeForCreateCharacterCanva, 66, 68, "width must be a multiple of 4 pixels, got 66"},
		{"canva", pixellab.NewImageSizeForCreateCharacterCanva, 68, 70, "height must be a multiple of 4 pixels, got 70"},
		// Pixen's area and its small sizes
		{"pixen", pixellab.NewImageSizeForCreateImagePixen, 768, 256, ""},
		{"pixen", pixellab.NewImageSizeForCreateImagePixen, 16, 16, ""},
		{
			"pixen", pixellab.NewImageSizeForCreateImagePixen, 768, 768,
			"width times height must be at most 512x512 pixels, got 768x768",
		},
		{
			"pixen", pixellab.NewImageSizeForCreateImagePixen, 16, 64,
			"width and height must be equal when either is below 32 pixels, got 16x64",
		},
		// the sides a tile may have
		{"tileset", pixellab.NewTileSizeForCreateTileset, 24, 64, ""},
		{"tileset", pixellab.NewTileSizeForCreateTileset, 48, 32, "width must be 16, 24, 32 or 64 pixels, got 48"},
		{"tileset", pixellab.NewTileSizeForCreateTileset, 32, 8, "height must be 16, 24, 32 or 64 pixels, got 8"},
		{"sidescroller", pixellab.NewTileSizeForCreateSidescrollerTileset, 32, 24, "height must be 16 or 32 pixels, got 24"},
	} {
		got, err := tc.new(tc.width, tc.height)

		switch {
		case tc.err == "" && err != nil:
			t.Errorf("%s %dx%d: %v", tc.name, tc.width, tc.height, err)
		case tc.err == "" && got != (pixellab.ImageSize{Width: tc.width, Height: tc.height}):
			t.Errorf("%s %dx%d: got %+v", tc.name, tc.width, tc.height, got)
		case tc.err != "" && (err == nil || err.Error() != tc.err):
			t.Errorf("%s %dx%d: err = %v, want %q", tc.name, tc.width, tc.height, err, tc.err)
		case tc.err != "" && got != (pixellab.ImageSize{}):
			t.Errorf("%s %dx%d: rejected but got %+v", tc.name, tc.width, tc.height, got)
		}
	}
}
