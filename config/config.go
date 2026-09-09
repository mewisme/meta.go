package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
	"go.mewis.me/fbgo/internal/fsutil"
	"gopkg.in/yaml.v3"
)

const CurrentSchemaVersion = 1
const DefaultFileName = "config.toml"

var (
	ErrAmbiguousConfig   = errors.New("multiple default config files found")
	ErrUnsupportedFormat = errors.New("unsupported config format")
)

type Config struct {
	SchemaVersion  int           `json:"schema_version" yaml:"schema_version" toml:"schema_version"`
	DefaultProfile string        `json:"default_profile,omitempty" yaml:"default_profile,omitempty" toml:"default_profile,omitempty"`
	LogLevel       string        `json:"log_level,omitempty" yaml:"log_level,omitempty" toml:"log_level,omitempty"`
	Timeout        time.Duration `json:"-" yaml:"-" toml:"-"`
	TimeoutText    string        `json:"timeout,omitempty" yaml:"timeout,omitempty" toml:"timeout,omitempty"`
	E2EE           bool          `json:"e2ee" yaml:"e2ee" toml:"e2ee"`
	EventBuffer    int           `json:"event_buffer,omitempty" yaml:"event_buffer,omitempty" toml:"event_buffer,omitempty"`
}

func Default() Config {
	return Config{SchemaVersion: CurrentSchemaVersion, LogLevel: "info", Timeout: 30 * time.Second, TimeoutText: "30s", E2EE: true, EventBuffer: 100}
}

func (c *Config) Validate() error {
	if c.SchemaVersion == 0 {
		c.SchemaVersion = CurrentSchemaVersion
	}
	if c.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("unsupported config schema version %d", c.SchemaVersion)
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	switch c.LogLevel {
	case "error", "warn", "info", "debug", "trace":
	default:
		return fmt.Errorf("invalid log_level %q", c.LogLevel)
	}
	if c.TimeoutText == "" {
		c.TimeoutText = "30s"
	}
	duration, err := time.ParseDuration(c.TimeoutText)
	if err != nil || duration <= 0 {
		return fmt.Errorf("invalid timeout %q", c.TimeoutText)
	}
	c.Timeout = duration
	if c.EventBuffer == 0 {
		c.EventBuffer = 100
	}
	if c.EventBuffer < 1 {
		return errors.New("event_buffer must be positive")
	}
	return nil
}

func DefaultDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "fbgo"), nil
}

func DefaultPath(dir string) (string, error) {
	if dir == "" {
		var err error
		dir, err = DefaultDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(dir, DefaultFileName), nil
}

func InitDefault(dir string) (string, error) {
	path, err := DefaultPath(dir)
	if err != nil {
		return "", err
	}
	config := Default()
	data, err := encode(".toml", config)
	if err != nil {
		return "", err
	}
	if err := fsutil.ExclusiveWriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func Discover(dir string) (string, error) {
	if dir == "" {
		var err error
		dir, err = DefaultDir()
		if err != nil {
			return "", err
		}
	}
	var found []string
	for _, name := range []string{"config.toml", "config.yaml", "config.yml", "config.json"} {
		path := filepath.Join(dir, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			found = append(found, path)
		}
	}
	switch len(found) {
	case 0:
		return "", os.ErrNotExist
	case 1:
		return found[0], nil
	default:
		return "", fmt.Errorf("%w: %s", ErrAmbiguousConfig, strings.Join(found, ", "))
	}
}

func LoadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	config := Default()
	if err := decode(filepath.Ext(path), data, &config); err != nil {
		return Config{}, err
	}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func SaveFile(path string, config Config) error {
	if err := config.Validate(); err != nil {
		return err
	}
	data, err := encode(filepath.Ext(path), config)
	if err != nil {
		return err
	}
	return fsutil.AtomicWriteFile(path, data, 0o600)
}

func decode(ext string, data []byte, target *Config) error {
	switch strings.ToLower(ext) {
	case ".json":
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		return decoder.Decode(target)
	case ".yaml", ".yml":
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		return decoder.Decode(target)
	case ".toml":
		return toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields().Decode(target)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedFormat, ext)
	}
}

func encode(ext string, value Config) ([]byte, error) {
	switch strings.ToLower(ext) {
	case ".json":
		data, err := json.MarshalIndent(value, "", "  ")
		return append(data, '\n'), err
	case ".yaml", ".yml":
		return yaml.Marshal(value)
	case ".toml":
		return toml.Marshal(value)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, ext)
	}
}
