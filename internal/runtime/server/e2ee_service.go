package server

import (
	"context"

	fberrors "go.mewis.me/meta.go/errors"
	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
	"go.mewis.me/meta.go/internal/runtime/session"
	"go.mewis.me/meta.go/model"
)

type e2eeService struct {
	metav1.UnimplementedE2EEServiceServer
	server *Server
}

func (s *e2eeService) service(sessionID string) (session.E2EE, error) {
	sess, err := (&messengerService{server: s.server}).runtimeSession(sessionID)
	if err != nil {
		return nil, err
	}
	service := sess.Client().E2EEService()
	if service == nil {
		return nil, fberrors.ErrE2EENotReady
	}
	return service, nil
}

func (s *e2eeService) SendText(ctx context.Context, req *metav1.E2EEServiceSendTextRequest) (*metav1.E2EEServiceSendTextResponse, error) {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	result, err := service.SendE2EE(ctx, model.E2EESendRequest{ChatJID: req.GetChatJid(), FacebookUserID: model.ID(req.GetFacebookUserId()), Text: req.GetText(), ReplyTo: model.ID(req.GetReplyTo()), ReplySenderJID: req.GetReplySenderJid()})
	if err != nil {
		return nil, grpcError(err)
	}
	return &metav1.E2EEServiceSendTextResponse{MessageId: result.MessageID.String(), Timestamp: timestampOrNil(result.Timestamp)}, nil
}

func (s *e2eeService) React(ctx context.Context, req *metav1.E2EEServiceReactRequest) (*metav1.E2EEServiceReactResponse, error) {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.ReactE2EE(ctx, model.E2EEReactionRequest{ChatJID: req.GetChatJid(), MessageID: model.ID(req.GetMessageId()), SenderJID: req.GetSenderJid(), Reaction: req.GetReaction()}); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.E2EEServiceReactResponse{}, nil
}

func (s *e2eeService) Edit(ctx context.Context, req *metav1.E2EEServiceEditRequest) (*metav1.E2EEServiceEditResponse, error) {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.EditE2EE(ctx, req.GetChatJid(), model.ID(req.GetMessageId()), req.GetText()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.E2EEServiceEditResponse{}, nil
}

func (s *e2eeService) Unsend(ctx context.Context, req *metav1.E2EEServiceUnsendRequest) (*metav1.E2EEServiceUnsendResponse, error) {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.UnsendE2EE(ctx, req.GetChatJid(), model.ID(req.GetMessageId())); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.E2EEServiceUnsendResponse{}, nil
}

func (s *e2eeService) SetTyping(ctx context.Context, req *metav1.E2EEServiceSetTypingRequest) (*metav1.E2EEServiceSetTypingResponse, error) {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.TypingE2EE(ctx, req.GetChatJid(), req.GetTyping()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.E2EEServiceSetTypingResponse{}, nil
}

func (s *e2eeService) MarkRead(ctx context.Context, req *metav1.E2EEServiceMarkReadRequest) (*metav1.E2EEServiceMarkReadResponse, error) {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	timestamp, err := timeFromProto(req.GetTimestamp(), "timestamp")
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.ReadE2EE(ctx, model.E2EEReadRequest{ChatJID: req.GetChatJid(), SenderJID: req.GetSenderJid(), MessageIDs: idsFromStrings(req.GetMessageIds()), Timestamp: timestamp}); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.E2EEServiceMarkReadResponse{}, nil
}
