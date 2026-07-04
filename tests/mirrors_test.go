package tests

import (
	"context"
	"testing"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/repository"
	"github.com/stretchr/testify/assert"
)

// Test whether all mirror sources work properly
func TestAllMirrors(t *testing.T) {
	// Skip the test if running in short mode (CI environments)
	if testing.Short() {
		t.Skip("Skipping mirror tests in short mode")
	}

	// Create context and set timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Common Gem packages to test
	testGems := []string{
		"rails",
		"rack",
		"activesupport",
		"rake",
	}

	// Create repositories with different mirror sources
	repos := map[string]repository.Repository{
		"Official":  repository.NewRepository(),
		"RubyChina": repository.NewRubyChinaRepository(),
		"TsingHua":  repository.NewTSingHuaRepository(),
		"AliYun":    repository.NewAliYunRepository(),
	}

	// Iterate over all mirror sources
	for name, repo := range repos {
		t.Run(name, func(t *testing.T) {
			// Test get package info
			for _, gemName := range testGems {
				t.Run("GetPackage-"+gemName, func(t *testing.T) {
					pkg, err := repo.GetPackage(ctx, gemName)
					assert.NoError(t, err, "get package info failed")
					assert.NotNil(t, pkg, "package info is empty")
					if pkg != nil {
						assert.Equal(t, gemName, pkg.Name, "package name mismatch")
						assert.NotEmpty(t, pkg.Version, "version info is empty")
					} else {
						// Skip further assertions if pkg is nil
						t.SkipNow()
					}
				})
			}

			// Test search
			t.Run("Search", func(t *testing.T) {
				results, err := repo.Search(ctx, "rails", 1)
				assert.NoError(t, err, "search failed")
				assert.NotEmpty(t, results, "search results are empty")
				// Skip further assertions if results is empty
				if len(results) == 0 {
					t.SkipNow()
				}
			})

			// Test get download stats
			t.Run("Downloads", func(t *testing.T) {
				stats, err := repo.Downloads(ctx)
				assert.NoError(t, err, "get download stats failed")
				assert.NotNil(t, stats, "download stats are empty")
				// Skip further assertions if stats is nil
				if stats == nil {
					t.SkipNow()
				}
				assert.Greater(t, stats.TotalDownloads, 0, "download count should be greater than 0")
			})

			// Test get latest gems
			t.Run("LatestGems", func(t *testing.T) {
				gems, err := repo.LatestGems(ctx)
				assert.NoError(t, err, "get latest gems failed")
				assert.NotEmpty(t, gems, "latest gems list is empty")
				// Skip further assertions if gems is empty
				if len(gems) == 0 {
					t.SkipNow()
				}
			})
		})
	}
}
