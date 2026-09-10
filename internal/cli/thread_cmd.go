package cli

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	fberrors "go.mewis.me/meta.go/errors"
	"go.mewis.me/meta.go/model"
)

func newThreadCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "thread", Short: "Thread query and administration"}
	cmd.AddCommand(newThreadListCommand(opts), newThreadGetCommand(opts), newThreadPollCommand(opts), newThreadMuteCommand(opts), newThreadPhotoCommand(opts), newThreadDeleteCommand(opts), newThreadCreateDMCommand(opts), newThreadSearchCommand(opts), newThreadContactCommand(opts), newThreadAdminCommand(opts), newThreadNameCommand(opts), newThreadEmojiCommand(opts), newThreadNicknameCommand(opts))
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
		result, err := c.Threads.List(cmd.Context(), limit)
		if err != nil {
			return err
		}
		if opts.json {
			return writeValue(cmd.OutOrStdout(), true, opts.jqo, result, "")
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
		thread, err := c.Threads.Get(cmd.Context(), model.ID(args[0]))
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, thread, thread.ID.String()+"\t"+thread.Name)
	}}
}

func newThreadPollCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "poll", Short: "Poll operations"}
	options := []string{}
	create := &cobra.Command{Use: "create <thread-id> <question>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Threads.CreatePoll(cmd.Context(), model.ID(args[0]), args[1], options); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]bool{"created": true}, "poll created")
	}}
	create.Flags().StringSliceVarP(&options, "option", "o", nil, "poll option (repeatable)")
	vote := &cobra.Command{Use: "vote <thread-id> <poll-id> <option-id> [option-id...]", Args: cobra.MinimumNArgs(3), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		ids := make([]model.ID, len(args)-2)
		for i := 2; i < len(args); i++ {
			ids[i-2] = model.ID(args[i])
		}
		if err := c.Threads.VotePoll(cmd.Context(), model.ID(args[0]), model.ID(args[1]), ids); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]bool{"updated": true}, "poll updated")
	}}
	cmd.AddCommand(create, vote)
	return cmd
}

func newThreadMuteCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "mute <thread-id> <duration|forever|off>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		var duration time.Duration
		switch strings.ToLower(strings.TrimSpace(args[1])) {
		case "forever":
			duration = -time.Second
		case "off", "unmute":
			duration = 0
		default:
			parsed, err := time.ParseDuration(args[1])
			if err != nil || parsed < 0 {
				return fmt.Errorf("%w: invalid mute duration %q", fberrors.ErrInvalidInput, args[1])
			}
			duration = parsed
		}
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Threads.Mute(cmd.Context(), model.ID(args[0]), duration); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]string{"thread_id": args[0], "mute": args[1]}, "thread mute updated")
	}}
}

func newThreadPhotoCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "photo <thread-id> <file>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		file, err := os.Open(args[1])
		if err != nil {
			return err
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			return err
		}
		contentType := mime.TypeByExtension(filepath.Ext(args[1]))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Threads.SetPhoto(cmd.Context(), model.ID(args[0]), model.AttachmentInput{Name: filepath.Base(args[1]), ContentType: contentType, Reader: file, Size: info.Size()}); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]bool{"updated": true}, "thread photo updated")
	}}
}

func newThreadDeleteCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "delete <thread-id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Threads.Delete(cmd.Context(), model.ID(args[0])); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]bool{"deleted": true}, "thread deleted")
	}}
}

func newThreadCreateDMCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "create-dm <user-id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		id, err := c.Threads.CreateDM(cmd.Context(), model.ID(args[0]))
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]model.ID{"thread_id": id}, id.String())
	}}
}

func newThreadSearchCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "search <query>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		users, err := c.Threads.SearchUsers(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		if opts.json {
			return writeValue(cmd.OutOrStdout(), true, opts.jqo, users, "")
		}
		for _, user := range users {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", user.ID, user.Name)
		}
		return nil
	}}
}

func newThreadContactCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "contact <user-id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		user, err := c.Threads.GetContact(cmd.Context(), model.ID(args[0]))
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, user, user.ID.String()+"\t"+user.Name)
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
		if err := c.Threads.SetAdmin(cmd.Context(), model.ID(args[0]), model.ID(args[1]), !remove); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]any{"thread_id": args[0], "user_id": args[1], "admin": !remove}, "admin updated")
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
		if err := c.Threads.SetName(cmd.Context(), model.ID(args[0]), args[1]); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]bool{"updated": true}, "thread name updated")
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
		if err := c.Threads.SetEmoji(cmd.Context(), model.ID(args[0]), args[1]); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]bool{"updated": true}, "thread emoji updated")
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
		if err := c.Threads.SetNickname(cmd.Context(), model.ID(args[0]), model.ID(args[1]), args[2]); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]bool{"updated": true}, "nickname updated")
	}}
}
