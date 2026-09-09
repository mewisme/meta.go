// Package meta contains the in-process Messenger and E2EE transport engine.
package meta

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	fberrors "go.mewis.me/fbgo/errors"
	"go.mewis.me/fbgo/model"
)

var (
	ErrClosed       = errors.New("meta engine closed")
	ErrNotConnected = fberrors.ErrNotConnected
	ErrE2EENotReady = fberrors.ErrE2EENotReady
)

type Account struct {
	ID       model.ID
	Name     string
	Username string
}

type EventKind = model.EventKind
type Event = model.Event
type ReactionEvent = model.ReactionEvent
type TypingEvent = model.TypingEvent
type ReadReceiptEvent = model.ReadReceiptEvent
type DeliveryReceiptEvent = model.DeliveryReceiptEvent
type MessageEditEvent = model.MessageEditEvent
type MessageUnsendEvent = model.MessageUnsendEvent
type ThreadUpdateEvent = model.ThreadUpdateEvent
type MessageRequest = model.MessageRequest
type Theme = model.Theme
type Note = model.Note

const (
	EventReady           = model.EventReady
	EventReconnected     = model.EventReconnected
	EventDisconnected    = model.EventDisconnected
	EventError           = model.EventError
	EventMessage         = model.EventMessage
	EventReaction        = model.EventReaction
	EventTyping          = model.EventTyping
	EventReadReceipt     = model.EventReadReceipt
	EventDeliveryReceipt = model.EventDeliveryReceipt
	EventMessageEdit     = model.EventMessageEdit
	EventMessageUnsend   = model.EventMessageUnsend
	EventThreadUpdate    = model.EventThreadUpdate
	EventE2EEReady       = model.EventE2EEReady
	EventE2EEReceipt     = model.EventE2EEReceipt
)

type SendTextRequest struct {
	ThreadID      model.ID
	Text          string
	ReplyTo       model.ID
	Mentions      []model.Mention
	AttachmentIDs []model.ID
	StickerID     model.ID
	URL           string
}

type UploadRequest = model.UploadInput
type UploadResult = model.UploadResult

type backend interface {
	SetEventHandler(func(context.Context, any))
	Bootstrap(context.Context) (Account, error)
	Connect(context.Context, context.Context) error
	Disconnect()
	SendText(context.Context, SendTextRequest) (model.SendResult, error)
	Forward(context.Context, model.ID, model.ID) (model.SendResult, error)
	Upload(context.Context, UploadRequest) (UploadResult, error)
	React(context.Context, model.ID, model.ID, string) error
	Edit(context.Context, model.ID, string) error
	Unsend(context.Context, model.ID) error
	Typing(context.Context, model.ID, bool, bool, int64) error
	Read(context.Context, model.ID, time.Time) error
	ListMessageRequests(context.Context) ([]MessageRequest, error)
	ListThemes(context.Context) ([]Theme, error)
	SetTheme(context.Context, model.ID, model.ID) error
	CurrentNote(context.Context) (*Note, error)
	CreateNote(context.Context, string, string) (*Note, error)
	DeleteNote(context.Context, model.ID) error
	ListThreads(context.Context, int) (model.ThreadList, error)
	GetThread(context.Context, model.ID) (*model.Thread, error)
	CreatePoll(context.Context, model.ID, string, []string) error
	VotePoll(context.Context, model.ID, model.ID, []model.ID) error
	MuteThread(context.Context, model.ID, time.Duration) error
	SetThreadPhoto(context.Context, model.ID, model.AttachmentInput) error
	DeleteThread(context.Context, model.ID) error
	CreateDM(context.Context, model.ID) (model.ID, error)
	SearchMessengerUsers(context.Context, string) ([]model.User, error)
	GetMessengerContact(context.Context, model.ID) (*model.User, error)
	SetThreadAdmin(context.Context, model.ID, model.ID, bool) error
	SetThreadName(context.Context, model.ID, string) error
	SetThreadEmoji(context.Context, model.ID, string) error
	SetThreadNickname(context.Context, model.ID, model.ID, string) error
	GetFacebookUser(context.Context, model.ID) (*model.FacebookUser, error)
	SearchFacebook(context.Context, string, int) ([]model.SearchResult, error)
	ListNotifications(context.Context, int) ([]model.Notification, error)
	SetFacebookBio(context.Context, string, bool) error
	CreateAdditionalProfile(context.Context, string, string) error
	UnfriendFacebookUser(context.Context, model.ID) error
	SetFacebookBlocked(context.Context, model.ID, bool) error
	CreateFacebookPost(context.Context, string) (*model.Post, error)
	ArchiveFacebookPost(context.Context, model.ID, model.PostOwnership) error
	DeleteFacebookPost(context.Context, model.ID, model.PostOwnership) error
	CreateMarketplaceListing(context.Context, model.MarketplaceListingInput) (*model.MarketplaceListing, error)
	GetMarketplaceListing(context.Context, model.ID) (*model.MarketplaceListing, error)
	SetProfessionalMode(context.Context, bool) error
	SendE2EE(context.Context, model.E2EESendRequest) (model.SendResult, error)
	SendE2EEMedia(context.Context, model.E2EEMediaInput) (model.SendResult, error)
	DownloadE2EEMedia(context.Context, model.E2EEMediaDownload) ([]byte, error)
	ReactE2EE(context.Context, model.E2EEReactionRequest) error
	EditE2EE(context.Context, string, model.ID, string) error
	UnsendE2EE(context.Context, string, model.ID) error
	TypingE2EE(context.Context, string, bool) error
	ReadE2EE(context.Context, model.E2EEReadRequest) error
	ConnectE2EE(context.Context, model.ID) error
	E2EEConnected() bool
}

