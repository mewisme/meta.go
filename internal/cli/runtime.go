package cli

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/zalando/go-keyring"

	"go.mewis.me/meta.go"
	"go.mewis.me/meta.go/auth"
	"go.mewis.me/meta.go/config"
	fberrors "go.mewis.me/meta.go/errors"
	"go.mewis.me/meta.go/storage"
)

const (
	ExitOK       = 0
	ExitUsage    = 2
	ExitAuth     = 3
	ExitNetwork  = 4
	ExitProtocol = 5
	ExitE2EE     = 6
	ExitRemote   = 7
	ExitInternal = 10
)

type outputEnvelope struct {
	OK   bool `json:"ok"`
	Data any  `json:"data,omitempty"`
}

type appRuntime struct {
	configPath string
	config     config.Config
	profiles   *storage.FileProfileStore
	secrets    storage.SecretStore
	manager    auth.ProfileManager
}

func ExitCode(err error) int { return classifyExitCode(err) }

func writeValue(out io.Writer, asJSON bool, selector string, value any, text string) error {
	if asJSON {
		return writeJSONOutput(out, outputEnvelope{OK: true, Data: value}, selector)
	}
	_, err := fmt.Fprintln(out, text)
	return err
}

func resolveConfigPath(opts *options, allowMissing bool) (string, error) {
	if opts != nil && strings.TrimSpace(opts.config) != "" {
		return filepath.Clean(opts.config), nil
	}
	if env := strings.TrimSpace(os.Getenv("META_CONFIG")); env != "" {
		return filepath.Clean(env), nil
	}
	path, err := config.Discover("")
	if err == nil {
		return path, nil
	}
	if allowMissing && errors.Is(err, os.ErrNotExist) {
		return config.DefaultPath("")
	}
	return "", err
}

func loadRuntime(ctx context.Context, opts *options) (*appRuntime, error) {
	path, err := resolveConfigPath(opts, false)
	if err != nil {
		return nil, err
	}
	cfg, err := config.LoadFile(path)
	if err != nil {
		return nil, err
	}
	if err := applyGlobalLogLevel(opts.logLevel, cfg.LogLevel); err != nil {
		return nil, err
	}
	if opts.timeout != "" {
		d, err := time.ParseDuration(opts.timeout)
		if err != nil || d <= 0 {
			return nil, fmt.Errorf("%w: invalid timeout", fberrors.ErrInvalidInput)
		}
		cfg.Timeout = d
		cfg.TimeoutText = opts.timeout
	}
	dir := filepath.Dir(path)
	profiles := storage.NewFileProfileStore(filepath.Join(dir, "profiles.json"))
	secretBackend := storage.NewFileSecretStore(filepath.Join(dir, "secrets.json"))
	key, err := loadMasterKey(path)
	if err != nil {
		return nil, err
	}
	secrets, err := storage.NewEncryptedSecretStore(secretBackend, key)
	if err != nil {
		return nil, err
	}
	manager := auth.ProfileManager{Profiles: profiles, Secrets: secrets}
	_ = ctx
	return &appRuntime{configPath: path, config: cfg, profiles: profiles, secrets: secrets, manager: manager}, nil
}

func loadMasterKey(configPath string) ([]byte, error) {
	if raw := strings.TrimSpace(os.Getenv("META_MASTER_KEY")); raw != "" {
		return decodeMasterKey(raw)
	}
	canonical, err := canonicalConfigPath(configPath)
	if err != nil {
		return nil, err
	}
	identity := sha256.Sum256([]byte(canonical))
	user := hex.EncodeToString(identity[:16])
	raw, err := keyring.Get("meta", user)
	if err == nil {
		return decodeMasterKey(raw)
	}
	if !errors.Is(err, keyring.ErrNotFound) {
		return nil, fmt.Errorf("OS keyring unavailable: %w; set META_MASTER_KEY for headless use", err)
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := keyring.Set("meta", user, base64.RawStdEncoding.EncodeToString(key)); err != nil {
		return nil, fmt.Errorf("store meta master key: %w", err)
	}
	return key, nil
}

func canonicalConfigPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve config path: %w", err)
	}
	abs = filepath.Clean(abs)
	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return resolved, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return abs, nil
	}
	return "", fmt.Errorf("resolve config symlinks: %w", err)
}

func decodeMasterKey(raw string) ([]byte, error) {
	for _, decode := range []func(string) ([]byte, error){hex.DecodeString, base64.RawStdEncoding.DecodeString, base64.StdEncoding.DecodeString} {
		key, err := decode(raw)
		if err == nil && len(key) == 32 {
			return key, nil
		}
	}
	return nil, errors.New("META_MASTER_KEY must encode exactly 32 bytes as hex or base64")
}

func (r *appRuntime) profileName(opts *options) string {
	if opts != nil && strings.TrimSpace(opts.profile) != "" {
		return strings.TrimSpace(opts.profile)
	}
	if env := strings.TrimSpace(os.Getenv("META_PROFILE")); env != "" {
		return env
	}
	return strings.TrimSpace(r.config.DefaultProfile)
}

func (r *appRuntime) client(ctx context.Context, opts *options) (*meta.Client, error) {
	name := r.profileName(opts)
	if name == "" {
		return nil, fmt.Errorf("%w: no profile selected", fberrors.ErrInvalidInput)
	}
	profile, err := r.profiles.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	logger, err := cliLogger(opts.logLevel, r.config.LogLevel)
	if err != nil {
		return nil, err
	}
	client, err := meta.NewClient(meta.WithProfile(profile), meta.WithSecretStore(r.secrets), meta.WithTimeout(r.config.Timeout), meta.WithE2EE(r.config.E2EE), meta.WithEventBuffer(r.config.EventBuffer), meta.WithLogger(logger))
	if err != nil {
		return nil, err
	}
	if err := client.Connect(ctx); err != nil {
		client.Close()
		return nil, err
	}
	return client, nil
}

func cliLogger(flagLevel, configLevel string) (*slog.Logger, error) {
	level, zeroLevel, err := resolveLogLevels(flagLevel, configLevel)
	if err != nil {
		return nil, err
	}
	zerolog.SetGlobalLevel(zeroLevel)
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})), nil
}

func applyGlobalLogLevel(flagLevel, configLevel string) error {
	_, zeroLevel, err := resolveLogLevels(flagLevel, configLevel)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(zeroLevel)
	return nil
}

func resolveLogLevels(flagLevel, configLevel string) (slog.Level, zerolog.Level, error) {
	name := strings.ToLower(strings.TrimSpace(flagLevel))
	if name == "" {
		name = strings.ToLower(strings.TrimSpace(configLevel))
	}
	if name == "" {
		name = "info"
	}
	switch name {
	case "error":
		return slog.LevelError, zerolog.ErrorLevel, nil
	case "warn":
		return slog.LevelWarn, zerolog.WarnLevel, nil
	case "info":
		return slog.LevelInfo, zerolog.InfoLevel, nil
	case "debug":
		return slog.LevelDebug, zerolog.DebugLevel, nil
	case "trace":
		return slog.Level(-8), zerolog.TraceLevel, nil
	default:
		return 0, zerolog.NoLevel, fmt.Errorf("%w: invalid log level %q", fberrors.ErrInvalidInput, name)
	}
}
