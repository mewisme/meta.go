package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	fberrors "go.mewis.me/meta.go/errors"
	facebookservice "go.mewis.me/meta.go/facebook"
	"go.mewis.me/meta.go/model"
)

func newFacebookCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "facebook", Short: "Facebook profile, social and Marketplace operations"}
	cmd.AddCommand(newFacebookUserCommand(opts), newFacebookSearchCommand(opts), newFacebookNotificationsCommand(opts), newFacebookBioCommand(opts), newFacebookAdditionalProfileCommand(opts), newFacebookUnfriendCommand(opts), newFacebookBlockCommand(opts), newFacebookPostCommand(opts), newFacebookMarketplaceCommand(opts), newFacebookProfessionalCommand(opts))
	return cmd
}

func newFacebookUserCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "user <user-id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		value, err := c.Facebook.User(cmd.Context(), model.ID(args[0]))
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, value, value.ID.String()+"\t"+value.Name)
	}}
}

func newFacebookSearchCommand(opts *options) *cobra.Command {
	limit := 5
	cmd := &cobra.Command{Use: "search <query>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		values, err := c.Facebook.Search(cmd.Context(), args[0], limit)
		if err != nil {
			return err
		}
		if opts.json {
			return writeValue(cmd.OutOrStdout(), true, opts.jqo, values, "")
		}
		for _, value := range values {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", value.ID, value.Name, value.URL)
		}
		return nil
	}}
	cmd.Flags().IntVar(&limit, "limit", 5, "maximum search results")
	return cmd
}

func newFacebookNotificationsCommand(opts *options) *cobra.Command {
	limit := 15
	cmd := &cobra.Command{Use: "notifications", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		values, err := c.Facebook.Notifications(cmd.Context(), limit)
		if err != nil {
			return err
		}
		if opts.json {
			return writeValue(cmd.OutOrStdout(), true, opts.jqo, values, "")
		}
		for _, value := range values {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), value.Text)
		}
		return nil
	}}
	cmd.Flags().IntVar(&limit, "limit", 15, "maximum notifications")
	return cmd
}

func newFacebookBioCommand(opts *options) *cobra.Command {
	publish := false
	cmd := &cobra.Command{Use: "bio <text>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Facebook.SetBio(cmd.Context(), args[0], publish); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]bool{"updated": true}, "bio updated")
	}}
	cmd.Flags().BoolVar(&publish, "publish", false, "publish a feed story for the bio change")
	return cmd
}

func newFacebookAdditionalProfileCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "additional-profile <name> <username>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Facebook.CreateAdditionalProfile(cmd.Context(), args[0], args[1]); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]bool{"created": true}, "additional profile created")
	}}
}

func newFacebookUnfriendCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "unfriend <user-id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Facebook.Unfriend(cmd.Context(), model.ID(args[0])); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]bool{"unfriended": true}, "user unfriended")
	}}
}

func newFacebookBlockCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "block", Short: "Block or unblock users"}
	for _, item := range []struct {
		name    string
		blocked bool
	}{{"add", true}, {"remove", false}} {
		item := item
		cmd.AddCommand(&cobra.Command{Use: item.name + " <user-id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRuntime(cmd.Context(), opts)
			if err != nil {
				return err
			}
			c, err := r.client(cmd.Context(), opts)
			if err != nil {
				return err
			}
			defer c.Close()
			if err := c.Facebook.SetBlocked(cmd.Context(), model.ID(args[0]), item.blocked); err != nil {
				return err
			}
			return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]bool{"blocked": item.blocked}, "block state updated")
		}})
	}
	return cmd
}

func newFacebookPostCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "post", Short: "Facebook post operations"}
	cmd.AddCommand(&cobra.Command{Use: "create <text>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		post, err := c.Facebook.CreatePost(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, post, post.URL)
	}})
	for _, action := range []string{"archive", "delete"} {
		action := action
		ownership := "owned"
		sub := &cobra.Command{Use: action + " <post-id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			kind := model.PostOwnership(ownership)
			if kind != model.PostOwned && kind != model.PostShared {
				return fmt.Errorf("%w: ownership must be owned or shared", fberrors.ErrInvalidInput)
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
			if action == "archive" {
				err = c.Facebook.ArchivePost(cmd.Context(), model.ID(args[0]), kind)
			} else {
				err = c.Facebook.DeletePost(cmd.Context(), model.ID(args[0]), kind)
			}
			if err != nil {
				return err
			}
			return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]bool{action + "d": true}, "post "+action+"d")
		}}
		sub.Flags().StringVar(&ownership, "ownership", "owned", "post ownership: owned or shared")
		cmd.AddCommand(sub)
	}
	return cmd
}

func newFacebookMarketplaceCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "marketplace", Short: "Marketplace operations"}
	cmd.AddCommand(&cobra.Command{Use: "categories", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		values := cCategories()
		if opts.json {
			return writeValue(cmd.OutOrStdout(), true, opts.jqo, values, "")
		}
		for _, value := range values {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), value)
		}
		return nil
	}})
	cmd.AddCommand(&cobra.Command{Use: "get <listing-id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		r, err := loadRuntime(cmd.Context(), opts)
		if err != nil {
			return err
		}
		c, err := r.client(cmd.Context(), opts)
		if err != nil {
			return err
		}
		defer c.Close()
		value, err := c.Facebook.MarketplaceListing(cmd.Context(), model.ID(args[0]))
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, value, value.ID.String()+"\t"+value.Title)
	}})
	cmd.AddCommand(&cobra.Command{Use: "create <input.json>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		var input model.MarketplaceListingInput
		if err := decodeJSONInput(data, opts.jqi, &input); err != nil {
			return err
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
		value, err := c.Facebook.CreateMarketplaceListing(cmd.Context(), input)
		if err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, value, value.URL)
	}})
	return cmd
}

func cCategories() []string { return facebookservice.MarketplaceCategories() }

func newFacebookProfessionalCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "professional <on|off>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		value := strings.ToLower(args[0])
		if value != "on" && value != "off" {
			return fmt.Errorf("%w: professional mode must be on or off", fberrors.ErrInvalidInput)
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
		if err := c.Facebook.SetProfessionalMode(cmd.Context(), value == "on"); err != nil {
			return err
		}
		return writeValue(cmd.OutOrStdout(), opts.json, opts.jqo, map[string]bool{"enabled": value == "on"}, "professional mode "+value)
	}}
}
