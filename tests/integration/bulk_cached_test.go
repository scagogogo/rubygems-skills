package integration

import (
	"context"
	"testing"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/cache"
	"github.com/scagogogo/rubygems-skills/pkg/repository"
	"github.com/stretchr/testify/assert"
)

// Test repository bulk operations combined with cache
func TestBulkOperationsWithCache(t *testing.T) {
	// Create a context with a longer timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create memory cache and repository
	memCache := cache.NewMemoryCache(5*time.Minute, 30*time.Minute)
	baseRepo := repository.NewRepository()
	cachedRepo := repository.NewCachedRepository(baseRepo, 5*time.Minute, memCache)
	defer cachedRepo.Close()

	// Prepare test data
	gems := []string{
		"rails",
		"rack",
		"activesupport",
		"rake",
		"nokogiri",
	}

	// Create bulk operation options
	options := repository.NewBulkOptions().WithMaxConcurrency(3)

	// Test bulk operations first, then get from cache
	t.Run("cache get after bulk operations", func(t *testing.T) {
		// Clear cache
		cachedRepo.ClearCache()

		// First use base repository to bulk get
		startTime := time.Now()
		results1 := baseRepo.BulkGetPackages(ctx, gems, options)
		duration1 := time.Since(startTime)

		// Verify results
		assert.Equal(t, len(gems), len(results1), "result count should match request count")
		for _, result := range results1 {
			assert.NoError(t, result.Error, "getting package %s should not return error", result.Key)
			assert.NotNil(t, result.Value, "package info returned for %s should not be nil", result.Key)

			// Manually cache result
			if result.Error == nil {
				cacheKey := "package:" + result.Key
				memCache.SetWithExpiration(cacheKey, result.Value, 5*time.Minute)
			}
		}

		// Wait a moment to ensure it is not network jitter causing the speed difference
		time.Sleep(500 * time.Millisecond)

		// Use cached repository to get one by one
		startTime = time.Now()
		for _, gemName := range gems {
			pkg, err := cachedRepo.GetPackage(ctx, gemName)
			assert.NoError(t, err, "getting package %s from cache should not return error", gemName)
			assert.NotNil(t, pkg, "package info returned for %s from cache should not be nil", gemName)
		}
		duration2 := time.Since(startTime)

		// Getting from cache should be significantly faster than API get
		t.Logf("bulk API get duration: %v, cache get one by one duration: %v", duration1, duration2)
		assert.True(t, duration2 < duration1, "getting one by one from cache should be faster than bulk API get")
	})

	// Test cached repository with version info
	t.Run("version info cache", func(t *testing.T) {
		// Clear cache
		cachedRepo.ClearCache()

		// Use normal repository to get data
		startTime := time.Now()
		results1 := baseRepo.BulkGetVersions(ctx, gems[:3], options)
		duration1 := time.Since(startTime)

		// Verify results
		assert.Equal(t, 3, len(results1), "result count should match request count")
		for _, result := range results1 {
			assert.NoError(t, result.Error, "getting versions for package %s should not return error", result.Key)
			assert.NotNil(t, result.Value, "versions returned for %s should not be nil", result.Key)

			// Manually cache result
			if result.Error == nil {
				cacheKey := "versions:" + result.Key
				memCache.SetWithExpiration(cacheKey, result.Value, 5*time.Minute)
			}
		}

		// Wait a moment to ensure it is not network jitter causing the speed difference
		time.Sleep(500 * time.Millisecond)

		// Get data from cache
		startTime = time.Now()
		for _, gemName := range gems[:3] {
			versions, err := cachedRepo.GetGemVersions(ctx, gemName)
			assert.NoError(t, err, "getting versions for package %s from cache should not return error", gemName)
			assert.NotNil(t, versions, "versions returned for %s from cache should not be nil", gemName)
		}
		duration2 := time.Since(startTime)

		// Cache should be faster
		t.Logf("bulk get versions duration: %v, cache get versions duration: %v", duration1, duration2)
		assert.True(t, duration2 < duration1, "getting versions from cache should be faster than bulk get")
	})

	// Test reverse dependency get and cache
	t.Run("reverse dependency cache", func(t *testing.T) {
		// Clear cache
		cachedRepo.ClearCache()

		// Bulk get reverse dependencies
		startTime := time.Now()
		results := baseRepo.BulkGetReverseDependencies(ctx, gems[:2], options)
		duration1 := time.Since(startTime)

		// Manually cache results
		for _, result := range results {
			if result.Error == nil {
				cacheKey := "reverse_dependencies:" + result.Key
				memCache.SetWithExpiration(cacheKey, result.Value, 5*time.Minute)
			}
		}

		// Wait a moment
		time.Sleep(500 * time.Millisecond)

		// Get from cache
		startTime = time.Now()
		for _, gemName := range gems[:2] {
			deps, err := cachedRepo.GetReverseDependencies(ctx, gemName)
			assert.NoError(t, err, "getting reverse dependencies for package %s from cache should not return error", gemName)
			assert.NotNil(t, deps, "reverse dependencies returned for %s from cache should not be nil", gemName)
		}
		duration2 := time.Since(startTime)

		// Cache should be faster
		t.Logf("bulk get reverse dependencies duration: %v, cache get reverse dependencies duration: %v", duration1, duration2)
		assert.True(t, duration2 < duration1, "getting reverse dependencies from cache should be faster than bulk get")
	})
}

// Test using cache and bulk query together
func TestCacheStatsAndExpiration(t *testing.T) {
	// Skip long running test
	if testing.Short() {
		t.Skip("skip cache expiration test in short mode")
	}

	// Create a cache with a short cache period
	shortCache := cache.NewMemoryCache(2*time.Second, 1*time.Second)
	baseRepo := repository.NewRepository()
	cachedRepo := repository.NewCachedRepository(baseRepo, 2*time.Second, shortCache)
	defer cachedRepo.Close()

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test data
	gemName := "rails"

	// First get package info, manually cache
	pkg, err := baseRepo.GetPackage(ctx, gemName)
	assert.NoError(t, err, "getting package info should not return error")
	assert.NotNil(t, pkg, "package info should not be nil")

	// Manually cache
	cacheKey := "package:" + gemName
	shortCache.SetWithExpiration(cacheKey, pkg, 2*time.Second)

	// Verify cache status
	cacheStats1 := cachedRepo.GetCacheStats()
	assert.Equal(t, 1, cacheStats1, "there should be one item in cache")

	// Get from cache
	cachedPkg, err := cachedRepo.GetPackage(ctx, gemName)
	assert.NoError(t, err, "getting package info from cache should not return error")
	assert.Equal(t, pkg.Name, cachedPkg.Name, "cached package name should be the same as original package name")

	// Wait for cache to expire
	time.Sleep(3 * time.Second)

	// Cache should already be cleared
	cacheStats2 := cachedRepo.GetCacheStats()
	assert.Equal(t, 0, cacheStats2, "cache should be empty")

	// Get again, should re-fetch from API
	pkg2, err := cachedRepo.GetPackage(ctx, gemName)
	assert.NoError(t, err, "re-getting package info should not return error")
	assert.NotNil(t, pkg2, "package info should not be nil")
}