type Engine struct {
	backend backend

	ctx    context.Context
	cancel context.CancelFunc

	connectMu      sync.Mutex
	eventMu        sync.RWMutex
	handlerMu      sync.RWMutex
	closeOnce      sync.Once
	closed         atomic.Bool
	connected      atomic.Bool
	e2eeReady      atomic.Bool
	reconnect      atomic.Uint64
	dropped        atomic.Uint64
	lastSend       atomic.Int64
	lastRecv       atomic.Int64
	requestTimeout time.Duration
	regularState   atomic.Value
	e2eeState      atomic.Value
	lastError      atomic.Value

	events   chan Event
	handlers map[uint64]func(Event)
	nextID   uint64
}

func newEngine(parent context.Context, transport backend, eventBuffer int) *Engine {
	if parent == nil {
		parent = context.Background()
	}
	if eventBuffer < 1 {
		eventBuffer = 100
	}
	ctx, cancel := context.WithCancel(parent)
	e := &Engine{backend: transport, ctx: ctx, cancel: cancel, events: make(chan Event, eventBuffer), handlers: map[uint64]func(Event){}}
	e.regularState.Store(model.ConnectionDisconnected)
	e.e2eeState.Store(model.ConnectionDisconnected)
	e.lastError.Store("")
	transport.SetEventHandler(e.handleTransportEvent)
	return e
}

func (e *Engine) Connect(ctx context.Context) (Account, error) {
	e.connectMu.Lock()
	defer e.connectMu.Unlock()
	if e.closed.Load() {
		return Account{}, ErrClosed
	}
	e.regularState.Store(model.ConnectionConnecting)
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	account, err := e.backend.Bootstrap(ctx)
	if err != nil {
		e.regularState.Store(model.ConnectionFailed)
		e.recordError(err)
		return Account{}, err
	}
	if err := e.backend.Connect(e.ctx, ctx); err != nil {
		e.regularState.Store(model.ConnectionFailed)
		e.recordError(err)
		return Account{}, err
	}
	e.connected.Store(true)
	e.regularState.Store(model.ConnectionConnected)
	return account, nil
}

