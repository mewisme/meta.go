package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.mewis.me/meta.go"
	"go.mewis.me/meta.go/auth"
	"go.mewis.me/meta.go/config"
	fberrors "go.mewis.me/meta.go/errors"
	"go.mewis.me/meta.go/model"
)

func TestWriteVersionText(t *testing.T) {
	var out bytes.Buffer
	info := meta.VersionInfo{Version: "v1.2.3", Commit: "abc", GoVersion: "go1.test"}
	if err := writeVersion(&out, false, "", info); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "meta v1.2.3 (abc) go1.test\n" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestWriteVersionJSON(t *testing.T) {
	var out bytes.Buffer
	info := meta.VersionInfo{Version: "v1.2.3", Commit: "abc", GoVersion: "go1.test"}
	if err := writeVersion(&out, true, "", info); err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		OK   bool             `json:"ok"`
		Data meta.VersionInfo `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if !decoded.OK || decoded.Data.Version != info.Version || decoded.Data.Commit != info.Commit || !strings.HasSuffix(out.String(), "\n") {
		t.Fatalf("unexpected JSON output: %q", out.String())
	}
}

func TestJQOutputFiltersFinalEnvelopeAndImpliesJSON(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run([]string{"--jqo", ".data.version", "version"}, strings.NewReader(""), &out, &errOut)
	if code != ExitOK {
		t.Fatalf("unexpected exit code %d: %s", code, errOut.String())
	}
	if got := strings.TrimSpace(out.String()); got != `"dev"` {
		t.Fatalf("unexpected jqo output: %q", got)
	}
}

func TestJQOutputSupportsMultipleAndEmptyResults(t *testing.T) {
	var out bytes.Buffer
	value := outputEnvelope{OK: true, Data: map[string]any{"items": []int{1, 2}}}
	if err := writeJSONOutput(&out, value, ".data.items[]"); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "1\n2\n" {
		t.Fatalf("unexpected multi-result output: %q", got)
	}
	out.Reset()
	if err := writeJSONOutput(&out, value, ".data.items[] | select(. > 10)"); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("expected empty jq output, got %q", out.String())
	}
}

func TestInvalidJQOutputReturnsJSONUsageError(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run([]string{"--jqo", ".[", "version"}, strings.NewReader(""), &out, &errOut)
	if code != ExitUsage {
		t.Fatalf("expected usage exit code, got %d", code)
	}
	var payload errorOutputEnvelope
	if err := json.Unmarshal(errOut.Bytes(), &payload); err != nil {
		t.Fatalf("invalid error JSON: %q %v", errOut.String(), err)
	}
	if payload.OK || payload.Error.Code != ExitUsage {
		t.Fatalf("unexpected error envelope: %#v", payload)
	}
}

func TestCLIConfigProfileAuthWorkflow(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "custom.json")
	run := func(input string, args ...string) (string, error) {
		cmd := New()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetIn(strings.NewReader(input))
		cmd.SetArgs(args)
		err := cmd.Execute()
		return out.String(), err
	}
	if _, err := run("", "--config", configPath, "config", "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatal(err)
	}
	t.Setenv("META_CONFIG", configPath)
	t.Setenv("META_MASTER_KEY", strings.Repeat("01", 32))
	if _, err := run("", "profile", "create", "default"); err != nil {
		t.Fatal(err)
	}
	if _, err := run("", "profile", "use", "default"); err != nil {
		t.Fatal(err)
	}
	if _, err := run("c_user=123; xs=secret-xs\n", "auth", "import", "-"); err != nil {
		t.Fatal(err)
	}
	status, err := run("", "--json", "auth", "status")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(status, `"authenticated":true`) || strings.Contains(status, "secret-xs") {
		t.Fatalf("unexpected auth status: %s", status)
	}
	secrets, err := os.ReadFile(filepath.Join(dir, "secrets.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(secrets, []byte("secret-xs")) || bytes.Contains(secrets, []byte("c_user=123")) {
		t.Fatal("plaintext cookies leaked into secrets.json")
	}
	doctor, err := run("", "doctor")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doctor, "version") || !strings.Contains(doctor, "secret_backend") || !strings.Contains(doctor, "cookies") || !strings.Contains(doctor, "c_user,xs") || strings.Contains(doctor, "secret-xs") {
		t.Fatalf("unexpected doctor output: %s", doctor)
	}
}

func TestRunJSONErrorEnvelopeAndExitCode(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run([]string{"--json", "config", "set"}, strings.NewReader(""), &out, &errOut)
	if code != ExitUsage {
		t.Fatalf("expected usage exit code, got %d", code)
	}
	var payload struct {
		OK    bool `json:"ok"`
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(errOut.Bytes(), &payload); err != nil {
		t.Fatalf("invalid error JSON: %q %v", errOut.String(), err)
	}
	if payload.OK || payload.Error.Code != ExitUsage || payload.Error.Message == "" {
		t.Fatalf("unexpected error envelope: %#v", payload)
	}
}

func TestGlobalOptionValidation(t *testing.T) {
	for _, args := range [][]string{{"--timeout", "bad", "version"}, {"--timeout", "0s", "version"}, {"--log-level", "verbose", "version"}} {
		var out, errOut bytes.Buffer
		if code := Run(args, strings.NewReader(""), &out, &errOut); code != ExitUsage {
			t.Fatalf("args %v: expected usage code, got %d", args, code)
		}
	}
}

func TestProfileResolutionPrecedence(t *testing.T) {
	r := &appRuntime{config: config.Config{DefaultProfile: "configured"}}
	t.Setenv("META_PROFILE", "environment")
	if got := r.profileName(&options{}); got != "environment" {
		t.Fatalf("env profile not selected: %q", got)
	}
	if got := r.profileName(&options{profile: "flag"}); got != "flag" {
		t.Fatalf("flag profile not selected: %q", got)
	}
}

func TestExitCodeContract(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{fberrors.ErrInvalidInput, ExitUsage}, {fberrors.ErrUnauthorized, ExitAuth}, {context.DeadlineExceeded, ExitNetwork},
		{fberrors.ErrProtocolChanged, ExitProtocol}, {fberrors.ErrE2EENotReady, ExitE2EE}, {fberrors.ErrPermissionDenied, ExitRemote}, {fberrors.ErrRateLimited, ExitRemote}, {auth.ErrTwoFactorRequired, ExitAuth}, {errors.New("internal"), ExitInternal},
	}
	for _, test := range tests {
		if got := ExitCode(test.err); got != test.want {
			t.Fatalf("%v: got %d want %d", test.err, got, test.want)
		}
	}
}

func TestConfigInitExistingExplicitPathReturnsUsage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := config.InitFile(path); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	code := Run([]string{"--config", path, "config", "init"}, strings.NewReader(""), &out, &errOut)
	if code != ExitUsage {
		t.Fatalf("expected usage for existing config, got %d", code)
	}
}

func TestConfigInitRefusesDifferentSecondDefaultFormat(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	dir := filepath.Join(root, "meta")
	if _, err := config.InitDefault(dir); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	code := Run([]string{"config", "init", "--format", "json"}, strings.NewReader(""), &out, &errOut)
	if code != ExitUsage {
		t.Fatalf("expected usage for second default config, got %d (%s)", code, errOut.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "config.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unexpected second config file: %v", err)
	}
}

func TestMarketplaceInputJSONContract(t *testing.T) {
	var input model.MarketplaceListingInput
	data := []byte(`{"title":"Item","price":"10","currency":"USD","category":"Tools","photoIds":["1"],"location":{"latitude":10.1,"longitude":106.1}}`)
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatal(err)
	}
	if input.Title != "Item" || len(input.PhotoIDs) != 1 || input.Location.Latitude != 10.1 {
		t.Fatalf("unexpected marketplace input: %#v", input)
	}
}
