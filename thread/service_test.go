package thread

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	fberrors "go.mewis.me/meta.go/errors"
	"go.mewis.me/meta.go/model"
)

type fakeBackend struct{}

func (*fakeBackend) ListThreads(context.Context, int) (model.ThreadList, error) {
	return model.ThreadList{Threads: []model.Thread{{ID: "1"}}, SyncSequenceID: 2}, nil
}
func (*fakeBackend) GetThread(context.Context, model.ID) (*model.Thread, error) {
	return &model.Thread{ID: "1"}, nil
}
func (*fakeBackend) CreatePoll(context.Context, model.ID, string, []string) error     { return nil }
func (*fakeBackend) VotePoll(context.Context, model.ID, model.ID, []model.ID) error   { return nil }
func (*fakeBackend) MuteThread(context.Context, model.ID, time.Duration) error        { return nil }
func (*fakeBackend) MuteThreadCalls(context.Context, model.ID, time.Duration) error   { return nil }
func (*fakeBackend) SetThreadApprovalMode(context.Context, model.ID, bool) error      { return nil }
func (*fakeBackend) SetThreadArchived(context.Context, model.ID, bool) error          { return nil }
func (*fakeBackend) SetMessagePinned(context.Context, model.ID, model.ID, bool) error { return nil }
func (*fakeBackend) SetThreadPhoto(context.Context, model.ID, model.AttachmentInput) error {
	return nil
}
func (*fakeBackend) DeleteThread(context.Context, model.ID) error         { return nil }
func (*fakeBackend) CreateDM(context.Context, model.ID) (model.ID, error) { return "9", nil }
func (*fakeBackend) SearchMessengerUsers(context.Context, string) ([]model.User, error) {
	return []model.User{{ID: "1", Name: "User"}}, nil
}
func (*fakeBackend) GetMessengerContact(context.Context, model.ID) (*model.User, error) {
	return &model.User{ID: "1", Name: "User"}, nil
}
func (*fakeBackend) SetThreadAdmin(context.Context, model.ID, model.ID, bool) error      { return nil }
func (*fakeBackend) SetThreadName(context.Context, model.ID, string) error               { return nil }
func (*fakeBackend) SetThreadEmoji(context.Context, model.ID, string) error              { return nil }
func (*fakeBackend) SetThreadNickname(context.Context, model.ID, model.ID, string) error { return nil }

func TestServiceValidationAndDelegation(t *testing.T) {
	service := NewService(new(fakeBackend))
	list, err := service.List(context.Background(), 0)
	if err != nil || len(list.Threads) != 1 {
		t.Fatalf("unexpected list: %#v %v", list, err)
	}
	if _, err := service.List(context.Background(), 101); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid limit, got %v", err)
	}
	if err := service.SetName(context.Background(), "1", " "); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid name, got %v", err)
	}
	if err := service.SetAdmin(context.Background(), "", "2", true); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid IDs, got %v", err)
	}
	if err := service.CreatePoll(context.Background(), "1", "Question", []string{"A", "B"}); err != nil {
		t.Fatal(err)
	}
	if err := service.CreatePoll(context.Background(), "1", "Question", []string{"A"}); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid poll, got %v", err)
	}
	if err := service.VotePoll(context.Background(), "1", "2", []model.ID{"3"}); err != nil {
		t.Fatal(err)
	}
	if err := service.Mute(context.Background(), "1", -time.Second); err != nil {
		t.Fatal(err)
	}
	if err := service.MuteCalls(context.Background(), "1", -time.Second); err != nil {
		t.Fatal(err)
	}
	if err := service.SetApprovalMode(context.Background(), "1", true); err != nil {
		t.Fatal(err)
	}
	if err := service.SetArchived(context.Background(), "1", true); err != nil {
		t.Fatal(err)
	}
	if err := service.PinMessage(context.Background(), "1", "mid.1"); err != nil {
		t.Fatal(err)
	}
	if err := service.UnpinMessage(context.Background(), "1", "mid.1"); err != nil {
		t.Fatal(err)
	}
	if err := service.PinMessage(context.Background(), "", "mid.1"); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid pin target, got %v", err)
	}
	if err := service.SetPhoto(context.Background(), "1", model.AttachmentInput{Reader: bytes.NewBufferString("x"), Size: 1}); err != nil {
		t.Fatal(err)
	}
	if id, err := service.CreateDM(context.Background(), "2"); err != nil || id != "9" {
		t.Fatalf("unexpected create dm: %q %v", id, err)
	}
	if users, err := service.SearchUsers(context.Background(), "user"); err != nil || len(users) != 1 {
		t.Fatalf("unexpected users: %#v %v", users, err)
	}
	if user, err := service.GetContact(context.Background(), "1"); err != nil || user.ID != "1" {
		t.Fatalf("unexpected contact: %#v %v", user, err)
	}
}
