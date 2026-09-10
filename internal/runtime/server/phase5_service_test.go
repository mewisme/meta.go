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
	"google.golang.org/protobuf/types/known/timestamppb"

	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
	"go.mewis.me/meta.go/internal/runtime/session"
	"go.mewis.me/meta.go/model"
)

type phase5E2EE struct {
	session.E2EE
	send     model.E2EESendRequest
	reaction model.E2EEReactionRequest
	edit     struct {
		chat    string
		message model.ID
		text    string
	}
	unsend struct {
		chat    string
		message model.ID
	}
	typing struct {
		chat   string
		typing bool
	}
	read model.E2EEReadRequest
}

func (f *phase5E2EE) SendE2EE(_ context.Context, req model.E2EESendRequest) (model.SendResult, error) {
	f.send = req
	return model.SendResult{MessageID: "e1", Timestamp: time.Unix(1_700_000_000, 0).UTC()}, nil
}
func (f *phase5E2EE) ReactE2EE(_ context.Context, req model.E2EEReactionRequest) error {
	f.reaction = req
	return nil
}
func (f *phase5E2EE) EditE2EE(_ context.Context, chat string, message model.ID, text string) error {
	f.edit.chat, f.edit.message, f.edit.text = chat, message, text
	return nil
}
func (f *phase5E2EE) UnsendE2EE(_ context.Context, chat string, message model.ID) error {
	f.unsend.chat, f.unsend.message = chat, message
	return nil
}
func (f *phase5E2EE) TypingE2EE(_ context.Context, chat string, typing bool) error {
	f.typing.chat, f.typing.typing = chat, typing
	return nil
}
func (f *phase5E2EE) ReadE2EE(_ context.Context, req model.E2EEReadRequest) error {
	f.read = req
	return nil
}

type phase5Facebook struct {
	session.Facebook
	archivedOwnership model.PostOwnership
	listingInput      model.MarketplaceListingInput
	professional      bool
}

func (f *phase5Facebook) User(context.Context, model.ID) (*model.FacebookUser, error) {
	return &model.FacebookUser{ID: "u1", Name: "Mew", Username: "mew", NonFriend: true}, nil
}
func (f *phase5Facebook) Search(context.Context, string, int) ([]model.SearchResult, error) {
	return []model.SearchResult{{ID: "u1", Name: "Mew", URL: "https://example.invalid/u1"}}, nil
}
func (f *phase5Facebook) Notifications(context.Context, int) ([]model.Notification, error) {
	return []model.Notification{{ID: "n1", Text: "hello", Timestamp: time.Unix(1_700_000_000, 0).UTC()}}, nil
}
func (f *phase5Facebook) CreatePost(context.Context, string) (*model.Post, error) {
	return &model.Post{ID: "p1", URL: "https://example.invalid/p1"}, nil
}
func (f *phase5Facebook) ArchivePost(_ context.Context, _ model.ID, ownership model.PostOwnership) error {
	f.archivedOwnership = ownership
	return nil
}
func (f *phase5Facebook) CreateMarketplaceListing(_ context.Context, input model.MarketplaceListingInput) (*model.MarketplaceListing, error) {
	f.listingInput = input
	return &model.MarketplaceListing{ID: "l1", Title: input.Title, Price: input.Price, Currency: input.Currency, Seller: model.FacebookUser{ID: "u1"}, Location: input.Location}, nil
}
func (f *phase5Facebook) MarketplaceListing(context.Context, model.ID) (*model.MarketplaceListing, error) {
	return &model.MarketplaceListing{ID: "l1", Title: "item", Seller: model.FacebookUser{ID: "u1"}}, nil
}
func (f *phase5Facebook) SetProfessionalMode(_ context.Context, enabled bool) error {
	f.professional = enabled
	return nil
}

