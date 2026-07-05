package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/scagogogo/rubygems-skills/pkg/models"
)

func pushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push [gem-file]",
		Short: "Publish (push) a .gem file to RubyGems.org (requires --token)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read gem file: %w", err)
			}
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			resp, err := newWriteRepo().PushGem(ctx, data)
			if err != nil {
				return handleErr(err)
			}
			fmt.Println(resp)
			return nil
		},
	}
}

func yankCmd() *cobra.Command {
	var platform string
	c := &cobra.Command{
		Use:   "yank [gem] [version]",
		Short: "Yank (unpublish) a gem version (requires --token)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			var resp string
			var err error
			if platform != "" {
				resp, err = newWriteRepo().YankGemWithPlatform(ctx, args[0], args[1], platform)
			} else {
				resp, err = newWriteRepo().YankGem(ctx, args[0], args[1])
			}
			if err != nil {
				return handleErr(err)
			}
			fmt.Println(resp)
			return nil
		},
	}
	c.Flags().StringVar(&platform, "platform", "", "Gem platform (e.g. x86_64-linux)")
	return c
}

func addOwnerCmd() *cobra.Command {
	var role string
	c := &cobra.Command{
		Use:   "add-owner [gem] [email]",
		Short: "Add an owner to a gem (requires --token)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			if err := newWriteRepo().AddGemOwner(ctx, args[0], args[1], role); err != nil {
				return handleErr(err)
			}
			fmt.Printf("Added owner %s to %s (role: %s)\n", args[1], args[0], role)
			return nil
		},
	}
	c.Flags().StringVar(&role, "role", "owner", "Owner role")
	return c
}

func removeOwnerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove-owner [gem] [email]",
		Short: "Remove an owner from a gem (requires --token)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			if err := newWriteRepo().RemoveGemOwner(ctx, args[0], args[1]); err != nil {
				return handleErr(err)
			}
			fmt.Printf("Removed owner %s from %s\n", args[1], args[0])
			return nil
		},
	}
}

func updateOwnerCmd() *cobra.Command {
	var role string
	c := &cobra.Command{
		Use:   "update-owner [gem] [email]",
		Short: "Update an owner's role on a gem (requires --token)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if role == "" {
				return fmt.Errorf("--role is required")
			}
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			if err := newWriteRepo().UpdateGemOwnerRole(ctx, args[0], args[1], role); err != nil {
				return handleErr(err)
			}
			fmt.Printf("Updated owner %s on %s to role: %s\n", args[1], args[0], role)
			return nil
		},
	}
	c.Flags().StringVar(&role, "role", "", "New owner role")
	markRequired(c, "role")
	return c
}

func listWebhooksCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-webhooks",
		Short: "List webhooks (requires --token)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			hooks, err := newWriteRepo().ListWebhooks(ctx)
			if err != nil {
				return handleErr(err)
			}
			printOutput(hooks)
			return nil
		},
	}
}

func createWebhookCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create-webhook [gem] [url]",
		Short: "Create a webhook for a gem (requires --token)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			if err := newWriteRepo().CreateWebhook(ctx, args[0], args[1]); err != nil {
				return handleErr(err)
			}
			fmt.Printf("Created webhook %s for %s\n", args[1], args[0])
			return nil
		},
	}
}

func deleteWebhookCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete-webhook [gem] [url]",
		Short: "Delete a webhook for a gem (requires --token)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			if err := newWriteRepo().DeleteWebhook(ctx, args[0], args[1]); err != nil {
				return handleErr(err)
			}
			fmt.Printf("Deleted webhook %s for %s\n", args[1], args[0])
			return nil
		},
	}
}

func fireWebhookCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fire-webhook [gem] [url]",
		Short: "Test-fire a webhook for a gem (requires --token)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			if err := newWriteRepo().FireWebhook(ctx, args[0], args[1]); err != nil {
				return handleErr(err)
			}
			fmt.Printf("Fired webhook %s for %s\n", args[1], args[0])
			return nil
		},
	}
}

func getAPIKeyCmd() *cobra.Command {
	var user, pass string
	c := &cobra.Command{
		Use:   "get-api-key",
		Short: "Retrieve a legacy API key (HTTP Basic auth)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			key, err := newWriteRepo().GetAPIKey(ctx, user, pass)
			if err != nil {
				return handleErr(err)
			}
			printOutput(key)
			return nil
		},
	}
	c.Flags().StringVar(&user, "user", "", "RubyGems.org username")
	c.Flags().StringVar(&pass, "password", "", "RubyGems.org password (read from stdin if empty)")
	markRequired(c, "user")
	return c
}

func createAPIKeyCmd() *cobra.Command {
	var user, pass, name, mfa string
	var scopes []string
	c := &cobra.Command{
		Use:   "create-api-key",
		Short: "Create a scoped API key (HTTP Basic auth)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &models.CreateAPIKeyRequest{
				Name:   name,
				Scopes: scopes,
				MFA:    mfa,
			}
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			key, err := newWriteRepo().CreateAPIKey(ctx, user, pass, req)
			if err != nil {
				return handleErr(err)
			}
			printOutput(key)
			return nil
		},
	}
	c.Flags().StringVar(&user, "user", "", "RubyGems.org username")
	c.Flags().StringVar(&pass, "password", "", "RubyGems.org password")
	c.Flags().StringVar(&name, "name", "", "API key name")
	c.Flags().StringSliceVar(&scopes, "scopes", nil, "Scopes (e.g. push_rubygem,yank_rubygem)")
	c.Flags().StringVar(&mfa, "mfa", "", "MFA setting: enabled or disabled")
	markRequired(c, "user", "name")
	return c
}

func updateAPIKeyCmd() *cobra.Command {
	var user, pass, apiKey string
	var scopes []string
	c := &cobra.Command{
		Use:   "update-api-key",
		Short: "Update an API key's scopes (HTTP Basic auth)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &models.UpdateAPIKeyRequest{
				APIKey: apiKey,
				Scopes: scopes,
			}
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			key, err := newWriteRepo().UpdateAPIKey(ctx, user, pass, req)
			if err != nil {
				return handleErr(err)
			}
			printOutput(key)
			return nil
		},
	}
	c.Flags().StringVar(&user, "user", "", "RubyGems.org username")
	c.Flags().StringVar(&pass, "password", "", "RubyGems.org password")
	c.Flags().StringVar(&apiKey, "api-key", "", "Existing API key value")
	c.Flags().StringSliceVar(&scopes, "scopes", nil, "New scopes")
	markRequired(c, "user", "api-key")
	return c
}

func myProfileCmd() *cobra.Command {
	var user, pass string
	c := &cobra.Command{
		Use:   "my-profile",
		Short: "Get the full authenticated user profile (HTTP Basic auth)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			profile, err := newWriteRepo().GetMyProfile(ctx, user, pass)
			if err != nil {
				return handleErr(err)
			}
			printOutput(profile)
			return nil
		},
	}
	c.Flags().StringVar(&user, "user", "", "RubyGems.org username")
	c.Flags().StringVar(&pass, "password", "", "RubyGems.org password")
	markRequired(c, "user")
	return c
}
