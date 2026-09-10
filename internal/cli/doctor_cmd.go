package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"go.mewis.me/meta.go"
	"go.mewis.me/meta.go/auth"
	"go.mewis.me/meta.go/storage"
)

type doctorCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}
type doctorReport struct {
	Checks []doctorCheck `json:"checks"`
	Online bool          `json:"online"`
}

func newDoctorCommand(opts *options) *cobra.Command {
	online := false
	cmd := &cobra.Command{Use: "doctor", Short: "Check meta configuration and connectivity", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		report := doctorReport{Online: online}
		version := meta.BuildVersion()
		report.Checks = append(report.Checks, doctorCheck{Name: "version", OK: true, Detail: fmt.Sprintf("%s %s %s", version.Version, version.Commit, version.GoVersion)})
		path, err := resolveConfigPath(opts, false)
		report.Checks = append(report.Checks, doctorCheck{Name: "config", OK: err == nil, Detail: path})
		if err != nil {
			return writeDoctor(cmd, opts, report)
		}
		appendPermissionChecks(&report, path)
		r, err := loadRuntime(cmd.Context(), opts)
		report.Checks = append(report.Checks, doctorCheck{Name: "runtime", OK: err == nil})
		if err != nil {
			return writeDoctor(cmd, opts, report)
		}
		report.Checks = append(report.Checks, doctorCheck{Name: "secret_backend", OK: true, Detail: secretBackendDetail()})
		name := r.profileName(opts)
		_, profileErr := r.profiles.Get(cmd.Context(), name)
		report.Checks = append(report.Checks, doctorCheck{Name: "profile", OK: name != "" && profileErr == nil, Detail: name})
		cookies, cookieErr := r.manager.LoadCookies(cmd.Context(), name)
		report.Checks = append(report.Checks, doctorCheck{Name: "cookies", OK: cookieErr == nil, Detail: localAuthDetail(cookies, cookieErr)})
		report.Checks = append(report.Checks, localE2EEStateCheck(cmd, r, name, profileErr))
		if online && profileErr == nil && cookieErr == nil {
			client, err := r.client(cmd.Context(), opts)
			report.Checks = append(report.Checks, doctorCheck{Name: "facebook_connect", OK: err == nil})
			if client != nil {
				health := client.Health()
				report.Checks = append(report.Checks, doctorCheck{Name: "regular_health", OK: health.Regular == meta.ConnectionConnected, Detail: string(health.Regular)})
				if r.config.E2EE {
					report.Checks = append(report.Checks, doctorCheck{Name: "e2ee_health", OK: health.E2EE == meta.ConnectionConnected, Detail: string(health.E2EE)})
				} else {
					report.Checks = append(report.Checks, doctorCheck{Name: "e2ee_health", OK: true, Detail: "disabled"})
				}
				client.Close()
			}
		}
		return writeDoctor(cmd, opts, report)
	}}
	cmd.Flags().BoolVar(&online, "online", false, "include Facebook connectivity checks")
	return cmd
}

func appendPermissionChecks(report *doctorReport, path string) {
	for _, item := range []struct {
		name string
		path string
	}{{"config_file_permissions", path}, {"config_dir_permissions", filepath.Dir(path)}} {
		info, err := os.Stat(item.path)
		if err != nil {
			report.Checks = append(report.Checks, doctorCheck{Name: item.name, OK: false, Detail: err.Error()})
			continue
		}
		if runtime.GOOS == "windows" {
			report.Checks = append(report.Checks, doctorCheck{Name: item.name, OK: true, Detail: "ACL-managed"})
			continue
		}
		report.Checks = append(report.Checks, doctorCheck{Name: item.name, OK: info.Mode().Perm()&0o077 == 0, Detail: fmt.Sprintf("%#o", info.Mode().Perm())})
	}
}

func secretBackendDetail() string {
	if strings.TrimSpace(os.Getenv("META_MASTER_KEY")) != "" {
		return "encrypted-file+environment-master-key"
	}
	return "encrypted-file+os-keyring"
}

func localAuthDetail(cookies auth.Cookies, err error) string {
	if errors.Is(err, storage.ErrNotFound) {
		return "not configured"
	}
	if err != nil {
		return "invalid"
	}
	names := make([]string, 0, len(cookies))
	for name := range cookies {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}

func localE2EEStateCheck(cmd *cobra.Command, r *appRuntime, profile string, profileErr error) doctorCheck {
	if !r.config.E2EE {
		return doctorCheck{Name: "e2ee_state", OK: true, Detail: "disabled"}
	}
	if profileErr != nil || strings.TrimSpace(profile) == "" {
		return doctorCheck{Name: "e2ee_state", OK: false, Detail: "profile unavailable"}
	}
	_, err := r.secrets.Get(cmd.Context(), profile, "e2ee_state")
	if errors.Is(err, storage.ErrNotFound) {
		return doctorCheck{Name: "e2ee_state", OK: true, Detail: "not initialized"}
	}
	if err != nil {
		return doctorCheck{Name: "e2ee_state", OK: false, Detail: "unreadable"}
	}
	return doctorCheck{Name: "e2ee_state", OK: true, Detail: "initialized"}
}

func writeDoctor(cmd *cobra.Command, opts *options, report doctorReport) error {
	if opts.json {
		return writeValue(cmd.OutOrStdout(), true, opts.jqo, report, "")
	}
	for _, check := range report.Checks {
		state := "ok"
		if !check.OK {
			state = "fail"
		}
		if check.Detail != "" {
			cmd.Printf("%-24s %s (%s)\n", check.Name, state, check.Detail)
		} else {
			cmd.Printf("%-24s %s\n", check.Name, state)
		}
	}
	return nil
}
