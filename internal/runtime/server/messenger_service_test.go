package server

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/durationpb"

	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
	"go.mewis.me/meta.go/internal/runtime/session"
	"go.mewis.me/meta.go/model"
)

type phase4Messenger struct {
	session.Messenger
	sendRequest model.SendRequest
	themes      []model.Theme
}

func (f *phase4Messenger) Send(_ context.Context, req model.SendRequest) (model.SendResult, error) {
	f.sendRequest = req
	return model.SendResult{MessageID: "m1", Timestamp: time.Unix(1_700_000_000, 0).UTC()}, nil
}

func (f *phase4Messenger) Themes(context.Context) ([]model.Theme, error) { return f.themes, nil }

type phase4Threads struct {
	session.Threads
	searchRequest model.MessageSearchRequest
	pinCalls      []bool
	muteDuration  time.Duration
}

func (f *phase4Threads) List(context.Context, int) (model.ThreadList, error) {
	return model.ThreadList{Threads: []model.Thread{{ID: "t1", Name: "thread", Participants: []model.User{{ID: "u1", Name: "Mew"}}}}, SyncSequenceID: 9}, nil
}

func (f *phase4Threads) SearchMessages(_ context.Context, req model.MessageSearchRequest) (*model.MessageSearchPage, error) {
	f.searchRequest = req
	next := "next"
	return &model.MessageSearchPage{Results: []model.MessageSearchResult{{MessageID: "m1", ThreadID: req.ThreadID, Text: req.Query}}, ResultCount: 1, HasNextPage: true, NextCursor: &next}, nil
}

func (f *phase4Threads) PinMessage(context.Context, model.ID, model.ID) error {
	f.pinCalls = append(f.pinCalls, true)
	return nil
}

func (f *phase4Threads) UnpinMessage(context.Context, model.ID, model.ID) error {
	f.pinCalls = append(f.pinCalls, false)
	return nil
}

func (f *phase4Threads) Mute(_ context.Context, _ model.ID, duration time.Duration) error {
	f.muteDuration = duration
	return nil
}

func (f *phase4Threads) GetContact(context.Context, model.ID) (*model.User, error) {
	return &model.User{ID: "u1", Name: "Mew", MessengerRestricted: true, MessengerBlockStatus: model.MessengerBlockMessageBlocked}, nil
}

