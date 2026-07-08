package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/scagogogo/rubygems-skills/pkg/cache"
	"github.com/scagogogo/rubygems-skills/pkg/repository"
)

// Global flag values shared across all subcommands.
var (
	flagMirror   string
	flagServer   string // custom server URL (overrides --mirror)
	flagToken    string
	flagProxy    string
	flagTimeout  int
	flagJSON     bool
	flagCache    bool
	flagCacheTTL int

	// Retry flags
	flagRetry          bool
	flagRetryAttempts  int
	flagRetryWait      int
	flagRetryBackoff   bool
)

// persistentFlags registers global flags on the given command (used for root).
func persistentFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&flagMirror, "mirror", "default", "Mirror source: default, ruby-china, tsinghua, aliyun")
	cmd.PersistentFlags().StringVar(&flagServer, "server", "", "Custom gem server URL (overrides --mirror)")
	cmd.PersistentFlags().StringVar(&flagToken, "token", "", "API token (raises rate-limit quota, required for write/authed ops)")
	cmd.PersistentFlags().StringVar(&flagProxy, "proxy", "", "HTTP proxy URL")
	cmd.PersistentFlags().IntVar(&flagTimeout, "timeout", 30, "Request timeout in seconds")
	cmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output in JSON format")
	cmd.PersistentFlags().BoolVar(&flagCache, "cache", false, "Enable in-memory cache")
	cmd.PersistentFlags().IntVar(&flagCacheTTL, "cache-ttl", 5, "Cache TTL in minutes (only with --cache)")

	cmd.PersistentFlags().BoolVar(&flagRetry, "retry", false, "Enable retry with backoff")
	cmd.PersistentFlags().IntVar(&flagRetryAttempts, "retry-attempts", 3, "Max retry attempts (only with --retry)")
	cmd.PersistentFlags().IntVar(&flagRetryWait, "retry-wait", 1, "Initial retry wait in seconds (only with --retry)")
	cmd.PersistentFlags().BoolVar(&flagRetryBackoff, "retry-backoff", true, "Use exponential backoff (only with --retry)")
}

// buildOptions constructs *repository.Options from global flags.
func buildOptions() *repository.Options {
	opts := repository.NewOptions()
	if flagToken != "" {
		opts.SetToken(flagToken)
	}
	if flagProxy != "" {
		opts.SetProxy(flagProxy)
	}
	if flagRetry {
		opts.SetRetryOptions(repository.NewDefaultRetryOptions().
			WithMaxAttempts(flagRetryAttempts).
			WithWaitTime(time.Duration(flagRetryWait) * time.Second).
			WithExponentialBackoff(flagRetryBackoff))
	}
	return opts
}

// newRepo builds a read Repository from global flags, optionally wrapped with cache.
func newRepo() repository.Repository {
	return newRepoFunc()
}

// newRepoFunc is the factory used by newRepo. Tests swap it to inject a stubbed
// Repository (e.g. one pointed at an httptest.Server) without touching the real
// flag-driven construction.
var newRepoFunc = defaultNewRepo

// defaultNewRepo is the flag-driven Repository construction used in production.
func defaultNewRepo() repository.Repository {
	var repo repository.Repository
	switch {
	case flagServer != "":
		repo = repository.NewCustomRepository(flagServer)
	case flagMirror == "ruby-china":
		repo = repository.NewRubyChinaRepository()
	case flagMirror == "tsinghua":
		repo = repository.NewTSingHuaRepository()
	case flagMirror == "aliyun":
		repo = repository.NewAliYunRepository()
	default:
		repo = repository.NewRepository(buildOptions())
	}

	if flagCache {
		ttl := time.Duration(flagCacheTTL) * time.Minute
		mem := cache.NewMemoryCache(ttl, ttl*2)
		repo = repository.NewCachedRepository(repo, ttl, mem)
	}
	return repo
}

// newWriteRepo builds a WriteRepository from global flags (requires --token or basic auth).
func newWriteRepo() *repository.WriteRepositoryImpl {
	return newWriteRepoFunc()
}

// newWriteRepoFunc is the factory used by newWriteRepo; tests may swap it.
var newWriteRepoFunc = defaultNewWriteRepo

// defaultNewWriteRepo is the flag-driven WriteRepository construction used in production.
func defaultNewWriteRepo() *repository.WriteRepositoryImpl {
	return repository.NewWriteRepository(buildOptions())
}

// ctxWithTimeout builds a context from the global --timeout flag.
func ctxWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(flagTimeout)*time.Second)
}

// printJSON pretty-prints v as JSON and exits non-zero on error.
func printJSON(v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Printf("JSON serialization failed: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Println(string(b))
}

// printOutput prints v either as JSON (when --json) or via the pretty printer.
func printOutput(v interface{}) {
	if flagJSON {
		printJSON(v)
		return
	}
	prettyPrint(v)
}

// exitCode lets RunE handlers signal a non-zero exit without os.Exit (which would
// skip cobra's deferred output flush). main() reads this after Execute.
var exitCode int

// markRequired flags the named flags as required on c. It panics if a flag is
// missing — these are static, compile-time-known flag names, so a failure is a
// programming error, not a runtime condition. Wrapping c.MarkFlagRequired this
// way satisfies errcheck.
func markRequired(c *cobra.Command, names ...string) {
	for _, n := range names {
		if err := c.MarkFlagRequired(n); err != nil {
			panic(fmt.Errorf("mark flag %q required: %w", n, err))
		}
	}
}