func (e *Engine) SendText(ctx context.Context, req SendTextRequest) (model.SendResult, error) {
	if !e.connected.Load() {
		return model.SendResult{}, ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	result, err := e.backend.SendText(ctx, req)
	if err == nil {
		e.recordSend(result.Timestamp)
	} else {
		e.recordError(err)
	}
	return result, err
}

func (e *Engine) Send(ctx context.Context, req model.SendRequest) (model.SendResult, error) {
	if !e.connected.Load() {
		return model.SendResult{}, ErrNotConnected
	}
	if req.Encryption == model.EncryptionRequired {
		return model.SendResult{}, ErrE2EENotReady
	}
	attachmentIDs := make([]model.ID, 0, len(req.Attachments))
	for _, attachment := range req.Attachments {
		result, err := e.Upload(ctx, UploadRequest{ThreadID: req.ThreadID, Name: attachment.Name, ContentType: attachment.ContentType, Reader: attachment.Reader, Size: attachment.Size})
		if err != nil {
			return model.SendResult{}, err
		}
		if result.ID.Empty() {
			return model.SendResult{}, errors.New("media upload returned empty attachment ID")
		}
		attachmentIDs = append(attachmentIDs, result.ID)
	}
	var replyTo model.ID
	if req.ReplyTo != nil {
		replyTo = req.ReplyTo.MessageID
	}
	return e.SendText(ctx, SendTextRequest{ThreadID: req.ThreadID, Text: req.Text, ReplyTo: replyTo, Mentions: req.Mentions, AttachmentIDs: attachmentIDs, StickerID: req.StickerID, URL: req.URL})
}

func (e *Engine) Forward(ctx context.Context, threadID, messageID model.ID) (model.SendResult, error) {
	if !e.connected.Load() {
		return model.SendResult{}, ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	result, err := e.backend.Forward(ctx, threadID, messageID)
	if err == nil {
		e.recordSend(result.Timestamp)
	} else {
		e.recordError(err)
	}
	return result, err
}

func (e *Engine) Upload(ctx context.Context, req UploadRequest) (UploadResult, error) {
	if !e.connected.Load() {
		return UploadResult{}, ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.Upload(ctx, req)
}

func (e *Engine) React(ctx context.Context, threadID, messageID model.ID, reaction string) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.React(ctx, threadID, messageID, reaction)
}

func (e *Engine) Edit(ctx context.Context, messageID model.ID, text string) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.Edit(ctx, messageID, text)
}

func (e *Engine) Unsend(ctx context.Context, messageID model.ID) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.Unsend(ctx, messageID)
}

func (e *Engine) Typing(ctx context.Context, threadID model.ID, typing, group bool, threadType int64) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.Typing(ctx, threadID, typing, group, threadType)
}

func (e *Engine) Read(ctx context.Context, threadID model.ID, watermark time.Time) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.Read(ctx, threadID, watermark)
}

func (e *Engine) ListMessageRequests(ctx context.Context) ([]MessageRequest, error) {
	if !e.connected.Load() {
		return nil, ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.ListMessageRequests(ctx)
}

func (e *Engine) ListThemes(ctx context.Context) ([]Theme, error) {
	if !e.connected.Load() {
		return nil, ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.ListThemes(ctx)
}

func (e *Engine) FindTheme(ctx context.Context, query string) (*Theme, error) {
	themes, err := e.ListThemes(ctx)
	if err != nil {
		return nil, err
	}
	normalized := strings.ToLower(strings.TrimSpace(query))
	if normalized == "" {
		return nil, errors.New("theme query is required")
	}
	for index := range themes {
		if themes[index].ID.String() == normalized {
			return &themes[index], nil
		}
	}
	for index := range themes {
		if strings.EqualFold(themes[index].Name, normalized) {
			return &themes[index], nil
		}
	}
	for index := range themes {
		if strings.Contains(strings.ToLower(themes[index].Name), normalized) {
			return &themes[index], nil
		}
	}
	return nil, errors.New("theme not found")
}

func (e *Engine) SetTheme(ctx context.Context, threadID, themeID model.ID) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.SetTheme(ctx, threadID, themeID)
}

func (e *Engine) CurrentNote(ctx context.Context) (*Note, error) {
	if !e.connected.Load() {
		return nil, ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.CurrentNote(ctx)
}

func (e *Engine) CreateNote(ctx context.Context, text, privacy string) (*Note, error) {
	if !e.connected.Load() {
		return nil, ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.CreateNote(ctx, text, privacy)
}

func (e *Engine) DeleteNote(ctx context.Context, noteID model.ID) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.DeleteNote(ctx, noteID)
}

func (e *Engine) RecreateNote(ctx context.Context, oldNoteID model.ID, text, privacy string) (*Note, error) {
	if err := e.DeleteNote(ctx, oldNoteID); err != nil {
		return nil, err
	}
	return e.CreateNote(ctx, text, privacy)
}

func (e *Engine) ListThreads(ctx context.Context, limit int) (model.ThreadList, error) {
	if !e.connected.Load() {
		return model.ThreadList{}, ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.ListThreads(ctx, limit)
}

func (e *Engine) GetThread(ctx context.Context, threadID model.ID) (*model.Thread, error) {
	if !e.connected.Load() {
		return nil, ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.GetThread(ctx, threadID)
}

func (e *Engine) CreatePoll(ctx context.Context, threadID model.ID, question string, options []string) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.CreatePoll(ctx, threadID, question, options)
}

func (e *Engine) VotePoll(ctx context.Context, threadID, pollID model.ID, optionIDs []model.ID) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.VotePoll(ctx, threadID, pollID, optionIDs)
}

func (e *Engine) MuteThread(ctx context.Context, threadID model.ID, duration time.Duration) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.MuteThread(ctx, threadID, duration)
}

func (e *Engine) SetThreadPhoto(ctx context.Context, threadID model.ID, input model.AttachmentInput) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.SetThreadPhoto(ctx, threadID, input)
}

