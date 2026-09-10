package meta

import (
	"testing"
	"time"

	"go.mewis.me/meta-extra/pkg/messagix"
	"go.mewis.me/meta-extra/pkg/messagix/table"

	"go.mewis.me/meta.go/model"
)

func TestParsePositiveID(t *testing.T) {
	if got, err := parsePositiveID("123", "thread"); err != nil || got != 123 {
		t.Fatalf("unexpected parsed ID: %d %v", got, err)
	}
	for _, value := range []model.ID{"", "0", "-1", "abc"} {
		if _, err := parsePositiveID(value, "thread"); err == nil {
			t.Fatalf("expected invalid ID %q", value)
		}
	}
}

func TestMessengerGenderNormalization(t *testing.T) {
	for gender, want := range map[table.Gender]string{table.FEMALE_SINGULAR: "female", table.MALE_SINGULAR: "male", table.NEUTER_SINGULAR: "neutral", table.UNKNOWN_SINGULAR: ""} {
		if got := messengerGender(gender); got != want {
			t.Fatalf("gender %d: got %q want %q", gender, got, want)
		}
	}
}

func TestNormalizePollDetails(t *testing.T) {
	details := normalizePollDetails(&messagix.PollDetails{ID: 2, ThreadKey: 1, Title: "Question", LastUpdateMessageID: "mid.1", LastUpdateMessageTimestampMS: 1700000000123, Options: []messagix.PollOption{{ID: 3, Text: "A", SortKeyCreationTimestamp: 1700000000000}}, Votes: []messagix.PollVote{{OptionID: 3, ContactID: 4, TimestampMS: 1700000000100, ThreadKey: 1, MessageID: "mid.2"}}})
	if details == nil || details.ID != "2" || details.ThreadID != "1" || details.LastUpdateMessageID != "mid.1" || len(details.Options) != 1 || len(details.Votes) != 1 {
		t.Fatalf("unexpected poll details: %#v", details)
	}
	if details.LastUpdateMessageTimestamp.UnixMilli() != 1700000000123 || details.Options[0].SortKeyCreationTimestamp.UnixMilli() != 1700000000000 || details.Votes[0].Timestamp.UnixMilli() != 1700000000100 {
		t.Fatalf("unexpected normalized timestamps: %#v", details)
	}
}

func TestNormalizeMessageSearchPage(t *testing.T) {
	cursor := "next"
	page := normalizeMessageSearchPage(&messagix.MessageSearchPage{ResultCount: 1, HasNextPage: true, NextCursor: &cursor, Results: []messagix.MessageSearchResult{{MessageID: "mid.1", ThreadKey: 1, ThreadType: table.GROUP_THREAD, GlobalIndex: 2, SenderName: "Mew", TimestampMS: 1700000000123, Text: "hello", Highlights: []messagix.MessageSearchHighlight{{Offset: 0, Length: 5}}}}})
	if page == nil || page.ResultCount != 1 || !page.HasNextPage || page.NextCursor == nil || *page.NextCursor != cursor || len(page.Results) != 1 {
		t.Fatalf("unexpected page: %#v", page)
	}
	result := page.Results[0]
	if result.MessageID != "mid.1" || result.ThreadID != "1" || result.ThreadType != "group" || result.Timestamp.UnixMilli() != 1700000000123 || len(result.Highlights) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestUnixMilliAndThreadTypeNormalization(t *testing.T) {
	if !unixMilli(0).IsZero() || !unixMilli(-1).IsZero() {
		t.Fatal("non-positive timestamp must normalize to zero time")
	}
	if got := unixMilli(1000); !got.Equal(time.Unix(1, 0)) {
		t.Fatalf("unexpected timestamp: %v", got)
	}
	if messengerThreadType(table.ONE_TO_ONE) != "oneToOne" || messengerThreadType(table.UNKNOWN_THREAD_TYPE) != "unknown" {
		t.Fatal("unexpected thread type normalization")
	}
}
