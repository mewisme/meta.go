package server

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"go.mewis.me/meta.go/auth"
	fberrors "go.mewis.me/meta.go/errors"
	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
	"go.mewis.me/meta.go/internal/runtime/session"
	"go.mewis.me/meta.go/model"
)

type runtimeFakeClient struct {
	account   model.User
	snapshot  auth.AuthSnapshot
	refreshes int
	lastAuth  *auth.Source
	health    model.HealthSnapshot
	connects  int
	closes    int
	handler   func(model.Event)
	messenger session.Messenger
	threads   session.Threads
	e2ee      session.E2EE
	facebook  session.Facebook
}

func (f *runtimeFakeClient) Connect(context.Context) error {
	f.connects++
	return nil
}

func (f *runtimeFakeClient) RefreshAuth(_ context.Context, source *auth.Source) (auth.AuthSnapshot, error) {
	f.refreshes++
	f.lastAuth = source
	return f.snapshot, nil
}
func (f *runtimeFakeClient) AuthSnapshot(context.Context) (auth.AuthSnapshot, error) {
	return f.snapshot, nil
}

func (f *runtimeFakeClient) Close() error {
	f.closes++
	return nil
}

func (f *runtimeFakeClient) Health() model.HealthSnapshot        { return f.health }
func (f *runtimeFakeClient) Account() model.User                 { return f.account }
func (f *runtimeFakeClient) MessengerService() session.Messenger { return f.messenger }
func (f *runtimeFakeClient) ThreadService() session.Threads      { return f.threads }
func (f *runtimeFakeClient) E2EEService() session.E2EE           { return f.e2ee }
func (f *runtimeFakeClient) FacebookService() session.Facebook   { return f.facebook }
func (f *runtimeFakeClient) Subscribe(handler func(model.Event)) func() {
	f.handler = handler
	return func() { f.handler = nil }
}

func (f *runtimeFakeClient) emit(event model.Event) {
	if f.handler != nil {
		f.handler(event)
	}
}

