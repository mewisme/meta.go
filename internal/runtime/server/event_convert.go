package server

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
	"go.mewis.me/meta.go/internal/runtime/session"
	"go.mewis.me/meta.go/model"
)

func eventToProto(event session.Event) (*metav1.Event, bool) {
	result := &metav1.Event{SessionId: event.SessionID, Sequence: event.Sequence, EmittedAt: timestamppb.New(event.EmittedAt)}
	switch payload := event.Payload; payload.Kind {
	case model.EventReady:
		result.Payload = &metav1.Event_Ready{Ready: &metav1.ReadyEvent{IsNewSession: payload.IsNewSession}}
	case model.EventReconnected:
		result.Payload = &metav1.Event_Reconnected{Reconnected: &metav1.ReconnectedEvent{}}
	case model.EventDisconnected:
		result.Payload = &metav1.Event_Disconnected{Disconnected: &metav1.DisconnectedEvent{Transport: transportToProto(payload.Transport)}}
	case model.EventError:
		result.Payload = &metav1.Event_Error{Error: &metav1.ErrorEvent{Error: errorDetail(payload.Error)}}
	case model.EventMessage:
		result.Payload = &metav1.Event_Message{Message: &metav1.MessageEvent{Message: messageToProto(payload.Message)}}
	case model.EventReaction:
		if payload.Reaction == nil {
			return nil, false
		}
		result.Payload = &metav1.Event_Reaction{Reaction: &metav1.ReactionEvent{MessageId: payload.Reaction.MessageID.String(), ThreadId: payload.Reaction.ThreadID.String(), ActorId: payload.Reaction.ActorID.String(), Reaction: payload.Reaction.Reaction}}
	case model.EventTyping:
		if payload.Typing == nil {
			return nil, false
		}
		result.Payload = &metav1.Event_Typing{Typing: &metav1.TypingEvent{ThreadId: payload.Typing.ThreadID.String(), SenderId: payload.Typing.SenderID.String(), Typing: payload.Typing.Typing}}
	case model.EventReadReceipt:
		if payload.ReadReceipt == nil {
			return nil, false
		}
		result.Payload = &metav1.Event_ReadReceipt{ReadReceipt: &metav1.ReadReceiptEvent{ThreadId: payload.ReadReceipt.ThreadID.String(), ReaderId: payload.ReadReceipt.ReaderID.String(), Watermark: timestampOrNil(payload.ReadReceipt.Watermark)}}
	case model.EventDeliveryReceipt:
		if payload.DeliveryReceipt == nil {
			return nil, false
		}
		result.Payload = &metav1.Event_DeliveryReceipt{DeliveryReceipt: &metav1.DeliveryReceiptEvent{ThreadId: payload.DeliveryReceipt.ThreadID.String(), RecipientId: payload.DeliveryReceipt.RecipientID.String(), Watermark: timestampOrNil(payload.DeliveryReceipt.Watermark)}}
	case model.EventMessageEdit:
		if payload.MessageEdit == nil {
			return nil, false
		}
		result.Payload = &metav1.Event_MessageEdit{MessageEdit: &metav1.MessageEditEvent{MessageId: payload.MessageEdit.MessageID.String(), Text: payload.MessageEdit.Text, EditCount: payload.MessageEdit.EditCount}}
	case model.EventMessageUnsend:
		if payload.MessageUnsend == nil {
			return nil, false
		}
		result.Payload = &metav1.Event_MessageUnsend{MessageUnsend: &metav1.MessageUnsendEvent{MessageId: payload.MessageUnsend.MessageID.String(), ThreadId: payload.MessageUnsend.ThreadID.String()}}
	case model.EventThreadUpdate:
		if payload.ThreadUpdate == nil {
			return nil, false
		}
		result.Payload = &metav1.Event_ThreadUpdate{ThreadUpdate: &metav1.ThreadUpdateEvent{ThreadId: payload.ThreadUpdate.ThreadID.String(), Field: payload.ThreadUpdate.Field, Value: payload.ThreadUpdate.Value}}
	case model.EventThreadSystem:
		if payload.ThreadSystem == nil {
			return nil, false
		}
		result.Payload = &metav1.Event_ThreadSystem{ThreadSystem: threadSystemToProto(payload.ThreadSystem)}
	case model.EventE2EEReady:
		result.Payload = &metav1.Event_E2EeReady{E2EeReady: &metav1.E2EEReadyEvent{}}
	case model.EventE2EEReceipt:
		if payload.E2EEReceipt == nil {
			return nil, false
		}
		messageIDs := make([]string, len(payload.E2EEReceipt.MessageIDs))
		for i, id := range payload.E2EEReceipt.MessageIDs {
			messageIDs[i] = id.String()
		}
		result.Payload = &metav1.Event_E2EeReceipt{E2EeReceipt: &metav1.E2EEReceiptEvent{Type: payload.E2EEReceipt.Type, ChatJid: payload.E2EEReceipt.ChatJID, SenderJid: payload.E2EEReceipt.SenderJID, MessageIds: messageIDs}}
	default:
		return nil, false
	}
	return result, true
}

