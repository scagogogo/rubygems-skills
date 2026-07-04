package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/scagogogo/rubygems-skills/pkg/repository"
)

func bulkCommonFlags(c *cobra.Command, concurrency *int) {
	c.Flags().IntVar(concurrency, "concurrency", 5, "Max concurrency")
}

// parseGems accepts multiple positional args, or a single comma-separated arg.
func parseGems(args []string) []string {
	var gems []string
	for _, a := range args {
		for _, g := range strings.Split(a, ",") {
			if g = strings.TrimSpace(g); g != "" {
				gems = append(gems, g)
			}
		}
	}
	return gems
}

func bulkGetCmd() *cobra.Command {
	var concurrency int
	c := &cobra.Command{
		Use:   "bulk-get [gems...]",
		Short: "Bulk get package info for multiple gems",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gems := parseGems(args)
			opts := repository.NewBulkOptions().WithMaxConcurrency(concurrency)
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			results := newRepo().BulkGetPackages(ctx, gems, opts)
			if flagJSON {
				printOutput(results)
				return nil
			}
			for _, r := range results {
				if r.Error != nil {
					fmt.Printf("%s: FAILED: %v\n", r.Key, r.Error)
					continue
				}
				fmt.Printf("%s: %s %s (%d downloads)\n", r.Key, r.Value.Name, r.Value.Version, r.Value.Downloads)
			}
			return nil
		},
	}
	bulkCommonFlags(c, &concurrency)
	return c
}

func bulkVersionsCmd() *cobra.Command {
	var concurrency int
	c := &cobra.Command{
		Use:   "bulk-versions [gems...]",
		Short: "Bulk get version info for multiple gems",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gems := parseGems(args)
			opts := repository.NewBulkOptions().WithMaxConcurrency(concurrency)
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			results := newRepo().BulkGetVersions(ctx, gems, opts)
			if flagJSON {
				printOutput(results)
				return nil
			}
			for _, r := range results {
				if r.Error != nil {
					fmt.Printf("%s: FAILED: %v\n", r.Key, r.Error)
					continue
				}
				fmt.Printf("%s: %d versions\n", r.Key, len(r.Value))
			}
			return nil
		},
	}
	bulkCommonFlags(c, &concurrency)
	return c
}

func bulkDepsCmd() *cobra.Command {
	var concurrency int
	c := &cobra.Command{
		Use:   "bulk-deps [gems...]",
		Short: "Bulk get dependency info for multiple gems",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gems := parseGems(args)
			opts := repository.NewBulkOptions().WithMaxConcurrency(concurrency)
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			results := newRepo().BulkGetDependencies(ctx, gems, opts)
			if flagJSON {
				printOutput(results)
				return nil
			}
			for _, r := range results {
				if r.Error != nil {
					fmt.Printf("%s: FAILED: %v\n", r.Key, r.Error)
					continue
				}
				fmt.Printf("%s: %d dependencies\n", r.Key, len(r.Value))
			}
			return nil
		},
	}
	bulkCommonFlags(c, &concurrency)
	return c
}

func bulkRdepsCmd() *cobra.Command {
	var concurrency int
	c := &cobra.Command{
		Use:   "bulk-rdeps [gems...]",
		Short: "Bulk get reverse dependencies for multiple gems",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gems := parseGems(args)
			opts := repository.NewBulkOptions().WithMaxConcurrency(concurrency)
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			results := newRepo().BulkGetReverseDependencies(ctx, gems, opts)
			if flagJSON {
				printOutput(results)
				return nil
			}
			for _, r := range results {
				if r.Error != nil {
					fmt.Printf("%s: FAILED: %v\n", r.Key, r.Error)
					continue
				}
				fmt.Printf("%s: %d reverse dependencies\n", r.Key, len(r.Value))
			}
			return nil
		},
	}
	bulkCommonFlags(c, &concurrency)
	return c
}
