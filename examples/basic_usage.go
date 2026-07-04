package main

import (
	"context"
	"fmt"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/repository"
)

func main() {
	// Create a context with a 5-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Basic usage
	basicUsage(ctx)

	// Use mirror source
	useMirror(ctx)

	// Use retry strategy
	useRetry(ctx)

	// Use token authentication
	useToken(ctx)

	// Search package
	searchPackage(ctx)

	// Get package version list
	getVersions(ctx)

	// Get download statistics
	getDownloadStats(ctx)
}

func basicUsage(ctx context.Context) {
	fmt.Println("=== Basic Usage ===")
	repo := repository.NewRepository()

	pkg, err := repo.GetPackage(ctx, "rails")
	if err != nil {
		fmt.Printf("failed to get package info: %v\n", err)
		return
	}

	fmt.Printf("package name: %s\n", pkg.Name)
	fmt.Printf("version: %s\n", pkg.Version)
	fmt.Printf("authors: %s\n", pkg.Authors)
	fmt.Printf("summary: %s\n", pkg.Info)
	fmt.Printf("downloads: %d\n", pkg.Downloads)
	fmt.Printf("homepage: %s\n", pkg.HomepageURI)
	fmt.Println()
}

func useMirror(ctx context.Context) {
	fmt.Println("=== Use Mirror Source ===")

	// Use Ruby China mirror source
	repo := repository.NewRubyChinaRepository()

	pkg, err := repo.GetPackage(ctx, "rails")
	if err != nil {
		fmt.Printf("failed to get package info: %v\n", err)
		return
	}

	fmt.Printf("package name: %s\n", pkg.Name)
	fmt.Printf("version: %s\n", pkg.Version)
	fmt.Println()

	// You can also try other mirror sources
	// repo = repository.NewTSingHuaRepository()
	// repo = repository.NewAliYunRepository()
}

func useRetry(ctx context.Context) {
	fmt.Println("=== Use Retry Strategy ===")

	// Custom retry strategy
	retryOptions := repository.NewDefaultRetryOptions().
		WithMaxAttempts(3).
		WithWaitTime(1 * time.Second)

	options := repository.NewOptions().SetRetryOptions(retryOptions)
	repo := repository.NewRepository(options)

	pkg, err := repo.GetPackage(ctx, "rails")
	if err != nil {
		fmt.Printf("failed to get package info: %v\n", err)
		return
	}

	fmt.Printf("package name: %s\n", pkg.Name)
	fmt.Printf("version: %s\n", pkg.Version)
	fmt.Println()
}

func useToken(ctx context.Context) {
	fmt.Println("=== Use Token Authentication ===")

	// Please replace with your actual token
	options := repository.NewOptions().SetToken("your-api-token")
	repo := repository.NewRepository(options)

	// This is just an example, no actual token is used
	fmt.Println("note: please replace with your actual token")
	fmt.Println("Repository object with token authentication created: ", repo != nil)
	fmt.Println()
}

func searchPackage(ctx context.Context) {
	fmt.Println("=== Search Package ===")
	repo := repository.NewRepository()

	// Search packages containing "rails", first page
	packages, err := repo.Search(ctx, "rails", 1)
	if err != nil {
		fmt.Printf("search failed: %v\n", err)
		return
	}

	fmt.Printf("found %d results:\n", len(packages))
	for i, pkg := range packages {
		if i >= 5 {
			fmt.Println("... more results omitted ...")
			break
		}
		fmt.Printf("%d. %s (version: %s, downloads: %d)\n", i+1, pkg.Name, pkg.Version, pkg.Downloads)
	}
	fmt.Println()
}

func getVersions(ctx context.Context) {
	fmt.Println("=== Get Version List ===")
	repo := repository.NewRepository()

	versions, err := repo.GetGemVersions(ctx, "rails")
	if err != nil {
		fmt.Printf("failed to get versions: %v\n", err)
		return
	}

	fmt.Printf("rails has %d versions:\n", len(versions))
	for i, ver := range versions {
		if i >= 5 {
			fmt.Println("... more versions omitted ...")
			break
		}
		fmt.Printf("%d. %s (downloads: %d, released: %s)\n",
			i+1, ver.Number, ver.DownloadsCount, ver.CreatedAt.Format("2006-01-02"))
	}
	fmt.Println()
}

func getDownloadStats(ctx context.Context) {
	fmt.Println("=== Download Statistics ===")
	repo := repository.NewRepository()

	// Get repository total downloads
	repoStats, err := repo.Downloads(ctx)
	if err != nil {
		fmt.Printf("failed to get download statistics: %v\n", err)
		return
	}

	fmt.Printf("RubyGems repository total downloads: %d\n", repoStats.TotalDownloads)

	// Get downloads for a specific version
	verStats, err := repo.VersionDownloads(ctx, "rails", "7.0.5")
	if err != nil {
		fmt.Printf("failed to get version download statistics: %v\n", err)
		return
	}

	fmt.Printf("rails 7.0.5 version downloads: %d\n", verStats.VersionDownloads)
	fmt.Printf("rails total downloads: %d\n", verStats.TotalDownloads)
	fmt.Println()
}
