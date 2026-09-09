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
)

type SendTextRequest struct {
	ThreadID      model.ID
	Text          string
	ReplyTo       model.ID
	Mentions      []model.Mention
	AttachmentIDs []model.ID
}

type UploadRequest = model.UploadInput
type UploadResult = model.UploadResult

type backend interface {
	SetEventHandler(func(context.Context, any))
	Bootstrap(context.Context) (Account, error)
	Connect(context.Context, context.Context) error
	Disconnect()
	SendText(context.Context, SendTextRequest) (model.SendResult, error)
	Upload(context.Context, UploadRequest) (UploadResult, error)
	React(context.Context, model.ID, model.ID, string) error
	Edit(context.Context, model.ID, string) error
	Unsend(context.Context, model.ID) error
	ListMessageRequests(context.Context) ([]MessageRequest, error)
	ListThemes(context.Context) ([]Theme, error)
	SetTheme(context.Context, model.ID, model.ID) error
	CurrentNote(context.Context) (*Note, error)
	CreateNote(context.Context, string, string) (*Note, error)
	DeleteNote(context.Context, model.ID) error
	ConnectE2EE(context.Context, model.ID) error
	E2EEConnected() bool
}

type Engine struct {
	backend backend

	ctx    context.Context
	cancel context.CancelFunc

	connectMu sync.Mutex
	eventMu   sync.RWMutex
	closeOnce sync.Once
	closed    atomic.Bool
	connected atomic.Bool
	e2eeReady atomic.Bool
	reconnect atomic.Uint64
	dropped   atomic.Uint64
	lastRecv  atomic.Int64

	events chan Event
}

func newEngine(parent context.Context, transport backend, eventBuffer int) *Engine {
	if parent == nil {
		parent = context.Background()
	}
	if eventBuffer < 1 {
		eventBuffer = 100
	}
	ctx, cancel := context.WithCancel(parent)
	e := &Engine{backend: transport, ctx: ctx, cancel: cancel, events: make(chan Event, eventBuffer)}
	transport.SetEventHandler(e.handleTransportEvent)
	return e
}

func (e *Engine) Connect(ctx context.Context) (Account, error) {
	e.connectMu.Lock()
	defer e.connectMu.Unlock()
	if e.closed.Load() {
		return Account{}, ErrClosed
	}
	ctx, cancel := mergeContext(e.ctx, ctx)
	defer cancel()
	account, err := e.backend.Bootstrap(ctx)
	if err != nil {
		return Account{}, err
	}
	if err := e.backend.Connect(e.ctx, ctx); err != nil {
		return Account{}, err
	}
	e.connected.Store(true)
	return account, nil
}

func (e *Engine) SendText(ctx context.Context, req SendTextRequest) (model.SendResult, error) {
	if !e.connected.Load() {
		return model.SendResult{}, ErrNotConnected
	}
	ctx, cancel := mergeContext(e.ctx, ctx)
	defer cancel()
	return e.backend.SendText(ctx, req)
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
	return e.SendText(ctx, SendTextRequest{ThreadID: req.ThreadID, Text: req.Text, ReplyTo: replyTo, Mentions: req.Mentions, AttachmentIDs: attachmentIDs})
}

func (e *Engine) Upload(ctx context.Context, req UploadRequest) (UploadResult, error) {
	if !e.connected.Load() {
		return UploadResult{}, ErrNotConnected
	}
	ctx, cancel := mergeContext(e.ctx, ctx)
	defer cancel()
	return e.backend.Upload(ctx, req)
}

func (e *Engine) React(ctx context.Context, threadID, messageID model.ID, reaction string) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := mergeContext(e.ctx, ctx)
	defer cancel()
	return e.backend.React(ctx, threadID, messageID, reaction)
}

func (e *Engine) Edit(ctx context.Context, messageID model.ID, text string) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := mergeContext(e.ctx, ctx)
	defer cancel()
	return e.backend.Edit(ctx, messageID, text)
}

