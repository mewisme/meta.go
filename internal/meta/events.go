package meta

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mau.fi/mautrix-meta/pkg/messagix"
	"go.mau.fi/mautrix-meta/pkg/messagix/table"
	"go.mau.fi/whatsmeow/proto/waConsumerApplication"
	"go.mau.fi/whatsmeow/proto/waMediaTransport"
	waEvents "go.mau.fi/whatsmeow/types/events"
	"go.mewis.me/fbgo/model"
)

func (e *Engine) handleTransportEvent(_ context.Context, event any) {
	switch evt := event.(type) {
	case *messagix.Event_Ready:
		e.connected.Store(true)
		e.regularState.Store(model.ConnectionConnected)
		e.emit(Event{Kind: EventReady, IsNewSession: evt.IsNewSession})
	case *messagix.Event_Reconnected:
		e.connected.Store(true)
		e.regularState.Store(model.ConnectionConnected)
		e.reconnect.Add(1)
		e.emit(Event{Kind: EventReconnected})
	case *messagix.Event_SocketError:
		e.connected.Store(false)
		e.regularState.Store(model.ConnectionFailed)
		e.recordError(evt.Err)
		e.emit(Event{Kind: EventError, Error: evt.Err})
	case *messagix.Event_PermanentError:
		e.connected.Store(false)
		e.regularState.Store(model.ConnectionFailed)
		e.recordError(evt.Err)
		e.emit(Event{Kind: EventError, Error: evt.Err})
	case *messagix.Event_PublishResponse:
		if evt.Table != nil {
			e.emitTable(evt.Table)
		}
	case *waEvents.Connected:
		e.e2eeReady.Store(true)
		e.e2eeState.Store(model.ConnectionConnected)
		e.emit(Event{Kind: EventE2EEReady})
	case *waEvents.Disconnected:
		e.e2eeReady.Store(false)
		e.e2eeState.Store(model.ConnectionDisconnected)
		e.emit(Event{Kind: EventDisconnected, Transport: model.TransportE2EE})
	case *waEvents.FBMessage:
		e.emitE2EEMessage(evt)
	case *waEvents.Receipt:
		ids := make([]model.ID, len(evt.MessageIDs))
		for index, id := range evt.MessageIDs {
			ids[index] = model.ID(id)
		}
		e.lastRecv.Store(time.Now().UnixMilli())
		e.emit(Event{Kind: EventE2EEReceipt, E2EEReceipt: &model.E2EEReceiptEvent{Type: string(evt.Type), ChatJID: evt.Chat.String(), SenderJID: evt.Sender.String(), MessageIDs: ids}})
	case *waEvents.UndecryptableMessage:
		e.lastRecv.Store(time.Now().UnixMilli())
		err := fmt.Errorf("E2EE message %s could not be decrypted", evt.Info.ID)
		e.recordError(err)
		e.emit(Event{Kind: EventError, Error: err})
	}
}

