// RubyGems CLI
//
// Query RubyGems repository for package info, search, version list,
// dependencies, etc. Supports official source and multiple mirror sources.
//
// Usage examples:
//
//	# Get package info
//	rubygems-cli -get -gem rails
//
//	# Search packages
//	rubygems-cli -search -query rails -limit 10
//
//	# Get version list
//	rubygems-cli -versions -gem rails -limit 20
//
//	# Get dependency info
//	rubygems-cli -deps -gem rails
//
//	# Get reverse dependencies
//	rubygems-cli -rdeps -gem rails -limit 50
//
//	# Output in JSON format
//	rubygems-cli -get -gem rails -json
//
//	# Use mirror source
//	rubygems-cli -get -gem rails -mirror ruby-china
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/cache"
	"github.com/scagogogo/rubygems-skills/pkg/install"
	"github.com/scagogogo/rubygems-skills/pkg/models"
	"github.com/scagogogo/rubygems-skills/pkg/repository"
)

var (
	// Operation commands
	get        = flag.Bool("get", false, "Get package info")
	search     = flag.Bool("search", false, "Search packages")
	versions   = flag.Bool("versions", false, "Get version list")
	deps       = flag.Bool("deps", false, "Get dependency info")
	rdeps      = flag.Bool("rdeps", false, "Get reverse dependency info")
	installCmd = flag.Bool("install", false, "Auto-install Ruby/RubyGems")
	help       = flag.Bool("help", false, "Show help")

	// Parameters
	gem    = flag.String("gem", "", "Package name")
	query  = flag.String("query", "", "Search query string")
	limit  = flag.Int("limit", 10, "Result limit")
	mirror = flag.String("mirror", "default", "Mirror selection (default, ruby-china, tsinghua, aliyun)")

	// Output settings
	useJSON  = flag.Bool("json", false, "Output in JSON format")
	useCache = flag.Bool("cache", false, "Enable caching")

	// Install options
	installForce     = flag.Bool("force", false, "Force reinstall")
	installNoDev     = flag.Bool("no-dev", false, "Skip dev headers")
	installNoBundler = flag.Bool("no-bundler", false, "Skip bundler")
	installNoUpdate  = flag.Bool("no-update", false, "Skip package index update")
	installNoSudo    = flag.Bool("no-sudo", false, "Don't use sudo")
)

