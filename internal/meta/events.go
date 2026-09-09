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
		e.reconnect.Add(1)
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
	e.lastRecv.Store(time.Now().UnixMilli())
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
			Timestamp: time.UnixMilli(msg.TimestampMs), Attachments: wrappedAttachments(msg), Transport: model.TransportMessenger, Encryption: model.EncryptionNone,
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
	for _, receipt := range tbl.LSUpdateDeliveryReceipt {
		if receipt == nil {
			continue
		}
		e.emit(Event{Kind: EventDeliveryReceipt, Data: DeliveryReceiptEvent{ThreadID: id64(receipt.ThreadKey), RecipientID: id64(receipt.ContactId), Watermark: time.UnixMilli(receipt.DeliveredWatermarkTimestampMs)}})
	}
	for _, edit := range tbl.LSEditMessage {
		if edit == nil || edit.MessageID == "" {
			continue
		}
		e.emit(Event{Kind: EventMessageEdit, Data: MessageEditEvent{MessageID: model.ID(edit.MessageID), Text: edit.Text, EditCount: edit.EditCount}})
	}
	for _, deleted := range tbl.LSDeleteMessage {
		if deleted == nil || deleted.MessageId == "" {
			continue
		}
		e.emit(Event{Kind: EventMessageUnsend, Data: MessageUnsendEvent{MessageID: model.ID(deleted.MessageId), ThreadID: id64(deleted.ThreadKey)}})
	}
	for _, update := range tbl.LSSyncUpdateThreadName {
		if update == nil {
			continue
		}
		e.emit(Event{Kind: EventThreadUpdate, Data: ThreadUpdateEvent{ThreadID: id64(update.ThreadKey), Field: "name", Value: update.ThreadName}})
	}
	for _, update := range tbl.LSSetThreadImageURL {
		if update == nil {
			continue
		}
		e.emit(Event{Kind: EventThreadUpdate, Data: ThreadUpdateEvent{ThreadID: id64(update.ThreadKey), Field: "image", Value: update.ImageURL}})
	}
}

func wrappedAttachments(msg *table.WrappedMessage) []model.Attachment {
	attachments := make([]model.Attachment, 0, len(msg.Attachments)+len(msg.BlobAttachments)+len(msg.Stickers))
	for _, attachment := range msg.Attachments {
		if attachment == nil {
			continue
		}
		attachments = append(attachments, model.Attachment{ID: model.ID(attachment.AttachmentFbid), Type: attachmentType(attachment.AttachmentMimeType), URL: firstNonEmpty(attachment.PlayableUrl, attachment.ImageUrl), PreviewURL: attachment.PreviewUrl, FileName: attachment.Filename, ContentType: firstNonEmpty(attachment.AttachmentMimeType, attachment.PlayableUrlMimeType, attachment.PreviewUrlMimeType), Size: attachment.Filesize, Width: int(attachment.PreviewWidth), Height: int(attachment.PreviewHeight), Duration: int(attachment.PlayableDurationMs)})
	}
	for _, attachment := range msg.BlobAttachments {
		if attachment == nil {
			continue
		}
		attachments = append(attachments, model.Attachment{ID: model.ID(attachment.AttachmentFbid), Type: attachmentType(attachment.AttachmentMimeType), URL: attachment.PlayableUrl, PreviewURL: attachment.PreviewUrl, FileName: attachment.Filename, ContentType: firstNonEmpty(attachment.AttachmentMimeType, attachment.PlayableUrlMimeType, attachment.PreviewUrlMimeType), Size: attachment.Filesize, Width: int(attachment.PreviewWidth), Height: int(attachment.PreviewHeight), Duration: int(attachment.PlayableDurationMs)})
	}
	for _, attachment := range msg.Stickers {
		if attachment == nil {
			continue
		}
		attachments = append(attachments, model.Attachment{ID: model.ID(attachment.AttachmentFbid), Type: "sticker", URL: firstNonEmpty(attachment.PlayableUrl, attachment.ImageUrl), PreviewURL: attachment.PreviewUrl, ContentType: firstNonEmpty(attachment.PlayableUrlMimeType, attachment.PreviewUrlMimeType, attachment.ImageUrlMimeType), Width: int(attachment.PreviewWidth), Height: int(attachment.PreviewHeight)})
	}
	return attachments
}

func attachmentType(contentType string) string {
	switch {
	case len(contentType) >= 6 && contentType[:6] == "image/":
		return "image"
	case len(contentType) >= 6 && contentType[:6] == "video/":
		return "video"
	case len(contentType) >= 6 && contentType[:6] == "audio/":
		return "audio"
	default:
		return "file"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func id64(value int64) model.ID { return model.ID(strconv.FormatInt(value, 10)) }
