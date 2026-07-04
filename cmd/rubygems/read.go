package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/scagogogo/rubygems-skills/pkg/models"
	"github.com/scagogogo/rubygems-skills/pkg/repository"
)

// ===== Read subcommands =====

func getCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [gem]",
		Short: "Get detailed package info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			pkg, err := newRepo().GetPackage(ctx, args[0])
			if err != nil {
				return handleErr(err)
			}
			printOutput(pkg)
			return nil
		},
	}
}

func searchCmd() *cobra.Command {
	var page, limit int
	c := &cobra.Command{
		Use:   "search [query]",
		Short: "Search packages",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			results, err := newRepo().Search(ctx, args[0], page)
			if err != nil {
				return handleErr(err)
			}
			if len(results) > limit {
				results = results[:limit]
			}
			if flagJSON {
				printOutput(results)
				return nil
			}
			fmt.Printf("Search results for '%s' (top %d):\n", args[0], len(results))
			for i, p := range results {
				fmt.Printf("%d. %s (version: %s, downloads: %d)\n", i+1, p.Name, p.Version, p.Downloads)
			}
			return nil
		},
	}
	c.Flags().IntVar(&page, "page", 1, "Page number")
	c.Flags().IntVar(&limit, "limit", 10, "Result limit")
	return c
}

func autocompleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "autocomplete [query]",
		Short: "Search autocomplete suggestions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			suggestions, err := newRepo().SearchAutocomplete(ctx, args[0])
			if err != nil {
				return handleErr(err)
			}
			printOutput(suggestions)
			return nil
		},
	}
}

func versionsCmd() *cobra.Command {
	var limit int
	c := &cobra.Command{
		Use:   "versions [gem]",
		Short: "List all versions of a gem",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			versions, err := newRepo().GetGemVersions(ctx, args[0])
			if err != nil {
				return handleErr(err)
			}
			if len(versions) > limit {
				versions = versions[:limit]
			}
			if flagJSON {
				printOutput(versions)
				return nil
			}
			fmt.Printf("Version list for %s (top %d):\n", args[0], len(versions))
			for i, v := range versions {
				fmt.Printf("%d. %s (downloads: %d, released: %s)\n", i+1, v.Number, v.DownloadsCount, v.CreatedAt.Format("2006-01-02"))
			}
			return nil
		},
	}
	c.Flags().IntVar(&limit, "limit", 10, "Result limit")
	return c
}

func latestVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "latest-version [gem]",
		Short: "Get the latest version of a gem",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			v, err := newRepo().GetGemLatestVersion(ctx, args[0])
			if err != nil {
				return handleErr(err)
			}
			printOutput(v)
			return nil
		},
	}
}

func versionDetailCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version-detail [gem] [version]",
		Short: "Get detailed version info (API v2, includes spec_sha, yanked, full deps)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			d, err := newRepo().GetGemVersionDetail(ctx, args[0], args[1])
			if err != nil {
				return handleErr(err)
			}
			printOutput(d)
			return nil
		},
	}
}

func versionContentsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version-contents [gem] [version]",
		Short: "Get file checksums/manifest for a gem version (API v2)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			c, err := newRepo().GetGemVersionContents(ctx, args[0], args[1])
			if err != nil {
				return handleErr(err)
			}
			printOutput(c)
			return nil
		},
	}
}

func downloadsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "downloads",
		Short: "Get total repository download count",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			d, err := newRepo().Downloads(ctx)
			if err != nil {
				return handleErr(err)
			}
			printOutput(d)
			return nil
		},
	}
}

func versionDownloadsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version-downloads [gem] [version]",
		Short: "Get download count for a specific gem version",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			d, err := newRepo().VersionDownloads(ctx, args[0], args[1])
			if err != nil {
				return handleErr(err)
			}
			printOutput(d)
			return nil
		},
	}
}