func TestRuntimeSessionLifecycle(t *testing.T) {
	listener, err := Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	fake := &runtimeFakeClient{account: model.User{ID: "42", Name: "Mew", Username: "mew"}, health: model.HealthSnapshot{Regular: model.ConnectionConnected, E2EE: model.ConnectionConnecting, ReconnectCount: 2}}
	var captured session.Config
	manager := session.NewManager(func(config session.Config) (session.Client, error) {
		captured = config
		return fake, nil
	})
	runtimeServer, err := New(Config{Token: "test-token", Sessions: manager, Capabilities: []string{"session.lifecycle", "runtime.info", "session.lifecycle"}})
	if err != nil {
		t.Fatal(err)
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- runtimeServer.Serve(listener) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := runtimeServer.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("shutdown: %v", err)
		}
	})
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	runtimeClient := metav1.NewRuntimeServiceClient(conn)
	if _, err := runtimeClient.GetInfo(t.Context(), &metav1.GetInfoRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
	ctx := metadata.AppendToOutgoingContext(t.Context(), authorizationKey, "Bearer test-token")
	info, err := runtimeClient.GetInfo(ctx, &metav1.GetInfoRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if info.Protocol.GetMajor() != 1 || info.Protocol.GetMinor() != 0 || len(info.Capabilities) != 2 || info.Capabilities[0] != "runtime.info" || info.Capabilities[1] != "session.lifecycle" {
		t.Fatalf("unexpected runtime info: %#v", info)
	}
	sessions := metav1.NewSessionServiceClient(conn)
	created, err := sessions.CreateSession(ctx, &metav1.CreateSessionRequest{Cookies: map[string]string{"c_user": "42", "xs": "secret"}, E2Ee: true, EventBuffer: 16, Timeout: durationpb.New(5 * time.Second)})
	if err != nil {
		t.Fatal(err)
	}
	if created.SessionId == "" || manager.Len() != 1 || !captured.E2EE || captured.EventBuffer != 16 || captured.Timeout != 5*time.Second {
		t.Fatalf("unexpected session: response=%#v config=%#v len=%d", created, captured, manager.Len())
	}
	connected, err := sessions.Connect(ctx, &metav1.ConnectRequest{SessionId: created.SessionId})
	if err != nil {
		t.Fatal(err)
	}
	if connected.Account.GetId() != "42" || connected.Account.GetUsername() != "mew" || fake.connects != 1 {
		t.Fatalf("unexpected connect response: %#v connects=%d", connected, fake.connects)
	}
	stream, err := sessions.SubscribeEvents(ctx, &metav1.SubscribeEventsRequest{SessionId: created.SessionId, Buffer: 2})
	if err != nil {
		t.Fatal(err)
	}
	sess, err := manager.Get(created.SessionId)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for sess.SubscriberCount() != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if sess.SubscriberCount() != 1 {
		t.Fatal("event stream did not register subscriber")
	}
	fake.emit(model.Event{Kind: model.EventReady, IsNewSession: true})
	fake.emit(model.Event{Kind: model.EventReaction, Reaction: &model.ReactionEvent{MessageID: "m1", ThreadID: "t1", ActorID: "42", Reaction: "+1"}})
	firstEvent, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	secondEvent, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if firstEvent.Event.GetSequence() != 1 || !firstEvent.Event.GetReady().GetIsNewSession() || secondEvent.Event.GetSequence() != 2 || secondEvent.Event.GetReaction().GetMessageId() != "m1" {
		t.Fatalf("unexpected streamed events: %#v %#v", firstEvent, secondEvent)
	}
	fake.snapshot = auth.AuthSnapshot{Cookies: auth.Cookies{"c_user": "42", "xs": "fresh"}, AppState: auth.AppState{{Key: "c_user", Value: "42"}, {Key: "xs", Value: "fresh"}}, Session: auth.Session{FBID: "42", Name: "Mew", Username: "mew", DTSG: "d", Jazoest: "j", LSD: "l", SessionID: "facebook-session", ClientRevision: 7, BootstrappedAt: time.Unix(1_700_000_000, 0).UTC()}}
	refresh, err := sessions.RefreshAuth(ctx, &metav1.RefreshAuthRequest{SessionId: created.SessionId, Auth: &metav1.SessionAuth{Source: &metav1.SessionAuth_AppState{AppState: &metav1.AppState{Cookies: []*metav1.AppStateCookie{{Key: "c_user", Value: "42"}, {Key: "xs", Value: "fresh"}}}}}})
	if err != nil {
		t.Fatal(err)
	}
	if fake.refreshes != 1 || fake.lastAuth == nil || len(fake.lastAuth.AppState) != 2 || refresh.GetSnapshot().GetCookies().GetValues()["xs"] != "fresh" || refresh.GetSnapshot().GetSession().GetSessionId() != "facebook-session" || sess.SubscriberCount() != 1 {
		t.Fatalf("unexpected auth refresh state: refresh=%#v calls=%d auth=%#v subscribers=%d", refresh, fake.refreshes, fake.lastAuth, sess.SubscriberCount())
	}
	fake.emit(model.Event{Kind: model.EventReconnected})
	thirdEvent, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if thirdEvent.Event.GetSequence() != 3 || thirdEvent.Event.GetReconnected() == nil {
		t.Fatalf("event stream did not survive refresh: %#v", thirdEvent)
	}
	snapshot, err := sessions.GetAuthSnapshot(ctx, &metav1.GetAuthSnapshotRequest{SessionId: created.SessionId})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.GetSnapshot().GetSession().GetAccountId() != "42" || snapshot.GetSnapshot().GetAppState().GetCookies()[1].GetValue() != "fresh" {
		t.Fatalf("unexpected auth snapshot: %#v", snapshot)
	}
	health, err := sessions.GetHealth(ctx, &metav1.GetHealthRequest{SessionId: created.SessionId})
	if err != nil {
		t.Fatal(err)
	}
	if health.Health.GetRegular() != metav1.ConnectionState_CONNECTION_STATE_CONNECTED || health.Health.GetE2Ee() != metav1.ConnectionState_CONNECTION_STATE_CONNECTING || health.Health.GetReconnectCount() != 2 {
		t.Fatalf("unexpected health: %#v", health)
	}
	if _, err := sessions.CloseSession(ctx, &metav1.CloseSessionRequest{SessionId: created.SessionId}); err != nil {
		t.Fatal(err)
	}
	if fake.closes != 1 || manager.Len() != 0 {
		t.Fatalf("unexpected close state: closes=%d len=%d", fake.closes, manager.Len())
	}
	_, err = sessions.GetHealth(ctx, &metav1.GetHealthRequest{SessionId: created.SessionId})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected not found, got %v", err)
	}
	details := status.Convert(err).Details()
	if len(details) != 1 {
		t.Fatalf("expected stable error detail, got %#v", details)
	}
	detail, ok := details[0].(*metav1.ErrorDetail)
	if !ok || detail.Code != "session_not_found" {
		t.Fatalf("unexpected error detail: %#v", details[0])
	}
}