func TestE2EEAndFacebookRPCContract(t *testing.T) {
	e2ee := &phase5E2EE{}
	facebook := &phase5Facebook{}
	fake := &runtimeFakeClient{e2ee: e2ee, facebook: facebook}
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

	e2eeClient := metav1.NewE2EEServiceClient(conn)
	sent, err := e2eeClient.SendText(ctx, &metav1.E2EEServiceSendTextRequest{SessionId: created.ID(), ChatJid: "chat", FacebookUserId: "u1", Text: "hello", ReplyTo: "m0", ReplySenderJid: "sender"})
	if err != nil || sent.GetMessageId() != "e1" || e2ee.send.ChatJID != "chat" || e2ee.send.FacebookUserID != "u1" || e2ee.send.ReplyTo != "m0" {
		t.Fatalf("e2ee send mismatch: response=%#v captured=%#v err=%v", sent, e2ee.send, err)
	}
	if _, err := e2eeClient.React(ctx, &metav1.E2EEServiceReactRequest{SessionId: created.ID(), ChatJid: "chat", MessageId: "e1", SenderJid: "sender", Reaction: "+1"}); err != nil {
		t.Fatal(err)
	}
	if e2ee.reaction.MessageID != "e1" || e2ee.reaction.Reaction != "+1" {
		t.Fatalf("e2ee reaction mismatch: %#v", e2ee.reaction)
	}
	if _, err := e2eeClient.Edit(ctx, &metav1.E2EEServiceEditRequest{SessionId: created.ID(), ChatJid: "chat", MessageId: "e1", Text: "edited"}); err != nil {
		t.Fatal(err)
	}
	if e2ee.edit.message != "e1" || e2ee.edit.text != "edited" {
		t.Fatalf("e2ee edit mismatch: %#v", e2ee.edit)
	}
	if _, err := e2eeClient.Unsend(ctx, &metav1.E2EEServiceUnsendRequest{SessionId: created.ID(), ChatJid: "chat", MessageId: "e1"}); err != nil {
		t.Fatal(err)
	}
	if e2ee.unsend.message != "e1" {
		t.Fatalf("e2ee unsend mismatch: %#v", e2ee.unsend)
	}
	if _, err := e2eeClient.SetTyping(ctx, &metav1.E2EEServiceSetTypingRequest{SessionId: created.ID(), ChatJid: "chat", Typing: true}); err != nil {
		t.Fatal(err)
	}
	if !e2ee.typing.typing || e2ee.typing.chat != "chat" {
		t.Fatalf("e2ee typing mismatch: %#v", e2ee.typing)
	}
	readAt := time.Unix(1_700_000_100, 0).UTC()
	if _, err := e2eeClient.MarkRead(ctx, &metav1.E2EEServiceMarkReadRequest{SessionId: created.ID(), ChatJid: "chat", SenderJid: "sender", MessageIds: []string{"e1", "e2"}, Timestamp: timestamppb.New(readAt)}); err != nil {
		t.Fatal(err)
	}
	if e2ee.read.ChatJID != "chat" || !slices.Equal(e2ee.read.MessageIDs, []model.ID{"e1", "e2"}) || !e2ee.read.Timestamp.Equal(readAt) {
		t.Fatalf("e2ee read mismatch: %#v", e2ee.read)
	}

	facebookClient := metav1.NewFacebookServiceClient(conn)
	user, err := facebookClient.GetUser(ctx, &metav1.FacebookServiceGetUserRequest{SessionId: created.ID(), UserId: "u1"})
	if err != nil || user.GetUser().GetId() != "u1" || user.GetUser().GetUsername() != "mew" || !user.GetUser().GetNonFriend() {
		t.Fatalf("facebook user mismatch: %#v err=%v", user, err)
	}
	search, err := facebookClient.Search(ctx, &metav1.FacebookServiceSearchRequest{SessionId: created.ID(), Query: "mew", Limit: 5})
	if err != nil || len(search.GetResults()) != 1 || search.GetResults()[0].GetId() != "u1" {
		t.Fatalf("facebook search mismatch: %#v err=%v", search, err)
	}
	notifications, err := facebookClient.ListNotifications(ctx, &metav1.FacebookServiceListNotificationsRequest{SessionId: created.ID(), Limit: 5})
	if err != nil || len(notifications.GetNotifications()) != 1 || notifications.GetNotifications()[0].GetId() != "n1" {
		t.Fatalf("facebook notification mismatch: %#v err=%v", notifications, err)
	}
	post, err := facebookClient.CreatePost(ctx, &metav1.FacebookServiceCreatePostRequest{SessionId: created.ID(), Text: "hello"})
	if err != nil || post.GetPost().GetId() != "p1" {
		t.Fatalf("facebook post mismatch: %#v err=%v", post, err)
	}
	if _, err := facebookClient.ArchivePost(ctx, &metav1.FacebookServiceArchivePostRequest{SessionId: created.ID(), PostId: "p1", Ownership: metav1.FacebookPostOwnership_FACEBOOK_POST_OWNERSHIP_SHARED}); err != nil {
		t.Fatal(err)
	}
	if facebook.archivedOwnership != model.PostShared {
		t.Fatalf("post ownership mismatch: %q", facebook.archivedOwnership)
	}
	listing, err := facebookClient.CreateMarketplaceListing(ctx, &metav1.FacebookServiceCreateMarketplaceListingRequest{SessionId: created.ID(), Listing: &metav1.MarketplaceListingInput{Title: "item", Price: "1", Currency: "USD", Category: "Tools", PhotoIds: []string{"photo1"}, Location: &metav1.MarketplaceLocation{Latitude: 10, Longitude: 20, Name: "place"}}})
	if err != nil || listing.GetListing().GetId() != "l1" || !slices.Equal(facebook.listingInput.PhotoIDs, []model.ID{"photo1"}) || facebook.listingInput.Location.Name != "place" {
		t.Fatalf("marketplace mismatch: response=%#v input=%#v err=%v", listing, facebook.listingInput, err)
	}
	gotListing, err := facebookClient.GetMarketplaceListing(ctx, &metav1.FacebookServiceGetMarketplaceListingRequest{SessionId: created.ID(), ListingId: "l1"})
	if err != nil || gotListing.GetListing().GetSeller().GetId() != "u1" {
		t.Fatalf("marketplace get mismatch: %#v err=%v", gotListing, err)
	}
	if _, err := facebookClient.SetProfessionalMode(ctx, &metav1.FacebookServiceSetProfessionalModeRequest{SessionId: created.ID(), Enabled: true}); err != nil || !facebook.professional {
		t.Fatalf("professional mode mismatch: enabled=%v err=%v", facebook.professional, err)
	}
}

