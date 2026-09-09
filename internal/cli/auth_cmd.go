package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"go.mewis.me/fbgo/auth"
	fberrors "go.mewis.me/fbgo/errors"
	"go.mewis.me/fbgo/storage"
	"golang.org/x/term"
)

type loginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	TOTP     string `json:"totp"`
}

func newAuthCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "auth", Short: "Manage authentication"}
	cmd.AddCommand(newAuthImportCommand(opts), newAuthLoginCommand(opts), newAuthStatusCommand(opts), newAuthRefreshCommand(opts), newAuthLogoutCommand(opts))
	return cmd
}

func newAuthImportCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "import [file|-]", Short: "Import cookies from a file or stdin", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		name := r.profileName(opts)
		if name == "" {
			return fmt.Errorf("%w: no profile selected", fberrors.ErrInvalidInput)
		}
		var data []byte
		if len(args) == 0 || args[0] == "-" {
			data, err = io.ReadAll(cmd.InOrStdin())
		} else {
			data, err = os.ReadFile(args[0])
		}
		if err != nil {
			return err
		}
		trimmed := strings.TrimSpace(string(data))
		var cookies auth.Cookies
		if strings.HasPrefix(trimmed, "[") {
			cookies, err = auth.ParseBrowserCookieJSON(data)
		} else {
			cookies, err = auth.ParseCookieString(trimmed)
		}
		if err != nil {
			return err
		}
		if err := r.manager.ImportCookies(cmd.Context(), name, cookies); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, map[string]any{"profile": name, "imported": true}, "cookies imported for "+name)
	}}
}

func newAuthLoginCommand(opts *options) *cobra.Command {
	inputPath := ""
	cmd := &cobra.Command{Use: "login", Short: "Authenticate with credentials", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		name := r.profileName(opts)
		if name == "" {
			return fmt.Errorf("%w: no profile selected", fberrors.ErrInvalidInput)
		}
		preset := loginInput{}
		if inputPath != "" {
			data, err := os.ReadFile(inputPath)
			if err != nil {
				return err
			}
			preset, err = decodeLoginInput(data)
			if err != nil {
				return err
			}
		}
		login := auth.NewCredentialLogin(zerolog.Nop())
		responses := map[string]string{}
		for attempts := 0; attempts < 12; attempts++ {
			challenge, cookies, err := login.Step(cmd.Context(), responses)
			if err != nil {
				return err
			}
			if cookies != nil {
				if err := r.manager.ImportCookies(cmd.Context(), name, cookies); err != nil {
					return err
				}
				return writeValue(cmd.OutOrStdout(), opts.json, map[string]any{"profile": name, "authenticated": true}, "authenticated as "+name)
			}
			if challenge == nil {
				return errors.New("login returned no challenge")
			}
			responses = make(map[string]string, len(challenge.Fields))
			for _, field := range challenge.Fields {
				value, err := resolveLoginField(cmd, field, preset)
				if err != nil {
					return err
				}
				responses[field.ID] = value
			}
		}
		return &fberrors.ProtocolError{Operation: "credential login challenge flow", Cause: fberrors.ErrProtocolChanged}
	}}
	cmd.Flags().StringVar(&inputPath, "input", "", "read credentials from JSON file instead of prompting")
	return cmd
}

func decodeLoginInput(data []byte) (loginInput, error) {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var input loginInput
	if err := decoder.Decode(&input); err != nil {
		return loginInput{}, fmt.Errorf("%w: invalid login input: %v", fberrors.ErrInvalidInput, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return loginInput{}, fmt.Errorf("%w: login input must contain exactly one JSON object", fberrors.ErrInvalidInput)
		}
		return loginInput{}, fmt.Errorf("%w: invalid login input: %v", fberrors.ErrInvalidInput, err)
	}
	return input, nil
}

