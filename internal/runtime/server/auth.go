package server

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const authorizationKey = "authorization"

func newToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func authorized(ctx context.Context, token string) bool {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false
	}
	want := "Bearer " + token
	for _, value := range md.Get(authorizationKey) {
		if len(value) == len(want) && subtle.ConstantTimeCompare([]byte(value), []byte(want)) == 1 {
			return true
		}
	}
	return false
}

func unaryAuth(token string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !authorized(ctx, token) {
			return nil, status.Error(codes.Unauthenticated, "runtime authentication required")
		}
		return handler(ctx, req)
	}
}

func streamAuth(token string) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !authorized(stream.Context(), token) {
			return status.Error(codes.Unauthenticated, "runtime authentication required")
		}
		return handler(srv, stream)
	}
}
