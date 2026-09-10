package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"go.mewis.me/meta.go/model"
)

func TestMessengerTypingAndReadCommandsExist(t *testing.T) {
	root := New()
	for _, path := range [][]string{{"messenger", "typing"}, {"messenger", "read"}} {
		if _, _, err := root.Find(path); err != nil {
			t.Fatalf("missing command %v: %v", path, err)
		}
	}
}

func TestMessengerE2EEFlags(t *testing.T) {
	root := New()
	tests := []struct {
		path  []string
		flags []string
	}{
		{[]string{"messenger", "send"}, []string{"e2ee", "regular", "sticker-id", "url"}},
		{[]string{"messenger", "media", "upload"}, []string{"e2ee", "kind", "caption"}},
		{[]string{"messenger", "react"}, []string{"e2ee", "sender-jid"}},
		{[]string{"messenger", "edit"}, []string{"e2ee", "chat-jid"}},
		{[]string{"messenger", "unsend"}, []string{"e2ee", "chat-jid"}},
	}
	for _, test := range tests {
		cmd, _, err := root.Find(test.path)
		if err != nil {
			t.Fatalf("missing command %v: %v", test.path, err)
		}
		for _, name := range test.flags {
			if cmd.Flags().Lookup(name) == nil {
				t.Fatalf("%v missing --%s", test.path, name)
			}
		}
	}
}

func TestMessengerForwardCommandExists(t *testing.T) {
	root := New()
	if _, _, err := root.Find([]string{"messenger", "forward"}); err != nil {
		t.Fatalf("missing messenger forward: %v", err)
	}
}

func TestMessengerRichContactCommandsExist(t *testing.T) {
	root := New()
	for _, path := range [][]string{{"messenger", "share-contact"}, {"messenger", "restrict"}, {"messenger", "message-block"}} {
		if _, _, err := root.Find(path); err != nil {
			t.Fatalf("missing command %v: %v", path, err)
		}
	}
}

func TestResolveE2EEMediaKind(t *testing.T) {
	tests := []struct {
		value string
		mime  string
		want  model.E2EEMediaKind
	}{
		{"", "image/png", model.E2EEMediaImage},
		{"", "video/mp4", model.E2EEMediaVideo},
		{"", "audio/mpeg", model.E2EEMediaAudio},
		{"", "application/pdf", model.E2EEMediaDocument},
		{"sticker", "image/webp", model.E2EEMediaSticker},
		{"file", "application/octet-stream", model.E2EEMediaDocument},
	}
	for _, test := range tests {
		got, err := resolveE2EEMediaKind(test.value, test.mime)
		if err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Fatalf("value=%q mime=%q: got %q want %q", test.value, test.mime, got, test.want)
		}
	}
	if _, err := resolveE2EEMediaKind("unknown", "application/octet-stream"); err == nil {
		t.Fatal("expected invalid E2EE media kind error")
	}
}

func TestWriteEventHumanAndNDJSON(t *testing.T) {
	event := model.Event{Kind: model.EventMessage, Message: &model.Message{ID: "1"}}
	var human bytes.Buffer
	if err := writeEvent(&human, false, "", event); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(human.String(), "message\t") {
		t.Fatalf("unexpected human event: %q", human.String())
	}
	var ndjson bytes.Buffer
	if err := writeEvent(&ndjson, true, "", event); err != nil {
		t.Fatal(err)
	}
	var decoded model.Event
	if err := json.Unmarshal(ndjson.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Kind != model.EventMessage || !strings.HasSuffix(ndjson.String(), "\n") {
		t.Fatalf("unexpected NDJSON event: %q", ndjson.String())
	}
	ndjson.Reset()
	if err := writeEvent(&ndjson, true, ".kind", event); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(ndjson.String()); got != `"message"` {
		t.Fatalf("unexpected filtered NDJSON event: %q", got)
	}
}
