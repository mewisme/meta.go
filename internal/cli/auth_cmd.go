package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
	"go.mewis.me/fbgo/auth"
	fberrors "go.mewis.me/fbgo/errors"
	"go.mewis.me/fbgo/storage"
	"golang.org/x/term"
)

type loginInput struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
	TOTP       string `json:"totp"`
	OTP        string `json:"otp"`
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
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]any{"profile": name, "imported": true}, "cookies imported for "+name)
	}}
}

func newAuthLoginCommand(opts *options) *cobra.Command {
	var inputPath, identifier, password, totp, otp string
	cmd := &cobra.Command{Use: "login", Short: "Authenticate with credentials", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		defer func() {
			password, totp, otp = "", "", ""
		}()
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		name := r.profileName(opts)
		if name == "" {
			return fmt.Errorf("%w: no profile selected", fberrors.ErrInvalidInput)
		}
		preset := loginInput{}
		defer func() {
			preset.Password, preset.TOTP, preset.OTP = "", "", ""
		}()
		if inputPath != "" {
			var data []byte
			defer func() { clear(data) }()
			if inputPath == "-" {
				data, err = io.ReadAll(cmd.InOrStdin())
			} else {
				data, err = os.ReadFile(inputPath)
			}
			if err != nil {
				return err
			}
			if err := decodeJSONInput(data, opts.jqi, &preset); err != nil {
				return err
			}
		} else if strings.TrimSpace(opts.jqi) != "" {
			return fmt.Errorf("%w: --jqi requires --input", fberrors.ErrInvalidInput)
		}
		if cmd.Flags().Changed("identifier") {
			preset.Identifier = identifier
		}
		if cmd.Flags().Changed("password") {
			preset.Password = password
		}
		if cmd.Flags().Changed("totp") {
			preset.TOTP, preset.OTP = totp, ""
		}
		if cmd.Flags().Changed("otp") {
			preset.OTP, preset.TOTP = otp, ""
		}
		if err := collectBaseCredentials(cmd, &preset); err != nil {
			return err
		}
		if err := validateLoginInput(preset); err != nil {
			return err
		}
		login := auth.NewCredentialLogin()
		cookies, err := login.Login(cmd.Context(), auth.Credentials{Identifier: preset.Identifier, Password: preset.Password, TOTP: preset.TOTP, OTP: preset.OTP})
		if errors.Is(err, auth.ErrTwoFactorRequired) && preset.TOTP == "" && preset.OTP == "" && isInteractive(cmd) {
			if err := promptOTP(cmd, &preset.OTP); err != nil {
				return err
			}
			cookies, err = login.Login(cmd.Context(), auth.Credentials{Identifier: preset.Identifier, Password: preset.Password, OTP: preset.OTP})
		}
		if err != nil {
			return err
		}
		if err := r.manager.ImportCookies(cmd.Context(), name, cookies); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]any{"profile": name, "authenticated": true}, "authenticated as "+name)
	}}
	cmd.Flags().StringVar(&inputPath, "input", "", "read credentials from JSON file or - for stdin")
	cmd.Flags().StringVar(&identifier, "identifier", "", "login email, phone number, or username")
	cmd.Flags().StringVar(&password, "password", "", "password (visible to shell history/process list; omit to prompt securely)")
	cmd.Flags().StringVar(&totp, "totp", "", "TOTP secret (visible to shell history/process list; omit to prompt securely)")
	cmd.Flags().StringVar(&otp, "otp", "", "6-digit one-time password (visible to shell history/process list; omit to prompt securely)")
	cmd.MarkFlagsMutuallyExclusive("totp", "otp")
	return cmd
}

func collectBaseCredentials(cmd *cobra.Command, input *loginInput) error {
	if input.Identifier != "" && input.Password != "" {
		return nil
	}
	if !isInteractive(cmd) {
		if input.Identifier == "" {
			return fmt.Errorf("%w: identifier is required", fberrors.ErrInvalidInput)
		}
		return fmt.Errorf("%w: password is required", fberrors.ErrInvalidInput)
	}
	fields := make([]huh.Field, 0, 2)
	if input.Identifier == "" {
		fields = append(fields, huh.NewInput().Title("Identifier").Value(&input.Identifier).Validate(huh.ValidateNotEmpty()))
	}
	if input.Password == "" {
		fields = append(fields, huh.NewInput().Title("Password").Password(true).Value(&input.Password).Validate(huh.ValidateNotEmpty()))
	}
	return huh.NewForm(huh.NewGroup(fields...)).WithInput(cmd.InOrStdin()).WithOutput(cmd.ErrOrStderr()).RunWithContext(cmd.Context())
}

func promptOTP(cmd *cobra.Command, value *string) error {
	field := huh.NewInput().Title("One-time password").Description("Enter the current 6-digit authentication code").Password(true).Value(value).Validate(func(value string) error {
		if len(value) != 6 {
			return errors.New("OTP must be exactly 6 digits")
		}
		_, err := strconv.ParseUint(value, 10, 32)
		return err
	})
	return huh.NewForm(huh.NewGroup(field)).WithInput(cmd.InOrStdin()).WithOutput(cmd.ErrOrStderr()).RunWithContext(cmd.Context())
}

func validateLoginInput(input loginInput) error {
	if input.TOTP != "" && input.OTP != "" {
		return fmt.Errorf("%w: totp and otp are mutually exclusive", fberrors.ErrInvalidInput)
	}
	if input.TOTP != "" {
		if _, err := auth.TOTP(input.TOTP, time.Now()); err != nil {
			return fmt.Errorf("%w: invalid TOTP secret", fberrors.ErrInvalidInput)
		}
	}
	if input.OTP != "" {
		if len(input.OTP) != 6 {
			return fmt.Errorf("%w: otp must be exactly 6 digits", fberrors.ErrInvalidInput)
		}
		if _, err := strconv.ParseUint(input.OTP, 10, 32); err != nil {
			return fmt.Errorf("%w: otp must be exactly 6 digits", fberrors.ErrInvalidInput)
		}
	}
	return nil
}

func isInteractive(cmd *cobra.Command) bool {
	file, ok := cmd.InOrStdin().(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
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
			return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]any{"profile": name, "authenticated": false}, "not authenticated")
		}
		if err != nil {
			return err
		}
		data := map[string]any{"profile": name, "authenticated": true, "online": false}
		if online {
			session, err := (auth.SessionValidator{}).Validate(cmd.Context(), cookies)
			if err != nil {
				return err
			}
			data["online"] = true
			data["account_id"] = session.FBID
			data["name"] = session.Name
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, data, "authenticated as "+name)
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
		session, err := r.manager.Refresh(cmd.Context(), name, auth.SessionValidator{})
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]any{"profile": name, "account_id": session.FBID, "refreshed": true}, "session refreshed")
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
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]any{"profile": name, "logged_out": true, "e2ee_preserved": keepE2EE}, "logged out")
	}}
	cmd.Flags().BoolVar(&keepE2EE, "keep-e2ee", false, "preserve E2EE device state")
	return cmd
}
