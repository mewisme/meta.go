package server

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"

	"google.golang.org/grpc"

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

func (s *e2eeService) SendMedia(stream grpc.ClientStreamingServer[metav1.E2EEServiceSendMediaRequest, metav1.E2EEServiceSendMediaResponse]) error {
	first, err := stream.Recv()
	if err != nil {
		return grpcError(err)
	}
	metadata := first.GetMetadata()
	if metadata == nil {
		return grpcError(fmt.Errorf("%w: first E2EE media frame must contain metadata", fberrors.ErrInvalidInput))
	}
	service, err := s.service(metadata.GetSessionId())
	if err != nil {
		return grpcError(err)
	}
	kind, err := e2eeMediaKindFromProto(metadata.GetKind())
	if err != nil {
		return grpcError(err)
	}
	recv := func() ([]byte, error) {
		frame, err := stream.Recv()
		if err != nil {
			return nil, err
		}
		if frame.GetMetadata() != nil {
			return nil, fmt.Errorf("%w: E2EE media metadata may only be sent once", fberrors.ErrInvalidInput)
		}
		chunk := frame.GetChunk()
		if chunk == nil {
			return nil, fmt.Errorf("%w: E2EE media frame must contain a chunk", fberrors.ErrInvalidInput)
		}
		return chunk.GetData(), nil
	}
	result, hash, err := receiveMedia(stream.Context(), metadata.GetSize(), metadata.GetSha256(), recv, func(reader io.Reader) (model.SendResult, error) {
		return service.SendE2EEMedia(stream.Context(), model.E2EEMediaInput{ChatJID: metadata.GetChatJid(), FacebookUserID: model.ID(metadata.GetFacebookUserId()), Kind: kind, Name: metadata.GetName(), ContentType: metadata.GetContentType(), Reader: reader, Size: metadata.GetSize(), Caption: metadata.GetCaption(), Width: int(metadata.GetWidth()), Height: int(metadata.GetHeight()), Duration: int(metadata.GetDuration()), Voice: metadata.GetVoice(), ReplyTo: model.ID(metadata.GetReplyTo()), ReplySenderJID: metadata.GetReplySenderJid()})
	})
	if err != nil {
		return grpcError(err)
	}
	return stream.SendAndClose(&metav1.E2EEServiceSendMediaResponse{MessageId: result.MessageID.String(), Timestamp: timestampOrNil(result.Timestamp), Sha256: hash})
}

func (s *e2eeService) DownloadMedia(req *metav1.E2EEServiceDownloadMediaRequest, stream grpc.ServerStreamingServer[metav1.E2EEServiceDownloadMediaResponse]) error {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return grpcError(err)
	}
	reference, err := e2eeReferenceFromProto(req.GetReference())
	if err != nil {
		return grpcError(err)
	}
	data, err := service.DownloadE2EE(stream.Context(), model.E2EEMediaDownload{Reference: reference})
	if err != nil {
		return grpcError(err)
	}
	if len(data) > maxMediaBytes {
		return grpcError(fmt.Errorf("%w: downloaded media exceeds %d bytes", fberrors.ErrInvalidInput, maxMediaBytes))
	}
	hash := sha256.Sum256(data)
	if err := stream.Send(&metav1.E2EEServiceDownloadMediaResponse{Payload: &metav1.E2EEServiceDownloadMediaResponse_Metadata{Metadata: &metav1.E2EEServiceDownloadMediaMetadata{Size: int64(len(data)), ContentType: reference.ContentType, Sha256: hash[:]}}}); err != nil {
		return err
	}
	for offset := 0; offset < len(data); offset += maxMediaChunkBytes {
		end := min(offset+maxMediaChunkBytes, len(data))
		if err := stream.Send(&metav1.E2EEServiceDownloadMediaResponse{Payload: &metav1.E2EEServiceDownloadMediaResponse_Chunk{Chunk: &metav1.MediaChunk{Data: data[offset:end]}}}); err != nil {
			return err
		}
	}
	return nil
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

func e2eeMediaKindFromProto(kind metav1.E2EEMediaKind) (model.E2EEMediaKind, error) {
	switch kind {
	case metav1.E2EEMediaKind_E2EE_MEDIA_KIND_IMAGE:
		return model.E2EEMediaImage, nil
	case metav1.E2EEMediaKind_E2EE_MEDIA_KIND_VIDEO:
		return model.E2EEMediaVideo, nil
	case metav1.E2EEMediaKind_E2EE_MEDIA_KIND_AUDIO:
		return model.E2EEMediaAudio, nil
	case metav1.E2EEMediaKind_E2EE_MEDIA_KIND_DOCUMENT:
		return model.E2EEMediaDocument, nil
	case metav1.E2EEMediaKind_E2EE_MEDIA_KIND_STICKER:
		return model.E2EEMediaSticker, nil
	default:
		return "", fmt.Errorf("%w: unsupported E2EE media kind", fberrors.ErrInvalidInput)
	}
}

func e2eeReferenceFromProto(reference *metav1.E2EEMediaReference) (model.E2EEMediaReference, error) {
	if reference == nil {
		return model.E2EEMediaReference{}, fmt.Errorf("%w: E2EE media reference is required", fberrors.ErrInvalidInput)
	}
	kind := model.E2EEMediaKind(reference.GetKind())
	switch kind {
	case model.E2EEMediaImage, model.E2EEMediaVideo, model.E2EEMediaAudio, model.E2EEMediaDocument, model.E2EEMediaSticker:
	default:
		return model.E2EEMediaReference{}, fmt.Errorf("%w: unsupported E2EE media kind", fberrors.ErrInvalidInput)
	}
	return model.E2EEMediaReference{Kind: kind, DirectPath: reference.GetDirectPath(), MediaKey: append([]byte(nil), reference.GetMediaKey()...), FileSHA256: append([]byte(nil), reference.GetFileSha256()...), FileEncSHA256: append([]byte(nil), reference.GetFileEncSha256()...), ContentType: reference.GetContentType(), Size: reference.GetSize()}, nil
}