func messageToProto(message *model.Message) *metav1.Message {
	if message == nil {
		return nil
	}
	result := &metav1.Message{Id: message.ID.String(), ThreadId: message.ThreadID.String(), SenderId: message.SenderID.String(), Text: message.Text, Timestamp: timestampOrNil(message.Timestamp), Encryption: encryptionToProto(message.Encryption), Transport: transportToProto(message.Transport)}
	if message.ReplyTo != nil {
		result.ReplyTo = &metav1.ReplyReference{MessageId: message.ReplyTo.MessageID.String(), SenderId: message.ReplyTo.SenderID.String()}
	}
	result.Mentions = make([]*metav1.Mention, 0, len(message.Mentions))
	for _, mention := range message.Mentions {
		result.Mentions = append(result.Mentions, &metav1.Mention{UserId: mention.UserID.String(), Offset: int32(mention.Offset), Length: int32(mention.Length)})
	}
	result.Attachments = make([]*metav1.Attachment, 0, len(message.Attachments))
	for _, attachment := range message.Attachments {
		item := &metav1.Attachment{Id: attachment.ID.String(), Type: attachment.Type, Url: attachment.URL, PreviewUrl: attachment.PreviewURL, FileName: attachment.FileName, ContentType: attachment.ContentType, Size: attachment.Size, Width: int32(attachment.Width), Height: int32(attachment.Height), Duration: int32(attachment.Duration)}
		if attachment.E2EE != nil {
			item.E2Ee = &metav1.E2EEMediaReference{Kind: string(attachment.E2EE.Kind), DirectPath: attachment.E2EE.DirectPath, MediaKey: append([]byte(nil), attachment.E2EE.MediaKey...), FileSha256: append([]byte(nil), attachment.E2EE.FileSHA256...), FileEncSha256: append([]byte(nil), attachment.E2EE.FileEncSHA256...), ContentType: attachment.E2EE.ContentType, Size: attachment.E2EE.Size}
		}
		result.Attachments = append(result.Attachments, item)
	}
	return result
}

func threadSystemToProto(event *model.ThreadSystemEvent) *metav1.ThreadSystemEvent {
	return &metav1.ThreadSystemEvent{Kind: threadSystemKindToProto(event.Kind), ThreadId: event.ThreadID.String(), ParticipantId: event.ParticipantID.String(), MessageId: event.MessageID.String(), PollId: event.PollID.String(), Nickname: event.Nickname, Emoji: event.Emoji, Enabled: event.Enabled, Pinned: event.Pinned, IsAdmin: event.IsAdmin}
}

func threadSystemKindToProto(kind model.ThreadSystemKind) metav1.ThreadSystemKind {
	switch kind {
	case model.ThreadSystemNicknameUpdated:
		return metav1.ThreadSystemKind_THREAD_SYSTEM_KIND_NICKNAME_UPDATED
	case model.ThreadSystemEmojiUpdated:
		return metav1.ThreadSystemKind_THREAD_SYSTEM_KIND_EMOJI_UPDATED
	case model.ThreadSystemApprovalUpdated:
		return metav1.ThreadSystemKind_THREAD_SYSTEM_KIND_APPROVAL_MODE_UPDATED
	case model.ThreadSystemThemeUpdated:
		return metav1.ThreadSystemKind_THREAD_SYSTEM_KIND_THEME_UPDATED
	case model.ThreadSystemMemberAdded:
		return metav1.ThreadSystemKind_THREAD_SYSTEM_KIND_MEMBER_ADDED
	case model.ThreadSystemMemberRemoved:
		return metav1.ThreadSystemKind_THREAD_SYSTEM_KIND_MEMBER_REMOVED
	case model.ThreadSystemAdminUpdated:
		return metav1.ThreadSystemKind_THREAD_SYSTEM_KIND_PARTICIPANT_ADMIN_UPDATED
	case model.ThreadSystemPinUpdated:
		return metav1.ThreadSystemKind_THREAD_SYSTEM_KIND_MESSAGE_PIN_UPDATED
	case model.ThreadSystemPollUpdated:
		return metav1.ThreadSystemKind_THREAD_SYSTEM_KIND_POLL_UPDATED
	default:
		return metav1.ThreadSystemKind_THREAD_SYSTEM_KIND_UNSPECIFIED
	}
}

func transportToProto(kind model.TransportKind) metav1.TransportKind {
	switch kind {
	case model.TransportMessenger:
		return metav1.TransportKind_TRANSPORT_KIND_MESSENGER
	case model.TransportE2EE:
		return metav1.TransportKind_TRANSPORT_KIND_E2EE
	default:
		return metav1.TransportKind_TRANSPORT_KIND_UNSPECIFIED
	}
}

func encryptionToProto(kind model.EncryptionKind) metav1.EncryptionKind {
	switch kind {
	case model.EncryptionNone:
		return metav1.EncryptionKind_ENCRYPTION_KIND_NONE
	case model.EncryptionE2EE:
		return metav1.EncryptionKind_ENCRYPTION_KIND_E2EE
	default:
		return metav1.EncryptionKind_ENCRYPTION_KIND_UNSPECIFIED
	}
}