func TestMessengerRPCContract(t *testing.T) {
	messenger := &phase4Messenger{themes: []model.Theme{{ID: "theme1", Name: "Theme"}}}
	threads := &phase4Threads{}
	fake := &runtimeFakeClient{messenger: messenger, threads: threads}
	manager := session.NewManager(func(session.Config) (session.Client, error) { return fake, nil })
	created, err := manager.Create(session.Config{})
	if err != nil {
		t.Fatal(err)
	}
	listener, err := Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	runtimeServer, err := New(Config{Token: "test-token", Sessions: manager})
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = runtimeServer.Serve(listener) }()
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
	ctx := metadata.AppendToOutgoingContext(t.Context(), authorizationKey, "Bearer test-token")
	client := metav1.NewMessengerServiceClient(conn)

	cursor := "cursor"
	sent, err := client.Send(ctx, &metav1.SendRequest{SessionId: created.ID(), ThreadId: "t1", Text: "hello", ReplyTo: &metav1.ReplyReference{MessageId: "m0"}, Mentions: []*metav1.Mention{{UserId: "u1", Offset: 0, Length: 5}}, Encryption: metav1.EncryptionPolicy_ENCRYPTION_POLICY_DISABLED})
	if err != nil {
		t.Fatal(err)
	}
	if sent.GetMessageId() != "m1" || messenger.sendRequest.ThreadID != "t1" || messenger.sendRequest.ReplyTo == nil || len(messenger.sendRequest.Mentions) != 1 || messenger.sendRequest.Encryption != model.EncryptionDisabled {
		t.Fatalf("send adapter mismatch: response=%#v request=%#v", sent, messenger.sendRequest)
	}
	themes, err := client.ListThemes(ctx, &metav1.ListThemesRequest{SessionId: created.ID()})
	if err != nil || len(themes.GetThemes()) != 1 || themes.GetThemes()[0].GetId() != "theme1" {
		t.Fatalf("theme adapter mismatch: %#v err=%v", themes, err)
	}
	listed, err := client.ListThreads(ctx, &metav1.ListThreadsRequest{SessionId: created.ID(), Limit: 20})
	if err != nil || listed.GetSyncSequenceId() != 9 || len(listed.GetThreads()) != 1 || listed.GetThreads()[0].GetParticipants()[0].GetId() != "u1" {
		t.Fatalf("thread adapter mismatch: %#v err=%v", listed, err)
	}
	searched, err := client.SearchMessages(ctx, &metav1.SearchMessagesRequest{SessionId: created.ID(), ThreadId: "t1", Query: "hello", Cursor: &cursor})
	if err != nil || len(searched.GetResults()) != 1 || searched.GetNextCursor() != "next" || threads.searchRequest.Cursor == nil || *threads.searchRequest.Cursor != cursor {
		t.Fatalf("search adapter mismatch: %#v captured=%#v err=%v", searched, threads.searchRequest, err)
	}
	if _, err := client.SetMessagePinned(ctx, &metav1.SetMessagePinnedRequest{SessionId: created.ID(), ThreadId: "t1", MessageId: "m1", Pinned: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.SetMessagePinned(ctx, &metav1.SetMessagePinnedRequest{SessionId: created.ID(), ThreadId: "t1", MessageId: "m1", Pinned: false}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(threads.pinCalls, []bool{true, false}) {
		t.Fatalf("pin adapter mismatch: %#v", threads.pinCalls)
	}
	if _, err := client.MuteThread(ctx, &metav1.MuteThreadRequest{SessionId: created.ID(), ThreadId: "t1", Indefinitely: true}); err != nil || threads.muteDuration >= 0 {
		t.Fatalf("mute adapter mismatch: %v err=%v", threads.muteDuration, err)
	}
	contact, err := client.GetContact(ctx, &metav1.GetContactRequest{SessionId: created.ID(), UserId: "u1"})
	if err != nil || contact.GetUser().GetId() != "u1" || !contact.GetUser().GetMessengerRestricted() || contact.GetUser().GetMessengerBlockStatus() != metav1.MessengerBlockStatus_MESSENGER_BLOCK_STATUS_MESSAGE_BLOCKED {
		t.Fatalf("contact adapter mismatch: %#v err=%v", contact, err)
	}
	if _, err := client.MuteThread(ctx, &metav1.MuteThreadRequest{SessionId: created.ID(), ThreadId: "t1", Duration: durationpb.New(-time.Minute)}); err == nil {
		t.Fatal("expected invalid negative duration")
	}
}

func TestMessengerRPCSurfaceComplete(t *testing.T) {
	expected := []string{"Send", "Forward", "ShareContact", "React", "Edit", "Unsend", "SetTyping", "MarkRead", "ListMessageRequests", "ListThemes", "FindTheme", "SetTheme", "GetCurrentNote", "CreateNote", "DeleteNote", "RecreateNote", "SetRestricted", "SetMessageBlocked", "ListThreads", "GetThread", "CreatePoll", "VotePoll", "ListPinnedMessages", "GetPollDetails", "SearchMessages", "MuteThread", "MuteThreadCalls", "SetApprovalMode", "SetArchived", "SetMessagePinned", "DeleteThread", "CreateDM", "SearchUsers", "GetContact", "SetAdmin", "SetThreadName", "SetThreadEmoji", "SetNickname"}
	actual := make([]string, 0, len(metav1.MessengerService_ServiceDesc.Methods))
	for _, method := range metav1.MessengerService_ServiceDesc.Methods {
		actual = append(actual, method.MethodName)
	}
	if !slices.Equal(actual, expected) {
		t.Fatalf("RPC surface mismatch:\nactual=%v\nexpected=%v", actual, expected)
	}
}