func (e *Engine) emitE2EEMessage(evt *waEvents.FBMessage) {
	if evt == nil {
		return
	}
	e.lastRecv.Store(time.Now().UnixMilli())
	consumer := evt.GetConsumerApplication()
	if consumer == nil || consumer.GetPayload() == nil {
		return
	}
	payload := consumer.GetPayload()
	if applicationData := payload.GetApplicationData(); applicationData != nil {
		if revoke := applicationData.GetRevoke(); revoke != nil && revoke.GetKey() != nil {
			e.emit(Event{Kind: EventMessageUnsend, MessageUnsend: &model.MessageUnsendEvent{MessageID: model.ID(revoke.GetKey().GetID()), ThreadID: model.ID(evt.Info.Chat.User)}})
		}
		return
	}
	content := payload.GetContent()
	if content == nil {
		return
	}
	switch message := content.GetContent().(type) {
	case *waConsumerApplication.ConsumerApplication_Content_MessageText:
		text := ""
		if message.MessageText != nil {
			text = message.MessageText.GetText()
		}
		e.emit(Event{Kind: EventMessage, Message: &model.Message{ID: model.ID(evt.Info.ID), ThreadID: model.ID(evt.Info.Chat.User), SenderID: model.ID(evt.Info.Sender.User), Text: text, Timestamp: evt.Info.Timestamp, Transport: model.TransportE2EE, Encryption: model.EncryptionE2EE, ReplyTo: e2eeReply(evt)}})
	case *waConsumerApplication.ConsumerApplication_Content_ReactionMessage:
		if message.ReactionMessage == nil || message.ReactionMessage.GetKey() == nil {
			return
		}
		e.emit(Event{Kind: EventReaction, Reaction: &model.ReactionEvent{MessageID: model.ID(message.ReactionMessage.GetKey().GetID()), ThreadID: model.ID(evt.Info.Chat.User), ActorID: model.ID(evt.Info.Sender.User), Reaction: message.ReactionMessage.GetText()}})
	case *waConsumerApplication.ConsumerApplication_Content_EditMessage:
		if message.EditMessage == nil || message.EditMessage.GetKey() == nil {
			return
		}
		e.emit(Event{Kind: EventMessageEdit, MessageEdit: &model.MessageEditEvent{MessageID: model.ID(message.EditMessage.GetKey().GetID()), Text: message.EditMessage.GetMessage().GetText(), EditCount: 1}})
	case *waConsumerApplication.ConsumerApplication_Content_ImageMessage:
		if message.ImageMessage == nil {
			return
		}
		decoded, err := message.ImageMessage.Decode()
		if err != nil {
			e.emit(Event{Kind: EventError, Error: fmt.Errorf("decode E2EE image: %w", err)})
			return
		}
		attachment := e2eeAttachment(model.E2EEMediaImage, "", decoded.GetIntegral().GetTransport(), int(decoded.GetAncillary().GetWidth()), int(decoded.GetAncillary().GetHeight()), 0)
		e.emitE2EEMediaMessage(evt, message.ImageMessage.GetCaption().GetText(), attachment)
	case *waConsumerApplication.ConsumerApplication_Content_VideoMessage:
		if message.VideoMessage == nil {
			return
		}
		decoded, err := message.VideoMessage.Decode()
		if err != nil {
			e.emit(Event{Kind: EventError, Error: fmt.Errorf("decode E2EE video: %w", err)})
			return
		}
		attachment := e2eeAttachment(model.E2EEMediaVideo, "", decoded.GetIntegral().GetTransport(), int(decoded.GetAncillary().GetWidth()), int(decoded.GetAncillary().GetHeight()), int(decoded.GetAncillary().GetSeconds()))
		e.emitE2EEMediaMessage(evt, message.VideoMessage.GetCaption().GetText(), attachment)
	case *waConsumerApplication.ConsumerApplication_Content_AudioMessage:
		if message.AudioMessage == nil {
			return
		}
		decoded, err := message.AudioMessage.Decode()
		if err != nil {
			e.emit(Event{Kind: EventError, Error: fmt.Errorf("decode E2EE audio: %w", err)})
			return
		}
		attachment := e2eeAttachment(model.E2EEMediaAudio, "", decoded.GetIntegral().GetTransport(), 0, 0, int(decoded.GetAncillary().GetSeconds()))
		if message.AudioMessage.GetPTT() {
			attachment.Type = "voice"
		}
		e.emitE2EEMediaMessage(evt, "", attachment)
	case *waConsumerApplication.ConsumerApplication_Content_DocumentMessage:
		if message.DocumentMessage == nil {
			return
		}
		decoded, err := message.DocumentMessage.Decode()
		if err != nil {
			e.emit(Event{Kind: EventError, Error: fmt.Errorf("decode E2EE document: %w", err)})
			return
		}
		attachment := e2eeAttachment(model.E2EEMediaDocument, message.DocumentMessage.GetFileName(), decoded.GetIntegral().GetTransport(), 0, 0, 0)
		e.emitE2EEMediaMessage(evt, "", attachment)
	case *waConsumerApplication.ConsumerApplication_Content_StickerMessage:
		if message.StickerMessage == nil {
			return
		}
		decoded, err := message.StickerMessage.Decode()
		if err != nil {
			e.emit(Event{Kind: EventError, Error: fmt.Errorf("decode E2EE sticker: %w", err)})
			return
		}
		attachment := e2eeAttachment(model.E2EEMediaSticker, "", decoded.GetIntegral().GetTransport(), int(decoded.GetAncillary().GetWidth()), int(decoded.GetAncillary().GetHeight()), 0)
		e.emitE2EEMediaMessage(evt, "", attachment)
	}
}

func (e *Engine) emitE2EEMediaMessage(evt *waEvents.FBMessage, text string, attachment model.Attachment) {
	e.emit(Event{Kind: EventMessage, Message: &model.Message{ID: model.ID(evt.Info.ID), ThreadID: model.ID(evt.Info.Chat.User), SenderID: model.ID(evt.Info.Sender.User), Text: text, Timestamp: evt.Info.Timestamp, Attachments: []model.Attachment{attachment}, Transport: model.TransportE2EE, Encryption: model.EncryptionE2EE, ReplyTo: e2eeReply(evt)}})
}