func TestCreateSessionAcceptsNewAuthAndRejectsConflict(t *testing.T) {
	tests := []struct {
		name  string
		auth  *metav1.SessionAuth
		check func(*auth.Source) bool
	}{
		{"cookies", &metav1.SessionAuth{Source: &metav1.SessionAuth_Cookies{Cookies: &metav1.CookieMap{Values: map[string]string{"c_user": "1", "xs": "x"}}}}, func(source *auth.Source) bool { return source != nil && source.Cookies["xs"] == "x" }},
		{"appstate", &metav1.SessionAuth{Source: &metav1.SessionAuth_AppState{AppState: &metav1.AppState{Cookies: []*metav1.AppStateCookie{{Key: "c_user", Value: "1"}, {Key: "xs", Value: "x"}}}}}, func(source *auth.Source) bool { return source != nil && len(source.AppState) == 2 }},
		{"credentials", &metav1.SessionAuth{Source: &metav1.SessionAuth_Credentials{Credentials: &metav1.Credentials{Identifier: "user", Password: "pw", SecondFactor: &metav1.Credentials_Otp{Otp: "123456"}}}}, func(source *auth.Source) bool {
			return source != nil && source.Credentials != nil && source.Credentials.OTP == "123456"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var captured session.Config
			manager := session.NewManager(func(config session.Config) (session.Client, error) {
				captured = config
				return &runtimeFakeClient{}, nil
			})
			service := &sessionService{server: &Server{sessions: manager}}
			created, err := service.CreateSession(t.Context(), &metav1.CreateSessionRequest{Auth: test.auth})
			if err != nil || created.GetSessionId() == "" || !test.check(captured.Auth) || captured.Cookies != nil {
				t.Fatalf("unexpected create result: response=%#v config=%#v err=%v", created, captured, err)
			}
			_ = manager.CloseAll()
		})
	}

	manager := session.NewManager(func(session.Config) (session.Client, error) { return &runtimeFakeClient{}, nil })
	service := &sessionService{server: &Server{sessions: manager}}
	for _, request := range []*metav1.CreateSessionRequest{
		{},
		{Cookies: map[string]string{"c_user": "1", "xs": "x"}, Auth: tests[0].auth},
	} {
		if _, err := service.CreateSession(t.Context(), request); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected invalid argument for %#v, got %v", request, err)
		}
	}
}

func TestListenRejectsNonLoopback(t *testing.T) {
	for _, address := range []string{"0.0.0.0:0", ":0", "192.0.2.1:0"} {
		if listener, err := Listen(address); err == nil {
			_ = listener.Close()
			t.Fatalf("expected %q to be rejected", address)
		}
	}
	listener, err := Listen("[::1]:0")
	if err != nil {
		t.Fatal(err)
	}
	_ = listener.Close()
}

func TestNewGeneratesToken(t *testing.T) {
	runtimeServer, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(runtimeServer.Token()) < 32 {
		t.Fatalf("unexpected generated token length: %d", len(runtimeServer.Token()))
	}
}

func TestRuntimeCapabilitiesIncludeAuthFeatures(t *testing.T) {
	capabilities := RuntimeCapabilities()
	for _, capability := range []string{"session.auth.app_state", "session.auth.credentials", "session.auth.refresh"} {
		if !slices.Contains(capabilities, capability) {
			t.Fatalf("missing runtime capability %q", capability)
		}
	}
	capabilities[0] = "mutated"
	if RuntimeCapabilities()[0] == "mutated" {
		t.Fatal("runtime capabilities must return a copy")
	}
}

func TestGrpcErrorStableDetails(t *testing.T) {
	err := grpcError(errors.Join(fberrors.ErrInvalidInput, errors.New("bad input")))
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("unexpected status code: %v", status.Code(err))
	}
	details := status.Convert(err).Details()
	if len(details) != 1 {
		t.Fatalf("unexpected details: %#v", details)
	}
	detail, ok := details[0].(*metav1.ErrorDetail)
	if !ok || detail.Category != metav1.ErrorCategory_ERROR_CATEGORY_INVALID_INPUT || detail.Code != "invalid_input" {
		t.Fatalf("unexpected detail: %#v", details[0])
	}
}

func TestLoopbackHost(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "::1", "localhost"} {
		if !loopbackHost(host) {
			t.Fatalf("expected %q to be loopback", host)
		}
	}
	for _, host := range []string{"", "0.0.0.0", "example.com"} {
		if loopbackHost(host) {
			t.Fatalf("expected %q not to be loopback", host)
		}
	}
}
