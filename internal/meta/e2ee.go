package meta

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"go.mau.fi/mautrix-meta/pkg/messagix/methods"
	"go.mau.fi/mautrix-meta/pkg/messagix/socket"
	"go.mau.fi/mautrix-meta/pkg/messagix/table"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waConsumerApplication"
	"go.mau.fi/whatsmeow/proto/waMediaTransport"
	"go.mau.fi/whatsmeow/proto/waMsgApplication"
	waTypes "go.mau.fi/whatsmeow/types"
	"go.mewis.me/fbgo/model"
	"google.golang.org/protobuf/proto"
)

func (b *messagixBackend) SendE2EE(ctx context.Context, req model.E2EESendRequest) (model.SendResult, error) {
	client, err := b.requireE2EE()
	if err != nil {
		return model.SendResult{}, err
	}
	chat, err := e2eeChatJID(req.ChatJID, req.FacebookUserID)
	if err != nil {
		return model.SendResult{}, err
	}
	if err := b.ensureE2EEDM(ctx, chat); err != nil {
		return model.SendResult{}, err
	}
	text := req.Text
	message := &waConsumerApplication.ConsumerApplication{Payload: &waConsumerApplication.ConsumerApplication_Payload{Payload: &waConsumerApplication.ConsumerApplication_Payload_Content{Content: &waConsumerApplication.ConsumerApplication_Content{Content: &waConsumerApplication.ConsumerApplication_Content_MessageText{MessageText: &waCommon.MessageText{Text: &text}}}}}}
	metadata := replyMetadata(req.ReplyTo, req.ReplySenderJID)
	id := strconv.FormatInt(time.Now().UnixNano(), 10)
	resp, err := client.SendFBMessage(ctx, chat, message, metadata, whatsmeow.SendRequestExtra{ID: id})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "timed out waiting for message send response") {
			return model.SendResult{MessageID: model.ID(id), Timestamp: time.Now()}, nil
		}
		return model.SendResult{}, err
	}
	return model.SendResult{MessageID: model.ID(id), Timestamp: resp.Timestamp}, nil
}

func (b *messagixBackend) SendE2EEMedia(ctx context.Context, req model.E2EEMediaInput) (model.SendResult, error) {
	client, err := b.requireE2EE()
	if err != nil {
		return model.SendResult{}, err
	}
	chat, err := e2eeChatJID(req.ChatJID, req.FacebookUserID)
	if err != nil {
		return model.SendResult{}, err
	}
	if err := b.ensureE2EEDM(ctx, chat); err != nil {
		return model.SendResult{}, err
	}
	if req.Reader == nil {
		return model.SendResult{}, errors.New("nil E2EE media reader")
	}
	if req.Size > maxUploadBytes {
		return model.SendResult{}, fmt.Errorf("E2EE upload exceeds %d-byte limit", maxUploadBytes)
	}
	data, err := io.ReadAll(io.LimitReader(req.Reader, maxUploadBytes+1))
	if err != nil {
		return model.SendResult{}, err
	}
	if int64(len(data)) > maxUploadBytes {
		return model.SendResult{}, fmt.Errorf("E2EE upload exceeds %d-byte limit", maxUploadBytes)
	}
	mediaType, err := e2eeMediaType(req.Kind)
	if err != nil {
		return model.SendResult{}, err
	}
	uploaded, err := client.Upload(ctx, data, mediaType)
	if err != nil {
		return model.SendResult{}, err
	}
	contentType := strings.TrimSpace(req.ContentType)
	if contentType == "" {
		contentType = defaultE2EEContentType(req.Kind)
	}
	transport := &waMediaTransport.WAMediaTransport{
		Integral:  &waMediaTransport.WAMediaTransport_Integral{FileSHA256: uploaded.FileSHA256, MediaKey: uploaded.MediaKey, FileEncSHA256: uploaded.FileEncSHA256, DirectPath: &uploaded.DirectPath, MediaKeyTimestamp: proto.Int64(time.Now().Unix())},
		Ancillary: &waMediaTransport.WAMediaTransport_Ancillary{FileLength: proto.Uint64(uint64(len(data))), Mimetype: &contentType, ObjectID: &uploaded.ObjectID},
	}
	message, err := e2eeMediaMessage(req, transport)
	if err != nil {
		return model.SendResult{}, err
	}
	id := strconv.FormatInt(time.Now().UnixNano(), 10)
	resp, err := client.SendFBMessage(ctx, chat, message, replyMetadata(req.ReplyTo, req.ReplySenderJID), whatsmeow.SendRequestExtra{ID: id, MediaHandle: uploaded.Handle})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "timed out waiting for message send response") {
			return model.SendResult{MessageID: model.ID(id), Timestamp: time.Now()}, nil
		}
		return model.SendResult{}, err
	}
	return model.SendResult{MessageID: model.ID(id), Timestamp: resp.Timestamp}, nil
}

