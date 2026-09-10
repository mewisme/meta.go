package cli

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	fberrors "go.mewis.me/meta.go/errors"
	"go.mewis.me/meta.go/internal/fsutil"
	"go.mewis.me/meta.go/messenger"
	"go.mewis.me/meta.go/model"
)

func newMessengerCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "messenger", Short: "Messenger operations"}
	cmd.AddCommand(newMessengerListenCommand(opts), newMessengerSendCommand(opts), newMessengerShareContactCommand(opts), newMessengerForwardCommand(opts), newMessengerReactCommand(opts), newMessengerEditCommand(opts), newMessengerUnsendCommand(opts), newMessengerTypingCommand(opts), newMessengerReadCommand(opts), newMessengerRestrictCommand(opts), newMessengerMessageBlockCommand(opts), newMediaCommand(opts), newThemeCommand(opts), newNoteCommand(opts), newRequestsCommand(opts))
	return cmd
}

func newMessengerShareContactCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "share-contact <thread-id> <contact-id> [text]", Args: cobra.RangeArgs(2, 3), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		client, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer client.Close()
		text := ""
		if len(args) == 3 {
			text = args[2]
		}
		if err := client.Messenger.ShareContact(cmd.Context(), model.ID(args[0]), model.ID(args[1]), text); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]bool{"shared": true}, "contact shared")
	}}
}

func newMessengerRestrictCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "restrict <user-id> <on|off>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		restricted, err := parseOnOff(args[1], "restrict")
		if err != nil {
			return err
		}
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		client, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer client.Close()
		if err := client.Messenger.SetRestricted(cmd.Context(), model.ID(args[0]), restricted); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]any{"user_id": args[0], "restricted": restricted}, "messenger restriction updated")
	}}
}

func newMessengerMessageBlockCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "message-block <user-id> <on|off>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		blocked, err := parseOnOff(args[1], "message block")
		if err != nil {
			return err
		}
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		client, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer client.Close()
		if err := client.Messenger.SetMessageBlocked(cmd.Context(), model.ID(args[0]), blocked); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]any{"user_id": args[0], "message_blocked": blocked}, "messenger message block updated")
	}}
}

func newMessengerForwardCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "forward <thread-id> <message-id>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		client, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer client.Close()
		result, err := client.Messenger.Forward(cmd.Context(), model.ID(args[0]), model.ID(args[1]))
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, result, result.MessageID.String())
	}}
}

func newMessengerListenCommand(opts *options) *cobra.Command {
	ndjson := false
	cmd := &cobra.Command{Use: "listen", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		client, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer client.Close()
		for {
			select {
			case <-cmd.Context().Done():
				return cmd.Context().Err()
			case event, ok := <-client.Events():
				if !ok {
					return nil
				}
				if err := writeEvent(cmd.OutOrStdout(), ndjson || opts.json, opts.jqo, event); err != nil {
					return err
				}
			}
		}
	}}
	cmd.Flags().BoolVar(&ndjson, "ndjson", false, "stream one JSON event per line")
	return cmd
}