func e2eeAttachment(kind model.E2EEMediaKind, name string, transport *waMediaTransport.WAMediaTransport, width, height, duration int) model.Attachment {
	attachment := model.Attachment{Type: string(kind), FileName: name, Width: width, Height: height, Duration: duration}
	if transport == nil {
		return attachment
	}
	integral, ancillary := transport.GetIntegral(), transport.GetAncillary()
	if ancillary != nil {
		attachment.ContentType = ancillary.GetMimetype()
		attachment.Size = int64(ancillary.GetFileLength())
	}
	if integral == nil {
		return attachment
	}
	attachment.E2EE = &model.E2EEMediaReference{Kind: kind, DirectPath: integral.GetDirectPath(), MediaKey: clone(integral.GetMediaKey()), FileSHA256: clone(integral.GetFileSHA256()), FileEncSHA256: clone(integral.GetFileEncSHA256()), ContentType: attachment.ContentType, Size: attachment.Size}
	return attachment
}

func e2eeReply(evt *waEvents.FBMessage) *model.ReplyReference {
	if evt == nil || evt.FBApplication == nil || evt.FBApplication.GetMetadata() == nil {
		return nil
	}
	quoted := evt.FBApplication.GetMetadata().GetQuotedMessage()
	if quoted == nil || quoted.GetStanzaID() == "" {
		return nil
	}
	result := &model.ReplyReference{MessageID: model.ID(quoted.GetStanzaID())}
	participant := quoted.GetParticipant()
	if at := strings.IndexByte(participant, '@'); at > 0 {
		result.SenderID = model.ID(participant[:at])
	}
	return result
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
		e.emit(Event{Kind: EventMessage, Message: &model.Message{
			ID: model.ID(msg.MessageId), ThreadID: id64(msg.ThreadKey), SenderID: id64(msg.SenderId), Text: msg.Text,
			Timestamp: time.UnixMilli(msg.TimestampMs), Attachments: wrappedAttachments(msg), Transport: model.TransportMessenger, Encryption: model.EncryptionNone,
		}})
	}
	for _, reaction := range tbl.LSUpsertReaction {
		if reaction == nil {
			continue
		}
		e.emit(Event{Kind: EventReaction, Reaction: &ReactionEvent{MessageID: model.ID(reaction.MessageId), ThreadID: id64(reaction.ThreadKey), ActorID: id64(reaction.ActorId), Reaction: reaction.Reaction}})
	}
	for _, reaction := range tbl.LSDeleteReaction {
		if reaction == nil {
			continue
		}
		e.emit(Event{Kind: EventReaction, Reaction: &ReactionEvent{MessageID: model.ID(reaction.MessageId), ThreadID: id64(reaction.ThreadKey), ActorID: id64(reaction.ActorId)}})
	}
	for _, typing := range tbl.LSUpdateTypingIndicator {
		if typing == nil {
			continue
		}
		e.emit(Event{Kind: EventTyping, Typing: &TypingEvent{ThreadID: id64(typing.ThreadKey), SenderID: id64(typing.SenderId), Typing: typing.IsTyping}})
	}
	for _, receipt := range tbl.LSUpdateReadReceipt {
		if receipt == nil {
			continue
		}
		e.emit(Event{Kind: EventReadReceipt, ReadReceipt: &ReadReceiptEvent{ThreadID: id64(receipt.ThreadKey), ReaderID: id64(receipt.ContactId), Watermark: time.UnixMilli(receipt.ReadWatermarkTimestampMs)}})
	}
	for _, receipt := range tbl.LSUpdateDeliveryReceipt {
		if receipt == nil {
			continue
		}
		e.emit(Event{Kind: EventDeliveryReceipt, DeliveryReceipt: &DeliveryReceiptEvent{ThreadID: id64(receipt.ThreadKey), RecipientID: id64(receipt.ContactId), Watermark: time.UnixMilli(receipt.DeliveredWatermarkTimestampMs)}})
	}
	for _, edit := range tbl.LSEditMessage {
		if edit == nil || edit.MessageID == "" {
			continue
		}
		e.emit(Event{Kind: EventMessageEdit, MessageEdit: &MessageEditEvent{MessageID: model.ID(edit.MessageID), Text: edit.Text, EditCount: edit.EditCount}})
	}
	for _, deleted := range tbl.LSDeleteMessage {
		if deleted == nil || deleted.MessageId == "" {
			continue
		}
		e.emit(Event{Kind: EventMessageUnsend, MessageUnsend: &MessageUnsendEvent{MessageID: model.ID(deleted.MessageId), ThreadID: id64(deleted.ThreadKey)}})
	}
	for _, update := range tbl.LSSyncUpdateThreadName {
		if update == nil {
			continue
		}
		e.emit(Event{Kind: EventThreadUpdate, ThreadUpdate: &ThreadUpdateEvent{ThreadID: id64(update.ThreadKey), Field: "name", Value: update.ThreadName}})
	}
	for _, update := range tbl.LSSetThreadImageURL {
		if update == nil {
			continue
		}
		e.emit(Event{Kind: EventThreadUpdate, ThreadUpdate: &ThreadUpdateEvent{ThreadID: id64(update.ThreadKey), Field: "image", Value: update.ImageURL}})
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