func (e *Engine) DeleteThread(ctx context.Context, threadID model.ID) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.DeleteThread(ctx, threadID)
}

func (e *Engine) CreateDM(ctx context.Context, userID model.ID) (model.ID, error) {
	if !e.connected.Load() {
		return "", ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.CreateDM(ctx, userID)
}

func (e *Engine) SearchMessengerUsers(ctx context.Context, query string) ([]model.User, error) {
	if !e.connected.Load() {
		return nil, ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.SearchMessengerUsers(ctx, query)
}

func (e *Engine) GetMessengerContact(ctx context.Context, userID model.ID) (*model.User, error) {
	if !e.connected.Load() {
		return nil, ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.GetMessengerContact(ctx, userID)
}

func (e *Engine) SetThreadAdmin(ctx context.Context, threadID, userID model.ID, admin bool) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.SetThreadAdmin(ctx, threadID, userID, admin)
}

func (e *Engine) SetThreadName(ctx context.Context, threadID model.ID, name string) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.SetThreadName(ctx, threadID, name)
}

func (e *Engine) SetThreadEmoji(ctx context.Context, threadID model.ID, emoji string) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.SetThreadEmoji(ctx, threadID, emoji)
}

func (e *Engine) SetThreadNickname(ctx context.Context, threadID, userID model.ID, nickname string) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.SetThreadNickname(ctx, threadID, userID, nickname)
}

func (e *Engine) GetFacebookUser(ctx context.Context, userID model.ID) (*model.FacebookUser, error) {
	if !e.connected.Load() {
		return nil, ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.GetFacebookUser(ctx, userID)
}

func (e *Engine) SearchFacebook(ctx context.Context, query string, limit int) ([]model.SearchResult, error) {
	if !e.connected.Load() {
		return nil, ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.SearchFacebook(ctx, query, limit)
}

func (e *Engine) ListNotifications(ctx context.Context, limit int) ([]model.Notification, error) {
	if !e.connected.Load() {
		return nil, ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.ListNotifications(ctx, limit)
}

func (e *Engine) SetFacebookBio(ctx context.Context, bio string, publish bool) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.SetFacebookBio(ctx, bio, publish)
}

func (e *Engine) CreateAdditionalProfile(ctx context.Context, name, username string) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.CreateAdditionalProfile(ctx, name, username)
}

func (e *Engine) UnfriendFacebookUser(ctx context.Context, userID model.ID) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.UnfriendFacebookUser(ctx, userID)
}

func (e *Engine) SetFacebookBlocked(ctx context.Context, userID model.ID, blocked bool) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.SetFacebookBlocked(ctx, userID, blocked)
}

