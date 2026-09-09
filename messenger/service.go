// Package messenger exposes regular and end-to-end encrypted Messenger operations.
package messenger

import (
	"context"
	"errors"
	"fmt"
	"strings"

	fberrors "go.mewis.me/fbgo/errors"
	"go.mewis.me/fbgo/model"
)

var ErrUnavailable = errors.New("messenger service unavailable")

type Backend interface {
	Send(context.Context, model.SendRequest) (model.SendResult, error)
	Upload(context.Context, model.UploadInput) (model.UploadResult, error)
	React(context.Context, model.ID, model.ID, string) error
	Edit(context.Context, model.ID, string) error
	Unsend(context.Context, model.ID) error
	ListMessageRequests(context.Context) ([]model.MessageRequest, error)
	ListThemes(context.Context) ([]model.Theme, error)
	FindTheme(context.Context, string) (*model.Theme, error)
	SetTheme(context.Context, model.ID, model.ID) error
	CurrentNote(context.Context) (*model.Note, error)
	CreateNote(context.Context, string, string) (*model.Note, error)
	DeleteNote(context.Context, model.ID) error
	RecreateNote(context.Context, model.ID, string, string) (*model.Note, error)
}

type Service struct{ backend Backend }

func NewService(backend Backend) *Service { return &Service{backend: backend} }

func (s *Service) Send(ctx context.Context, req model.SendRequest) (model.SendResult, error) {
	if err := s.ready(); err != nil {
		return model.SendResult{}, err
	}
	if req.ThreadID.Empty() {
		return model.SendResult{}, invalid("thread ID is required")
	}
	if strings.TrimSpace(req.Text) == "" && len(req.Attachments) == 0 {
		return model.SendResult{}, invalid("message text or attachment is required")
	}
	if req.ReplyTo != nil && req.ReplyTo.MessageID.Empty() {
		return model.SendResult{}, invalid("reply message ID is required")
	}
	for _, mention := range req.Mentions {
		if mention.UserID.Empty() || mention.Offset < 0 || mention.Length <= 0 {
			return model.SendResult{}, invalid("mention contains invalid user ID, offset or length")
		}
	}
	for _, attachment := range req.Attachments {
		if attachment.Reader == nil {
			return model.SendResult{}, invalid("attachment reader is required")
		}
		if attachment.Size < 0 {
			return model.SendResult{}, invalid("attachment size cannot be negative")
		}
	}
	return s.backend.Send(ctx, req)
}

func (s *Service) Upload(ctx context.Context, input model.UploadInput) (model.UploadResult, error) {
	if err := s.ready(); err != nil {
		return model.UploadResult{}, err
	}
	if input.ThreadID.Empty() || input.Reader == nil {
		return model.UploadResult{}, invalid("thread ID and media reader are required")
	}
	if input.Size < 0 {
		return model.UploadResult{}, invalid("media size cannot be negative")
	}
	return s.backend.Upload(ctx, input)
}

func (s *Service) React(ctx context.Context, threadID, messageID model.ID, reaction string) error {
	if err := s.ready(); err != nil {
		return err
	}
	if threadID.Empty() || messageID.Empty() {
		return invalid("thread ID and message ID are required")
	}
	return s.backend.React(ctx, threadID, messageID, reaction)
}

func (s *Service) RemoveReaction(ctx context.Context, threadID, messageID model.ID) error {
	return s.React(ctx, threadID, messageID, "")
}

func (s *Service) Edit(ctx context.Context, messageID model.ID, text string) error {
	if err := s.ready(); err != nil {
		return err
	}
	if messageID.Empty() || text == "" {
		return invalid("message ID and replacement text are required")
	}
	return s.backend.Edit(ctx, messageID, text)
}

func (s *Service) Unsend(ctx context.Context, messageID model.ID) error {
	if err := s.ready(); err != nil {
		return err
	}
	if messageID.Empty() {
		return invalid("message ID is required")
	}
	return s.backend.Unsend(ctx, messageID)
}

func (s *Service) MessageRequests(ctx context.Context) ([]model.MessageRequest, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.backend.ListMessageRequests(ctx)
}

func (s *Service) Themes(ctx context.Context) ([]model.Theme, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.backend.ListThemes(ctx)
}

func (s *Service) FindTheme(ctx context.Context, query string) (*model.Theme, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(query) == "" {
		return nil, invalid("theme query is required")
	}
	return s.backend.FindTheme(ctx, query)
}

func (s *Service) SetTheme(ctx context.Context, threadID, themeID model.ID) error {
	if err := s.ready(); err != nil {
		return err
	}
	if threadID.Empty() || themeID.Empty() {
		return invalid("thread ID and theme ID are required")
	}
	return s.backend.SetTheme(ctx, threadID, themeID)
}

func (s *Service) CurrentNote(ctx context.Context) (*model.Note, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.backend.CurrentNote(ctx)
}

func (s *Service) CreateNote(ctx context.Context, text, privacy string) (*model.Note, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(text) == "" {
		return nil, invalid("note text is required")
	}
	return s.backend.CreateNote(ctx, text, privacy)
}

func (s *Service) DeleteNote(ctx context.Context, noteID model.ID) error {
	if err := s.ready(); err != nil {
		return err
	}
	if noteID.Empty() {
		return invalid("note ID is required")
	}
	return s.backend.DeleteNote(ctx, noteID)
}

func (s *Service) RecreateNote(ctx context.Context, oldNoteID model.ID, text, privacy string) (*model.Note, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if oldNoteID.Empty() || strings.TrimSpace(text) == "" {
		return nil, invalid("old note ID and new note text are required")
	}
	return s.backend.RecreateNote(ctx, oldNoteID, text, privacy)
}

func (s *Service) ready() error {
	if s == nil || s.backend == nil {
		return ErrUnavailable
	}
	return nil
}

func invalid(message string) error { return fmt.Errorf("%w: %s", fberrors.ErrInvalidInput, message) }
