package meta

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"go.mau.fi/mautrix-meta/pkg/messagix"
	"go.mewis.me/fbgo/model"
)

type fakeBackend struct {
	mu           sync.Mutex
	handler      func(context.Context, any)
	connected    bool
	disconnected bool
}

func (f *fakeBackend) SetEventHandler(handler func(context.Context, any)) { f.handler = handler }
func (f *fakeBackend) Bootstrap(context.Context) (Account, error) {
	return Account{ID: "1", Name: "Test"}, nil
}
func (f *fakeBackend) Connect(context.Context, context.Context) error {
	f.mu.Lock()
	f.connected = true
	f.mu.Unlock()
	return nil
}
func (f *fakeBackend) Disconnect() {
	f.mu.Lock()
	f.disconnected = true
	f.mu.Unlock()
}
func (f *fakeBackend) SendText(_ context.Context, req SendTextRequest) (model.SendResult, error) {
	if req.Text == "" {
		return model.SendResult{}, errors.New("empty")
	}
	return model.SendResult{MessageID: "mid.1", Timestamp: time.Unix(1, 0)}, nil
}
func (f *fakeBackend) Upload(_ context.Context, req UploadRequest) (UploadResult, error) {
	if req.Reader == nil {
		return UploadResult{}, errors.New("nil reader")
	}
	_, _ = io.ReadAll(req.Reader)
	return UploadResult{ID: "2", Name: req.Name}, nil
}
func (f *fakeBackend) React(context.Context, model.ID, model.ID, string) error       { return nil }
func (f *fakeBackend) Edit(context.Context, model.ID, string) error                  { return nil }
func (f *fakeBackend) Unsend(context.Context, model.ID) error                        { return nil }
func (f *fakeBackend) ListMessageRequests(context.Context) ([]MessageRequest, error) { return nil, nil }
func (f *fakeBackend) ListThemes(context.Context) ([]Theme, error)                   { return nil, nil }
func (f *fakeBackend) SetTheme(context.Context, model.ID, model.ID) error            { return nil }
func (f *fakeBackend) CurrentNote(context.Context) (*Note, error)                    { return nil, nil }
func (f *fakeBackend) CreateNote(context.Context, string, string) (*Note, error)     { return nil, nil }
func (f *fakeBackend) DeleteNote(context.Context, model.ID) error                    { return nil }
func (f *fakeBackend) SendE2EE(context.Context, model.E2EESendRequest) (model.SendResult, error) {
	return model.SendResult{MessageID: "e2ee.1", Timestamp: time.Unix(2, 0)}, nil
}
func (f *fakeBackend) SendE2EEMedia(context.Context, model.E2EEMediaInput) (model.SendResult, error) {
	return model.SendResult{MessageID: "e2ee.media.1", Timestamp: time.Unix(3, 0)}, nil
}
func (f *fakeBackend) DownloadE2EEMedia(context.Context, model.E2EEMediaDownload) ([]byte, error) {
	return []byte("media"), nil
}
func (f *fakeBackend) ReactE2EE(context.Context, model.E2EEReactionRequest) error { return nil }
func (f *fakeBackend) EditE2EE(context.Context, string, model.ID, string) error   { return nil }
func (f *fakeBackend) UnsendE2EE(context.Context, string, model.ID) error         { return nil }
func (f *fakeBackend) TypingE2EE(context.Context, string, bool) error             { return nil }
func (f *fakeBackend) ReadE2EE(context.Context, model.E2EEReadRequest) error      { return nil }
func (f *fakeBackend) ConnectE2EE(context.Context, model.ID) error                { return nil }
func (f *fakeBackend) E2EEConnected() bool                                        { return true }

