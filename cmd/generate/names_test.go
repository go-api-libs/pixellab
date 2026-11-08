package main

import (
	"slices"
	"testing"
)

func TestCommonWords(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		names []string
		want  string
	}{
		{[]string{"UpdateCharacterTagsRequest", "UpdateObjectTagsRequest"}, "UpdateTagsRequest"},
		{[]string{"DeleteCharacterAnimationResponse", "DeleteCharacterAnimationDirectionResponse"}, "DeleteCharacterAnimationResponse"},
		{[]string{"DeleteTilesProResponse", "DeleteIsometricTileResponse"}, ""},
		{[]string{"CreateCharacterAnimationRequest", "AnimateCharacterRequest"}, ""},
	} {
		if got := commonWords(tc.names); got != tc.want {
			t.Errorf("commonWords(%v) = %q, want %q", tc.names, got, tc.want)
		}
	}

	if got, want := words("CreateUIAssetV3Request"), []string{"Create", "UI", "Asset", "V", "3", "Request"}; !slices.Equal(got, want) {
		t.Errorf("words = %q, want %q", got, want)
	}
}
