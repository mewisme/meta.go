package meta

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"go.mau.fi/util/exhttp"
	"go.mau.fi/whatsmeow"
	"go.mewis.me/fbgo/internal/webapi"
	"go.mewis.me/fbgo/model"
	"go.mewis.me/meta-extra/pkg/messagix"
	"go.mewis.me/meta-extra/pkg/messagix/cookies"
	metaHTTP "go.mewis.me/meta-extra/pkg/messagix/httpclient"
	"go.mewis.me/meta-extra/pkg/messagix/socket"
	"go.mewis.me/meta-extra/pkg/messagix/table"
	metaTypes "go.mewis.me/meta-extra/pkg/messagix/types"
)

const maxUploadBytes int64 = 100 << 20

type Config struct {
	Cookies     map[string]string
	Platform    string
	Logger      zerolog.Logger
	EventBuffer int
	DeviceStore *DeviceStore
	HTTPClient  *http.Client
	Timeout     time.Duration
}

type messagixBackend struct {
	client         *messagix.Client
	deviceStore    *DeviceStore
	e2ee           *whatsmeow.Client
	handler        func(context.Context, any)
	requestCounter webapi.RequestCounter
	browserStateMu sync.Mutex
	browserState   *browserFormState
	lifetimeCtx    context.Context
}

func New(parent context.Context, cfg Config) (*Engine, error) {
	platform := metaTypes.PlatformFromString(cfg.Platform)
	if !platform.IsValid() {
		platform = metaTypes.Facebook
	}
	jar := &cookies.Cookies{Platform: platform}
	values := make(map[cookies.MetaCookieName]string, len(cfg.Cookies))
	for key, value := range cfg.Cookies {
		values[cookies.MetaCookieName(key)] = value
	}
	jar.UpdateValues(values)
	settings := exhttp.ClientSettings{}
	if cfg.HTTPClient != nil {
		transport := cfg.HTTPClient.Transport
		if transport == nil {
			transport = http.DefaultTransport
		}
		settings.TransportOverride = func(exhttp.ClientSettings) http.RoundTripper { return transport }
	}
	client := messagix.NewClient(jar, cfg.Logger, &messagix.Config{ClientSettings: settings})
	deviceStore := cfg.DeviceStore
	if deviceStore == nil {
		var err error
		deviceStore, err = NewMemoryDeviceStore()
		if err != nil {
			return nil, err
		}
	}
	client.SetDevice(deviceStore.Device())
	engine := newEngine(parent, &messagixBackend{client: client, deviceStore: deviceStore}, cfg.EventBuffer)
	engine.requestTimeout = cfg.Timeout
	return engine, nil
}

func (b *messagixBackend) SetEventHandler(handler func(context.Context, any)) {
	b.handler = handler
	b.client.SetEventHandler(handler)
}

func (b *messagixBackend) transportContext() context.Context {
	if b.lifetimeCtx != nil {
		return b.lifetimeCtx
	}
	return context.Background()
}

func (b *messagixBackend) Bootstrap(ctx context.Context) (Account, error) {
	user, _, err := b.client.LoadMessagesPage(ctx)
	if err != nil {
		return Account{}, err
	}
	return Account{ID: id64(user.GetFBID()), Name: user.GetName(), Username: user.GetUsername()}, nil
}

func (b *messagixBackend) Connect(lifetimeCtx, startupCtx context.Context) error {
	b.lifetimeCtx = lifetimeCtx
	if err := b.client.Connect(lifetimeCtx); err != nil {
		return err
	}
	if err := b.client.WaitUntilCanSendMessages(startupCtx, 30*time.Second); err != nil {
		b.client.Disconnect()
		return err
	}
	return nil
}
func (b *messagixBackend) Disconnect() {
	if b.e2ee != nil && b.e2ee.IsConnected() {
		b.e2ee.Disconnect()
	}
	b.client.Disconnect()
}

func (b *messagixBackend) ConnectE2EE(ctx context.Context, accountID model.ID) error {
	if b.E2EEConnected() {
		return nil
	}
	fbid, err := strconv.ParseInt(accountID.String(), 10, 64)
	if err != nil || fbid == 0 {
		return fmt.Errorf("invalid account ID %q", accountID)
	}
	e2ee, err := b.client.PrepareE2EEClient()
	if err != nil {
		return err
	}
	if b.handler != nil {
		handlerCtx := b.transportContext()
		e2ee.AddEventHandler(func(event any) { b.handler(handlerCtx, event) })
	}
	b.e2ee = e2ee
	if err := b.client.RegisterE2EE(ctx, fbid); err != nil {
		return err
	}
	if err := b.deviceStore.Device().Save(ctx); err != nil {
		return err
	}
	if err := e2ee.ConnectContext(b.transportContext()); err != nil {
		return err
	}
	if !e2ee.WaitForConnection(15*time.Second) || !e2ee.IsLoggedIn() {
		e2ee.Disconnect()
		return ErrE2EENotReady
	}
	return nil
}

