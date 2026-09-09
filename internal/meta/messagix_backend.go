package meta

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"go.mau.fi/mautrix-meta/pkg/messagix"
	"go.mau.fi/mautrix-meta/pkg/messagix/cookies"
	"go.mau.fi/mautrix-meta/pkg/messagix/socket"
	"go.mau.fi/mautrix-meta/pkg/messagix/table"
	metaTypes "go.mau.fi/mautrix-meta/pkg/messagix/types"
	"go.mau.fi/whatsmeow"
	"go.mewis.me/fbgo/model"
)

const maxUploadBytes int64 = 100 << 20

type Config struct {
	Cookies     map[string]string
	Platform    string
	Logger      zerolog.Logger
	EventBuffer int
	DeviceStore *DeviceStore
}

type messagixBackend struct {
	client      *messagix.Client
	deviceStore *DeviceStore
	e2ee        *whatsmeow.Client
	handler     func(context.Context, any)
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
	client := messagix.NewClient(jar, cfg.Logger, &messagix.Config{})
	deviceStore := cfg.DeviceStore
	if deviceStore == nil {
		var err error
		deviceStore, err = NewMemoryDeviceStore()
		if err != nil {
			return nil, err
		}
	}
	client.SetDevice(deviceStore.Device())
	return newEngine(parent, &messagixBackend{client: client, deviceStore: deviceStore}, cfg.EventBuffer), nil
}

func (b *messagixBackend) SetEventHandler(handler func(context.Context, any)) {
	b.handler = handler
	b.client.SetEventHandler(handler)
}

func (b *messagixBackend) Bootstrap(ctx context.Context) (Account, error) {
	user, _, err := b.client.LoadMessagesPage(ctx)
	if err != nil {
		return Account{}, err
	}
	return Account{ID: id64(user.GetFBID()), Name: user.GetName(), Username: user.GetUsername()}, nil
}

func (b *messagixBackend) Connect(lifetimeCtx, startupCtx context.Context) error {
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
		e2ee.AddEventHandler(func(event any) { b.handler(ctx, event) })
	}
	b.e2ee = e2ee
	if err := b.client.RegisterE2EE(ctx, fbid); err != nil {
		return err
	}
	if err := b.deviceStore.Device().Save(ctx); err != nil {
		return err
	}
	return e2ee.ConnectContext(ctx)
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
	task := &socket.SendMessageTask{ThreadId: threadID, Otid: otid, Text: req.Text, Source: table.MESSENGER_INBOX_IN_THREAD, SendType: table.TEXT, SyncGroup: 1}
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
	return model.SendResult{MessageID: messageID, Timestamp: time.Now()}, nil
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
	response, err := b.client.SendMercuryUploadRequest(ctx, threadID, &messagix.MercuryUploadMedia{Filename: req.Name, MimeType: req.ContentType, MediaData: data, IsVoiceClip: req.Voice})
	if err != nil {
		return UploadResult{}, err
	}
	var id int64
	if response != nil && response.Payload.RealMetadata != nil {
		id = response.Payload.RealMetadata.GetFbId()
	}
	return UploadResult{ID: id64(id), Name: req.Name}, nil
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