func newMessengerSendCommand(opts *options) *cobra.Command {
	attachments := []string{}
	stickerID, externalURL := "", ""
	regular, e2ee := false, false
	cmd := &cobra.Command{Use: "send <thread-id|chat-jid> [text]", Args: cobra.RangeArgs(1, 2), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		client, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer client.Close()
		text := ""
		if len(args) == 2 {
			text = args[1]
		}
		if regular && e2ee {
			return fmt.Errorf("%w: --regular and --e2ee are mutually exclusive", fberrors.ErrInvalidInput)
		}
		if e2ee {
			if len(attachments) > 0 || stickerID != "" || externalURL != "" {
				return fmt.Errorf("%w: regular attachments, stickers and external URLs cannot be used with --e2ee", fberrors.ErrInvalidInput)
			}
			req := model.E2EESendRequest{Text: text}
			if strings.Contains(args[0], "@") {
				req.ChatJID = args[0]
			} else {
				req.FacebookUserID = model.ID(args[0])
			}
			result, err := client.Messenger.SendE2EE(cmd.Context(), req)
			if err != nil {
				return err
			}
			return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, result, result.MessageID.String())
		}
		req := model.SendRequest{ThreadID: model.ID(args[0]), Text: text, StickerID: model.ID(stickerID), URL: externalURL, Encryption: model.EncryptionDisabled}
		opened := make([]*os.File, 0, len(attachments))
		defer func() {
			for _, f := range opened {
				_ = f.Close()
			}
		}()
		for _, path := range attachments {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			opened = append(opened, f)
			info, err := f.Stat()
			if err != nil {
				return err
			}
			contentType := mime.TypeByExtension(filepath.Ext(path))
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			req.Attachments = append(req.Attachments, model.AttachmentInput{Name: filepath.Base(path), ContentType: contentType, Reader: f, Size: info.Size()})
		}
		result, err := client.Messenger.Send(cmd.Context(), req)
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, result, result.MessageID.String())
	}}
	cmd.Flags().StringSliceVarP(&attachments, "attachment", "a", nil, "attachment file path (repeatable)")
	cmd.Flags().StringVar(&stickerID, "sticker-id", "", "regular Messenger sticker ID")
	cmd.Flags().StringVar(&externalURL, "url", "", "regular Messenger external media URL")
	cmd.Flags().BoolVar(&regular, "regular", false, "force regular Messenger transport")
	cmd.Flags().BoolVar(&e2ee, "e2ee", false, "send through E2EE Messenger transport")
	return cmd
}

func newMessengerReactCommand(opts *options) *cobra.Command {
	e2ee, senderJID := false, ""
	cmd := &cobra.Command{Use: "react <thread-id|chat-jid> <message-id> [reaction]", Args: cobra.RangeArgs(2, 3), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		client, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer client.Close()
		reaction := ""
		if len(args) == 3 {
			reaction = args[2]
		}
		if e2ee {
			if strings.TrimSpace(senderJID) == "" {
				return fmt.Errorf("%w: --sender-jid is required for E2EE reactions", fberrors.ErrInvalidInput)
			}
			err = client.Messenger.ReactE2EE(cmd.Context(), model.E2EEReactionRequest{ChatJID: args[0], MessageID: model.ID(args[1]), SenderJID: senderJID, Reaction: reaction})
		} else {
			if senderJID != "" {
				return fmt.Errorf("%w: --sender-jid requires --e2ee", fberrors.ErrInvalidInput)
			}
			err = client.Messenger.React(cmd.Context(), model.ID(args[0]), model.ID(args[1]), reaction)
		}
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]any{"updated": true, "e2ee": e2ee}, "reaction updated")
	}}
	cmd.Flags().BoolVar(&e2ee, "e2ee", false, "react through E2EE Messenger transport")
	cmd.Flags().StringVar(&senderJID, "sender-jid", "", "E2EE original message sender JID")
	return cmd
}

func newMessengerEditCommand(opts *options) *cobra.Command {
	e2ee, chatJID := false, ""
	cmd := &cobra.Command{Use: "edit <message-id> <text>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		client, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer client.Close()
		if e2ee {
			if strings.TrimSpace(chatJID) == "" {
				return fmt.Errorf("%w: --chat-jid is required for E2EE edits", fberrors.ErrInvalidInput)
			}
			err = client.Messenger.EditE2EE(cmd.Context(), chatJID, model.ID(args[0]), args[1])
		} else {
			if chatJID != "" {
				return fmt.Errorf("%w: --chat-jid requires --e2ee", fberrors.ErrInvalidInput)
			}
			err = client.Messenger.Edit(cmd.Context(), model.ID(args[0]), args[1])
		}
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]any{"edited": true, "e2ee": e2ee}, "message edited")
	}}
	cmd.Flags().BoolVar(&e2ee, "e2ee", false, "edit through E2EE Messenger transport")
	cmd.Flags().StringVar(&chatJID, "chat-jid", "", "E2EE chat JID")
	return cmd
}