func topDownloadsCmd() *cobra.Command {
	var limit int
	c := &cobra.Command{
		Use:   "top-downloads",
		Short: "Get top 50 most downloaded gems",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			gems, err := newRepo().TopDownloads(ctx)
			if err != nil {
				return handleErr(err)
			}
			if len(gems) > limit {
				gems = gems[:limit]
			}
			if flagJSON {
				printOutput(gems)
				return nil
			}
			fmt.Printf("Top %d downloaded gems:\n", len(gems))
			for i, g := range gems {
				fmt.Printf("%d. %s (%d downloads)\n", i+1, g.Name, g.Downloads)
			}
			return nil
		},
	}
	c.Flags().IntVar(&limit, "limit", 10, "Result limit")
	return c
}

func depsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deps [gems...]",
		Short: "Get dependency info for one or more gems",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			deps, err := newRepo().GetDependencies(ctx, args...)
			if err != nil {
				return handleErr(err)
			}
			if flagJSON {
				printOutput(deps)
				return nil
			}
			fmt.Printf("Dependencies for %v:\n", args)
			runtime, dev := make([]*models.DependencyInfo, 0), make([]*models.DependencyInfo, 0)
			for _, d := range deps {
				switch d.DependentType {
				case "runtime":
					runtime = append(runtime, d)
				case "development":
					dev = append(dev, d)
				}
			}
			if len(runtime) > 0 {
				fmt.Println("  Runtime dependencies:")
				for _, d := range runtime {
					fmt.Printf("    - %s %s\n", d.Name, d.Requirements)
				}
			}
			if len(dev) > 0 {
				fmt.Println("  Development dependencies:")
				for _, d := range dev {
					fmt.Printf("    - %s %s\n", d.Name, d.Requirements)
				}
			}
			if len(runtime) == 0 && len(dev) == 0 {
				fmt.Println("  No dependencies")
			}
			return nil
		},
	}
}

func rdepsCmd() *cobra.Command {
	var limit int
	c := &cobra.Command{
		Use:   "rdeps [gem]",
		Short: "Get packages that depend on a gem (reverse dependencies)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			rdeps, err := newRepo().GetReverseDependencies(ctx, args[0])
			if err != nil {
				return handleErr(err)
			}
			if len(rdeps) > limit {
				rdeps = rdeps[:limit]
			}
			if flagJSON {
				printOutput(rdeps)
				return nil
			}
			fmt.Printf("Packages depending on %s (top %d):\n", args[0], len(rdeps))
			for i, d := range rdeps {
				fmt.Printf("%d. %s\n", i+1, d)
			}
			return nil
		},
	}
	c.Flags().IntVar(&limit, "limit", 10, "Result limit")
	return c
}

func versionRdepsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version-rdeps [fullName]",
		Short: "Get packages depending on a specific version (fullName = gemname-version, e.g. rack-2.2.7)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			rdeps, err := newRepo().GetVersionReverseDependencies(ctx, args[0])
			if err != nil {
				return handleErr(err)
			}
			printOutput(rdeps)
			return nil
		},
	}
}

func latestGemsCmd() *cobra.Command {
	var limit int
	c := &cobra.Command{
		Use:   "latest-gems",
		Short: "Get recently published gems",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			gems, err := newRepo().LatestGems(ctx)
			if err != nil {
				return handleErr(err)
			}
			if len(gems) > limit {
				gems = gems[:limit]
			}
			printOutput(gems)
			return nil
		},
	}
	c.Flags().IntVar(&limit, "limit", 10, "Result limit")
	return c
}

func justUpdatedCmd() *cobra.Command {
	var limit int
	c := &cobra.Command{
		Use:   "just-updated",
		Short: "Get recently updated gems",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			gems, err := newRepo().JustUpdatedGems(ctx)
			if err != nil {
				return handleErr(err)
			}
			if len(gems) > limit {
				gems = gems[:limit]
			}
			printOutput(gems)
			return nil
		},
	}
	c.Flags().IntVar(&limit, "limit", 10, "Result limit")
	return c
}

func userProfileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "user-profile [handle]",
		Short: "Get a user's public profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			p, err := newRepo().GetUserProfile(ctx, args[0])
			if err != nil {
				return handleErr(err)
			}
			printOutput(p)
			return nil
		},
	}
}

func ownedGemsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "owned-gems",
		Short: "List gems owned by the authenticated user (requires --token)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			gems, err := newRepo().GetOwnedGems(ctx)
			if err != nil {
				return handleErr(err)
			}
			printOutput(gems)
			return nil
		},
	}
}

func gemsByOwnerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gems-by-owner [handle]",
		Short: "List gems owned by a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			gems, err := newRepo().GetGemsByOwner(ctx, args[0])
			if err != nil {
				return handleErr(err)
			}
			printOutput(gems)
			return nil
		},
	}
}

func gemOwnersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gem-owners [gem]",
		Short: "List owners of a gem",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			owners, err := newRepo().GetGemOwners(ctx, args[0])
			if err != nil {
				return handleErr(err)
			}
			printOutput(owners)
			return nil
		},
	}
}

func attestationsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "attestations [gem] [version]",
		Short: "Get sigstore attestations for a gem version",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			a, err := newRepo().GetAttestations(ctx, args[0], args[1])
			if err != nil {
				return handleErr(err)
			}
			printOutput(a)
			return nil
		},
	}
}

func mfaStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mfa-status",
		Short: "Check MFA status for the authenticated user (requires --token)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			s, err := newRepo().GetMFAStatus(ctx)
			if err != nil {
				return handleErr(err)
			}
			printOutput(s)
			return nil
		},
	}
}

func timeframeCmd() *cobra.Command {
	var from, to string
	c := &cobra.Command{
		Use:   "timeframe",
		Short: "Get versions published within a time range (RFC3339 dates)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fromT, err := time.Parse(time.RFC3339, from)
			if err != nil {
				return fmt.Errorf("invalid --from (use RFC3339, e.g. 2024-01-01T00:00:00Z): %w", err)
			}
			toT, err := time.Parse(time.RFC3339, to)
			if err != nil {
				return fmt.Errorf("invalid --to (use RFC3339, e.g. 2024-12-31T23:59:59Z): %w", err)
			}
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			versions, err := newRepo().GetTimeFrameVersions(ctx, fromT, toT)
			if err != nil {
				return handleErr(err)
			}
			printOutput(versions)
			return nil
		},
	}
	c.Flags().StringVar(&from, "from", "", "Start time (RFC3339)")
	c.Flags().StringVar(&to, "to", "", "End time (RFC3339)")
	c.MarkFlagRequired("from")
	c.MarkFlagRequired("to")
	return c
}

// ===== Output helpers =====

// prettyPrint renders common model types in a human-friendly format.
func prettyPrint(v interface{}) {
	switch val := v.(type) {
	case *models.PackageInformation:
		fmt.Printf("Package name: %s\n", val.Name)
		fmt.Printf("Version: %s\n", val.Version)
		fmt.Printf("Authors: %s\n", val.Authors)
		fmt.Printf("Downloads: %d\n", val.Downloads)
		fmt.Printf("Platform: %s\n", val.Platform)
		fmt.Printf("Homepage: %s\n", val.HomepageURI)
		fmt.Printf("Licenses: %v\n", val.Licenses)
		fmt.Printf("Description: %s\n", val.Info)
		if val.Metadata.SourceCodeURI != "" {
			fmt.Printf("Source code: %s\n", val.Metadata.SourceCodeURI)
		}
	default:
		fmt.Printf("%+v\n", val)
	}
}

// handleErr maps SDK error types to readable CLI messages.
func handleErr(err error) error {
	if repository.IsNotFound(err) {
		return fmt.Errorf("not found (404)")
	}
	if repository.IsRateLimited(err) {
		return fmt.Errorf("rate limited (429), retry later")
	}
	if repository.IsUnauthorized(err) {
		return fmt.Errorf("unauthorized (401/403), check --token")
	}
	return err
}