func TestEngineLifecycle(t *testing.T) {
	backend := new(fakeBackend)
	engine := newEngine(context.Background(), backend, 2)
	account, err := engine.Connect(context.Background())
	if err != nil || account.ID != "1" || !engine.Connected() {
		t.Fatalf("unexpected connect result: %#v %v", account, err)
	}
	backend.handler(context.Background(), &messagix.Event_Ready{IsNewSession: true})
	select {
	case event := <-engine.Events():
		if event.Kind != EventReady {
			t.Fatalf("unexpected event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("event not delivered")
	}
	result, err := engine.SendText(context.Background(), SendTextRequest{ThreadID: "10", Text: "hello"})
	if err != nil || result.MessageID != "mid.1" {
		t.Fatalf("unexpected send result: %#v %v", result, err)
	}
	engine.Close()
	engine.Close()
	if engine.Connected() {
		t.Fatal("engine remained connected after close")
	}
	backend.mu.Lock()
	disconnected := backend.disconnected
	backend.mu.Unlock()
	if !disconnected {
		t.Fatal("backend was not disconnected")
	}
}

func TestEngineE2EELifecycleUsesSameEngine(t *testing.T) {
	backend := new(fakeBackend)
	engine := newEngine(context.Background(), backend, 2)
	if _, err := engine.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := engine.ConnectE2EE(context.Background(), "1"); err != nil {
		t.Fatal(err)
	}
	if !engine.E2EEConnected() {
		t.Fatal("E2EE readiness was not reflected by the engine")
	}
	engine.Close()
	if engine.E2EEConnected() {
		t.Fatal("E2EE remained ready after close")
	}
}

func TestEngineConcurrentEmitAndClose(t *testing.T) {
	backend := new(fakeBackend)
	engine := newEngine(context.Background(), backend, 1)
	if _, err := engine.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 1000; i++ {
			backend.handler(context.Background(), &messagix.Event_Reconnected{})
		}
	}()
	engine.Close()
	<-done
}

func TestEngineCloseCancelsInFlightConnect(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	backend := &blockingBackend{started: started, release: release}
	engine := newEngine(context.Background(), backend, 1)
	done := make(chan error, 1)
	go func() {
		_, err := engine.Connect(context.Background())
		done <- err
	}()
	<-started
	closeDone := make(chan struct{})
	go func() { engine.Close(); close(closeDone) }()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected canceled connect, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("connect did not observe cancellation")
	}
	close(release)
	<-closeDone
}

type blockingBackend struct {
	started chan struct{}
	release chan struct{}
}

func (b *blockingBackend) SetEventHandler(func(context.Context, any)) {}
func (b *blockingBackend) Bootstrap(ctx context.Context) (Account, error) {
	close(b.started)
	select {
	case <-ctx.Done():
		return Account{}, ctx.Err()
	case <-b.release:
		return Account{}, nil
	}
}
func (b *blockingBackend) Connect(context.Context, context.Context) error { return nil }
func (b *blockingBackend) Disconnect()                                    {}
func (b *blockingBackend) SendText(context.Context, SendTextRequest) (model.SendResult, error) {
	return model.SendResult{}, nil
}
func (b *blockingBackend) Upload(context.Context, UploadRequest) (UploadResult, error) {
	return UploadResult{}, nil
}
func (b *blockingBackend) React(context.Context, model.ID, model.ID, string) error { return nil }
func (b *blockingBackend) Edit(context.Context, model.ID, string) error            { return nil }
func (b *blockingBackend) Unsend(context.Context, model.ID) error                  { return nil }
func (b *blockingBackend) ListMessageRequests(context.Context) ([]MessageRequest, error) {
	return nil, nil
}
func (b *blockingBackend) ListThemes(context.Context) ([]Theme, error)               { return nil, nil }
func (b *blockingBackend) SetTheme(context.Context, model.ID, model.ID) error        { return nil }
func (b *blockingBackend) CurrentNote(context.Context) (*Note, error)                { return nil, nil }
func (b *blockingBackend) CreateNote(context.Context, string, string) (*Note, error) { return nil, nil }
func (b *blockingBackend) DeleteNote(context.Context, model.ID) error                { return nil }
func (b *blockingBackend) SendE2EE(context.Context, model.E2EESendRequest) (model.SendResult, error) {
	return model.SendResult{}, nil
}
func (b *blockingBackend) SendE2EEMedia(context.Context, model.E2EEMediaInput) (model.SendResult, error) {
	return model.SendResult{}, nil
}
func (b *blockingBackend) DownloadE2EEMedia(context.Context, model.E2EEMediaDownload) ([]byte, error) {
	return nil, nil
}
func (b *blockingBackend) ReactE2EE(context.Context, model.E2EEReactionRequest) error { return nil }
func (b *blockingBackend) EditE2EE(context.Context, string, model.ID, string) error   { return nil }
func (b *blockingBackend) UnsendE2EE(context.Context, string, model.ID) error         { return nil }
func (b *blockingBackend) TypingE2EE(context.Context, string, bool) error             { return nil }
func (b *blockingBackend) ReadE2EE(context.Context, model.E2EEReadRequest) error      { return nil }
func (b *blockingBackend) ConnectE2EE(context.Context, model.ID) error                { return nil }
func (b *blockingBackend) E2EEConnected() bool                                        { return false }
