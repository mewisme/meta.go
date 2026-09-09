package meta

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNormalizePrivacy(t *testing.T) {
	tests := map[string]string{"": "FRIENDS", "public": "FRIENDS", "EVERYONE": "FRIENDS", "friends": "FRIENDS", "custom": "CUSTOM"}
	for input, expected := range tests {
		if actual := normalizePrivacy(input); actual != expected {
			t.Fatalf("normalizePrivacy(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestThemeTaskNullPayload(t *testing.T) {
	task := &themeTask{ThreadKey: 1, ThemeFBID: 2, SyncGroup: 1, label: "43", queue: "thread_theme", includeNulls: true}
	payload, queue, _ := task.Create()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"thread_key":1,"theme_fbid":2,"sync_group":1,"source":null,"payload":null}` || queue != "thread_theme" {
		t.Fatalf("unexpected theme task: %s queue=%v", data, queue)
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