func main() {
	flag.Parse()

	if *help {
		printHelp()
		return
	}

	// Create repository client
	repo := createRepository(*mirror)

	if *useCache {
		memCache := cache.NewMemoryCache(5*time.Minute, 30*time.Minute)
		cachedRepo := repository.NewCachedRepository(repo, 5*time.Minute, memCache)
		defer cachedRepo.Close()
		repo = cachedRepo
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Execute the corresponding operation based on command
	switch {
	case *get:
		handleGet(ctx, repo)
	case *search:
		handleSearch(ctx, repo)
	case *versions:
		handleVersions(ctx, repo)
	case *deps:
		handleDeps(ctx, repo)
	case *rdeps:
		handleRdeps(ctx, repo)
	case *installCmd:
		handleInstall(ctx)
	default:
		// If no command specified but gem is provided, default to get
		if *gem != "" {
			handleGet(ctx, repo)
		} else {
			fmt.Fprintln(os.Stderr, "Error: please specify an operation or package name")
			fmt.Fprintln(os.Stderr, "Use -help for help")
			os.Exit(1)
		}
	}
}

func createRepository(mirror string) repository.Repository {
	switch mirror {
	case "ruby-china":
		return repository.NewRubyChinaRepository()
	case "tsinghua":
		return repository.NewTSingHuaRepository()
	case "aliyun":
		return repository.NewAliYunRepository()
	default:
		return repository.NewRepository()
	}
}

func handleGet(ctx context.Context, repo repository.Repository) {
	if *gem == "" {
		fmt.Fprintln(os.Stderr, "Error: -gem parameter is required to get package info")
		os.Exit(1)
	}

	pkg, err := repo.GetPackage(ctx, *gem)
	if err != nil {
		if repository.IsNotFound(err) {
			fmt.Fprintf(os.Stderr, "Package '%s' not found\n", *gem)
		} else if repository.IsRateLimited(err) {
			fmt.Fprintln(os.Stderr, "API rate limited, please retry later")
		} else {
			fmt.Fprintf(os.Stderr, "Failed to get package info: %v\n", err)
		}
		os.Exit(1)
	}

	printOutput(pkg)
}

func handleSearch(ctx context.Context, repo repository.Repository) {
	if *query == "" {
		fmt.Fprintln(os.Stderr, "Error: -query parameter is required for search")
		os.Exit(1)
	}

	results, err := repo.Search(ctx, *query, 1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Search failed: %v\n", err)
		os.Exit(1)
	}

	// Limit output count
	if len(results) > *limit {
		results = results[:*limit]
	}

	if *useJSON {
		printOutput(results)
		return
	}

	fmt.Printf("Search results for '%s' (top %d):\n", *query, len(results))
	for i, pkg := range results {
		fmt.Printf("%d. %s (version: %s, downloads: %d)\n",
			i+1, pkg.Name, pkg.Version, pkg.Downloads)
	}
}

func handleVersions(ctx context.Context, repo repository.Repository) {
	if *gem == "" {
		fmt.Fprintln(os.Stderr, "Error: -gem parameter is required to get version list")
		os.Exit(1)
	}

	versions, err := repo.GetGemVersions(ctx, *gem)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get version list: %v\n", err)
		os.Exit(1)
	}

	// Limit output count
	if len(versions) > *limit {
		versions = versions[:*limit]
	}

	if *useJSON {
		printOutput(versions)
		return
	}

	fmt.Printf("Version list for %s (top %d):\n", *gem, len(versions))
	for i, v := range versions {
		fmt.Printf("%d. %s (downloads: %d, released: %s)\n",
			i+1, v.Number, v.DownloadsCount, v.CreatedAt.Format("2006-01-02"))
	}
}

func handleDeps(ctx context.Context, repo repository.Repository) {
	if *gem == "" {
		fmt.Fprintln(os.Stderr, "Error: -gem parameter is required to get dependency info")
		os.Exit(1)
	}

	deps, err := repo.GetDependencies(ctx, *gem)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get dependency info: %v\n", err)
		os.Exit(1)
	}

	if *useJSON {
		printOutput(deps)
		return
	}

	fmt.Printf("Dependencies for %s:\n", *gem)
	runtimeDeps := make([]*models.DependencyInfo, 0)
	devDeps := make([]*models.DependencyInfo, 0)

	for _, d := range deps {
		switch d.DependentType {
		case "runtime":
			runtimeDeps = append(runtimeDeps, d)
		case "development":
			devDeps = append(devDeps, d)
		}
	}

	if len(runtimeDeps) > 0 {
		fmt.Println("  Runtime dependencies:")
		for _, d := range runtimeDeps {
			fmt.Printf("    - %s %s\n", d.Name, d.Requirements)
		}
	}

	if len(devDeps) > 0 {
		fmt.Println("  Development dependencies:")
		for _, d := range devDeps {
			fmt.Printf("    - %s %s\n", d.Name, d.Requirements)
		}
	}

	if len(runtimeDeps) == 0 && len(devDeps) == 0 {
		fmt.Println("  No dependencies")
	}
}

func handleRdeps(ctx context.Context, repo repository.Repository) {
	if *gem == "" {
		fmt.Fprintln(os.Stderr, "Error: -gem parameter is required to get reverse dependencies")
		os.Exit(1)
	}

	rdeps, err := repo.GetReverseDependencies(ctx, *gem)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get reverse dependencies: %v\n", err)
		os.Exit(1)
	}

	// Limit output count
	if len(rdeps) > *limit {
		rdeps = rdeps[:*limit]
	}

	if *useJSON {
		printOutput(rdeps)
		return
	}

	fmt.Printf("Packages depending on %s (top %d):\n", *gem, len(rdeps))
	for i, d := range rdeps {
		fmt.Printf("%d. %s\n", i+1, d)
	}
}