func TestPhase5RPCSurfacesComplete(t *testing.T) {
	e2eeExpected := []string{"SendText", "React", "Edit", "Unsend", "SetTyping", "MarkRead"}
	e2eeActual := make([]string, 0, len(metav1.E2EEService_ServiceDesc.Methods))
	for _, method := range metav1.E2EEService_ServiceDesc.Methods {
		e2eeActual = append(e2eeActual, method.MethodName)
	}
	if !slices.Equal(e2eeActual, e2eeExpected) {
		t.Fatalf("E2EE RPC surface mismatch: actual=%v expected=%v", e2eeActual, e2eeExpected)
	}
	facebookExpected := []string{"GetUser", "Search", "ListNotifications", "SetBio", "CreateAdditionalProfile", "Unfriend", "SetBlocked", "CreatePost", "ArchivePost", "DeletePost", "CreateMarketplaceListing", "GetMarketplaceListing", "SetProfessionalMode"}
	facebookActual := make([]string, 0, len(metav1.FacebookService_ServiceDesc.Methods))
	for _, method := range metav1.FacebookService_ServiceDesc.Methods {
		facebookActual = append(facebookActual, method.MethodName)
	}
	if !slices.Equal(facebookActual, facebookExpected) {
		t.Fatalf("Facebook RPC surface mismatch: actual=%v expected=%v", facebookActual, facebookExpected)
	}
}
