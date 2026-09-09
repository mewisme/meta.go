package thread

import (
	"context"
	"errors"
	"testing"

	fberrors "go.mewis.me/fbgo/errors"
	"go.mewis.me/fbgo/model"
)

type fakeBackend struct{}

func (*fakeBackend) ListThreads(context.Context, int) (model.ThreadList, error) {
	return model.ThreadList{Threads: []model.Thread{{ID: "1"}}, SyncSequenceID: 2}, nil
}
func (*fakeBackend) GetThread(context.Context, model.ID) (*model.Thread, error) {
	return &model.Thread{ID: "1"}, nil
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
}
