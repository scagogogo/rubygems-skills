// Command rubygems is a CLI for the RubyGems.org API, built on cobra.
//
// It exposes the full SDK — read queries, write operations, bulk operations,
// and the Ruby/RubyGems auto-installer — as subcommands with global flags for
// mirror selection, auth, proxy, cache, and retry.
//
// Quick examples:
//
//	rubygems get rails                      # package info
//	rubygems search rails --limit 10        # search
//	rubygems versions rails --limit 20      # version list
//	rubygems deps rails                     # dependencies
//	rubygems get rails --json               # JSON output
//	rubygems get rails --mirror ruby-china  # use a mirror
//	rubygems get rails --cache              # enable cache
//	rubygems install                        # auto-install Ruby/RubyGems
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := buildRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		exitCode = 1
	}
	os.Exit(exitCode)
}

// buildRootCmd assembles the root cobra command with all subcommands and global
// flags registered. Extracted from main() so tests can drive it via SetArgs and
// capture stdout/exitCode without spawning a process.
func buildRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "rubygems",
		Short: "RubyGems.org API CLI — query, search, publish, and auto-install",
		Long: `rubygems is a CLI for the RubyGems.org API.

It wraps the entire rubygems-skills SDK: package queries, search, versions,
downloads, dependencies, reverse dependencies, user/owner info, attestations,
webhooks, API key management, gem publishing, bulk operations, and a
cross-platform Ruby/RubyGems auto-installer.

Run ` + "`rubygems <command> --help`" + ` for details on a subcommand.
Global flags (mirror, token, proxy, cache, retry, json) apply to most commands.`,
		SilenceUsage: true,
	}
	persistentFlags(root)

	root.AddCommand(
		// read
		getCmd(),
		searchCmd(),
		autocompleteCmd(),
		versionsCmd(),
		latestVersionCmd(),
		versionDetailCmd(),
		versionContentsCmd(),
		downloadsCmd(),
		versionDownloadsCmd(),
		topDownloadsCmd(),
		depsCmd(),
		rdepsCmd(),
		versionRdepsCmd(),
		latestGemsCmd(),
		justUpdatedCmd(),
		userProfileCmd(),
		ownedGemsCmd(),
		gemsByOwnerCmd(),
		gemOwnersCmd(),
		attestationsCmd(),
		mfaStatusCmd(),
		timeframeCmd(),
		// bulk
		bulkGetCmd(),
		bulkVersionsCmd(),
		bulkDepsCmd(),
		bulkRdepsCmd(),
		// write
		pushCmd(),
		yankCmd(),
		addOwnerCmd(),
		removeOwnerCmd(),
		updateOwnerCmd(),
		listWebhooksCmd(),
		createWebhookCmd(),
		deleteWebhookCmd(),
		fireWebhookCmd(),
		getAPIKeyCmd(),
		createAPIKeyCmd(),
		updateAPIKeyCmd(),
		myProfileCmd(),
		// install
		installCmd(),
		platformCmd(),
	)
	return root
}
