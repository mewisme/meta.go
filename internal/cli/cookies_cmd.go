package cli

import "github.com/spf13/cobra"

func newCookiesCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "cookies", Short: "Manage authentication cookies"}
	importCmd := newAuthImportCommand(opts)
	importCmd.Use = "import [file|-]"
	cmd.AddCommand(importCmd)
	return cmd
}
