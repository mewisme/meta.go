package meta

import (
	"context"
	"strconv"
	"time"

	"go.mau.fi/mautrix-meta/pkg/messagix"
	"go.mau.fi/mautrix-meta/pkg/messagix/table"
	waEvents "go.mau.fi/whatsmeow/types/events"
	"go.mewis.me/fbgo/model"
)

func (e *Engine) handleTransportEvent(_ context.Context, event any) {
	switch evt := event.(type) {
	case *messagix.Event_Ready:
		e.connected.Store(true)
		e.emit(Event{Kind: EventReady, Data: evt.IsNewSession})
	case *messagix.Event_Reconnected:
		e.connected.Store(true)
		e.emit(Event{Kind: EventReconnected})
	case *messagix.Event_SocketError:
		e.connected.Store(false)
		e.emit(Event{Kind: EventError, Data: evt.Err})
	case *messagix.Event_PermanentError:
		e.connected.Store(false)
		e.emit(Event{Kind: EventError, Data: evt.Err})
	case *messagix.Event_PublishResponse:
		if evt.Table != nil {
			e.emitTable(evt.Table)
		}
	case *waEvents.Connected:
		e.e2eeReady.Store(true)
		e.emit(Event{Kind: EventE2EEReady})
	case *waEvents.Disconnected:
		e.e2eeReady.Store(false)
		e.emit(Event{Kind: EventDisconnected, Data: model.TransportE2EE})
	}
}

func (e *Engine) emitTable(tbl *table.LSTable) {
	_, inserted := tbl.WrapMessages()
	seen := make(map[string]struct{}, len(inserted))
	for _, msg := range inserted {
		if msg == nil || msg.MessageId == "" {
			continue
		}
		if _, exists := seen[msg.MessageId]; exists {
			continue
		}
		seen[msg.MessageId] = struct{}{}
		e.emit(Event{Kind: EventMessage, Data: model.Message{
			ID: model.ID(msg.MessageId), ThreadID: id64(msg.ThreadKey), SenderID: id64(msg.SenderId), Text: msg.Text,
			Timestamp: time.UnixMilli(msg.TimestampMs), Transport: model.TransportMessenger, Encryption: model.EncryptionNone,
		}})
	}
	for _, reaction := range tbl.LSUpsertReaction {
		if reaction == nil {
			continue
		}
		e.emit(Event{Kind: EventReaction, Data: ReactionEvent{MessageID: model.ID(reaction.MessageId), ThreadID: id64(reaction.ThreadKey), ActorID: id64(reaction.ActorId), Reaction: reaction.Reaction}})
	}
	for _, reaction := range tbl.LSDeleteReaction {
		if reaction == nil {
			continue
		}
		e.emit(Event{Kind: EventReaction, Data: ReactionEvent{MessageID: model.ID(reaction.MessageId), ThreadID: id64(reaction.ThreadKey), ActorID: id64(reaction.ActorId)}})
	}
	for _, typing := range tbl.LSUpdateTypingIndicator {
		if typing == nil {
			continue
		}
		e.emit(Event{Kind: EventTyping, Data: TypingEvent{ThreadID: id64(typing.ThreadKey), SenderID: id64(typing.SenderId), Typing: typing.IsTyping}})
	}
	for _, receipt := range tbl.LSUpdateReadReceipt {
		if receipt == nil {
			continue
		}
		e.emit(Event{Kind: EventReadReceipt, Data: ReadReceiptEvent{ThreadID: id64(receipt.ThreadKey), ReaderID: id64(receipt.ContactId), Watermark: time.UnixMilli(receipt.ReadWatermarkTimestampMs)}})
	}
}

func id64(value int64) model.ID { return model.ID(strconv.FormatInt(value, 10)) }
