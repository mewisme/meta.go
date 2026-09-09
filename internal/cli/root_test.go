package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"go.mewis.me/fbgo"
)

func TestWriteVersionText(t *testing.T) {
	var out bytes.Buffer
	info := fbgo.VersionInfo{Version: "v1.2.3", Commit: "abc", GoVersion: "go1.test"}
	if err := writeVersion(&out, false, info); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "fbgo v1.2.3 (abc) go1.test\n" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestWriteVersionJSON(t *testing.T) {
	var out bytes.Buffer
	info := fbgo.VersionInfo{Version: "v1.2.3", Commit: "abc", GoVersion: "go1.test"}
	if err := writeVersion(&out, true, info); err != nil {
		t.Fatal(err)
	}
	var decoded fbgo.VersionInfo
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Version != info.Version || decoded.Commit != info.Commit || !strings.HasSuffix(out.String(), "\n") {
		t.Fatalf("unexpected JSON output: %q", out.String())
	}
}
