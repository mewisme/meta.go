package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"go.mewis.me/meta.go/config"
	fberrors "go.mewis.me/meta.go/errors"
)

func newProfileCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "profile", Short: "Manage account profiles"}
	cmd.AddCommand(newProfileCreateCommand(opts), newProfileListCommand(opts), newProfileShowCommand(opts), newProfileUseCommand(opts), newProfileRenameCommand(opts), newProfileRemoveCommand(opts))
	return cmd
}

func newProfileCreateCommand(opts *options) *cobra.Command {
	displayName := ""
	cmd := &cobra.Command{Use: "create <name>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		profile, err := r.manager.Create(cmd.Context(), args[0], displayName)
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, profile, profile.Name)
	}}
	cmd.Flags().StringVar(&displayName, "display-name", "", "profile display name")
	return cmd
}

func newProfileListCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		profiles, err := r.profiles.List(cmd.Context())
		if err != nil {
			return err
		}
		if opts.json {
			return writeValue(cmd.OutOrStdout(), true, opts.jqo, profiles, "")
		}
		for _, profile := range profiles {
			marker := " "
			if profile.Name == r.config.DefaultProfile {
				marker = "*"
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", marker, profile.Name); err != nil {
				return err
			}
		}
		return nil
	}}
}

func newProfileShowCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "show [name]", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		name := r.profileName(opts)
		if len(args) == 1 {
			name = strings.TrimSpace(args[0])
		}
		if name == "" {
			return fmt.Errorf("%w: profile name is required", fberrors.ErrInvalidInput)
		}
		profile, err := r.profiles.Get(cmd.Context(), name)
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, profile, profile.Name)
	}}
}

func newProfileUseCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "use <name>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		name := strings.TrimSpace(args[0])
		if _, err := r.profiles.Get(cmd.Context(), name); err != nil {
			return err
		}
		r.config.DefaultProfile = name
		if err := config.SaveFile(r.configPath, r.config); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]string{"default_profile": name}, name)
	}}
}

func newProfileRenameCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "rename <old> <new>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		profile, err := r.manager.Rename(cmd.Context(), args[0], args[1])
		if err != nil {
			return err
		}
		if r.config.DefaultProfile == args[0] {
			r.config.DefaultProfile = profile.Name
			if err := config.SaveFile(r.configPath, r.config); err != nil {
				return err
			}
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, profile, profile.Name)
	}}
}

func newProfileRemoveCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "remove <name>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		name := strings.TrimSpace(args[0])
		if err := r.manager.Remove(cmd.Context(), name); err != nil {
			return err
		}
		if r.config.DefaultProfile == name {
			r.config.DefaultProfile = ""
			if err := config.SaveFile(r.configPath, r.config); err != nil {
				return err
			}
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]string{"removed": name}, name)
	}}
}