func (b *messagixBackend) DownloadE2EEMedia(ctx context.Context, req model.E2EEMediaDownload) ([]byte, error) {
	client, err := b.requireE2EE()
	if err != nil {
		return nil, err
	}
	ref := req.Reference
	directPath := ref.DirectPath
	transport := &waMediaTransport.WAMediaTransport_Integral{MediaKey: clone(ref.MediaKey), FileSHA256: clone(ref.FileSHA256), FileEncSHA256: clone(ref.FileEncSHA256), DirectPath: &directPath}
	mediaType, err := e2eeMediaType(ref.Kind)
	if err != nil {
		return nil, err
	}
	data, err := client.DownloadFB(ctx, transport, mediaType)
	if err != nil {
		return nil, err
	}
	if ref.Size > 0 && int64(len(data)) != ref.Size {
		return nil, fmt.Errorf("E2EE media size mismatch: expected %d, got %d", ref.Size, len(data))
	}
	return data, nil
}

func e2eeMediaType(kind model.E2EEMediaKind) (whatsmeow.MediaType, error) {
	switch kind {
	case model.E2EEMediaImage, model.E2EEMediaSticker:
		return whatsmeow.MediaImage, nil
	case model.E2EEMediaVideo:
		return whatsmeow.MediaVideo, nil
	case model.E2EEMediaAudio:
		return whatsmeow.MediaAudio, nil
	case model.E2EEMediaDocument:
		return whatsmeow.MediaDocument, nil
	default:
		return "", fmt.Errorf("unsupported E2EE media kind %q", kind)
	}
}

