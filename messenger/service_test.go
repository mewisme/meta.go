package messenger

import (
	"bytes"
	"context"
	"errors"
	"testing"

	fberrors "go.mewis.me/fbgo/errors"
	"go.mewis.me/fbgo/model"
)

type fakeBackend struct {
	lastSend model.SendRequest
	reaction string
}

func (f *fakeBackend) Send(_ context.Context, req model.SendRequest) (model.SendResult, error) {
	f.lastSend = req
	return model.SendResult{MessageID: "mid.1"}, nil
}
func (f *fakeBackend) Upload(context.Context, model.UploadInput) (model.UploadResult, error) {
	return model.UploadResult{ID: "1"}, nil
}
func (f *fakeBackend) React(_ context.Context, _, _ model.ID, reaction string) error {
	f.reaction = reaction
	return nil
}
func (f *fakeBackend) Edit(context.Context, model.ID, string) error { return nil }
func (f *fakeBackend) Unsend(context.Context, model.ID) error       { return nil }
func (f *fakeBackend) ListMessageRequests(context.Context) ([]model.MessageRequest, error) {
	return []model.MessageRequest{{SenderID: "1"}}, nil
}
func (f *fakeBackend) ListThemes(context.Context) ([]model.Theme, error) {
	return []model.Theme{{ID: "1", Name: "Default"}}, nil
}
func (f *fakeBackend) FindTheme(context.Context, string) (*model.Theme, error) {
	return &model.Theme{ID: "1", Name: "Default"}, nil
}
func (f *fakeBackend) SetTheme(context.Context, model.ID, model.ID) error { return nil }
func (f *fakeBackend) CurrentNote(context.Context) (*model.Note, error)   { return nil, nil }
func (f *fakeBackend) CreateNote(context.Context, string, string) (*model.Note, error) {
	return &model.Note{ID: "1"}, nil
}
func (f *fakeBackend) DeleteNote(context.Context, model.ID) error { return nil }
func (f *fakeBackend) RecreateNote(context.Context, model.ID, string, string) (*model.Note, error) {
	return &model.Note{ID: "2"}, nil
}

func TestServiceSendAndRemoveReaction(t *testing.T) {
	backend := new(fakeBackend)
	service := NewService(backend)
	result, err := service.Send(context.Background(), model.SendRequest{ThreadID: "10", Text: "hello", Attachments: []model.AttachmentInput{{Name: "a.txt", Reader: bytes.NewBufferString("a"), Size: 1}}})
	if err != nil || result.MessageID != "mid.1" {
		t.Fatalf("unexpected send result: %#v %v", result, err)
	}
	if backend.lastSend.ThreadID != "10" {
		t.Fatalf("send request was not forwarded: %#v", backend.lastSend)
	}
	if err := service.RemoveReaction(context.Background(), "10", "mid.1"); err != nil {
		t.Fatal(err)
	}
	if backend.reaction != "" {
		t.Fatalf("remove reaction must use empty reaction, got %q", backend.reaction)
	}
}

func TestServiceInputValidation(t *testing.T) {
	service := NewService(new(fakeBackend))
	_, err := service.Send(context.Background(), model.SendRequest{})
	if !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
	if err := service.Edit(context.Background(), "", "x"); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}
