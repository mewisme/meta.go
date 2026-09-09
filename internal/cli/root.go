package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.mewis.me/fbgo"
	fberrors "go.mewis.me/fbgo/errors"
)

type options struct {
	profile  string
	config   string
	timeout  string
	logLevel string
	json     bool
	noColor  bool
}

func New() *cobra.Command {
	opts := new(options)
	root := &cobra.Command{
		Use:               "fbgo",
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
			return writeVersion(cmd.OutOrStdout(), opts.json, fbgo.BuildVersion())
		},
	}
}

func writeVersion(out io.Writer, asJSON bool, info fbgo.VersionInfo) error {
	if asJSON {
		encoder := json.NewEncoder(out)
		encoder.SetEscapeHTML(false)
		return encoder.Encode(outputEnvelope{OK: true, Data: info})
	}
	_, err := fmt.Fprintf(out, "fbgo %s (%s) %s\n", info.Version, info.Commit, info.GoVersion)
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
		return applyGlobalLogLevel(opts.logLevel, "")
	}
	return nil
}
