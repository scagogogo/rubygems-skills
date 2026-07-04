package main

import (
	"context"
	"fmt"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/models"
	"github.com/scagogogo/rubygems-skills/pkg/repository"
)

func main() {
	// Create context, set timeout to 30 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create base repository instance
	repo := repository.NewRepository()

	// Define the list of gems to query in bulk
	gems := []string{
		"rails",
		"rack",
		"activesupport",
		"rake",
		"concurrent-ruby",
		"i18n",
		"minitest",
		"tzinfo",
		"nokogiri",
		"zeitwerk",
	}

	fmt.Println("start bulk get package info...")
	startTime := time.Now()

	// Create bulk operation options, set max concurrency to 5
	options := repository.NewBulkOptions().WithMaxConcurrency(5)

	// Bulk get package info
	results := repo.BulkGetPackages(ctx, gems, options)

	// Calculate elapsed time
	duration := time.Since(startTime)
	fmt.Printf("bulk get done, elapsed: %v\n\n", duration)

	// Display results
	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("failed to get %s: %v\n", result.Key, result.Error)
			continue
		}

		pkg := result.Value
		fmt.Printf("package name: %s\n", pkg.Name)
		fmt.Printf("  current version: %s\n", pkg.Version)
		fmt.Printf("  homepage: %s\n", pkg.HomepageURI)
		fmt.Printf("  downloads: %d\n", pkg.Downloads)
		fmt.Printf("  description: %s\n\n", pkg.Info)
	}

	// Compare with sequential execution time
	fmt.Println("start sequential get package info for comparison...")
	startTime = time.Now()

	// Sequential get package info
	sequentialResults := make([]*repository.BulkResult[*models.PackageInformation], 0, len(gems))
	for _, gemName := range gems {
		pkg, err := repo.GetPackage(ctx, gemName)
		sequentialResults = append(sequentialResults, &repository.BulkResult[*models.PackageInformation]{
			Key:   gemName,
			Value: pkg,
			Error: err,
		})
	}

	// Calculate elapsed time
	sequentialDuration := time.Since(startTime)
	fmt.Printf("sequential get done, elapsed: %v\n", sequentialDuration)
	fmt.Printf("concurrent processing is %.2f times faster than sequential processing\n\n", float64(sequentialDuration)/float64(duration))

	// Demonstrate bulk get version info
	fmt.Println("start bulk get version info...")
	startTime = time.Now()

	// Select the first 5 gems for version query
	selectedGems := gems[:5]
	versionResults := repo.BulkGetVersions(ctx, selectedGems, options)

	// Calculate elapsed time
	duration = time.Since(startTime)
	fmt.Printf("bulk get version info done, elapsed: %v\n\n", duration)

	// Display version info results
	for _, result := range versionResults {
		if result.Error != nil {
			fmt.Printf("failed to get version info for %s: %v\n", result.Key, result.Error)
			continue
		}

		versions := result.Value
		fmt.Printf("package name: %s\n", result.Key)
		fmt.Printf("  version count: %d\n", len(versions))
		if len(versions) > 0 {
			fmt.Printf("  latest few versions: %s", versions[0].Number)
			for i := 1; i < min(5, len(versions)); i++ {
				fmt.Printf(", %s", versions[i].Number)
			}
			fmt.Println()
		}
		fmt.Println()
	}

	// Demonstrate bulk get dependency info
	fmt.Println("start bulk get dependency info...")
	startTime = time.Now()

	// Get dependency info
	dependencyResults := repo.BulkGetDependencies(ctx, selectedGems, options)

	// Calculate elapsed time
	duration = time.Since(startTime)
	fmt.Printf("bulk get dependency info done, elapsed: %v\n\n", duration)

	// Display dependency info results
	for _, result := range dependencyResults {
		if result.Error != nil {
			fmt.Printf("failed to get dependency info for %s: %v\n", result.Key, result.Error)
			continue
		}

		dependencies := result.Value
		fmt.Printf("package name: %s\n", result.Key)
		fmt.Printf("  dependency count: %d\n", len(dependencies))
		if len(dependencies) > 0 {
			fmt.Println("  partial dependency list:")
			for i := 0; i < min(5, len(dependencies)); i++ {
				dep := dependencies[i]
				fmt.Printf("    - %s (requirements: %s)\n", dep.Name, dep.Requirements)
			}
		}
		fmt.Println()
	}
}

// Go 1.20 and below do not have a built-in min function, implemented manually
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
