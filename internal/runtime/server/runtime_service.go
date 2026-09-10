package server

import (
	"context"
	"runtime"

	metago "go.mewis.me/meta.go"
	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
)

type runtimeService struct {
	metav1.UnimplementedRuntimeServiceServer
	server *Server
}

func (s *runtimeService) GetInfo(context.Context, *metav1.GetInfoRequest) (*metav1.GetInfoResponse, error) {
	build := metago.BuildVersion()
	return &metav1.GetInfoResponse{
		Protocol:     &metav1.ProtocolVersion{Major: ProtocolMajor, Minor: ProtocolMinor},
		Build:        &metav1.BuildInfo{Version: build.Version, Commit: build.Commit, GoVersion: build.GoVersion, Os: runtime.GOOS, Arch: runtime.GOARCH},
		Capabilities: append([]string(nil), s.server.capabilities...),
	}, nil
}
