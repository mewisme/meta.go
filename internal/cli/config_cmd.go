package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"go.mewis.me/fbgo/config"
	fberrors "go.mewis.me/fbgo/errors"
)

func newConfigCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Manage fbgo configuration"}
	cmd.AddCommand(newConfigInitCommand(opts), newConfigShowCommand(opts), newConfigPathCommand(opts), newConfigSetCommand(opts))
	return cmd
}

func newConfigInitCommand(opts *options) *cobra.Command {
	format := "toml"
	cmd := &cobra.Command{Use: "init", Short: "Create a new configuration", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		path, err := resolveConfigPath(opts, true)
		if err != nil {
			return err
		}
		explicit := strings.TrimSpace(opts.config) != "" || strings.TrimSpace(os.Getenv("FBGO_CONFIG")) != ""
		created := path
		if explicit {
			ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
			if ext == "" {
				return fmt.Errorf("%w: explicit config path needs .toml, .json, .yaml or .yml extension", fberrors.ErrInvalidInput)
			}
			if cmd.Flags().Changed("format") && format != ext && !(format == "yaml" && ext == "yml") {
				return fmt.Errorf("%w: --format does not match explicit config path extension", fberrors.ErrInvalidInput)
			}
			if err := config.InitFile(path); err != nil {
				return err
			}
		} else {
			if _, statErr := os.Stat(path); statErr == nil {
				return os.ErrExist
			} else if !errors.Is(statErr, os.ErrNotExist) {
				return statErr
			}
			created, err = config.Init(filepath.Dir(path), format)
			if err != nil {
				return err
			}
		}
		return writeValue(cmd.OutOrStdout(), opts.json, map[string]string{"path": created}, created)
	}}
	cmd.Flags().StringVar(&format, "format", "toml", "config format: toml, json, yaml")
	return cmd
}

func newConfigShowCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "show", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		path, err := resolveConfigPath(opts, false)
		if err != nil {
			return err
		}
		cfg, err := config.LoadFile(path)
		if err != nil {
			return err
		}
		if opts.json {
			return writeValue(cmd.OutOrStdout(), true, cfg, "")
		}
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return err
	}}
}

func newConfigPathCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "path", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		path, err := resolveConfigPath(opts, true)
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, map[string]string{"path": path}, path)
	}}
}

func newConfigSetCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "set <key> <value>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		path, err := resolveConfigPath(opts, false)
		if err != nil {
			return err
		}
		cfg, err := config.LoadFile(path)
		if err != nil {
			return err
		}
		key, value := strings.ToLower(args[0]), args[1]
		switch key {
		case "default_profile":
			cfg.DefaultProfile = value
		case "log_level":
			cfg.LogLevel = value
		case "timeout":
			cfg.TimeoutText = value
		case "e2ee":
			v, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("%w: invalid bool", fberrors.ErrInvalidInput)
			}
			cfg.E2EE = v
		case "event_buffer":
			v, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("%w: invalid integer", fberrors.ErrInvalidInput)
			}
			cfg.EventBuffer = v
		default:
			return fmt.Errorf("%w: unknown config key %q", fberrors.ErrInvalidInput, key)
		}
		if err := config.SaveFile(path, cfg); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, map[string]any{"key": key, "value": value}, key+" = "+value)
	}}
}
