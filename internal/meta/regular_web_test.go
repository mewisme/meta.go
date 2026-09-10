package meta

import (
	"testing"
	"time"

	"go.mewis.me/meta-extra/pkg/messagix"
)

func TestNormalizePrivacy(t *testing.T) {
	tests := map[string]string{"": "FRIENDS", "public": "FRIENDS", "EVERYONE": "FRIENDS", "friends": "FRIENDS", "custom": "CUSTOM"}
	for input, expected := range tests {
		if actual := normalizePrivacy(input); actual != expected {
			t.Fatalf("normalizePrivacy(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestNormalizeThreadTheme(t *testing.T) {
	description, color := "desc", "#123456"
	value := messagix.ThreadThemeVariant{ID: "2", AccessibilityLabel: "Theme", Description: &description, ComposerBackgroundColor: &color, NormalThemeID: "1"}
	value.BackgroundAsset = &messagix.ThreadThemeAsset{}
	value.BackgroundAsset.Image.URI = "https://example.invalid/background"
	got := normalizeThreadTheme(value)
	if got.ID != "2" || got.Name != "Theme" || got.Description != description || got.ComposerBackgroundColor != color || got.NormalThemeID != "1" || got.BackgroundImage == "" {
		t.Fatalf("unexpected normalized theme: %#v", got)
	}
}

func TestParseFacebookTimestamp(t *testing.T) {
	if got := parseFacebookTimestamp("1700000000123"); got.UnixMilli() != 1700000000123 {
		t.Fatalf("unexpected millisecond timestamp: %v", got)
	}
	expected := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if got := parseFacebookTimestamp("2026-01-02T03:04:05Z"); !got.Equal(expected) {
		t.Fatalf("unexpected RFC3339 timestamp: %v", got)
	}
}

func TestUploadType(t *testing.T) {
	tests := map[string]string{"image/gif": "gif", "image/png": "image", "video/mp4": "video", "audio/ogg; codecs=opus": "audio", "application/pdf": "file", "": "file"}
	for input, expected := range tests {
		if actual := uploadType(input); actual != expected {
			t.Fatalf("uploadType(%q) = %q, want %q", input, actual, expected)
		}
	}
}
