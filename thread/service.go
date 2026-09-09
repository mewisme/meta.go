// Package thread exposes Messenger thread queries and mutations.
package thread

import (
	"context"
	"errors"
	"fmt"
	"strings"

	fberrors "go.mewis.me/fbgo/errors"
	"go.mewis.me/fbgo/model"
)

var ErrUnavailable = errors.New("thread service unavailable")

type Backend interface {
	ListThreads(context.Context, int) (model.ThreadList, error)
	GetThread(context.Context, model.ID) (*model.Thread, error)
	SetThreadAdmin(context.Context, model.ID, model.ID, bool) error
	SetThreadName(context.Context, model.ID, string) error
	SetThreadEmoji(context.Context, model.ID, string) error
	SetThreadNickname(context.Context, model.ID, model.ID, string) error
}

type Service struct{ backend Backend }

func NewService(backend Backend) *Service { return &Service{backend: backend} }

func (s *Service) List(ctx context.Context, limit int) (model.ThreadList, error) {
	if err := s.ready(); err != nil {
		return model.ThreadList{}, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		return model.ThreadList{}, invalid("thread list limit cannot exceed 100")
	}
	return s.backend.ListThreads(ctx, limit)
}

func (s *Service) Get(ctx context.Context, threadID model.ID) (*model.Thread, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if threadID.Empty() {
		return nil, invalid("thread ID is required")
	}
	return s.backend.GetThread(ctx, threadID)
}

func (s *Service) SetAdmin(ctx context.Context, threadID, userID model.ID, admin bool) error {
	if err := s.ready(); err != nil {
		return err
	}
	if threadID.Empty() || userID.Empty() {
		return invalid("thread ID and user ID are required")
	}
	return s.backend.SetThreadAdmin(ctx, threadID, userID, admin)
}

func (s *Service) SetName(ctx context.Context, threadID model.ID, name string) error {
	if err := s.ready(); err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if threadID.Empty() || name == "" {
		return invalid("thread ID and name are required")
	}
	return s.backend.SetThreadName(ctx, threadID, name)
}

func (s *Service) SetEmoji(ctx context.Context, threadID model.ID, emoji string) error {
	if err := s.ready(); err != nil {
		return err
	}
	if threadID.Empty() || strings.TrimSpace(emoji) == "" {
		return invalid("thread ID and emoji are required")
	}
	return s.backend.SetThreadEmoji(ctx, threadID, emoji)
}

func (s *Service) SetNickname(ctx context.Context, threadID, userID model.ID, nickname string) error {
	if err := s.ready(); err != nil {
		return err
	}
	if threadID.Empty() || userID.Empty() {
		return invalid("thread ID and user ID are required")
	}
	return s.backend.SetThreadNickname(ctx, threadID, userID, nickname)
}

func (s *Service) ready() error {
	if s == nil || s.backend == nil {
		return ErrUnavailable
	}
	return nil
}

func invalid(message string) error { return fmt.Errorf("%w: %s", fberrors.ErrInvalidInput, message) }