func resolveLoginField(cmd *cobra.Command, field auth.LoginField, preset loginInput) (string, error) {
	label := strings.ToLower(field.ID + " " + field.Name + " " + field.Description)
	switch {
	case strings.Contains(label, "email"), strings.Contains(label, "username"), strings.Contains(label, "login"):
		if preset.Email != "" {
			return preset.Email, nil
		}
	case strings.Contains(label, "password"):
		if preset.Password != "" {
			return preset.Password, nil
		}
	case strings.Contains(label, "totp"), strings.Contains(label, "authenticator"), strings.Contains(label, "two_factor"):
		if preset.TOTP != "" {
			return auth.ResolveOTP(preset.TOTP, time.Now())
		}
	}
	if preset.TOTP != "" {
		for _, option := range field.Options {
			if strings.EqualFold(strings.TrimSpace(option), "Authentication app") {
				return option, nil
			}
		}
	}
	if len(field.Options) == 1 {
		return field.Options[0], nil
	}
	return promptField(cmd, field)
}

func promptField(cmd *cobra.Command, field auth.LoginField) (string, error) {
	label := field.Name
	if label == "" {
		label = field.ID
	}
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "%s: ", label); err != nil {
		return "", err
	}
	if field.Secret {
		if f, ok := cmd.InOrStdin().(*os.File); ok && term.IsTerminal(int(f.Fd())) {
			data, err := term.ReadPassword(int(f.Fd()))
			_, _ = fmt.Fprintln(cmd.ErrOrStderr())
			return string(data), err
		}
	}
	reader := bufio.NewReader(cmd.InOrStdin())
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func newAuthStatusCommand(opts *options) *cobra.Command {
	online := false
	cmd := &cobra.Command{Use: "status", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		name := r.profileName(opts)
		if name == "" {
			return fmt.Errorf("%w: no profile selected", fberrors.ErrInvalidInput)
		}
		cookies, err := r.manager.LoadCookies(cmd.Context(), name)
		if errors.Is(err, storage.ErrNotFound) {
			return writeValue(cmd.OutOrStdout(), opts.json, map[string]any{"profile": name, "authenticated": false}, "not authenticated")
		}
		if err != nil {
			return err
		}
		data := map[string]any{"profile": name, "authenticated": true, "online": false}
		if online {
			session, err := (auth.SessionValidator{Logger: zerolog.Nop()}).Validate(cmd.Context(), cookies)
			if err != nil {
				return err
			}
			data["online"] = true
			data["account_id"] = session.FBID
			data["name"] = session.Name
		}
		return writeValue(cmd.OutOrStdout(), opts.json, data, "authenticated as "+name)
	}}
	cmd.Flags().BoolVar(&online, "online", false, "validate the session with Facebook")
	return cmd
}

func newAuthRefreshCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "refresh", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		name := r.profileName(opts)
		if name == "" {
			return fmt.Errorf("%w: no profile selected", fberrors.ErrInvalidInput)
		}
		session, err := r.manager.Refresh(cmd.Context(), name, auth.SessionValidator{Logger: zerolog.Nop()})
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, map[string]any{"profile": name, "account_id": session.FBID, "refreshed": true}, "session refreshed")
	}}
}

func newAuthLogoutCommand(opts *options) *cobra.Command {
	keepE2EE := false
	cmd := &cobra.Command{Use: "logout", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		name := r.profileName(opts)
		if name == "" {
			return fmt.Errorf("%w: no profile selected", fberrors.ErrInvalidInput)
		}
		if err := r.manager.Logout(cmd.Context(), name, auth.LogoutPolicy{RemoveCookies: true, RemoveSession: true, RemoveE2EE: !keepE2EE}); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, map[string]any{"profile": name, "logged_out": true, "e2ee_preserved": keepE2EE}, "logged out")
	}}
	cmd.Flags().BoolVar(&keepE2EE, "keep-e2ee", false, "preserve E2EE device state")
	return cmd
}
