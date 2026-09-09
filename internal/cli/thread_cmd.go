package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.mewis.me/fbgo/model"
)

func newThreadCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "thread", Short: "Thread query and administration"}
	cmd.AddCommand(newThreadListCommand(opts), newThreadGetCommand(opts), newThreadAdminCommand(opts), newThreadNameCommand(opts), newThreadEmojiCommand(opts), newThreadNicknameCommand(opts))
	return cmd
}

func newThreadListCommand(opts *options) *cobra.Command {
	limit := 50
	cmd := &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		result, err := c.Thread.List(cmd.Context(), limit)
		if err != nil {
			return err
		}
		if opts.json {
			return writeValue(cmd.OutOrStdout(), true, result, "")
		}
		for _, thread := range result.Threads {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%d\n", thread.ID, thread.Name, thread.MessageCount)
		}
		return nil
	}}
	cmd.Flags().IntVar(&limit, "limit", 50, "maximum threads")
	return cmd
}

func newThreadGetCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "info <thread-id>", Aliases: []string{"get"}, Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		thread, err := c.Thread.Get(cmd.Context(), model.ID(args[0]))
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, thread, thread.ID.String()+"\t"+thread.Name)
	}}
}

func newThreadAdminCommand(opts *options) *cobra.Command {
	remove := false
	cmd := &cobra.Command{Use: "admin <thread-id> <user-id>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Thread.SetAdmin(cmd.Context(), model.ID(args[0]), model.ID(args[1]), !remove); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, map[string]any{"thread_id": args[0], "user_id": args[1], "admin": !remove}, "admin updated")
	}}
	cmd.Flags().BoolVar(&remove, "remove", false, "remove admin status")
	return cmd
}

func newThreadNameCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "name <thread-id> <name>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Thread.SetName(cmd.Context(), model.ID(args[0]), args[1]); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, map[string]bool{"updated": true}, "thread name updated")
	}}
}

func newThreadEmojiCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "emoji <thread-id> <emoji>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Thread.SetEmoji(cmd.Context(), model.ID(args[0]), args[1]); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, map[string]bool{"updated": true}, "thread emoji updated")
	}}
}

func newThreadNicknameCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "nickname <thread-id> <user-id> <nickname>", Args: cobra.ExactArgs(3), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Thread.SetNickname(cmd.Context(), model.ID(args[0]), model.ID(args[1]), args[2]); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, map[string]bool{"updated": true}, "nickname updated")
	}}
}
