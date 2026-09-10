package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"

	"google.golang.org/grpc"

	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
	"go.mewis.me/meta.go/internal/runtime/session"
)

const (
	DefaultListenAddress = "127.0.0.1:0"
	ProtocolMajor        = 1
	ProtocolMinor        = 0
)

type Config struct {
	Token        string
	Sessions     *session.Manager
	Capabilities []string
}

type Server struct {
	grpc         *grpc.Server
	sessions     *session.Manager
	token        string
	capabilities []string
}

func New(config Config) (*Server, error) {
	token := strings.TrimSpace(config.Token)
	if token == "" {
		var err error
		token, err = newToken()
		if err != nil {
			return nil, fmt.Errorf("generate runtime auth token: %w", err)
		}
	}
	sessions := config.Sessions
	if sessions == nil {
		sessions = session.NewManager(nil)
	}
	capabilities := append([]string(nil), config.Capabilities...)
	sort.Strings(capabilities)
	capabilities = compactStrings(capabilities)
	s := &Server{sessions: sessions, token: token, capabilities: capabilities}
	s.grpc = grpc.NewServer(grpc.UnaryInterceptor(unaryAuth(token)), grpc.StreamInterceptor(streamAuth(token)))
	metav1.RegisterRuntimeServiceServer(s.grpc, &runtimeService{server: s})
	metav1.RegisterSessionServiceServer(s.grpc, &sessionService{server: s})
	metav1.RegisterMessengerServiceServer(s.grpc, &messengerService{server: s})
	return s, nil
}

func (s *Server) Token() string { return s.token }

func (s *Server) Serve(listener net.Listener) error { return s.grpc.Serve(listener) }

func (s *Server) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.grpc.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		s.grpc.Stop()
		<-done
	}
	return errors.Join(s.sessions.CloseAll(), ctx.Err())
}

func Listen(address string) (net.Listener, error) {
	if strings.TrimSpace(address) == "" {
		address = DefaultListenAddress
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid runtime listen address: %w", err)
	}
	if !loopbackHost(host) {
		return nil, fmt.Errorf("runtime listen address must be loopback")
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok || !addr.IP.IsLoopback() {
		_ = listener.Close()
		return nil, fmt.Errorf("runtime listener resolved outside loopback")
	}
	return listener, nil
}

func loopbackHost(host string) bool {
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}