func (b *messagixBackend) E2EEConnected() bool {
	return b.e2ee != nil && b.e2ee.IsConnected() && b.e2ee.IsLoggedIn()
}

func (b *messagixBackend) SendText(ctx context.Context, req SendTextRequest) (model.SendResult, error) {
	threadID, err := strconv.ParseInt(req.ThreadID.String(), 10, 64)
	if err != nil || threadID == 0 {
		return model.SendResult{}, fmt.Errorf("invalid thread ID %q", req.ThreadID)
	}
	if err := b.client.WaitUntilCanSendMessages(ctx, 10*time.Second); err != nil {
		return model.SendResult{}, err
	}
	otid := time.Now().UnixNano()
	sendType := table.TEXT
	attachmentIDs := make([]int64, 0, len(req.AttachmentIDs))
	for _, attachmentID := range req.AttachmentIDs {
		id, err := strconv.ParseInt(attachmentID.String(), 10, 64)
		if err != nil || id == 0 {
			return model.SendResult{}, fmt.Errorf("invalid attachment ID %q", attachmentID)
		}
		attachmentIDs = append(attachmentIDs, id)
	}
	stickerID := int64(0)
	if !req.StickerID.Empty() {
		stickerID, err = parsePositiveID(req.StickerID, "sticker")
		if err != nil {
			return model.SendResult{}, err
		}
		sendType = table.STICKER
	} else if len(attachmentIDs) > 0 {
		sendType = table.MEDIA
	} else if strings.TrimSpace(req.URL) != "" {
		sendType = table.EXTERNAL_MEDIA
	}
	task := &socket.SendMessageTask{ThreadId: threadID, Otid: otid, Text: req.Text, Source: table.MESSENGER_INBOX_IN_THREAD, SendType: sendType, AttachmentFBIds: attachmentIDs, StickerId: stickerID, Url: strings.TrimSpace(req.URL), SyncGroup: 1}
	if req.ReplyTo != "" {
		task.ReplyMetaData = &socket.ReplyMetaData{ReplyMessageId: req.ReplyTo.String(), ReplySourceType: 1}
	}
	if len(req.Mentions) > 0 {
		task.MentionData = mentionData(req.Mentions)
	}
	tbl, err := b.client.ExecuteTasks(ctx, task)
	if err != nil {
		return model.SendResult{}, err
	}
	return sendResult(tbl, otid), nil
}

func (b *messagixBackend) Forward(ctx context.Context, threadID, messageID model.ID) (model.SendResult, error) {
	thread, err := parsePositiveID(threadID, "thread")
	if err != nil {
		return model.SendResult{}, err
	}
	if messageID.Empty() {
		return model.SendResult{}, errors.New("message ID is required")
	}
	if err := b.client.WaitUntilCanSendMessages(ctx, 10*time.Second); err != nil {
		return model.SendResult{}, err
	}
	otid := time.Now().UnixNano()
	tbl, err := b.client.ExecuteTasks(ctx, &socket.SendMessageTask{ThreadId: thread, Otid: otid, Source: table.MESSENGER_INBOX_IN_THREAD, SendType: table.FORWARD, ForwardedMsgId: messageID.String(), SyncGroup: 1})
	if err != nil {
		return model.SendResult{}, err
	}
	return sendResult(tbl, otid), nil
}

func sendResult(tbl *table.LSTable, otid int64) model.SendResult {
	messageID := model.ID("mid.$" + strconv.FormatInt(otid, 10))
	if tbl != nil {
		otidString := strconv.FormatInt(otid, 10)
		for _, replacement := range tbl.LSReplaceOptimsiticMessage {
			if replacement != nil && replacement.OfflineThreadingId == otidString {
				messageID = model.ID(replacement.MessageId)
				break
			}
		}
	}
	return model.SendResult{MessageID: messageID, Timestamp: time.Now()}
}