func printOutput(v interface{}) {
	if *useJSON {
		bytes, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "JSON serialization failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(bytes))
	} else {
		// For default output, use formatted JSON to display struct
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
}

func handleInstall(ctx context.Context) {
	opts := install.NewInstallOptions().
		WithForceReinstall(*installForce).
		WithDevHeaders(!*installNoDev).
		WithBundler(!*installNoBundler).
		WithUpdatePackageIndex(!*installNoUpdate).
		WithSudo(!*installNoSudo)

	installer := install.NewInstaller(opts)

	// Detect and display platform info
	platform, err := installer.DetectPlatform()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Platform detection failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Detected platform: %s\n", platform)

	// Check if already installed
	installed, info, _ := installer.IsInstalled()
	if installed && !*installForce {
		fmt.Printf("Ruby already installed: %s (gem: %s)\n", info.RubyVersion, info.GemVersion)
		fmt.Println("Use -force to force reinstall")
		return
	}

	if installed && *installForce {
		fmt.Printf("Ruby already installed: %s, force reinstalling...\n", info.RubyVersion)
	}

	// Execute installation
	fmt.Println("Installing Ruby/RubyGems...")
	result, err := installer.Install(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Installation failed: %v\n", err)
		os.Exit(1)
	}

	// Display installation result
	if result.AlreadyInstalled {
		fmt.Printf("Ruby already installed: %s (gem: %s)\n", result.RubyVersion, result.GemVersion)
	} else {
		fmt.Println("Installation successful!")
		fmt.Printf("  Ruby version: %s\n", result.RubyVersion)
		fmt.Printf("  gem version: %s\n", result.GemVersion)
		fmt.Printf("  Ruby path: %s\n", result.RubyPath)
		fmt.Printf("  gem path: %s\n", result.GemPath)
		fmt.Printf("  Package manager: %s\n", result.PackageManager)
		if len(result.CommandsRun) > 0 {
			fmt.Println("  Commands executed:")
			for _, cmd := range result.CommandsRun {
				fmt.Printf("    %s\n", cmd)
			}
		}
	}
}

func printHelp() {
	fmt.Println("RubyGems CLI - query RubyGems repository info")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  rubygems-cli [command] [parameters]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  -get        Get package info")
	fmt.Println("  -search     Search packages")
	fmt.Println("  -versions   Get version list")
	fmt.Println("  -deps       Get dependency info")
	fmt.Println("  -rdeps      Get reverse dependency info")
	fmt.Println("  -install    Auto-install Ruby/RubyGems")
	fmt.Println()
	fmt.Println("Parameters:")
	fmt.Println("  -gem        Package name")
	fmt.Println("  -query      Search query string")
	fmt.Println("  -limit      Result limit (default: 10)")
	fmt.Println("  -mirror     Mirror selection (default, ruby-china, tsinghua, aliyun)")
	fmt.Println("  -json       Output in JSON format")
	fmt.Println("  -cache      Enable caching")
	fmt.Println()
	fmt.Println("Install options:")
	fmt.Println("  -force       Force reinstall")
	fmt.Println("  -no-dev      Skip dev headers (ruby-dev/ruby-devel)")
	fmt.Println("  -no-bundler  Skip bundler")
	fmt.Println("  -no-update   Skip package index update")
	fmt.Println("  -no-sudo     Don't use sudo")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  rubygems-cli -get -gem rails")
	fmt.Println("  rubygems-cli -get -gem rails -json")
	fmt.Println("  rubygems-cli -search -query rails -limit 10")
	fmt.Println("  rubygems-cli -versions -gem rails -limit 20")
	fmt.Println("  rubygems-cli -deps -gem rails")
	fmt.Println("  rubygems-cli -rdeps -gem rails -limit 50")
	fmt.Println("  rubygems-cli -get -gem rails -mirror ruby-china")
	fmt.Println("  rubygems-cli -get -gem rails -cache")
	fmt.Println("  rubygems-cli -install")
	fmt.Println("  rubygems-cli -install -force")
	fmt.Println("  rubygems-cli -install -no-dev -no-bundler")
}
