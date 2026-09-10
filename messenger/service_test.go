package messenger

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	fberrors "go.mewis.me/meta.go/errors"
	"go.mewis.me/meta.go/model"
)

type fakeBackend struct {
	lastSend       model.SendRequest
	reaction       string
	restricted     bool
	messageBlocked bool
}

func (f *fakeBackend) Send(_ context.Context, req model.SendRequest) (model.SendResult, error) {
	f.lastSend = req
	return model.SendResult{MessageID: "mid.1"}, nil
}
func (f *fakeBackend) ShareContact(context.Context, model.ID, model.ID, string) error { return nil }
func (f *fakeBackend) Forward(context.Context, model.ID, model.ID) (model.SendResult, error) {
	return model.SendResult{MessageID: "mid.forward"}, nil
}
func (f *fakeBackend) Upload(context.Context, model.UploadInput) (model.UploadResult, error) {
	return model.UploadResult{ID: "1"}, nil
}
func (f *fakeBackend) React(_ context.Context, _, _ model.ID, reaction string) error {
	f.reaction = reaction
	return nil
}
func (f *fakeBackend) Edit(context.Context, model.ID, string) error              { return nil }
func (f *fakeBackend) Unsend(context.Context, model.ID) error                    { return nil }
func (f *fakeBackend) Typing(context.Context, model.ID, bool, bool, int64) error { return nil }
func (f *fakeBackend) Read(context.Context, model.ID, time.Time) error           { return nil }
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
func (f *fakeBackend) SetRestricted(_ context.Context, _ model.ID, restricted bool) error {
	f.restricted = restricted
	return nil
}
func (f *fakeBackend) SetMessageBlocked(_ context.Context, _ model.ID, blocked bool) error {
	f.messageBlocked = blocked
	return nil
}
func (f *fakeBackend) SendE2EE(context.Context, model.E2EESendRequest) (model.SendResult, error) {
	return model.SendResult{MessageID: "e2ee.1"}, nil
}
func (f *fakeBackend) SendE2EEMedia(context.Context, model.E2EEMediaInput) (model.SendResult, error) {
	return model.SendResult{MessageID: "e2ee.media.1"}, nil
}
func (f *fakeBackend) DownloadE2EEMedia(context.Context, model.E2EEMediaDownload) ([]byte, error) {
	return []byte("media"), nil
}
func (f *fakeBackend) ReactE2EE(context.Context, model.E2EEReactionRequest) error { return nil }
func (f *fakeBackend) EditE2EE(context.Context, string, model.ID, string) error   { return nil }
func (f *fakeBackend) UnsendE2EE(context.Context, string, model.ID) error         { return nil }
func (f *fakeBackend) TypingE2EE(context.Context, string, bool) error             { return nil }
func (f *fakeBackend) ReadE2EE(context.Context, model.E2EEReadRequest) error      { return nil }

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

func TestServiceForward(t *testing.T) {
	service := NewService(new(fakeBackend))
	result, err := service.Forward(context.Background(), "10", "mid.1")
	if err != nil || result.MessageID != "mid.forward" {
		t.Fatalf("unexpected forward result: %#v %v", result, err)
	}
	if _, err := service.Forward(context.Background(), "", "mid.1"); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid forward target, got %v", err)
	}
}

func TestServiceContactShareAndRestriction(t *testing.T) {
	backend := new(fakeBackend)
	service := NewService(backend)
	if err := service.ShareContact(context.Background(), "10", "20", "hello"); err != nil {
		t.Fatalf("unexpected contact share error: %v", err)
	}
	if err := service.SetRestricted(context.Background(), "20", true); err != nil || !backend.restricted {
		t.Fatalf("restrict failed: restricted=%t err=%v", backend.restricted, err)
	}
	if err := service.SetMessageBlocked(context.Background(), "20", true); err != nil || !backend.messageBlocked {
		t.Fatalf("message block failed: blocked=%t err=%v", backend.messageBlocked, err)
	}
	if err := service.ShareContact(context.Background(), "", "20", ""); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid contact share target, got %v", err)
	}
	if err := service.SetRestricted(context.Background(), "", true); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid restrict target, got %v", err)
	}
	if err := service.SetMessageBlocked(context.Background(), "", true); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid block target, got %v", err)
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
	if err := service.Typing(context.Background(), "", true, false, 1); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid typing target, got %v", err)
	}
	if err := service.Typing(context.Background(), "1", true, false, -1); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid thread type, got %v", err)
	}
	if err := service.Read(context.Background(), "", time.Time{}); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid read target, got %v", err)
	}
	if _, err := service.Send(context.Background(), model.SendRequest{ThreadID: "1", Attachments: []model.AttachmentInput{{Reader: bytes.NewBufferString("x")}}, StickerID: "2"}); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected mutually exclusive send content, got %v", err)
	}
	if _, err := service.Send(context.Background(), model.SendRequest{ThreadID: "1", StickerID: "2"}); err != nil {
		t.Fatalf("sticker send rejected: %v", err)
	}
	if _, err := service.Send(context.Background(), model.SendRequest{ThreadID: "1", URL: "https://example.com/media"}); err != nil {
		t.Fatalf("external media send rejected: %v", err)
	}
}

func TestServiceE2EEMediaValidation(t *testing.T) {
	service := NewService(new(fakeBackend))
	_, err := service.SendE2EEMedia(context.Background(), model.E2EEMediaInput{})
	if !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid media input, got %v", err)
	}
	result, err := service.SendE2EEMedia(context.Background(), model.E2EEMediaInput{ChatJID: "1@msgr", Kind: model.E2EEMediaDocument, Reader: bytes.NewBufferString("x"), Size: 1})
	if err != nil || result.MessageID != "e2ee.media.1" {
		t.Fatalf("unexpected media send result: %#v %v", result, err)
	}
	data, err := service.DownloadE2EE(context.Background(), model.E2EEMediaDownload{Reference: model.E2EEMediaReference{Kind: model.E2EEMediaDocument, DirectPath: "/x", MediaKey: []byte{1}, FileSHA256: []byte{2}}})
	if err != nil || string(data) != "media" {
		t.Fatalf("unexpected media download result: %q %v", data, err)
	}
}
