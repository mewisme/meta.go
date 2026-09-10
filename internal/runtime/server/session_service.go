package server

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"

	"go.mewis.me/meta.go/auth"
	fberrors "go.mewis.me/meta.go/errors"
	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
	"go.mewis.me/meta.go/internal/runtime/session"
)

const maxSessionEventBuffer = 65536

type sessionService struct {
	metav1.UnimplementedSessionServiceServer
	server *Server
}

func (s *sessionService) CreateSession(ctx context.Context, req *metav1.CreateSessionRequest) (*metav1.CreateSessionResponse, error) {
	if req == nil || len(req.Cookies) == 0 {
		return nil, grpcError(fmt.Errorf("%w: cookies are required", fberrors.ErrInvalidInput))
	}
	if req.EventBuffer > maxSessionEventBuffer {
		return nil, grpcError(fmt.Errorf("%w: event buffer cannot exceed %d", fberrors.ErrInvalidInput, maxSessionEventBuffer))
	}
	var timeout time.Duration
	if req.Timeout != nil {
		if err := req.Timeout.CheckValid(); err != nil || req.Timeout.AsDuration() <= 0 {
			return nil, grpcError(fmt.Errorf("%w: timeout must be positive", fberrors.ErrInvalidInput))
		}
		timeout = req.Timeout.AsDuration()
	}
	cookies := make(auth.Cookies, len(req.Cookies))
	for key, value := range req.Cookies {
		cookies[key] = value
	}
	created, err := s.server.sessions.Create(session.Config{Cookies: cookies, E2EE: req.E2Ee, EventBuffer: int(req.EventBuffer), Timeout: timeout})
	if err != nil {
		return nil, grpcError(err)
	}
	return &metav1.CreateSessionResponse{SessionId: created.ID()}, nil
}

func (s *sessionService) Connect(ctx context.Context, req *metav1.ConnectRequest) (*metav1.ConnectResponse, error) {
	sess, err := s.session(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	account, err := sess.Connect(ctx)
	if err != nil {
		return nil, grpcError(err)
	}
	return &metav1.ConnectResponse{Account: accountToProto(account)}, nil
}

func (s *sessionService) CloseSession(_ context.Context, req *metav1.CloseSessionRequest) (*metav1.CloseSessionResponse, error) {
	if err := s.server.sessions.Close(req.GetSessionId()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.CloseSessionResponse{}, nil
}

func (s *sessionService) GetHealth(_ context.Context, req *metav1.GetHealthRequest) (*metav1.GetHealthResponse, error) {
	sess, err := s.session(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	return &metav1.GetHealthResponse{Health: healthToProto(sess.Health(), sess.SubscriberDropped())}, nil
}

func (s *sessionService) SubscribeEvents(req *metav1.SubscribeEventsRequest, stream grpc.ServerStreamingServer[metav1.SubscribeEventsResponse]) error {
	sess, err := s.session(req.GetSessionId())
	if err != nil {
		return grpcError(err)
	}
	if req.Buffer > maxSessionEventBuffer {
		return grpcError(fmt.Errorf("%w: subscriber buffer cannot exceed %d", fberrors.ErrInvalidInput, maxSessionEventBuffer))
	}
	subscription, err := sess.Subscribe(int(req.Buffer))
	if err != nil {
		return grpcError(err)
	}
	defer subscription.Close()
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case event, ok := <-subscription.Events:
			if !ok {
				return nil
			}
			converted, known := eventToProto(event)
			if !known {
				continue
			}
			if err := stream.Send(&metav1.SubscribeEventsResponse{Event: converted}); err != nil {
				return err
			}
		}
	}
}

func (s *sessionService) session(id string) (*session.Session, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: session ID is required", fberrors.ErrInvalidInput)
	}
	return s.server.sessions.Get(id)
}