func (e *Engine) CreateFacebookPost(ctx context.Context, text string) (*model.Post, error) {
	if !e.connected.Load() {
		return nil, ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.CreateFacebookPost(ctx, text)
}

func (e *Engine) ArchiveFacebookPost(ctx context.Context, postID model.ID, ownership model.PostOwnership) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.ArchiveFacebookPost(ctx, postID, ownership)
}

func (e *Engine) DeleteFacebookPost(ctx context.Context, postID model.ID, ownership model.PostOwnership) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.DeleteFacebookPost(ctx, postID, ownership)
}

func (e *Engine) CreateMarketplaceListing(ctx context.Context, input model.MarketplaceListingInput) (*model.MarketplaceListing, error) {
	if !e.connected.Load() {
		return nil, ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.CreateMarketplaceListing(ctx, input)
}

func (e *Engine) GetMarketplaceListing(ctx context.Context, listingID model.ID) (*model.MarketplaceListing, error) {
	if !e.connected.Load() {
		return nil, ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.GetMarketplaceListing(ctx, listingID)
}

func (e *Engine) SetProfessionalMode(ctx context.Context, enabled bool) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.SetProfessionalMode(ctx, enabled)
}

func (e *Engine) SendE2EE(ctx context.Context, req model.E2EESendRequest) (model.SendResult, error) {
	if !e.E2EEConnected() {
		return model.SendResult{}, ErrE2EENotReady
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	result, err := e.backend.SendE2EE(ctx, req)
	if err == nil {
		e.recordSend(result.Timestamp)
	} else {
		e.recordError(err)
	}
	return result, err
}

func (e *Engine) SendE2EEMedia(ctx context.Context, req model.E2EEMediaInput) (model.SendResult, error) {
	if !e.E2EEConnected() {
		return model.SendResult{}, ErrE2EENotReady
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	result, err := e.backend.SendE2EEMedia(ctx, req)
	if err == nil {
		e.recordSend(result.Timestamp)
	} else {
		e.recordError(err)
	}
	return result, err
}

func (e *Engine) DownloadE2EEMedia(ctx context.Context, req model.E2EEMediaDownload) ([]byte, error) {
	if !e.E2EEConnected() {
		return nil, ErrE2EENotReady
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.DownloadE2EEMedia(ctx, req)
}

func (e *Engine) ReactE2EE(ctx context.Context, req model.E2EEReactionRequest) error {
	if !e.E2EEConnected() {
		return ErrE2EENotReady
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.ReactE2EE(ctx, req)
}

func (e *Engine) EditE2EE(ctx context.Context, chatJID string, messageID model.ID, text string) error {
	if !e.E2EEConnected() {
		return ErrE2EENotReady
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.EditE2EE(ctx, chatJID, messageID, text)
}

func (e *Engine) UnsendE2EE(ctx context.Context, chatJID string, messageID model.ID) error {
	if !e.E2EEConnected() {
		return ErrE2EENotReady
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.UnsendE2EE(ctx, chatJID, messageID)
}

func (e *Engine) TypingE2EE(ctx context.Context, chatJID string, typing bool) error {
	if !e.E2EEConnected() {
		return ErrE2EENotReady
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.TypingE2EE(ctx, chatJID, typing)
}

func (e *Engine) ReadE2EE(ctx context.Context, req model.E2EEReadRequest) error {
	if !e.E2EEConnected() {
		return ErrE2EENotReady
	}
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	return e.backend.ReadE2EE(ctx, req)
}

func (e *Engine) ConnectE2EE(ctx context.Context, accountID model.ID) error {
	e.connectMu.Lock()
	defer e.connectMu.Unlock()
	if e.closed.Load() {
		return ErrClosed
	}
	if !e.connected.Load() {
		return ErrNotConnected
	}
	e.e2eeState.Store(model.ConnectionConnecting)
	ctx, cancel := e.requestContext(ctx)
	defer cancel()
	if err := e.backend.ConnectE2EE(ctx, accountID); err != nil {
		e.e2eeReady.Store(false)
		e.e2eeState.Store(model.ConnectionFailed)
		e.recordError(err)
		return err
	}
	e.e2eeReady.Store(e.backend.E2EEConnected())
	if e.e2eeReady.Load() {
		e.e2eeState.Store(model.ConnectionConnected)
	} else {
		e.e2eeState.Store(model.ConnectionConnecting)
	}
	return nil
}

func (e *Engine) Events() <-chan Event { return e.events }
func (e *Engine) Connected() bool      { return e.connected.Load() && !e.closed.Load() }
func (e *Engine) E2EEConnected() bool  { return e.e2eeReady.Load() && !e.closed.Load() }

func (e *Engine) On(handler func(Event)) func() {
	if e == nil || handler == nil || e.closed.Load() {
		return func() {}
	}
	e.handlerMu.Lock()
	e.nextID++
	id := e.nextID
	e.handlers[id] = handler
	e.handlerMu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			e.handlerMu.Lock()
			delete(e.handlers, id)
			e.handlerMu.Unlock()
		})
	}
}

func (e *Engine) Health() model.HealthSnapshot {
	regular, _ := e.regularState.Load().(model.ConnectionState)
	e2ee, _ := e.e2eeState.Load().(model.ConnectionState)
	if regular == "" {
		regular = model.ConnectionDisconnected
	}
	if e2ee == "" {
		e2ee = model.ConnectionDisconnected
	}
	snapshot := model.HealthSnapshot{Regular: regular, E2EE: e2ee, ReconnectCount: e.reconnect.Load(), DroppedEventCount: e.dropped.Load()}
	if unixMilli := e.lastSend.Load(); unixMilli > 0 {
		snapshot.LastSuccessfulSend = time.UnixMilli(unixMilli)
	}
	if unixMilli := e.lastRecv.Load(); unixMilli > 0 {
		snapshot.LastReceive = time.UnixMilli(unixMilli)
	}
	if category, _ := e.lastError.Load().(string); category != "" {
		snapshot.LastErrorCategory = category
	}
	return snapshot
}

func (e *Engine) Close() {
	e.closeOnce.Do(func() {
		e.closed.Store(true)
		e.connected.Store(false)
		e.e2eeReady.Store(false)
		e.regularState.Store(model.ConnectionDisconnected)
		e.e2eeState.Store(model.ConnectionDisconnected)
		e.cancel()
		e.connectMu.Lock()
		e.backend.Disconnect()
		e.connectMu.Unlock()
		e.eventMu.Lock()
		close(e.events)
		e.eventMu.Unlock()
	})
}

func (e *Engine) recordSend(timestamp time.Time) {
	if timestamp.IsZero() {
		timestamp = time.Now()
	}
	e.lastSend.Store(timestamp.UnixMilli())
}

func (e *Engine) recordError(err error) {
	if err == nil {
		return
	}
	e.lastError.Store(string(fberrors.Classify(err)))
}

func (e *Engine) emit(event Event) {
	e.eventMu.RLock()
	if e.closed.Load() {
		e.eventMu.RUnlock()
		return
	}
	select {
	case e.events <- event:
	default:
		select {
		case <-e.events:
			e.dropped.Add(1)
		default:
		}
		select {
		case e.events <- event:
		default:
			e.dropped.Add(1)
		}
	}
	e.eventMu.RUnlock()
	e.handlerMu.RLock()
	handlers := make([]func(Event), 0, len(e.handlers))
	for _, handler := range e.handlers {
		handlers = append(handlers, handler)
	}
	e.handlerMu.RUnlock()
	for _, handler := range handlers {
		handler(event)
	}
}

func (e *Engine) requestContext(request context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := mergeContext(e.ctx, request)
	if e.requestTimeout <= 0 {
		return ctx, cancel
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, cancel
	}
	timed, timeoutCancel := context.WithTimeout(ctx, e.requestTimeout)
	return timed, func() {
		timeoutCancel()
		cancel()
	}
}

func mergeContext(parent, request context.Context) (context.Context, context.CancelFunc) {
	if request == nil {
		request = context.Background()
	}
	ctx, cancel := context.WithCancel(request)
	stop := context.AfterFunc(parent, cancel)
	return ctx, func() {
		stop()
		cancel()
	}
}