func newMessengerUnsendCommand(opts *options) *cobra.Command {
	e2ee, chatJID := false, ""
	cmd := &cobra.Command{Use: "unsend <message-id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		client, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer client.Close()
		if e2ee {
			if strings.TrimSpace(chatJID) == "" {
				return fmt.Errorf("%w: --chat-jid is required for E2EE unsend", fberrors.ErrInvalidInput)
			}
			err = client.Messenger.UnsendE2EE(cmd.Context(), chatJID, model.ID(args[0]))
		} else {
			if chatJID != "" {
				return fmt.Errorf("%w: --chat-jid requires --e2ee", fberrors.ErrInvalidInput)
			}
			err = client.Messenger.Unsend(cmd.Context(), model.ID(args[0]))
		}
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]any{"unsent": true, "e2ee": e2ee}, "message unsent")
	}}
	cmd.Flags().BoolVar(&e2ee, "e2ee", false, "unsend through E2EE Messenger transport")
	cmd.Flags().StringVar(&chatJID, "chat-jid", "", "E2EE chat JID")
	return cmd
}

func newMessengerTypingCommand(opts *options) *cobra.Command {
	group, e2ee := false, false
	threadType := int64(1)
	cmd := &cobra.Command{Use: "typing <thread-id|chat-jid> <on|off>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		state := strings.ToLower(strings.TrimSpace(args[1]))
		if state != "on" && state != "off" {
			return fmt.Errorf("%w: typing state must be on or off", fberrors.ErrInvalidInput)
		}
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		client, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer client.Close()
		typing := state == "on"
		if e2ee {
			if group || threadType != 1 {
				return fmt.Errorf("%w: --group and --thread-type apply only to regular typing", fberrors.ErrInvalidInput)
			}
			err = client.Messenger.TypingE2EE(cmd.Context(), args[0], typing)
		} else {
			err = client.Messenger.Typing(cmd.Context(), model.ID(args[0]), typing, group, threadType)
		}
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]any{"typing": typing, "e2ee": e2ee}, "typing updated")
	}}
	cmd.Flags().BoolVar(&group, "group", false, "regular target is a group thread")
	cmd.Flags().Int64Var(&threadType, "thread-type", 1, "regular Messenger thread type")
	cmd.Flags().BoolVar(&e2ee, "e2ee", false, "send E2EE typing presence")
	return cmd
}

func newMessengerReadCommand(opts *options) *cobra.Command {
	watermark := int64(0)
	cmd := &cobra.Command{Use: "read <thread-id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if watermark < 0 {
			return fmt.Errorf("%w: watermark cannot be negative", fberrors.ErrInvalidInput)
		}
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		client, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer client.Close()
		var at time.Time
		if watermark > 0 {
			at = time.UnixMilli(watermark)
		}
		if err := client.Messenger.Read(cmd.Context(), model.ID(args[0]), at); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]any{"read": true, "watermark": watermark}, "thread marked read")
	}}
	cmd.Flags().Int64Var(&watermark, "watermark", 0, "last read watermark in Unix milliseconds (default now)")
	return cmd
}

func newMediaCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "media", Short: "Media operations"}
	cmd.AddCommand(newMediaUploadCommand(opts), newMediaDownloadCommand(opts))
	return cmd
}

