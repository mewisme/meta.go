// Package meta contains the in-process Messenger and E2EE transport engine.
package meta

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"go.mewis.me/fbgo/model"
)

var (
	ErrClosed       = errors.New("meta engine closed")
	ErrNotConnected = errors.New("meta engine not connected")
	ErrE2EENotReady = errors.New("meta engine E2EE not ready")
)

type Account struct {
	ID       model.ID
	Name     string
	Username string
}

type EventKind string

const (
	EventReady           EventKind = "ready"
	EventReconnected     EventKind = "reconnected"
	EventDisconnected    EventKind = "disconnected"
	EventError           EventKind = "error"
	EventMessage         EventKind = "message"
	EventReaction        EventKind = "reaction"
	EventTyping          EventKind = "typing"
	EventReadReceipt     EventKind = "readReceipt"
	EventDeliveryReceipt EventKind = "deliveryReceipt"
	EventMessageEdit     EventKind = "messageEdit"
	EventMessageUnsend   EventKind = "messageUnsend"
	EventThreadUpdate    EventKind = "threadUpdate"
	EventE2EEReady       EventKind = "e2eeReady"
)

type Event struct {
	Kind EventKind
	Data any
}

type ReactionEvent struct {
	MessageID model.ID
	ThreadID  model.ID
	ActorID   model.ID
	Reaction  string
}

type TypingEvent struct {
	ThreadID model.ID
	SenderID model.ID
	Typing   bool
}

type ReadReceiptEvent struct {
	ThreadID  model.ID
	ReaderID  model.ID
	Watermark time.Time
}

type DeliveryReceiptEvent struct {
	ThreadID    model.ID
	RecipientID model.ID
	Watermark   time.Time
}

type MessageEditEvent struct {
	MessageID model.ID
	Text      string
	EditCount int64
}

type MessageUnsendEvent struct {
	MessageID model.ID
	ThreadID  model.ID
}

type ThreadUpdateEvent struct {
	ThreadID model.ID
	Field    string
	Value    string
}

type SendTextRequest struct {
	ThreadID model.ID
	Text     string
	ReplyTo  model.ID
	Mentions []model.Mention
}

type UploadRequest struct {
	ThreadID    model.ID
	Name        string
	ContentType string
	Reader      io.Reader
	Size        int64
	Voice       bool
}

type UploadResult struct {
	ID   model.ID
	Name string
}

type backend interface {
	SetEventHandler(func(context.Context, any))
	Bootstrap(context.Context) (Account, error)
	Connect(context.Context, context.Context) error
	Disconnect()
	SendText(context.Context, SendTextRequest) (model.SendResult, error)
	Upload(context.Context, UploadRequest) (UploadResult, error)
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

func (e *Engine) Upload(ctx context.Context, req UploadRequest) (UploadResult, error) {
	if !e.connected.Load() {
		return UploadResult{}, ErrNotConnected
	}
	ctx, cancel := mergeContext(e.ctx, ctx)
	defer cancel()
	return e.backend.Upload(ctx, req)
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
