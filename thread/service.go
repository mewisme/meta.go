// Package thread exposes Messenger thread queries and mutations.
package thread

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	fberrors "go.mewis.me/meta.go/errors"
	"go.mewis.me/meta.go/model"
)

var ErrUnavailable = errors.New("thread service unavailable")

type Backend interface {
	ListThreads(context.Context, int) (model.ThreadList, error)
	GetThread(context.Context, model.ID) (*model.Thread, error)
	CreatePoll(context.Context, model.ID, string, []string) error
	VotePoll(context.Context, model.ID, model.ID, []model.ID) error
	ListPinnedMessages(context.Context, model.ID) ([]model.PinnedMessage, error)
	FetchPollDetails(context.Context, model.ID) (*model.PollDetails, error)
	SearchThreadMessages(context.Context, model.MessageSearchRequest) (*model.MessageSearchPage, error)
	MuteThread(context.Context, model.ID, time.Duration) error
	MuteThreadCalls(context.Context, model.ID, time.Duration) error
	SetThreadApprovalMode(context.Context, model.ID, bool) error
	SetThreadArchived(context.Context, model.ID, bool) error
	SetMessagePinned(context.Context, model.ID, model.ID, bool) error
	SetThreadPhoto(context.Context, model.ID, model.AttachmentInput) error
	DeleteThread(context.Context, model.ID) error
	CreateDM(context.Context, model.ID) (model.ID, error)
	SearchMessengerUsers(context.Context, string) ([]model.User, error)
	GetMessengerContact(context.Context, model.ID) (*model.User, error)
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

func (s *Service) CreatePoll(ctx context.Context, threadID model.ID, question string, options []string) error {
	if err := s.ready(); err != nil {
		return err
	}
	question = strings.TrimSpace(question)
	if threadID.Empty() || question == "" || len(options) < 2 {
		return invalid("thread ID, question and at least two options are required")
	}
	clean := make([]string, len(options))
	for i, option := range options {
		clean[i] = strings.TrimSpace(option)
		if clean[i] == "" {
			return invalid("poll options cannot be empty")
		}
	}
	return s.backend.CreatePoll(ctx, threadID, question, clean)
}

func (s *Service) VotePoll(ctx context.Context, threadID, pollID model.ID, optionIDs []model.ID) error {
	if err := s.ready(); err != nil {
		return err
	}
	if threadID.Empty() || pollID.Empty() || len(optionIDs) == 0 {
		return invalid("thread ID, poll ID and selected option IDs are required")
	}
	for _, optionID := range optionIDs {
		if optionID.Empty() {
			return invalid("poll option IDs cannot be empty")
		}
	}
	return s.backend.VotePoll(ctx, threadID, pollID, optionIDs)
}

func (s *Service) PinnedMessages(ctx context.Context, threadID model.ID) ([]model.PinnedMessage, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if threadID.Empty() {
		return nil, invalid("thread ID is required")
	}
	return s.backend.ListPinnedMessages(ctx, threadID)
}

func (s *Service) PollDetails(ctx context.Context, pollID model.ID) (*model.PollDetails, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if pollID.Empty() {
		return nil, invalid("poll ID is required")
	}
	return s.backend.FetchPollDetails(ctx, pollID)
}

func (s *Service) SearchMessages(ctx context.Context, req model.MessageSearchRequest) (*model.MessageSearchPage, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	req.Query = strings.TrimSpace(req.Query)
	if req.ThreadID.Empty() || req.Query == "" {
		return nil, invalid("thread ID and search query are required")
	}
	if req.Cursor != nil && strings.TrimSpace(*req.Cursor) == "" {
		return nil, invalid("message search cursor cannot be empty")
	}
	return s.backend.SearchThreadMessages(ctx, req)
}

// Mute mutes a thread for duration. Zero unmutes; any negative duration mutes indefinitely.
func (s *Service) Mute(ctx context.Context, threadID model.ID, duration time.Duration) error {
	if err := s.ready(); err != nil {
		return err
	}
	if threadID.Empty() {
		return invalid("thread ID is required")
	}
	return s.backend.MuteThread(ctx, threadID, duration)
}

// MuteCalls mutes call notifications for duration. Zero unmutes; any negative duration mutes indefinitely.
func (s *Service) MuteCalls(ctx context.Context, threadID model.ID, duration time.Duration) error {
	if err := s.ready(); err != nil {
		return err
	}
	if threadID.Empty() {
		return invalid("thread ID is required")
	}
	return s.backend.MuteThreadCalls(ctx, threadID, duration)
}

func (s *Service) SetApprovalMode(ctx context.Context, threadID model.ID, enabled bool) error {
	if err := s.ready(); err != nil {
		return err
	}
	if threadID.Empty() {
		return invalid("thread ID is required")
	}
	return s.backend.SetThreadApprovalMode(ctx, threadID, enabled)
}

func (s *Service) SetArchived(ctx context.Context, threadID model.ID, archived bool) error {
	if err := s.ready(); err != nil {
		return err
	}
	if threadID.Empty() {
		return invalid("thread ID is required")
	}
	return s.backend.SetThreadArchived(ctx, threadID, archived)
}

func (s *Service) PinMessage(ctx context.Context, threadID, messageID model.ID) error {
	return s.setMessagePinned(ctx, threadID, messageID, true)
}

func (s *Service) UnpinMessage(ctx context.Context, threadID, messageID model.ID) error {
	return s.setMessagePinned(ctx, threadID, messageID, false)
}

func (s *Service) setMessagePinned(ctx context.Context, threadID, messageID model.ID, pinned bool) error {
	if err := s.ready(); err != nil {
		return err
	}
	if threadID.Empty() || messageID.Empty() {
		return invalid("thread ID and message ID are required")
	}
	return s.backend.SetMessagePinned(ctx, threadID, messageID, pinned)
}

func (s *Service) SetPhoto(ctx context.Context, threadID model.ID, input model.AttachmentInput) error {
	if err := s.ready(); err != nil {
		return err
	}
	if threadID.Empty() || input.Reader == nil {
		return invalid("thread ID and photo reader are required")
	}
	if input.Size < 0 {
		return invalid("photo size cannot be negative")
	}
	return s.backend.SetThreadPhoto(ctx, threadID, input)
}

func (s *Service) Delete(ctx context.Context, threadID model.ID) error {
	if err := s.ready(); err != nil {
		return err
	}
	if threadID.Empty() {
		return invalid("thread ID is required")
	}
	return s.backend.DeleteThread(ctx, threadID)
}

func (s *Service) CreateDM(ctx context.Context, userID model.ID) (model.ID, error) {
	if err := s.ready(); err != nil {
		return "", err
	}
	if userID.Empty() {
		return "", invalid("user ID is required")
	}
	return s.backend.CreateDM(ctx, userID)
}

func (s *Service) SearchUsers(ctx context.Context, query string) ([]model.User, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, invalid("search query is required")
	}
	return s.backend.SearchMessengerUsers(ctx, query)
}

func (s *Service) GetContact(ctx context.Context, userID model.ID) (*model.User, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if userID.Empty() {
		return nil, invalid("user ID is required")
	}
	return s.backend.GetMessengerContact(ctx, userID)
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