func newMediaUploadCommand(opts *options) *cobra.Command {
	voice, e2ee := false, false
	kind, caption := "", ""
	cmd := &cobra.Command{Use: "upload <thread-id|chat-jid> <file>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		f, err := os.Open(args[1])
		if err != nil {
			return err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return err
		}
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		client, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer client.Close()
		ct := mime.TypeByExtension(filepath.Ext(args[1]))
		if ct == "" {
			ct = "application/octet-stream"
		}
		if e2ee {
			mediaKind, err := resolveE2EEMediaKind(kind, ct)
			if err != nil {
				return err
			}
			req := model.E2EEMediaInput{Kind: mediaKind, Name: filepath.Base(args[1]), ContentType: ct, Reader: f, Size: info.Size(), Caption: caption, Voice: voice}
			if strings.Contains(args[0], "@") {
				req.ChatJID = args[0]
			} else {
				req.FacebookUserID = model.ID(args[0])
			}
			result, err := client.Messenger.SendE2EEMedia(cmd.Context(), req)
			if err != nil {
				return err
			}
			return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, result, result.MessageID.String())
		}
		if kind != "" || caption != "" {
			return fmt.Errorf("%w: --kind and --caption require --e2ee", fberrors.ErrInvalidInput)
		}
		result, err := client.Messenger.Upload(cmd.Context(), model.UploadInput{ThreadID: model.ID(args[0]), Name: filepath.Base(args[1]), ContentType: ct, Reader: f, Size: info.Size(), Voice: voice})
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, result, result.ID.String())
	}}
	cmd.Flags().BoolVar(&voice, "voice", false, "mark audio upload as a voice message")
	cmd.Flags().BoolVar(&e2ee, "e2ee", false, "upload and send through E2EE Messenger transport")
	cmd.Flags().StringVar(&kind, "kind", "", "E2EE media kind: image, video, audio, document, sticker")
	cmd.Flags().StringVar(&caption, "caption", "", "E2EE media caption")
	return cmd
}

func resolveE2EEMediaKind(value, contentType string) (model.E2EEMediaKind, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "image":
		return model.E2EEMediaImage, nil
	case "video":
		return model.E2EEMediaVideo, nil
	case "audio":
		return model.E2EEMediaAudio, nil
	case "document", "file":
		return model.E2EEMediaDocument, nil
	case "sticker":
		return model.E2EEMediaSticker, nil
	case "":
		switch {
		case strings.HasPrefix(contentType, "image/"):
			return model.E2EEMediaImage, nil
		case strings.HasPrefix(contentType, "video/"):
			return model.E2EEMediaVideo, nil
		case strings.HasPrefix(contentType, "audio/"):
			return model.E2EEMediaAudio, nil
		default:
			return model.E2EEMediaDocument, nil
		}
	default:
		return "", fmt.Errorf("%w: unsupported E2EE media kind %q", fberrors.ErrInvalidInput, value)
	}
}

func newMediaDownloadCommand(opts *options) *cobra.Command {
	output := ""
	maxBytes := int64(100 << 20)
	cmd := &cobra.Command{Use: "download <url|reference.json>", Short: "Download regular media URL or E2EE media reference", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if opts.json && (output == "" || output == "-") {
			return fmt.Errorf("%w: --json media download requires --output", fberrors.ErrInvalidInput)
		}
		var reader io.ReadCloser
		if strings.HasPrefix(args[0], "https://") || strings.HasPrefix(args[0], "http://") {
			media, err := messenger.DownloadMedia(cmd.Context(), args[0], maxBytes)
			if err != nil {
				return err
			}
			reader = media.Body
		} else {
			data, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			var ref model.E2EEMediaReference
			if err := decodeJSONInput(data, opts.jqi, &ref); err != nil {
				return err
			}
			r, err := loadRuntime(cmd.Context(), opts)
			if err != nil {
				return err
			}
			client, err := r.client(cmd.Context(), opts)
			if err != nil {
				return err
			}
			defer client.Close()
			data, err = client.Messenger.DownloadE2EE(cmd.Context(), model.E2EEMediaDownload{Reference: ref})
			if err != nil {
				return err
			}
			reader = io.NopCloser(bytes.NewReader(data))
		}
		defer reader.Close()
		if output == "" || output == "-" {
			_, err := io.Copy(cmd.OutOrStdout(), reader)
			return err
		}
		written, err := fsutil.AtomicWriteReader(output, reader, 0o600)
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]any{"path": output, "bytes": written}, output)
	}}
	cmd.Flags().StringVarP(&output, "output", "o", "", "output path (default stdout)")
	cmd.Flags().Int64Var(&maxBytes, "max-bytes", 100<<20, "maximum regular media download size")
	return cmd
}

func newThemeCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "theme", Short: "Messenger theme operations"}
	cmd.AddCommand(&cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		values, err := c.Messenger.Themes(cmd.Context())
		if err != nil {
			return err
		}
		if opts.json {
			return writeValue(cmd.OutOrStdout(), true, opts.jqo, values, "")
		}
		for _, v := range values {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", v.ID, v.Name)
		}
		return nil
	}})
	cmd.AddCommand(&cobra.Command{Use: "find <query>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		value, err := c.Messenger.FindTheme(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, value, value.ID.String()+"\t"+value.Name)
	}})
	cmd.AddCommand(&cobra.Command{Use: "set <thread-id> <theme-id>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Messenger.SetTheme(cmd.Context(), model.ID(args[0]), model.ID(args[1])); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]bool{"updated": true}, "theme updated")
	}})
	return cmd
}

func newNoteCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "note", Short: "Messenger Notes operations"}
	cmd.AddCommand(&cobra.Command{Use: "current", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		note, err := c.Messenger.CurrentNote(cmd.Context())
		if err != nil {
			return err
		}
		if note == nil {
			return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, nil, "no active note")
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, note, note.Description)
	}})
	privacy := "FRIENDS"
	create := &cobra.Command{Use: "create <text>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		note, err := c.Messenger.CreateNote(cmd.Context(), args[0], privacy)
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, note, note.ID.String())
	}}
	create.Flags().StringVar(&privacy, "privacy", "FRIENDS", "note privacy")
	cmd.AddCommand(create)
	recreatePrivacy := "FRIENDS"
	recreate := &cobra.Command{Use: "recreate <note-id> <text>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		note, err := c.Messenger.RecreateNote(cmd.Context(), model.ID(args[0]), args[1], recreatePrivacy)
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, note, note.ID.String())
	}}
	recreate.Flags().StringVar(&recreatePrivacy, "privacy", "FRIENDS", "note privacy")
	cmd.AddCommand(recreate)
	cmd.AddCommand(&cobra.Command{Use: "delete <note-id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Messenger.DeleteNote(cmd.Context(), model.ID(args[0])); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]bool{"deleted": true}, "note deleted")
	}})
	return cmd
}

func newRequestsCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "requests", Short: "List message requests", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		values, err := c.Messenger.MessageRequests(cmd.Context())
		if err != nil {
			return err
		}
		if opts.json {
			return writeValue(cmd.OutOrStdout(), true, opts.jqo, values, "")
		}
		for _, v := range values {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", v.SenderID, strings.TrimSpace(v.Snippet))
		}
		return nil
	}}
}

func writeEvent(out io.Writer, asJSON bool, selector string, event model.Event) error {
	if asJSON {
		return writeJSONOutput(out, event, selector)
	}
	_, err := fmt.Fprintf(out, "%s\t%v\n", event.Kind, eventHumanValue(event))
	return err
}

func eventHumanValue(event model.Event) any {
	switch event.Kind {
	case model.EventReady:
		return event.IsNewSession
	case model.EventError:
		return event.Error
	case model.EventDisconnected:
		return event.Transport
	case model.EventMessage:
		return event.Message
	case model.EventMessageEdit:
		return event.MessageEdit
	case model.EventMessageUnsend:
		return event.MessageUnsend
	case model.EventReaction:
		return event.Reaction
	case model.EventTyping:
		return event.Typing
	case model.EventReadReceipt:
		return event.ReadReceipt
	case model.EventDeliveryReceipt:
		return event.DeliveryReceipt
	case model.EventThreadUpdate:
		return event.ThreadUpdate
	case model.EventE2EEReceipt:
		return event.E2EEReceipt
	default:
		return nil
	}
}
