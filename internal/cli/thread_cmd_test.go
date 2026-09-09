package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestThreadParityCommandsExist(t *testing.T) {
	root := New()
	for _, path := range [][]string{{"thread", "poll", "create"}, {"thread", "poll", "vote"}, {"thread", "mute"}, {"thread", "photo"}, {"thread", "delete"}, {"thread", "create-dm"}, {"thread", "search"}, {"thread", "contact"}} {
		if _, _, err := root.Find(path); err != nil {
			t.Fatalf("missing command %v: %v", path, err)
		}
	}
	create, _, err := root.Find([]string{"thread", "poll", "create"})
	if err != nil || create.Flags().Lookup("option") == nil {
		t.Fatal("thread poll create missing repeatable --option")
	}
}

func TestThreadMuteInvalidDurationIsUsageError(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run([]string{"thread", "mute", "1", "bad"}, strings.NewReader(""), &out, &errOut)
	if code != ExitUsage {
		t.Fatalf("expected usage exit code, got %d: %s", code, errOut.String())
	}
}
