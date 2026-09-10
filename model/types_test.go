package model

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestEventJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		event Event
		check func(*testing.T, Event)
	}{
		{"message", Event{Kind: EventMessage, Message: &Message{ID: "1", ThreadID: "2", Text: "hello"}}, func(t *testing.T, got Event) {
			if got.Message == nil || got.Message.ID != "1" || got.Message.ThreadID != "2" || got.Message.Text != "hello" {
				t.Fatalf("unexpected message: %#v", got.Message)
			}
		}},
		{"error", Event{Kind: EventError, Error: errors.New("boom")}, func(t *testing.T, got Event) {
			if got.Error == nil || got.Error.Error() != "boom" {
				t.Fatalf("unexpected error: %v", got.Error)
			}
		}},
		{"ready", Event{Kind: EventReady, IsNewSession: true}, func(t *testing.T, got Event) {
			if !got.IsNewSession {
				t.Fatal("new-session flag was lost")
			}
		}},
		{"thread-system", Event{Kind: EventThreadSystem, ThreadSystem: &ThreadSystemEvent{Kind: ThreadSystemPinUpdated, ThreadID: "10", MessageID: "m1", Pinned: true}}, func(t *testing.T, got Event) {
			if got.ThreadSystem == nil || got.ThreadSystem.Kind != ThreadSystemPinUpdated || got.ThreadSystem.ThreadID != "10" || got.ThreadSystem.MessageID != "m1" || !got.ThreadSystem.Pinned {
				t.Fatalf("unexpected thread system event: %#v", got.ThreadSystem)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := json.Marshal(test.event)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), `"kind":"`+string(test.event.Kind)+`"`) || !strings.Contains(string(data), `"data":`) {
				t.Fatalf("unexpected JSON contract: %s", data)
			}
			var decoded Event
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.Kind != test.event.Kind {
				t.Fatalf("kind mismatch: %q", decoded.Kind)
			}
			test.check(t, decoded)
		})
	}
}

func TestEventLifecycleJSONOmitsEmptyData(t *testing.T) {
	data, err := json.Marshal(Event{Kind: EventReconnected})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"kind":"reconnected"}` {
		t.Fatalf("unexpected lifecycle JSON: %s", data)
	}
}
