package server

import (
	"context"
	"fmt"

	fberrors "go.mewis.me/meta.go/errors"
	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
	"go.mewis.me/meta.go/internal/runtime/session"
	"go.mewis.me/meta.go/model"
)

type messengerService struct {
	metav1.UnimplementedMessengerServiceServer
	server *Server
}

func (s *messengerService) messenger(sessionID string) (session.Messenger, error) {
	sess, err := s.runtimeSession(sessionID)
	if err != nil {
		return nil, err
	}
	service := sess.Client().MessengerService()
	if service == nil {
		return nil, fberrors.ErrNotConnected
	}
	return service, nil
}

func (s *messengerService) threads(sessionID string) (session.Threads, error) {
	sess, err := s.runtimeSession(sessionID)
	if err != nil {
		return nil, err
	}
	service := sess.Client().ThreadService()
	if service == nil {
		return nil, fberrors.ErrNotConnected
	}
	return service, nil
}

func (s *messengerService) runtimeSession(id string) (*session.Session, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: session ID is required", fberrors.ErrInvalidInput)
	}
	return s.server.sessions.Get(id)
}

func (s *messengerService) Send(ctx context.Context, req *metav1.SendRequest) (*metav1.SendResponse, error) {
	service, err := s.messenger(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	result, err := service.Send(ctx, sendRequestFromProto(req))
	if err != nil {
		return nil, grpcError(err)
	}
	return &metav1.SendResponse{MessageId: result.MessageID.String(), Timestamp: timestampOrNil(result.Timestamp)}, nil
}

func (s *messengerService) Forward(ctx context.Context, req *metav1.ForwardRequest) (*metav1.ForwardResponse, error) {
	service, err := s.messenger(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	result, err := service.Forward(ctx, model.ID(req.GetThreadId()), model.ID(req.GetMessageId()))
	if err != nil {
		return nil, grpcError(err)
	}
	return &metav1.ForwardResponse{MessageId: result.MessageID.String(), Timestamp: timestampOrNil(result.Timestamp)}, nil
}

func (s *messengerService) ShareContact(ctx context.Context, req *metav1.ShareContactRequest) (*metav1.ShareContactResponse, error) {
	service, err := s.messenger(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.ShareContact(ctx, model.ID(req.GetThreadId()), model.ID(req.GetContactId()), req.GetText()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.ShareContactResponse{}, nil
}

func (s *messengerService) React(ctx context.Context, req *metav1.ReactRequest) (*metav1.ReactResponse, error) {
	service, err := s.messenger(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.React(ctx, model.ID(req.GetThreadId()), model.ID(req.GetMessageId()), req.GetReaction()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.ReactResponse{}, nil
}

func (s *messengerService) Edit(ctx context.Context, req *metav1.EditRequest) (*metav1.EditResponse, error) {
	service, err := s.messenger(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.Edit(ctx, model.ID(req.GetMessageId()), req.GetText()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.EditResponse{}, nil
}

func (s *messengerService) Unsend(ctx context.Context, req *metav1.UnsendRequest) (*metav1.UnsendResponse, error) {
	service, err := s.messenger(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.Unsend(ctx, model.ID(req.GetMessageId())); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.UnsendResponse{}, nil
}

func (s *messengerService) SetTyping(ctx context.Context, req *metav1.SetTypingRequest) (*metav1.SetTypingResponse, error) {
	service, err := s.messenger(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.Typing(ctx, model.ID(req.GetThreadId()), req.GetTyping(), req.GetGroup(), req.GetThreadType()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.SetTypingResponse{}, nil
}

func (s *messengerService) MarkRead(ctx context.Context, req *metav1.MarkReadRequest) (*metav1.MarkReadResponse, error) {
	service, err := s.messenger(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	watermark, err := timeFromProto(req.GetWatermark(), "watermark")
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.Read(ctx, model.ID(req.GetThreadId()), watermark); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.MarkReadResponse{}, nil
}

func (s *messengerService) ListMessageRequests(ctx context.Context, req *metav1.ListMessageRequestsRequest) (*metav1.ListMessageRequestsResponse, error) {
	service, err := s.messenger(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	items, err := service.MessageRequests(ctx)
	if err != nil {
		return nil, grpcError(err)
	}
	result := &metav1.ListMessageRequestsResponse{Requests: make([]*metav1.MessageRequest, 0, len(items))}
	for _, item := range items {
		result.Requests = append(result.Requests, messageRequestToProto(item))
	}
	return result, nil
}

func (s *messengerService) ListThemes(ctx context.Context, req *metav1.ListThemesRequest) (*metav1.ListThemesResponse, error) {
	service, err := s.messenger(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	items, err := service.Themes(ctx)
	if err != nil {
		return nil, grpcError(err)
	}
	result := &metav1.ListThemesResponse{Themes: make([]*metav1.Theme, 0, len(items))}
	for i := range items {
		result.Themes = append(result.Themes, themeToProto(&items[i]))
	}
	return result, nil
}

func (s *messengerService) FindTheme(ctx context.Context, req *metav1.FindThemeRequest) (*metav1.FindThemeResponse, error) {
	service, err := s.messenger(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	item, err := service.FindTheme(ctx, req.GetQuery())
	if err != nil {
		return nil, grpcError(err)
	}
	return &metav1.FindThemeResponse{Theme: themeToProto(item)}, nil
}

func (s *messengerService) SetTheme(ctx context.Context, req *metav1.SetThemeRequest) (*metav1.SetThemeResponse, error) {
	service, err := s.messenger(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.SetTheme(ctx, model.ID(req.GetThreadId()), model.ID(req.GetThemeId())); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.SetThemeResponse{}, nil
}

func (s *messengerService) GetCurrentNote(ctx context.Context, req *metav1.GetCurrentNoteRequest) (*metav1.GetCurrentNoteResponse, error) {
	service, err := s.messenger(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	note, err := service.CurrentNote(ctx)
	if err != nil {
		return nil, grpcError(err)
	}
	return &metav1.GetCurrentNoteResponse{Note: noteToProto(note)}, nil
}

func (s *messengerService) CreateNote(ctx context.Context, req *metav1.CreateNoteRequest) (*metav1.CreateNoteResponse, error) {
	service, err := s.messenger(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	note, err := service.CreateNote(ctx, req.GetText(), req.GetPrivacy())
	if err != nil {
		return nil, grpcError(err)
	}
	return &metav1.CreateNoteResponse{Note: noteToProto(note)}, nil
}

func (s *messengerService) DeleteNote(ctx context.Context, req *metav1.DeleteNoteRequest) (*metav1.DeleteNoteResponse, error) {
	service, err := s.messenger(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.DeleteNote(ctx, model.ID(req.GetNoteId())); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.DeleteNoteResponse{}, nil
}

func (s *messengerService) RecreateNote(ctx context.Context, req *metav1.RecreateNoteRequest) (*metav1.RecreateNoteResponse, error) {
	service, err := s.messenger(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	note, err := service.RecreateNote(ctx, model.ID(req.GetOldNoteId()), req.GetText(), req.GetPrivacy())
	if err != nil {
		return nil, grpcError(err)
	}
	return &metav1.RecreateNoteResponse{Note: noteToProto(note)}, nil
}

func (s *messengerService) SetRestricted(ctx context.Context, req *metav1.SetRestrictedRequest) (*metav1.SetRestrictedResponse, error) {
	service, err := s.messenger(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.SetRestricted(ctx, model.ID(req.GetUserId()), req.GetRestricted()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.SetRestrictedResponse{}, nil
}

func (s *messengerService) SetMessageBlocked(ctx context.Context, req *metav1.SetMessageBlockedRequest) (*metav1.SetMessageBlockedResponse, error) {
	service, err := s.messenger(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.SetMessageBlocked(ctx, model.ID(req.GetUserId()), req.GetBlocked()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.SetMessageBlockedResponse{}, nil
}

func (s *messengerService) ListThreads(ctx context.Context, req *metav1.ListThreadsRequest) (*metav1.ListThreadsResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	list, err := service.List(ctx, int(req.GetLimit()))
	if err != nil {
		return nil, grpcError(err)
	}
	result := &metav1.ListThreadsResponse{SyncSequenceId: list.SyncSequenceID, Threads: make([]*metav1.Thread, 0, len(list.Threads))}
	for _, item := range list.Threads {
		result.Threads = append(result.Threads, threadToProto(item))
	}
	return result, nil
}

func (s *messengerService) GetThread(ctx context.Context, req *metav1.GetThreadRequest) (*metav1.GetThreadResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	item, err := service.Get(ctx, model.ID(req.GetThreadId()))
	if err != nil {
		return nil, grpcError(err)
	}
	if item == nil {
		return &metav1.GetThreadResponse{}, nil
	}
	return &metav1.GetThreadResponse{Thread: threadToProto(*item)}, nil
}

func (s *messengerService) CreatePoll(ctx context.Context, req *metav1.CreatePollRequest) (*metav1.CreatePollResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.CreatePoll(ctx, model.ID(req.GetThreadId()), req.GetQuestion(), req.GetOptions()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.CreatePollResponse{}, nil
}

func (s *messengerService) VotePoll(ctx context.Context, req *metav1.VotePollRequest) (*metav1.VotePollResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.VotePoll(ctx, model.ID(req.GetThreadId()), model.ID(req.GetPollId()), idsFromStrings(req.GetOptionIds())); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.VotePollResponse{}, nil
}

func (s *messengerService) ListPinnedMessages(ctx context.Context, req *metav1.ListPinnedMessagesRequest) (*metav1.ListPinnedMessagesResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	items, err := service.PinnedMessages(ctx, model.ID(req.GetThreadId()))
	if err != nil {
		return nil, grpcError(err)
	}
	result := &metav1.ListPinnedMessagesResponse{Messages: make([]*metav1.PinnedMessage, 0, len(items))}
	for _, item := range items {
		result.Messages = append(result.Messages, pinnedMessageToProto(item))
	}
	return result, nil
}

func (s *messengerService) GetPollDetails(ctx context.Context, req *metav1.GetPollDetailsRequest) (*metav1.GetPollDetailsResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	poll, err := service.PollDetails(ctx, model.ID(req.GetPollId()))
	if err != nil {
		return nil, grpcError(err)
	}
	return &metav1.GetPollDetailsResponse{Poll: pollDetailsToProto(poll)}, nil
}

func (s *messengerService) SearchMessages(ctx context.Context, req *metav1.SearchMessagesRequest) (*metav1.SearchMessagesResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	page, err := service.SearchMessages(ctx, model.MessageSearchRequest{ThreadID: model.ID(req.GetThreadId()), Query: req.GetQuery(), Cursor: req.Cursor})
	if err != nil {
		return nil, grpcError(err)
	}
	result := &metav1.SearchMessagesResponse{ResultCount: page.ResultCount, HasNextPage: page.HasNextPage, NextCursor: page.NextCursor, Results: make([]*metav1.MessageSearchResult, 0, len(page.Results))}
	for _, item := range page.Results {
		result.Results = append(result.Results, messageSearchResultToProto(item))
	}
	return result, nil
}

func (s *messengerService) MuteThread(ctx context.Context, req *metav1.MuteThreadRequest) (*metav1.MuteThreadResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	duration, err := durationFromProto(req.GetDuration(), req.GetIndefinitely())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.Mute(ctx, model.ID(req.GetThreadId()), duration); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.MuteThreadResponse{}, nil
}

func (s *messengerService) MuteThreadCalls(ctx context.Context, req *metav1.MuteThreadCallsRequest) (*metav1.MuteThreadCallsResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	duration, err := durationFromProto(req.GetDuration(), req.GetIndefinitely())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.MuteCalls(ctx, model.ID(req.GetThreadId()), duration); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.MuteThreadCallsResponse{}, nil
}

func (s *messengerService) SetApprovalMode(ctx context.Context, req *metav1.SetApprovalModeRequest) (*metav1.SetApprovalModeResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.SetApprovalMode(ctx, model.ID(req.GetThreadId()), req.GetEnabled()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.SetApprovalModeResponse{}, nil
}

func (s *messengerService) SetArchived(ctx context.Context, req *metav1.SetArchivedRequest) (*metav1.SetArchivedResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.SetArchived(ctx, model.ID(req.GetThreadId()), req.GetArchived()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.SetArchivedResponse{}, nil
}

func (s *messengerService) SetMessagePinned(ctx context.Context, req *metav1.SetMessagePinnedRequest) (*metav1.SetMessagePinnedResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	var mutationErr error
	if req.GetPinned() {
		mutationErr = service.PinMessage(ctx, model.ID(req.GetThreadId()), model.ID(req.GetMessageId()))
	} else {
		mutationErr = service.UnpinMessage(ctx, model.ID(req.GetThreadId()), model.ID(req.GetMessageId()))
	}
	if mutationErr != nil {
		return nil, grpcError(mutationErr)
	}
	return &metav1.SetMessagePinnedResponse{}, nil
}

func (s *messengerService) DeleteThread(ctx context.Context, req *metav1.DeleteThreadRequest) (*metav1.DeleteThreadResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.Delete(ctx, model.ID(req.GetThreadId())); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.DeleteThreadResponse{}, nil
}

func (s *messengerService) CreateDM(ctx context.Context, req *metav1.CreateDMRequest) (*metav1.CreateDMResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	threadID, err := service.CreateDM(ctx, model.ID(req.GetUserId()))
	if err != nil {
		return nil, grpcError(err)
	}
	return &metav1.CreateDMResponse{ThreadId: threadID.String()}, nil
}

func (s *messengerService) SearchUsers(ctx context.Context, req *metav1.SearchUsersRequest) (*metav1.SearchUsersResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	users, err := service.SearchUsers(ctx, req.GetQuery())
	if err != nil {
		return nil, grpcError(err)
	}
	result := &metav1.SearchUsersResponse{Users: make([]*metav1.User, 0, len(users))}
	for _, user := range users {
		result.Users = append(result.Users, userToProto(user))
	}
	return result, nil
}

func (s *messengerService) GetContact(ctx context.Context, req *metav1.GetContactRequest) (*metav1.GetContactResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	user, err := service.GetContact(ctx, model.ID(req.GetUserId()))
	if err != nil {
		return nil, grpcError(err)
	}
	if user == nil {
		return &metav1.GetContactResponse{}, nil
	}
	return &metav1.GetContactResponse{User: userToProto(*user)}, nil
}

func (s *messengerService) SetAdmin(ctx context.Context, req *metav1.SetAdminRequest) (*metav1.SetAdminResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.SetAdmin(ctx, model.ID(req.GetThreadId()), model.ID(req.GetUserId()), req.GetAdmin()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.SetAdminResponse{}, nil
}

func (s *messengerService) SetThreadName(ctx context.Context, req *metav1.SetThreadNameRequest) (*metav1.SetThreadNameResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.SetName(ctx, model.ID(req.GetThreadId()), req.GetName()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.SetThreadNameResponse{}, nil
}

func (s *messengerService) SetThreadEmoji(ctx context.Context, req *metav1.SetThreadEmojiRequest) (*metav1.SetThreadEmojiResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.SetEmoji(ctx, model.ID(req.GetThreadId()), req.GetEmoji()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.SetThreadEmojiResponse{}, nil
}

func (s *messengerService) SetNickname(ctx context.Context, req *metav1.SetNicknameRequest) (*metav1.SetNicknameResponse, error) {
	service, err := s.threads(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.SetNickname(ctx, model.ID(req.GetThreadId()), model.ID(req.GetUserId()), req.GetNickname()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.SetNicknameResponse{}, nil
}
