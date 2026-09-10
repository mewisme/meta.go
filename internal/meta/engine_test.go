package meta

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"go.mewis.me/meta-extra/pkg/messagix"

	"go.mewis.me/meta.go/model"
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
func (f *fakeBackend) Forward(context.Context, model.ID, model.ID) (model.SendResult, error) {
	return model.SendResult{MessageID: "mid.forward"}, nil
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
func (f *fakeBackend) Typing(context.Context, model.ID, bool, bool, int64) error     { return nil }
func (f *fakeBackend) Read(context.Context, model.ID, time.Time) error               { return nil }
func (f *fakeBackend) ListMessageRequests(context.Context) ([]MessageRequest, error) { return nil, nil }
func (f *fakeBackend) ListThemes(context.Context) ([]Theme, error)                   { return nil, nil }
func (f *fakeBackend) SetTheme(context.Context, model.ID, model.ID) error            { return nil }
func (f *fakeBackend) CurrentNote(context.Context) (*Note, error)                    { return nil, nil }
func (f *fakeBackend) CreateNote(context.Context, string, string) (*Note, error)     { return nil, nil }
func (f *fakeBackend) DeleteNote(context.Context, model.ID) error                    { return nil }
func (f *fakeBackend) ListThreads(context.Context, int) (model.ThreadList, error) {
	return model.ThreadList{Threads: []model.Thread{{ID: "10"}}, SyncSequenceID: 1}, nil
}
func (f *fakeBackend) GetThread(context.Context, model.ID) (*model.Thread, error) {
	return &model.Thread{ID: "10"}, nil
}
func (f *fakeBackend) CreatePoll(context.Context, model.ID, string, []string) error   { return nil }
func (f *fakeBackend) VotePoll(context.Context, model.ID, model.ID, []model.ID) error { return nil }
func (f *fakeBackend) ListPinnedMessages(context.Context, model.ID) ([]model.PinnedMessage, error) {
	return nil, nil
}
func (f *fakeBackend) FetchPollDetails(context.Context, model.ID) (*model.PollDetails, error) {
	return nil, nil
}
func (f *fakeBackend) SearchThreadMessages(context.Context, model.MessageSearchRequest) (*model.MessageSearchPage, error) {
	return nil, nil
}
func (f *fakeBackend) MuteThread(context.Context, model.ID, time.Duration) error        { return nil }
func (f *fakeBackend) MuteThreadCalls(context.Context, model.ID, time.Duration) error   { return nil }
func (f *fakeBackend) SetThreadApprovalMode(context.Context, model.ID, bool) error      { return nil }
func (f *fakeBackend) SetThreadArchived(context.Context, model.ID, bool) error          { return nil }
func (f *fakeBackend) SetMessagePinned(context.Context, model.ID, model.ID, bool) error { return nil }
func (f *fakeBackend) SetThreadPhoto(context.Context, model.ID, model.AttachmentInput) error {
	return nil
}
func (f *fakeBackend) DeleteThread(context.Context, model.ID) error         { return nil }
func (f *fakeBackend) CreateDM(context.Context, model.ID) (model.ID, error) { return "10", nil }
func (f *fakeBackend) SearchMessengerUsers(context.Context, string) ([]model.User, error) {
	return nil, nil
}
func (f *fakeBackend) GetMessengerContact(context.Context, model.ID) (*model.User, error) {
	return &model.User{ID: "1"}, nil
}
func (f *fakeBackend) SetThreadAdmin(context.Context, model.ID, model.ID, bool) error { return nil }
func (f *fakeBackend) SetThreadName(context.Context, model.ID, string) error          { return nil }
func (f *fakeBackend) SetThreadEmoji(context.Context, model.ID, string) error         { return nil }
func (f *fakeBackend) SetThreadNickname(context.Context, model.ID, model.ID, string) error {
	return nil
}
func (f *fakeBackend) GetFacebookUser(context.Context, model.ID) (*model.FacebookUser, error) {
	return &model.FacebookUser{ID: "1"}, nil
}
func (f *fakeBackend) SearchFacebook(context.Context, string, int) ([]model.SearchResult, error) {
	return []model.SearchResult{{ID: "1"}}, nil
}
func (f *fakeBackend) ListNotifications(context.Context, int) ([]model.Notification, error) {
	return []model.Notification{{Text: "x"}}, nil
}
func (f *fakeBackend) SetFacebookBio(context.Context, string, bool) error            { return nil }
func (f *fakeBackend) CreateAdditionalProfile(context.Context, string, string) error { return nil }
func (f *fakeBackend) UnfriendFacebookUser(context.Context, model.ID) error          { return nil }
func (f *fakeBackend) SetFacebookBlocked(context.Context, model.ID, bool) error      { return nil }
func (f *fakeBackend) CreateFacebookPost(context.Context, string) (*model.Post, error) {
	return &model.Post{URL: "x"}, nil
}
func (f *fakeBackend) ArchiveFacebookPost(context.Context, model.ID, model.PostOwnership) error {
	return nil
}
func (f *fakeBackend) DeleteFacebookPost(context.Context, model.ID, model.PostOwnership) error {
	return nil
}
func (f *fakeBackend) CreateMarketplaceListing(context.Context, model.MarketplaceListingInput) (*model.MarketplaceListing, error) {
	return &model.MarketplaceListing{ID: "m1"}, nil
}
func (f *fakeBackend) GetMarketplaceListing(context.Context, model.ID) (*model.MarketplaceListing, error) {
	return &model.MarketplaceListing{ID: "m1"}, nil
}
func (f *fakeBackend) SetProfessionalMode(context.Context, bool) error { return nil }
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
	backend.handler(context.Background(), &messagix.ConnectedEvent{})
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
			backend.handler(context.Background(), &messagix.ReconnectedEvent{})
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
func (b *blockingBackend) Forward(context.Context, model.ID, model.ID) (model.SendResult, error) {
	return model.SendResult{}, nil
}
func (b *blockingBackend) Upload(context.Context, UploadRequest) (UploadResult, error) {
	return UploadResult{}, nil
}
func (b *blockingBackend) React(context.Context, model.ID, model.ID, string) error   { return nil }
func (b *blockingBackend) Edit(context.Context, model.ID, string) error              { return nil }
func (b *blockingBackend) Unsend(context.Context, model.ID) error                    { return nil }
func (b *blockingBackend) Typing(context.Context, model.ID, bool, bool, int64) error { return nil }
func (b *blockingBackend) Read(context.Context, model.ID, time.Time) error           { return nil }
func (b *blockingBackend) ListMessageRequests(context.Context) ([]MessageRequest, error) {
	return nil, nil
}
func (b *blockingBackend) ListThemes(context.Context) ([]Theme, error)               { return nil, nil }
func (b *blockingBackend) SetTheme(context.Context, model.ID, model.ID) error        { return nil }
func (b *blockingBackend) CurrentNote(context.Context) (*Note, error)                { return nil, nil }
func (b *blockingBackend) CreateNote(context.Context, string, string) (*Note, error) { return nil, nil }
func (b *blockingBackend) DeleteNote(context.Context, model.ID) error                { return nil }
func (b *blockingBackend) ListThreads(context.Context, int) (model.ThreadList, error) {
	return model.ThreadList{}, nil
}
func (b *blockingBackend) GetThread(context.Context, model.ID) (*model.Thread, error) {
	return nil, nil
}
func (b *blockingBackend) CreatePoll(context.Context, model.ID, string, []string) error   { return nil }
func (b *blockingBackend) VotePoll(context.Context, model.ID, model.ID, []model.ID) error { return nil }
func (b *blockingBackend) ListPinnedMessages(context.Context, model.ID) ([]model.PinnedMessage, error) {
	return nil, nil
}
func (b *blockingBackend) FetchPollDetails(context.Context, model.ID) (*model.PollDetails, error) {
	return nil, nil
}
func (b *blockingBackend) SearchThreadMessages(context.Context, model.MessageSearchRequest) (*model.MessageSearchPage, error) {
	return nil, nil
}
func (b *blockingBackend) MuteThread(context.Context, model.ID, time.Duration) error      { return nil }
func (b *blockingBackend) MuteThreadCalls(context.Context, model.ID, time.Duration) error { return nil }
func (b *blockingBackend) SetThreadApprovalMode(context.Context, model.ID, bool) error    { return nil }
func (b *blockingBackend) SetThreadArchived(context.Context, model.ID, bool) error        { return nil }
func (b *blockingBackend) SetMessagePinned(context.Context, model.ID, model.ID, bool) error {
	return nil
}
func (b *blockingBackend) SetThreadPhoto(context.Context, model.ID, model.AttachmentInput) error {
	return nil
}
func (b *blockingBackend) DeleteThread(context.Context, model.ID) error         { return nil }
func (b *blockingBackend) CreateDM(context.Context, model.ID) (model.ID, error) { return "", nil }
func (b *blockingBackend) SearchMessengerUsers(context.Context, string) ([]model.User, error) {
	return nil, nil
}
func (b *blockingBackend) GetMessengerContact(context.Context, model.ID) (*model.User, error) {
	return nil, nil
}
func (b *blockingBackend) SetThreadAdmin(context.Context, model.ID, model.ID, bool) error { return nil }
func (b *blockingBackend) SetThreadName(context.Context, model.ID, string) error          { return nil }
func (b *blockingBackend) SetThreadEmoji(context.Context, model.ID, string) error         { return nil }
func (b *blockingBackend) SetThreadNickname(context.Context, model.ID, model.ID, string) error {
	return nil
}
func (b *blockingBackend) GetFacebookUser(context.Context, model.ID) (*model.FacebookUser, error) {
	return nil, nil
}
func (b *blockingBackend) SearchFacebook(context.Context, string, int) ([]model.SearchResult, error) {
	return nil, nil
}
func (b *blockingBackend) ListNotifications(context.Context, int) ([]model.Notification, error) {
	return nil, nil
}
func (b *blockingBackend) SetFacebookBio(context.Context, string, bool) error            { return nil }
func (b *blockingBackend) CreateAdditionalProfile(context.Context, string, string) error { return nil }
func (b *blockingBackend) UnfriendFacebookUser(context.Context, model.ID) error          { return nil }
func (b *blockingBackend) SetFacebookBlocked(context.Context, model.ID, bool) error      { return nil }
func (b *blockingBackend) CreateFacebookPost(context.Context, string) (*model.Post, error) {
	return nil, nil
}
func (b *blockingBackend) ArchiveFacebookPost(context.Context, model.ID, model.PostOwnership) error {
	return nil
}
func (b *blockingBackend) DeleteFacebookPost(context.Context, model.ID, model.PostOwnership) error {
	return nil
}
func (b *blockingBackend) CreateMarketplaceListing(context.Context, model.MarketplaceListingInput) (*model.MarketplaceListing, error) {
	return nil, nil
}
func (b *blockingBackend) GetMarketplaceListing(context.Context, model.ID) (*model.MarketplaceListing, error) {
	return nil, nil
}
func (b *blockingBackend) SetProfessionalMode(context.Context, bool) error { return nil }
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
