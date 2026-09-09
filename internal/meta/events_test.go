package meta

import (
	"testing"
	"time"

	"go.mau.fi/mautrix-meta/pkg/messagix"
	"go.mau.fi/mautrix-meta/pkg/messagix/table"
	"go.mewis.me/fbgo/model"
)

func TestEmitTableNormalizesRealtimeEvents(t *testing.T) {
	backend := new(fakeBackend)
	engine := newEngine(nil, backend, 32)
	tbl := &table.LSTable{
		LSInsertMessage:         []*table.LSInsertMessage{{MessageId: "m1", ThreadKey: 10, SenderId: 20, Text: "hello", TimestampMs: 1234}},
		LSUpsertReaction:        []*table.LSUpsertReaction{{MessageId: "m1", ThreadKey: 10, ActorId: 30, Reaction: "❤"}},
		LSUpdateTypingIndicator: []*table.LSUpdateTypingIndicator{{ThreadKey: 10, SenderId: 20, IsTyping: true}},
		LSUpdateReadReceipt:     []*table.LSUpdateReadReceipt{{ThreadKey: 10, ContactId: 20, ReadWatermarkTimestampMs: 2000}},
		LSUpdateDeliveryReceipt: []*table.LSUpdateDeliveryReceipt{{ThreadKey: 10, ContactId: 20, DeliveredWatermarkTimestampMs: 2100}},
		LSEditMessage:           []*table.LSEditMessage{{MessageID: "m1", Text: "edited", EditCount: 1}},
		LSDeleteMessage:         []*table.LSDeleteMessage{{ThreadKey: 10, MessageId: "m2"}},
		LSSyncUpdateThreadName:  []*table.LSSyncUpdateThreadName{{ThreadKey: 10, ThreadName: "group"}},
	}
	engine.emitTable(tbl)
	want := map[EventKind]bool{EventMessage: false, EventReaction: false, EventTyping: false, EventReadReceipt: false, EventDeliveryReceipt: false, EventMessageEdit: false, EventMessageUnsend: false, EventThreadUpdate: false}
	for range len(want) {
		event := <-engine.Events()
		want[event.Kind] = true
	}
	for kind, seen := range want {
		if !seen {
			t.Fatalf("missing normalized event %s", kind)
		}
	}
	if engine.Health().LastReceive.IsZero() {
		t.Fatal("last receive was not updated")
	}
}

func TestMessageAttachmentNormalization(t *testing.T) {
	wrapped := &table.WrappedMessage{LSInsertMessage: &table.LSInsertMessage{MessageId: "m1"}, Attachments: []*table.LSInsertAttachment{{AttachmentFbid: "99", AttachmentMimeType: "image/jpeg", PreviewUrl: "https://preview.invalid/a", ImageUrl: "https://image.invalid/a", Filename: "a.jpg", Filesize: 42, PreviewWidth: 10, PreviewHeight: 20}}}
	attachments := wrappedAttachments(wrapped)
	if len(attachments) != 1 {
		t.Fatalf("unexpected attachment count: %d", len(attachments))
	}
	attachment := attachments[0]
	if attachment.ID != model.ID("99") || attachment.Type != "image" || attachment.PreviewURL == "" || attachment.FileName != "a.jpg" || attachment.Size != 42 {
		t.Fatalf("unexpected attachment: %#v", attachment)
	}
}

func TestEngineDropOldestMetrics(t *testing.T) {
	engine := newEngine(nil, new(fakeBackend), 1)
	engine.emit(Event{Kind: EventReady})
	engine.emit(Event{Kind: EventTyping})
	if got := engine.Health().DroppedEventCount; got != 1 {
		t.Fatalf("expected one dropped event, got %d", got)
	}
	if got := (<-engine.Events()).Kind; got != EventTyping {
		t.Fatalf("expected newest event, got %s", got)
	}
}

func TestReconnectHealthCounter(t *testing.T) {
	backend := new(fakeBackend)
	engine := newEngine(nil, backend, 4)
	engine.connected.Store(true)
	engine.handleTransportEvent(nil, &messagix.Event_Reconnected{})
	health := engine.Health()
	if health.ReconnectCount != 1 || health.Regular != model.ConnectionConnected {
		t.Fatalf("unexpected health: %#v", health)
	}
	engine.lastRecv.Store(time.Unix(3, 0).UnixMilli())
	if engine.Health().LastReceive.Unix() != 3 {
		t.Fatal("last receive timestamp was not exposed")
	}
}
