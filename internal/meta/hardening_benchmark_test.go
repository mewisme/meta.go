package meta

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"go.mewis.me/meta-extra/pkg/messagix/table"
	fberrors "go.mewis.me/fbgo/errors"
	"go.mewis.me/fbgo/model"
)

func BenchmarkEventConversion(b *testing.B) {
	wrapped := &table.WrappedMessage{LSInsertMessage: &table.LSInsertMessage{MessageId: "m1"}, Attachments: []*table.LSInsertAttachment{{AttachmentFbid: "99", AttachmentMimeType: "image/jpeg", ImageUrl: "https://example.com/a.jpg", Filename: "a.jpg", Filesize: 4096, PreviewWidth: 640, PreviewHeight: 480}}}
	b.ReportAllocs()
	for b.Loop() {
		if len(wrappedAttachments(wrapped)) != 1 {
			b.Fatal("attachment conversion failed")
		}
	}
}

func BenchmarkEventQueue(b *testing.B) {
	engine := newEngine(nil, new(fakeBackend), 64)
	defer engine.Close()
	event := Event{Kind: EventTyping, Typing: &model.TypingEvent{ThreadID: "1", SenderID: "2", Typing: true}}
	b.ReportAllocs()
	for b.Loop() {
		engine.emit(event)
	}
}

func BenchmarkEngineStartupConnect(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		engine := newEngine(context.Background(), new(fakeBackend), 64)
		if _, err := engine.Connect(context.Background()); err != nil {
			b.Fatal(err)
		}
		engine.Close()
	}
}

func TestEngineLoadRemainsBounded(t *testing.T) {
	runtime.GC()
	beforeGoroutines := runtime.NumGoroutine()
	engine := newEngine(nil, new(fakeBackend), 32)
	defer engine.Close()
	for i := 0; i < 10000; i++ {
		unsubscribe := engine.On(func(Event) {})
		engine.emit(Event{Kind: EventTyping})
		unsubscribe()
	}
	if len(engine.events) > cap(engine.events) || cap(engine.events) != 32 {
		t.Fatalf("event queue grew beyond bound: len=%d cap=%d", len(engine.events), cap(engine.events))
	}
	engine.handlerMu.RLock()
	handlers := len(engine.handlers)
	engine.handlerMu.RUnlock()
	if handlers != 0 {
		t.Fatalf("handler registrations leaked: %d", handlers)
	}
	if engine.dropped.Load() == 0 {
		t.Fatal("expected bounded queue to drop old events under sustained load")
	}
	runtime.GC()
	if after := runtime.NumGoroutine(); after > beforeGoroutines+8 {
		t.Fatalf("load test leaked goroutines: before=%d after=%d", beforeGoroutines, after)
	}
}

type failingBootstrapBackend struct {
	fakeBackend
	attempts atomic.Int64
}

type blockingSendBackend struct{ fakeBackend }

func (b *blockingSendBackend) SendText(ctx context.Context, _ SendTextRequest) (model.SendResult, error) {
	<-ctx.Done()
	return model.SendResult{}, ctx.Err()
}

func TestEngineDefaultRequestTimeout(t *testing.T) {
	engine := newEngine(nil, new(blockingSendBackend), 1)
	engine.requestTimeout = 20 * time.Millisecond
	defer engine.Close()
	if _, err := engine.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	_, err := engine.SendText(context.Background(), SendTextRequest{ThreadID: "1", Text: "x"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected default request timeout, got %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("default timeout took too long: %s", elapsed)
	}
}

func TestEngineCallerDeadlineOverridesDefaultTimeout(t *testing.T) {
	engine := newEngine(nil, new(blockingSendBackend), 1)
	engine.requestTimeout = time.Second
	defer engine.Close()
	if _, err := engine.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := engine.SendText(ctx, SendTextRequest{ThreadID: "1", Text: "x"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected caller deadline, got %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("caller deadline took too long: %s", elapsed)
	}
}

func (b *failingBootstrapBackend) Bootstrap(context.Context) (Account, error) {
	b.attempts.Add(1)
	return Account{}, context.DeadlineExceeded
}

func TestEngineConnectFailureIsObservableAndNotRetriedInternally(t *testing.T) {
	backend := new(failingBootstrapBackend)
	engine := newEngine(nil, backend, 1)
	defer engine.Close()
	if _, err := engine.Connect(context.Background()); err == nil {
		t.Fatal("expected connect failure")
	}
	health := engine.Health()
	if health.Regular != model.ConnectionFailed || health.LastErrorCategory != string(fberrors.ErrorCategoryNetwork) {
		t.Fatalf("unexpected failure health: %#v", health)
	}
	if backend.attempts.Load() != 1 {
		t.Fatalf("connect failure was retried internally: %d attempts", backend.attempts.Load())
	}
}
