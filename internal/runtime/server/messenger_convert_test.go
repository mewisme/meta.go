package server

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
	"go.mewis.me/meta.go/model"
)

func TestMessengerSendRequestConversion(t *testing.T) {
	request := sendRequestFromProto(&metav1.SendRequest{ThreadId: "t1", Text: "hello", ReplyTo: &metav1.ReplyReference{MessageId: "m0", SenderId: "u0"}, Mentions: []*metav1.Mention{{UserId: "u1", Offset: 1, Length: 2}}, StickerId: "s1", Url: "https://example.invalid", Encryption: metav1.EncryptionPolicy_ENCRYPTION_POLICY_DISABLED})
	if request.ThreadID != "t1" || request.Text != "hello" || request.ReplyTo == nil || request.ReplyTo.MessageID != "m0" || len(request.Mentions) != 1 || request.Mentions[0].UserID != "u1" || request.StickerID != "s1" || request.URL != "https://example.invalid" || request.Encryption != model.EncryptionDisabled {
		t.Fatalf("unexpected request: %#v", request)
	}
}

func TestMessengerModelConversions(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	thread := threadToProto(model.Thread{ID: "t1", Name: "group", Participants: []model.User{{ID: "u1", Name: "Mew", MessengerBlockStatus: model.MessengerBlockMessageBlocked}}, Admins: []model.ID{"u1"}, Nicknames: map[model.ID]string{"u1": "Boss"}, LastActivity: now})
	if thread.GetId() != "t1" || len(thread.GetParticipants()) != 1 || thread.GetParticipants()[0].GetMessengerBlockStatus() != metav1.MessengerBlockStatus_MESSENGER_BLOCK_STATUS_MESSAGE_BLOCKED || thread.GetNicknames()["u1"] != "Boss" || thread.GetLastActivity().AsTime() != now {
		t.Fatalf("thread conversion lost data: %#v", thread)
	}
	poll := pollDetailsToProto(&model.PollDetails{ID: "p1", ThreadID: "t1", Title: "Q", Options: []model.PollOption{{ID: "o1", Text: "A", SortKeyCreationTimestamp: now}}, Votes: []model.PollVote{{OptionID: "o1", ContactID: "u1", Timestamp: now, VoteCount: 1}}})
	if poll.GetId() != "p1" || len(poll.GetOptions()) != 1 || poll.GetOptions()[0].GetId() != "o1" || len(poll.GetVotes()) != 1 || poll.GetVotes()[0].GetContactId() != "u1" {
		t.Fatalf("poll conversion lost data: %#v", poll)
	}
	search := messageSearchResultToProto(model.MessageSearchResult{MessageID: "m1", ThreadID: "t1", Text: "hello", Highlights: []model.MessageSearchHighlight{{Offset: 2, Length: 3}}})
	if search.GetMessageId() != "m1" || len(search.GetHighlights()) != 1 || search.GetHighlights()[0].GetOffset() != 2 {
		t.Fatalf("search conversion lost data: %#v", search)
	}
	theme := themeToProto(&model.Theme{ID: "th1", Name: "Theme", GradientColors: []string{"#1", "#2"}, NormalThemeID: "base"})
	if theme.GetId() != "th1" || len(theme.GetGradientColors()) != 2 || theme.GetNormalThemeId() != "base" {
		t.Fatalf("theme conversion lost data: %#v", theme)
	}
}

func TestMessengerDurationConversion(t *testing.T) {
	duration, err := durationFromProto(durationpb.New(5*time.Minute), false)
	if err != nil || duration != 5*time.Minute {
		t.Fatalf("finite duration: %v %v", duration, err)
	}
	duration, err = durationFromProto(nil, true)
	if err != nil || duration >= 0 {
		t.Fatalf("indefinite duration: %v %v", duration, err)
	}
	if _, err := durationFromProto(durationpb.New(time.Minute), true); err == nil {
		t.Fatal("expected ambiguous duration to fail")
	}
	if _, err := durationFromProto(durationpb.New(-time.Minute), false); err == nil {
		t.Fatal("expected negative finite duration to fail")
	}
}
