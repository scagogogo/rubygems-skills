package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test getting versions within a time frame
func TestRepository_GetTimeFrameVersions(t *testing.T) {
	// Skip the test if running in short mode (CI environments)
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create repository instance
	repo := NewRepository()

	// Set the time range, choose the last 24 hours
	to := time.Now()
	from := to.Add(-24 * time.Hour)

	// Get versions within the time frame
	versions, err := repo.GetTimeFrameVersions(ctx, from, to)

	// Verify the result
	assert.NoError(t, err, "getting versions within a time frame should not return an error")
	assert.NotNil(t, versions, "returned version list should not be nil")

	// If no versions are returned, it just means no versions were published during this period, not an error
	if len(versions) > 0 {
		// Verify that the version creation time is within the specified range
		for _, version := range versions {
			assert.True(t, version.CreatedAt.After(from) && version.CreatedAt.Before(to.Add(time.Minute)),
				"version creation time should be within the specified range: %v-%v, actual: %v", from, to, version.CreatedAt)
			assert.NotEmpty(t, version.Number, "version number must not be empty")
		}
	}
}

// Test getting package dependencies
func TestRepository_GetDependencies(t *testing.T) {
	// Skip the test if running in short mode (CI environments)
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create repository instance
	repo := NewRepository()

	// Test single package dependencies
	t.Run("single package dependencies", func(t *testing.T) {
		// Choose a common package, it should have dependencies
		dependencies, err := repo.GetDependencies(ctx, "rails")

		assert.NoError(t, err, "getting dependencies should not return an error")
		assert.NotNil(t, dependencies, "dependency list should not be nil")
		assert.NotEmpty(t, dependencies, "Rails should have dependencies")

		// Check the fields of the dependency items
		for _, dep := range dependencies {
			assert.NotEmpty(t, dep.Name, "dependency name must not be empty")
			assert.NotEmpty(t, dep.Requirements, "dependency requirements must not be empty")
		}
	})

	// Test multiple package dependencies
	t.Run("multiple package dependencies", func(t *testing.T) {
		// Choose several common packages
		dependencies, err := repo.GetDependencies(ctx, "rails", "rack", "nokogiri")

		assert.NoError(t, err, "getting multiple package dependencies should not return an error")
		assert.NotNil(t, dependencies, "dependency list should not be nil")
		assert.NotEmpty(t, dependencies, "these packages should have dependencies")
	})

	// Test getting dependencies for a non-existent package
	t.Run("non-existent package", func(t *testing.T) {
		// Use a package name that almost certainly does not exist
		dependencies, err := repo.GetDependencies(ctx, "non_existent_package_xyz_123")

		// This should return an empty list instead of an error
		assert.NoError(t, err, "getting dependencies for a non-existent package should return an empty list, not an error")
		assert.Empty(t, dependencies, "non-existent package should not have dependencies")
	})
}

// Test getting reverse dependencies of a package
func TestRepository_GetReverseDependencies(t *testing.T) {
	// Skip the test if running in short mode (CI environments)
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create repository instance
	repo := NewRepository()

	// Test reverse dependencies of a common package
	t.Run("common package reverse dependencies", func(t *testing.T) {
		// Choose a base package, it should be depended on by many other packages
		dependencies, err := repo.GetReverseDependencies(ctx, "rack")

		assert.NoError(t, err, "getting reverse dependencies should not return an error")
		assert.NotNil(t, dependencies, "reverse dependencies list should not be nil")
		assert.NotEmpty(t, dependencies, "Rack should have reverse dependencies")
	})

	// Test getting reverse dependencies for a non-existent package
	t.Run("non-existent package", func(t *testing.T) {
		// Use a package name that almost certainly does not exist
		dependencies, err := repo.GetReverseDependencies(ctx, "non_existent_package_xyz_123")

		// This should return an empty list instead of an error
		assert.NoError(t, err, "getting reverse dependencies for a non-existent package should return an empty list, not an error")
		assert.Empty(t, dependencies, "non-existent package should not have reverse dependencies")
	})

	// Test reverse dependencies of a new package
	t.Run("new package reverse dependencies", func(t *testing.T) {
		// First get the latest package list
		latestGems, err := repo.LatestGems(ctx)
		if err != nil || len(latestGems) == 0 {
			t.Skip("cannot get latest package list")
			return
		}

		// Choose the most recently published package, it may not have reverse dependencies
		dependencies, err := repo.GetReverseDependencies(ctx, latestGems[0].Name)

		// Do not care whether there are dependencies, but it should not error
		assert.NoError(t, err, "getting reverse dependencies for a new package should not return an error")
		assert.NotNil(t, dependencies, "reverse dependencies list should not be nil")
	})
}

// Test using different mirror sources
func TestDifferentMirrors(t *testing.T) {
	// Skip the test if running in short mode (CI environments)
	if testing.Short() {
		t.Skip("Skipping mirror tests in short mode")
	}

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create repositories with different mirror sources
	repos := map[string]Repository{
		"default":    NewRepository(),
		"RubyChina": NewRubyChinaRepository(),
		"TsingHua":  NewTSingHuaRepository(),
		"AliYun":    NewAliYunRepository(),
	}

	// Test each mirror source
	for name, repo := range repos {
		t.Run(name, func(t *testing.T) {
			// Test getting package info
			pkg, err := repo.GetPackage(ctx, "rails")
			assert.NoError(t, err, "%s: failed to get package info", name)
			assert.NotNil(t, pkg, "%s: package info is nil", name)
			if pkg != nil {
				assert.Equal(t, "rails", pkg.Name, "%s: package name mismatch", name)
			} else {
				// Skip further assertions if pkg is nil
				t.SkipNow()
			}
		})
	}
}

// Test proxy settings (this test requires a valid proxy, not run in CI environment)
func TestProxySetting(t *testing.T) {
	// Skip tests in CI environment
	if testing.Short() {
		t.Skip("skip proxy test in short mode")
	}

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create repository options with proxy
	// Note: this is an example, replace with an actually usable proxy
	proxyURL := ""
	if proxyURL == "" {
		t.Skip("proxy URL not set, skip test")
	}

	options := NewOptions().SetProxy(proxyURL)
	repo := NewRepository(options)

	// Test whether basic functionality works
	pkg, err := repo.GetPackage(ctx, "rails")
	assert.NoError(t, err, "failed to get package info via proxy")
	assert.NotNil(t, pkg, "package info is nil")
	if pkg != nil {
		assert.Equal(t, "rails", pkg.Name, "package name mismatch")
	}
}

// Test token settings (this test requires a valid token, not run in CI environment)
func TestTokenSetting(t *testing.T) {
	// Skip tests in CI environment
	if testing.Short() {
		t.Skip("skip token test in short mode")
	}

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create repository options with token
	// Note: this is an example, replace with an actually usable token
	token := ""
	if token == "" {
		t.Skip("token not set, skip test")
	}

	options := NewOptions().SetToken(token)
	repo := NewRepository(options)

	// Test whether basic functionality works
	pkg, err := repo.GetPackage(ctx, "rails")
	assert.NoError(t, err, "failed to get package info using token")
	assert.NotNil(t, pkg, "package info is nil")
	if pkg != nil {
		assert.Equal(t, "rails", pkg.Name, "package name mismatch")
	}
}
