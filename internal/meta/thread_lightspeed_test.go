package meta

import (
	"testing"

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