func defaultE2EEContentType(kind model.E2EEMediaKind) string {
	switch kind {
	case model.E2EEMediaImage:
		return "image/jpeg"
	case model.E2EEMediaVideo:
		return "video/mp4"
	case model.E2EEMediaAudio:
		return "audio/ogg; codecs=opus"
	case model.E2EEMediaSticker:
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

func e2eeMediaMessage(req model.E2EEMediaInput, transport *waMediaTransport.WAMediaTransport) (*waConsumerApplication.ConsumerApplication, error) {
	content := &waConsumerApplication.ConsumerApplication_Content{}
	width, height := req.Width, req.Height
	if width <= 0 {
		if req.Kind == model.E2EEMediaSticker {
			width = 512
		} else {
			width = 400
		}
	}
	if height <= 0 {
		if req.Kind == model.E2EEMediaSticker {
			height = 512
		} else {
			height = 400
		}
	}
	switch req.Kind {
	case model.E2EEMediaImage:
		message := &waConsumerApplication.ConsumerApplication_ImageMessage{}
		if req.Caption != "" {
			message.Caption = &waCommon.MessageText{Text: &req.Caption}
		}
		transport.Ancillary.Thumbnail = &waMediaTransport.WAMediaTransport_Ancillary_Thumbnail{ThumbnailWidth: proto.Uint32(uint32(width)), ThumbnailHeight: proto.Uint32(uint32(height))}
		if err := message.Set(&waMediaTransport.ImageTransport{Integral: &waMediaTransport.ImageTransport_Integral{Transport: transport}, Ancillary: &waMediaTransport.ImageTransport_Ancillary{Width: proto.Uint32(uint32(width)), Height: proto.Uint32(uint32(height))}}); err != nil {
			return nil, err
		}
		content.Content = &waConsumerApplication.ConsumerApplication_Content_ImageMessage{ImageMessage: message}
	case model.E2EEMediaVideo:
		message := &waConsumerApplication.ConsumerApplication_VideoMessage{}
		if req.Caption != "" {
			message.Caption = &waCommon.MessageText{Text: &req.Caption}
		}
		transport.Ancillary.Thumbnail = &waMediaTransport.WAMediaTransport_Ancillary_Thumbnail{ThumbnailWidth: proto.Uint32(uint32(width)), ThumbnailHeight: proto.Uint32(uint32(height))}
		gif := false
		if err := message.Set(&waMediaTransport.VideoTransport{Integral: &waMediaTransport.VideoTransport_Integral{Transport: transport}, Ancillary: &waMediaTransport.VideoTransport_Ancillary{Width: proto.Uint32(uint32(width)), Height: proto.Uint32(uint32(height)), Seconds: proto.Uint32(uint32(max(req.Duration, 0))), GifPlayback: &gif}}); err != nil {
			return nil, err
		}
		content.Content = &waConsumerApplication.ConsumerApplication_Content_VideoMessage{VideoMessage: message}
	case model.E2EEMediaAudio:
		message := &waConsumerApplication.ConsumerApplication_AudioMessage{PTT: &req.Voice}
		if err := message.Set(&waMediaTransport.AudioTransport{Integral: &waMediaTransport.AudioTransport_Integral{Transport: transport}, Ancillary: &waMediaTransport.AudioTransport_Ancillary{Seconds: proto.Uint32(uint32(max(req.Duration, 0)))}}); err != nil {
			return nil, err
		}
		content.Content = &waConsumerApplication.ConsumerApplication_Content_AudioMessage{AudioMessage: message}
	case model.E2EEMediaDocument:
		message := &waConsumerApplication.ConsumerApplication_DocumentMessage{FileName: &req.Name}
		if err := message.Set(&waMediaTransport.DocumentTransport{Integral: &waMediaTransport.DocumentTransport_Integral{Transport: transport}, Ancillary: &waMediaTransport.DocumentTransport_Ancillary{}}); err != nil {
			return nil, err
		}
		content.Content = &waConsumerApplication.ConsumerApplication_Content_DocumentMessage{DocumentMessage: message}
	case model.E2EEMediaSticker:
		message := &waConsumerApplication.ConsumerApplication_StickerMessage{}
		transport.Ancillary.Thumbnail = &waMediaTransport.WAMediaTransport_Ancillary_Thumbnail{ThumbnailWidth: proto.Uint32(uint32(width)), ThumbnailHeight: proto.Uint32(uint32(height))}
		if err := message.Set(&waMediaTransport.StickerTransport{Integral: &waMediaTransport.StickerTransport_Integral{Transport: transport}, Ancillary: &waMediaTransport.StickerTransport_Ancillary{Width: proto.Uint32(uint32(width)), Height: proto.Uint32(uint32(height))}}); err != nil {
			return nil, err
		}
		content.Content = &waConsumerApplication.ConsumerApplication_Content_StickerMessage{StickerMessage: message}
	default:
		return nil, fmt.Errorf("unsupported E2EE media kind %q", req.Kind)
	}
	return &waConsumerApplication.ConsumerApplication{Payload: &waConsumerApplication.ConsumerApplication_Payload{Payload: &waConsumerApplication.ConsumerApplication_Payload_Content{Content: content}}}, nil
}

func (b *messagixBackend) ReactE2EE(ctx context.Context, req model.E2EEReactionRequest) error {
	client, err := b.requireE2EE()
	if err != nil {
		return err
	}
	chat, err := waTypes.ParseJID(req.ChatJID)
	if err != nil {
		return err
	}
	sender, err := waTypes.ParseJID(req.SenderJID)
	if err != nil {
		return err
	}
	key := client.BuildMessageKey(chat, sender, req.MessageID.String())
	reaction := req.Reaction
	message := &waConsumerApplication.ConsumerApplication{Payload: &waConsumerApplication.ConsumerApplication_Payload{Payload: &waConsumerApplication.ConsumerApplication_Payload_Content{Content: &waConsumerApplication.ConsumerApplication_Content{Content: &waConsumerApplication.ConsumerApplication_Content_ReactionMessage{ReactionMessage: &waConsumerApplication.ConsumerApplication_ReactionMessage{Key: key, Text: &reaction}}}}}}
	_, err = client.SendFBMessage(ctx, chat, message, nil, whatsmeow.SendRequestExtra{ID: strconv.FormatInt(time.Now().UnixNano(), 10)})
	return err
}

func (b *messagixBackend) EditE2EE(ctx context.Context, chatJID string, messageID model.ID, text string) error {
	client, err := b.requireE2EE()
	if err != nil {
		return err
	}
	chat, err := waTypes.ParseJID(chatJID)
	if err != nil {
		return err
	}
	key := client.BuildMessageKey(chat, waTypes.EmptyJID, messageID.String())
	timestamp := time.Now().UnixMilli()
	message := &waConsumerApplication.ConsumerApplication{Payload: &waConsumerApplication.ConsumerApplication_Payload{Payload: &waConsumerApplication.ConsumerApplication_Payload_Content{Content: &waConsumerApplication.ConsumerApplication_Content{Content: &waConsumerApplication.ConsumerApplication_Content_EditMessage{EditMessage: &waConsumerApplication.ConsumerApplication_EditMessage{Key: key, Message: &waCommon.MessageText{Text: &text}, TimestampMS: &timestamp}}}}}}
	_, err = client.SendFBMessage(ctx, chat, message, nil, whatsmeow.SendRequestExtra{ID: strconv.FormatInt(time.Now().UnixNano(), 10)})
	return err
}

func (b *messagixBackend) UnsendE2EE(ctx context.Context, chatJID string, messageID model.ID) error {
	client, err := b.requireE2EE()
	if err != nil {
		return err
	}
	chat, err := waTypes.ParseJID(chatJID)
	if err != nil {
		return err
	}
	key := client.BuildMessageKey(chat, waTypes.EmptyJID, messageID.String())
	message := &waConsumerApplication.ConsumerApplication{Payload: &waConsumerApplication.ConsumerApplication_Payload{Payload: &waConsumerApplication.ConsumerApplication_Payload_ApplicationData{ApplicationData: &waConsumerApplication.ConsumerApplication_ApplicationData{ApplicationContent: &waConsumerApplication.ConsumerApplication_ApplicationData_Revoke{Revoke: &waConsumerApplication.ConsumerApplication_RevokeMessage{Key: key}}}}}}
	_, err = client.SendFBMessage(ctx, chat, message, nil, whatsmeow.SendRequestExtra{ID: strconv.FormatInt(time.Now().UnixNano(), 10)})
	return err
}

func (b *messagixBackend) TypingE2EE(ctx context.Context, chatJID string, typing bool) error {
	client, err := b.requireE2EE()
	if err != nil {
		return err
	}
	chat, err := waTypes.ParseJID(chatJID)
	if err != nil {
		return err
	}
	state := waTypes.ChatPresencePaused
	if typing {
		state = waTypes.ChatPresenceComposing
	}
	return client.SendChatPresence(ctx, chat, state, waTypes.ChatPresenceMediaText)
}

func (b *messagixBackend) ReadE2EE(ctx context.Context, req model.E2EEReadRequest) error {
	client, err := b.requireE2EE()
	if err != nil {
		return err
	}
	chat, err := waTypes.ParseJID(req.ChatJID)
	if err != nil {
		return err
	}
	sender, err := waTypes.ParseJID(req.SenderJID)
	if err != nil {
		return err
	}
	ids := make([]waTypes.MessageID, len(req.MessageIDs))
	for i, id := range req.MessageIDs {
		ids[i] = waTypes.MessageID(id.String())
	}
	timestamp := req.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now()
	}
	return client.MarkRead(ctx, ids, timestamp, chat, sender)
}

func (b *messagixBackend) requireE2EE() (*whatsmeow.Client, error) {
	if b.e2ee == nil || !b.e2ee.IsConnected() || !b.e2ee.IsLoggedIn() {
		return nil, ErrE2EENotReady
	}
	return b.e2ee, nil
}

func e2eeChatJID(raw string, userID model.ID) (waTypes.JID, error) {
	if raw != "" {
		return waTypes.ParseJID(raw)
	}
	if userID.Empty() {
		return waTypes.EmptyJID, errors.New("E2EE chat target is required")
	}
	if _, err := strconv.ParseInt(userID.String(), 10, 64); err != nil {
		return waTypes.EmptyJID, fmt.Errorf("invalid Facebook user ID %q", userID)
	}
	return waTypes.NewJID(userID.String(), waTypes.MessengerServer), nil
}

func (b *messagixBackend) ensureE2EEDM(ctx context.Context, chat waTypes.JID) error {
	if chat.Server != waTypes.MessengerServer || chat.User == "" {
		return nil
	}
	threadID, err := strconv.ParseInt(chat.User, 10, 64)
	if err != nil {
		return err
	}
	resp, err := b.client.ExecuteTasks(ctx, &socket.CreateWhatsAppThreadTask{WAJID: threadID, OfflineThreadKey: methods.GenerateEpochID(), ThreadType: table.ENCRYPTED_OVER_WA_ONE_TO_ONE, FolderType: table.INBOX, BumpTimestampMS: time.Now().UnixMilli(), TAMThreadSubtype: 0})
	if err != nil || resp == nil || len(resp.LSIssueNewTask) == 0 {
		return err
	}
	tasks := make([]socket.Task, len(resp.LSIssueNewTask))
	for i, task := range resp.LSIssueNewTask {
		tasks[i] = task
	}
	_, err = b.client.ExecuteTasks(ctx, tasks...)
	return err
}

func replyMetadata(messageID model.ID, senderJID string) *waMsgApplication.MessageApplication_Metadata {
	if messageID.Empty() {
		return nil
	}
	id := messageID.String()
	metadata := &waMsgApplication.MessageApplication_Metadata{QuotedMessage: &waMsgApplication.MessageApplication_Metadata_QuotedMessage{StanzaID: &id}}
	if senderJID != "" {
		metadata.QuotedMessage.Participant = &senderJID
	}
	return metadata
}