func (e *Engine) Unsend(ctx context.Context, messageID model.ID) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := mergeContext(e.ctx, ctx)
	defer cancel()
	return e.backend.Unsend(ctx, messageID)
}

func (e *Engine) ListMessageRequests(ctx context.Context) ([]MessageRequest, error) {
	if !e.connected.Load() {
		return nil, ErrNotConnected
	}
	ctx, cancel := mergeContext(e.ctx, ctx)
	defer cancel()
	return e.backend.ListMessageRequests(ctx)
}

func (e *Engine) ListThemes(ctx context.Context) ([]Theme, error) {
	if !e.connected.Load() {
		return nil, ErrNotConnected
	}
	ctx, cancel := mergeContext(e.ctx, ctx)
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
	ctx, cancel := mergeContext(e.ctx, ctx)
	defer cancel()
	return e.backend.SetTheme(ctx, threadID, themeID)
}

func (e *Engine) CurrentNote(ctx context.Context) (*Note, error) {
	if !e.connected.Load() {
		return nil, ErrNotConnected
	}
	ctx, cancel := mergeContext(e.ctx, ctx)
	defer cancel()
	return e.backend.CurrentNote(ctx)
}

func (e *Engine) CreateNote(ctx context.Context, text, privacy string) (*Note, error) {
	if !e.connected.Load() {
		return nil, ErrNotConnected
	}
	ctx, cancel := mergeContext(e.ctx, ctx)
	defer cancel()
	return e.backend.CreateNote(ctx, text, privacy)
}

func (e *Engine) DeleteNote(ctx context.Context, noteID model.ID) error {
	if !e.connected.Load() {
		return ErrNotConnected
	}
	ctx, cancel := mergeContext(e.ctx, ctx)
	defer cancel()
	return e.backend.DeleteNote(ctx, noteID)
}

func (e *Engine) RecreateNote(ctx context.Context, oldNoteID model.ID, text, privacy string) (*Note, error) {
	if err := e.DeleteNote(ctx, oldNoteID); err != nil {
		return nil, err
	}
	return e.CreateNote(ctx, text, privacy)
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
	ctx, cancel := mergeContext(e.ctx, ctx)
	defer cancel()
	if err := e.backend.ConnectE2EE(ctx, accountID); err != nil {
		e.e2eeReady.Store(false)
		return err
	}
	e.e2eeReady.Store(e.backend.E2EEConnected())
	return nil
}

func (e *Engine) Events() <-chan Event { return e.events }
func (e *Engine) Connected() bool      { return e.connected.Load() && !e.closed.Load() }
func (e *Engine) E2EEConnected() bool  { return e.e2eeReady.Load() && !e.closed.Load() }

func (e *Engine) Health() model.HealthSnapshot {
	regular := model.ConnectionDisconnected
	if e.Connected() {
		regular = model.ConnectionConnected
	}
	e2ee := model.ConnectionDisconnected
	if e.E2EEConnected() {
		e2ee = model.ConnectionConnected
	}
	snapshot := model.HealthSnapshot{Regular: regular, E2EE: e2ee, ReconnectCount: e.reconnect.Load(), DroppedEventCount: e.dropped.Load()}
	if unixMilli := e.lastRecv.Load(); unixMilli > 0 {
		snapshot.LastReceive = time.UnixMilli(unixMilli)
	}
	return snapshot
}

func (e *Engine) Close() {
	e.closeOnce.Do(func() {
		e.closed.Store(true)
		e.connected.Store(false)
		e.e2eeReady.Store(false)
		e.cancel()
		e.connectMu.Lock()
		e.backend.Disconnect()
		e.connectMu.Unlock()
		e.eventMu.Lock()
		close(e.events)
		e.eventMu.Unlock()
	})
}

func (e *Engine) emit(event Event) {
	e.eventMu.RLock()
	defer e.eventMu.RUnlock()
	if e.closed.Load() {
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
