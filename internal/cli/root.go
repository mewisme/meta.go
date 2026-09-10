package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.mewis.me/meta.go"
	fberrors "go.mewis.me/meta.go/errors"
)

type options struct {
	profile  string
	config   string
	timeout  string
	logLevel string
	jqi      string
	jqo      string
	json     bool
	noColor  bool
}

func New() *cobra.Command {
	opts := new(options)
	root := &cobra.Command{
		Use:               "meta",
		Short:             "Facebook Messenger client and automation toolkit",
		SilenceUsage:      true,
		SilenceErrors:     true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return validateGlobalOptions(opts) },
	}
	flags := root.PersistentFlags()
	flags.StringVar(&opts.profile, "profile", "", "profile name")
	flags.StringVar(&opts.config, "config", "", "config file path")
	flags.StringVar(&opts.timeout, "timeout", "", "operation timeout")
	flags.StringVar(&opts.logLevel, "log-level", "", "log level: error, warn, info, debug, trace")
	flags.StringVar(&opts.jqi, "jqi", "", "filter JSON input before decoding")
	flags.StringVar(&opts.jqo, "jqo", "", "filter JSON output with jq; implies --json")
	flags.BoolVar(&opts.json, "json", false, "write machine-readable JSON")
	flags.BoolVar(&opts.noColor, "no-color", false, "disable color output")
	root.AddCommand(newVersionCommand(opts), newConfigCommand(opts), newProfileCommand(opts), newAuthCommand(opts), newCookiesCommand(opts), newDoctorCommand(opts), newMessengerCommand(opts), newThreadCommand(opts), newFacebookCommand(opts))
	return root
}

func Execute() error { return New().Execute() }

func newVersionCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print build version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return writeVersion(cmd.OutOrStdout(), opts.json, opts.jqo, meta.BuildVersion())
		},
	}
}

func writeVersion(out io.Writer, asJSON bool, selector string, info meta.VersionInfo) error {
	if asJSON {
		return writeJSONOutput(out, outputEnvelope{OK: true, Data: info}, selector)
	}
	_, err := fmt.Fprintf(out, "meta %s (%s) %s\n", info.Version, info.Commit, info.GoVersion)
	return err
}

func validateGlobalOptions(opts *options) error {
	if opts == nil {
		return nil
	}
	if value := strings.TrimSpace(opts.timeout); value != "" {
		duration, err := time.ParseDuration(value)
		if err != nil || duration <= 0 {
			return fmt.Errorf("%w: invalid timeout %q", fberrors.ErrInvalidInput, value)
		}
	}
	if strings.TrimSpace(opts.logLevel) != "" {
		if err := applyGlobalLogLevel(opts.logLevel, ""); err != nil {
			return err
		}
	}
	if strings.TrimSpace(opts.jqo) != "" {
		if _, err := compileJQ(opts.jqo); err != nil {
			return err
		}
		opts.json = true
	}
	return nil
}