func (b *messagixBackend) Upload(ctx context.Context, req UploadRequest) (UploadResult, error) {
	threadID, err := strconv.ParseInt(req.ThreadID.String(), 10, 64)
	if err != nil || threadID == 0 {
		return UploadResult{}, fmt.Errorf("invalid thread ID %q", req.ThreadID)
	}
	if req.Reader == nil {
		return UploadResult{}, errors.New("nil upload reader")
	}
	if req.Size > maxUploadBytes {
		return UploadResult{}, fmt.Errorf("upload exceeds %d-byte limit", maxUploadBytes)
	}
	data, err := io.ReadAll(io.LimitReader(req.Reader, maxUploadBytes+1))
	if err != nil {
		return UploadResult{}, err
	}
	if int64(len(data)) > maxUploadBytes {
		return UploadResult{}, fmt.Errorf("upload exceeds %d-byte limit", maxUploadBytes)
	}
	response, err := b.client.GetHTTP().SendMercuryUploadRequest(ctx, threadID, &metaHTTP.MercuryUploadMedia{Filename: req.Name, MimeType: req.ContentType, MediaData: data, IsVoiceClip: req.Voice})
	if err != nil {
		return UploadResult{}, err
	}
	var id int64
	if response != nil && response.Payload.RealMetadata != nil {
		id = response.Payload.RealMetadata.GetFbId()
	}
	return UploadResult{ID: id64(id), Name: req.Name, ContentType: req.ContentType, Type: uploadType(req.ContentType)}, nil
}

func uploadType(contentType string) string {
	contentType = strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
	switch {
	case contentType == "image/gif":
		return "gif"
	case strings.HasPrefix(contentType, "image/"):
		return "image"
	case strings.HasPrefix(contentType, "video/"):
		return "video"
	case strings.HasPrefix(contentType, "audio/"):
		return "audio"
	default:
		return "file"
	}
}

func (b *messagixBackend) React(ctx context.Context, threadID, messageID model.ID, reaction string) error {
	thread, err := strconv.ParseInt(threadID.String(), 10, 64)
	if err != nil || thread == 0 {
		return fmt.Errorf("invalid thread ID %q", threadID)
	}
	account, err := b.client.GetCurrentAccount()
	if err != nil {
		return err
	}
	_, err = b.client.ExecuteTasks(ctx, &socket.SendReactionTask{ThreadKey: thread, MessageID: messageID.String(), ActorID: account.GetFBID(), Reaction: reaction, SendAttribution: table.MESSENGER_INBOX_IN_THREAD})
	return err
}

func (b *messagixBackend) Edit(ctx context.Context, messageID model.ID, text string) error {
	if messageID.Empty() {
		return errors.New("message ID is required")
	}
	if text == "" {
		return errors.New("edit text is required")
	}
	_, err := b.client.ExecuteTasks(ctx, &socket.EditMessageTask{MessageID: messageID.String(), Text: text})
	return err
}

func (b *messagixBackend) Unsend(ctx context.Context, messageID model.ID) error {
	if messageID.Empty() {
		return errors.New("message ID is required")
	}
	_, err := b.client.ExecuteTasks(ctx, &socket.DeleteMessageTask{MessageId: messageID.String()})
	return err
}

func (b *messagixBackend) Typing(ctx context.Context, threadID model.ID, typing, group bool, threadType int64) error {
	thread, err := strconv.ParseInt(threadID.String(), 10, 64)
	if err != nil || thread == 0 {
		return fmt.Errorf("invalid thread ID %q", threadID)
	}
	if threadType <= 0 {
		threadType = 1
	}
	typingValue, groupValue := int64(0), int64(0)
	if typing {
		typingValue = 1
	}
	if group {
		groupValue = 1
	}
	return b.client.ExecuteStatelessTask(ctx, &socket.UpdatePresenceTask{ThreadKey: thread, IsGroupThread: groupValue, IsTyping: typingValue, Attribution: 0, SyncGroup: 1, ThreadType: threadType})
}

func (b *messagixBackend) Read(ctx context.Context, threadID model.ID, watermark time.Time) error {
	thread, err := strconv.ParseInt(threadID.String(), 10, 64)
	if err != nil || thread == 0 {
		return fmt.Errorf("invalid thread ID %q", threadID)
	}
	if watermark.IsZero() {
		watermark = time.Now()
	}
	_, err = b.client.ExecuteTasks(ctx, &socket.ThreadMarkReadTask{ThreadId: thread, LastReadWatermarkTs: watermark.UnixMilli(), SyncGroup: 1})
	return err
}

func mentionData(mentions []model.Mention) *socket.MentionData {
	ids, offsets, lengths, types := "", "", "", ""
	for index, mention := range mentions {
		separator := ""
		if index > 0 {
			separator = ","
		}
		ids += separator + mention.UserID.String()
		offsets += separator + strconv.Itoa(mention.Offset)
		lengths += separator + strconv.Itoa(mention.Length)
		types += separator + "p"
	}
	return &socket.MentionData{MentionIDs: ids, MentionOffsets: offsets, MentionLengths: lengths, MentionTypes: types}
}
