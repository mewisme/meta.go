package server

import (
	"errors"
	"testing"
	"time"

	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
	"go.mewis.me/meta.go/internal/runtime/session"
	"go.mewis.me/meta.go/model"
)

func TestEveryCurrentEventKindConverts(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	tests := []model.Event{
		{Kind: model.EventReady, IsNewSession: true},
		{Kind: model.EventReconnected},
		{Kind: model.EventDisconnected, Transport: model.TransportMessenger},
		{Kind: model.EventError, Error: errors.New("test")},
		{Kind: model.EventMessage, Message: &model.Message{ID: "m1", ThreadID: "t1", SenderID: "u1", Text: "hello", Timestamp: now, Transport: model.TransportMessenger, Encryption: model.EncryptionNone}},
		{Kind: model.EventMessageEdit, MessageEdit: &model.MessageEditEvent{MessageID: "m1", Text: "edit", EditCount: 1}},
		{Kind: model.EventMessageUnsend, MessageUnsend: &model.MessageUnsendEvent{MessageID: "m1", ThreadID: "t1"}},
		{Kind: model.EventReaction, Reaction: &model.ReactionEvent{MessageID: "m1", ThreadID: "t1", ActorID: "u1", Reaction: "+1"}},
		{Kind: model.EventTyping, Typing: &model.TypingEvent{ThreadID: "t1", SenderID: "u1", Typing: true}},
		{Kind: model.EventReadReceipt, ReadReceipt: &model.ReadReceiptEvent{ThreadID: "t1", ReaderID: "u1", Watermark: now}},
		{Kind: model.EventDeliveryReceipt, DeliveryReceipt: &model.DeliveryReceiptEvent{ThreadID: "t1", RecipientID: "u1", Watermark: now}},
		{Kind: model.EventThreadUpdate, ThreadUpdate: &model.ThreadUpdateEvent{ThreadID: "t1", Field: "name", Value: "test"}},
		{Kind: model.EventThreadSystem, ThreadSystem: &model.ThreadSystemEvent{Kind: model.ThreadSystemPinUpdated, ThreadID: "t1", MessageID: "m1", Pinned: true}},
		{Kind: model.EventE2EEReady},
		{Kind: model.EventE2EEReceipt, E2EEReceipt: &model.E2EEReceiptEvent{Type: "delivered", ChatJID: "chat", SenderJID: "sender", MessageIDs: []model.ID{"m1"}}},
	}
	for _, input := range tests {
		t.Run(string(input.Kind), func(t *testing.T) {
			converted, ok := eventToProto(session.Event{SessionID: "s1", Sequence: 7, EmittedAt: now, Payload: input})
			if !ok || converted == nil || converted.GetPayload() == nil || converted.GetSessionId() != "s1" || converted.GetSequence() != 7 {
				t.Fatalf("event did not convert: %#v", converted)
			}
		})
	}
}

func TestMessageConversionPreservesNestedData(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	message := &model.Message{
		ID: "m1", ThreadID: "t1", SenderID: "u1", Text: "hello", Timestamp: now, Transport: model.TransportE2EE, Encryption: model.EncryptionE2EE,
		ReplyTo:     &model.ReplyReference{MessageID: "m0", SenderID: "u2"},
		Mentions:    []model.Mention{{UserID: "u3", Offset: 1, Length: 2}},
		Attachments: []model.Attachment{{ID: "a1", Type: "image", URL: "https://example.invalid", Width: 10, Height: 20, E2EE: &model.E2EEMediaReference{Kind: model.E2EEMediaImage, DirectPath: "/path", MediaKey: []byte{1, 2}, Size: 42}}},
	}
	converted := messageToProto(message)
	if converted.GetReplyTo().GetMessageId() != "m0" || len(converted.Mentions) != 1 || converted.Mentions[0].GetUserId() != "u3" || len(converted.Attachments) != 1 || converted.Attachments[0].GetE2Ee().GetDirectPath() != "/path" {
		t.Fatalf("nested message data lost: %#v", converted)
	}
	if converted.GetTransport() != metav1.TransportKind_TRANSPORT_KIND_E2EE || converted.GetEncryption() != metav1.EncryptionKind_ENCRYPTION_KIND_E2EE {
		t.Fatalf("transport/encryption mismatch: %#v", converted)
	}
}
