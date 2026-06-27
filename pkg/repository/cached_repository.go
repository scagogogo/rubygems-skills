package repository

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/cache"
	"github.com/scagogogo/rubygems-skills/pkg/models"
)

// Compile-time check that CachedRepository implements the Repository interface
var _ Repository = (*CachedRepository)(nil)

const (
	// DefaultCacheExpiration default cache expiration time (10 minutes)
	DefaultCacheExpiration = 10 * time.Minute

	// DefaultCleanupInterval default cleanup interval (1 hour)
	DefaultCleanupInterval = 1 * time.Hour
)

// CachedRepository is a repository wrapper with caching functionality
// It implements the Repository interface and can seamlessly replace the base repository
// By caching API response data, it reduces duplicate requests and improves performance
type CachedRepository struct {
	repo          Repository    // Underlying repository implementation
	defaultTTL    time.Duration // Default cache expiration time
	cache         cache.Cache   // Cache implementation
	stopCleanupCh chan struct{} // Channel for stopping cleanup goroutine
}

// NewCachedRepository create a new cached repository instance
// Parameters:
//   - repo: underlying repository implementation
//   - ttl: default cache expiration time, time-to-live for all cache items
//   - cache: cache implementation, if nil, a new memory cache will be created
func NewCachedRepository(repo Repository, ttl time.Duration, cacheImpl cache.Cache) *CachedRepository {
	if cacheImpl == nil {
		// If cache not provided, create a memory cache with default cleanup interval of twice the cache time
		cacheImpl = cache.NewMemoryCache(ttl, ttl*2)
	}

	return &CachedRepository{
		repo:          repo,
		defaultTTL:    ttl,
		cache:         cacheImpl,
		stopCleanupCh: make(chan struct{}),
	}
}

// GetPackage get package info through cache
// Prefer getting from cache, call underlying repository method and cache result on cache miss
func (c *CachedRepository) GetPackage(ctx context.Context, gemName string) (*models.PackageInformation, error) {
	cacheKey := "package:" + gemName

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if pkg, ok := cachedValue.(*models.PackageInformation); ok {
			return pkg, nil
		}
	}

	// Cache miss, call underlying repository
	pkg, err := c.repo.GetPackage(ctx, gemName)
	if err != nil {
		return nil, err
	}

	// Cache result
	c.cache.SetWithExpiration(cacheKey, pkg, c.defaultTTL)
	return pkg, nil
}

// Search execute search operation through cache
// Search results may change over time, so search results have shorter cache time
func (c *CachedRepository) Search(ctx context.Context, query string, page int) ([]*models.PackageInformation, error) {
	cacheKey := "search:" + query + ":" + strconv.Itoa(page)

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if results, ok := cachedValue.([]*models.PackageInformation); ok {
			return results, nil
		}
	}

	// Cache miss, call underlying repository
	results, err := c.repo.Search(ctx, query, page)
	if err != nil {
		return nil, err
	}

	// Search results have shorter cache time, use half of default TTL
	c.cache.SetWithExpiration(cacheKey, results, c.defaultTTL/2)
	return results, nil
}

// GetGemVersions get package version list through cache
// Version list is relatively stable, use default cache time
func (c *CachedRepository) GetGemVersions(ctx context.Context, gemName string) ([]*models.Version, error) {
	cacheKey := "versions:" + gemName

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if versions, ok := cachedValue.([]*models.Version); ok {
			return versions, nil
		}
	}

	// Cache miss, call underlying repository
	versions, err := c.repo.GetGemVersions(ctx, gemName)
	if err != nil {
		return nil, err
	}

	c.cache.SetWithExpiration(cacheKey, versions, c.defaultTTL)
	return versions, nil
}

// GetGemLatestVersion get latest version of package through cache
// Latest version may update frequently, use shorter cache time
func (c *CachedRepository) GetGemLatestVersion(ctx context.Context, gemName string) (*models.LatestVersion, error) {
	cacheKey := "latest_version:" + gemName

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if version, ok := cachedValue.(*models.LatestVersion); ok {
			return version, nil
		}
	}

	// Cache miss, call underlying repository
	version, err := c.repo.GetGemLatestVersion(ctx, gemName)
	if err != nil {
		return nil, err
	}

	// Latest version has shorter cache time
	c.cache.SetWithExpiration(cacheKey, version, c.defaultTTL/2)
	return version, nil
}

// GetTimeFrameVersions get versions within time range through cache
// Timeframe query results are relatively stable, use default cache time
func (c *CachedRepository) GetTimeFrameVersions(ctx context.Context, from, to time.Time) ([]*models.Version, error) {
	cacheKey := "timeframe:" + from.Format(time.RFC3339) + ":" + to.Format(time.RFC3339)

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if versions, ok := cachedValue.([]*models.Version); ok {
			return versions, nil
		}
	}

	// Cache miss, call underlying repository
	versions, err := c.repo.GetTimeFrameVersions(ctx, from, to)
	if err != nil {
		return nil, err
	}

	c.cache.SetWithExpiration(cacheKey, versions, c.defaultTTL)
	return versions, nil
}

// Downloads get repository download statistics through cache
// Download statistics change frequently, use shorter cache time
func (c *CachedRepository) Downloads(ctx context.Context) (*models.RepositoryDownloadCount, error) {
	cacheKey := "downloads"

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if downloads, ok := cachedValue.(*models.RepositoryDownloadCount); ok {
			return downloads, nil
		}
	}

	// Cache miss, call underlying repository
	downloads, err := c.repo.Downloads(ctx)
	if err != nil {
		return nil, err
	}

	// Download statistics have shorter cache time
	c.cache.SetWithExpiration(cacheKey, downloads, c.defaultTTL/2)
	return downloads, nil
}

// VersionDownloads get specific version download statistics through cache
// Version download statistics change frequently, use shorter cache time
func (c *CachedRepository) VersionDownloads(ctx context.Context, gemName, gemVersion string) (*models.VersionDownloadCount, error) {
	cacheKey := "version_downloads:" + gemName + ":" + gemVersion

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if downloads, ok := cachedValue.(*models.VersionDownloadCount); ok {
			return downloads, nil
		}
	}

	// Cache miss, call underlying repository
	downloads, err := c.repo.VersionDownloads(ctx, gemName, gemVersion)
	if err != nil {
		return nil, err
	}

	// Version download statistics have shorter cache time
	c.cache.SetWithExpiration(cacheKey, downloads, c.defaultTTL/2)
	return downloads, nil
}

// GetDependencies get package dependencies through cache
// Dependencies are relatively stable, use default cache time
func (c *CachedRepository) GetDependencies(ctx context.Context, gemNames ...string) ([]*models.DependencyInfo, error) {
	// For multiple package names, use joined string as cache key
	cacheKey := "dependencies:" + strings.Join(gemNames, ",")

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if deps, ok := cachedValue.([]*models.DependencyInfo); ok {
			return deps, nil
		}
	}

	// Cache miss, call underlying repository
	deps, err := c.repo.GetDependencies(ctx, gemNames...)
	if err != nil {
		return nil, err
	}

	c.cache.SetWithExpiration(cacheKey, deps, c.defaultTTL)
	return deps, nil
}

// LatestGems get latest gem list through cache
// Latest list changes frequently, use shorter cache time
func (c *CachedRepository) LatestGems(ctx context.Context) ([]*models.PackageInformation, error) {
	cacheKey := "latest_gems"

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if gems, ok := cachedValue.([]*models.PackageInformation); ok {
			return gems, nil
		}
	}

	// Cache miss, call underlying repository
	gems, err := c.repo.LatestGems(ctx)
	if err != nil {
		return nil, err
	}

	// Latest list has shorter cache time
	c.cache.SetWithExpiration(cacheKey, gems, c.defaultTTL/4)
	return gems, nil
}

// GetReverseDependencies get reverse dependencies of package through cache
// Reverse dependencies are relatively stable, use default cache time
func (c *CachedRepository) GetReverseDependencies(ctx context.Context, gemName string) ([]string, error) {
	cacheKey := "reverse_dependencies:" + gemName

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if deps, ok := cachedValue.([]string); ok {
			return deps, nil
		}
	}

	// Cache miss, call underlying repository
	deps, err := c.repo.GetReverseDependencies(ctx, gemName)
	if err != nil {
		return nil, err
	}

	c.cache.SetWithExpiration(cacheKey, deps, c.defaultTTL)
	return deps, nil
}

// GetVersionReverseDependencies get reverse dependencies for a specific version through cache
// The fullName parameter should be in the format "gemname-version" (e.g., "rails-7.0.5")
// Version-level reverse dependencies are relatively stable, use default cache time
func (c *CachedRepository) GetVersionReverseDependencies(ctx context.Context, fullName string) ([]string, error) {
	cacheKey := "version_reverse_dependencies:" + fullName

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if deps, ok := cachedValue.([]string); ok {
			return deps, nil
		}
	}

	// Cache miss, call underlying repository
	deps, err := c.repo.GetVersionReverseDependencies(ctx, fullName)
	if err != nil {
		return nil, err
	}

	c.cache.SetWithExpiration(cacheKey, deps, c.defaultTTL)
	return deps, nil
}

// SearchAutocomplete get search autocomplete suggestions through cache
// Autocomplete results change frequently, use shorter cache time
func (c *CachedRepository) SearchAutocomplete(ctx context.Context, query string) ([]string, error) {
	cacheKey := "autocomplete:" + query

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if suggestions, ok := cachedValue.([]string); ok {
			return suggestions, nil
		}
	}

	// Cache miss, call underlying repository
	suggestions, err := c.repo.SearchAutocomplete(ctx, query)
	if err != nil {
		return nil, err
	}

	c.cache.SetWithExpiration(cacheKey, suggestions, c.defaultTTL/4)
	return suggestions, nil
}

// GetGemVersionDetail get detailed info of specific package version through cache
// Version details are relatively stable, use default cache time
func (c *CachedRepository) GetGemVersionDetail(ctx context.Context, gemName, version string) (*models.VersionDetail, error) {
	cacheKey := "version_detail:" + gemName + ":" + version

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if detail, ok := cachedValue.(*models.VersionDetail); ok {
			return detail, nil
		}
	}

	// Cache miss, call underlying repository
	detail, err := c.repo.GetGemVersionDetail(ctx, gemName, version)
	if err != nil {
		return nil, err
	}

	c.cache.SetWithExpiration(cacheKey, detail, c.defaultTTL)
	return detail, nil
}

// JustUpdatedGems get recently updated gems through cache
// Updated list changes frequently, use shorter cache time
func (c *CachedRepository) JustUpdatedGems(ctx context.Context) ([]*models.PackageInformation, error) {
	cacheKey := "just_updated_gems"

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if gems, ok := cachedValue.([]*models.PackageInformation); ok {
			return gems, nil
		}
	}

	// Cache miss, call underlying repository
	gems, err := c.repo.JustUpdatedGems(ctx)
	if err != nil {
		return nil, err
	}

	c.cache.SetWithExpiration(cacheKey, gems, c.defaultTTL/4)
	return gems, nil
}

// TopDownloads get top 50 most downloaded gems through cache
// Rankings change slowly, use default cache time
func (c *CachedRepository) TopDownloads(ctx context.Context) ([]*models.TopDownloadedGem, error) {
	cacheKey := "top_downloads"

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if gems, ok := cachedValue.([]*models.TopDownloadedGem); ok {
			return gems, nil
		}
	}

	// Cache miss, call underlying repository
	gems, err := c.repo.TopDownloads(ctx)
	if err != nil {
		return nil, err
	}

	c.cache.SetWithExpiration(cacheKey, gems, c.defaultTTL)
	return gems, nil
}

// GetUserProfile get user profile through cache
// User profiles are relatively stable, use default cache time
func (c *CachedRepository) GetUserProfile(ctx context.Context, handleOrID string) (*models.UserProfile, error) {
	cacheKey := "user_profile:" + handleOrID

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if profile, ok := cachedValue.(*models.UserProfile); ok {
			return profile, nil
		}
	}

	// Cache miss, call underlying repository
	profile, err := c.repo.GetUserProfile(ctx, handleOrID)
	if err != nil {
		return nil, err
	}

	c.cache.SetWithExpiration(cacheKey, profile, c.defaultTTL)
	return profile, nil
}

// GetOwnedGems get all gems owned by current authenticated user through cache
func (c *CachedRepository) GetOwnedGems(ctx context.Context) ([]*models.PackageInformation, error) {
	cacheKey := "owned_gems"

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if gems, ok := cachedValue.([]*models.PackageInformation); ok {
			return gems, nil
		}
	}

	// Cache miss, call underlying repository
	gems, err := c.repo.GetOwnedGems(ctx)
	if err != nil {
		return nil, err
	}

	c.cache.SetWithExpiration(cacheKey, gems, c.defaultTTL/2)
	return gems, nil
}

// GetGemsByOwner get all gems owned by specified user through cache
func (c *CachedRepository) GetGemsByOwner(ctx context.Context, handleOrID string) ([]*models.PackageInformation, error) {
	cacheKey := "gems_by_owner:" + handleOrID

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if gems, ok := cachedValue.([]*models.PackageInformation); ok {
			return gems, nil
		}
	}

	// Cache miss, call underlying repository
	gems, err := c.repo.GetGemsByOwner(ctx, handleOrID)
	if err != nil {
		return nil, err
	}

	c.cache.SetWithExpiration(cacheKey, gems, c.defaultTTL)
	return gems, nil
}

// GetGemOwners get all owners of specified gem through cache
// Owner list is relatively stable, use default cache time
func (c *CachedRepository) GetGemOwners(ctx context.Context, gemName string) ([]*models.Owner, error) {
	cacheKey := "gem_owners:" + gemName

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if owners, ok := cachedValue.([]*models.Owner); ok {
			return owners, nil
		}
	}

	// Cache miss, call underlying repository
	owners, err := c.repo.GetGemOwners(ctx, gemName)
	if err != nil {
		return nil, err
	}

	c.cache.SetWithExpiration(cacheKey, owners, c.defaultTTL)
	return owners, nil
}

// GetAttestations get sigstore attestations for specified gem version through cache
func (c *CachedRepository) GetAttestations(ctx context.Context, gemName, version string) ([]*models.Attestation, error) {
	cacheKey := "attestations:" + gemName + ":" + version

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if atts, ok := cachedValue.([]*models.Attestation); ok {
			return atts, nil
		}
	}

	// Cache miss, call underlying repository
	atts, err := c.repo.GetAttestations(ctx, gemName, version)
	if err != nil {
		return nil, err
	}

	c.cache.SetWithExpiration(cacheKey, atts, c.defaultTTL)
	return atts, nil
}

// GetGemVersionContents get file checksums for specified gem version through cache
func (c *CachedRepository) GetGemVersionContents(ctx context.Context, gemName, version string) (*models.VersionContent, error) {
	cacheKey := "version_contents:" + gemName + ":" + version

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if contents, ok := cachedValue.(*models.VersionContent); ok {
			return contents, nil
		}
	}

	// Cache miss, call underlying repository
	contents, err := c.repo.GetGemVersionContents(ctx, gemName, version)
	if err != nil {
		return nil, err
	}

	c.cache.SetWithExpiration(cacheKey, contents, c.defaultTTL)
	return contents, nil
}

// GetMFAStatus check MFA status for the authenticated user through cache
// MFA status changes rarely, use default cache time
func (c *CachedRepository) GetMFAStatus(ctx context.Context) (*models.MFAStatus, error) {
	cacheKey := "mfa_status"

	// Try to get from cache
	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if status, ok := cachedValue.(*models.MFAStatus); ok {
			return status, nil
		}
	}

	// Cache miss, call underlying repository
	status, err := c.repo.GetMFAStatus(ctx)
	if err != nil {
		return nil, err
	}

	c.cache.SetWithExpiration(cacheKey, status, c.defaultTTL)
	return status, nil
}

// Close close the cached repository and release resources
// This method should be called when the repository is no longer in use
func (c *CachedRepository) Close() {
	close(c.stopCleanupCh)
}

// ClearCache clear the cache
// Can be called when forced data refresh is needed
func (c *CachedRepository) ClearCache() {
	c.cache.Clear()
}

// GetCacheStats get cache statistics
// Return the number of items currently in cache
func (c *CachedRepository) GetCacheStats() int {
	return c.cache.Count()
}

// BulkGetPackages implements the Repository interface
func (c *CachedRepository) BulkGetPackages(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[*models.PackageInformation] {
	return c.repo.BulkGetPackages(ctx, gemNames, options)
}

// BulkGetVersions implements the Repository interface
func (c *CachedRepository) BulkGetVersions(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[[]*models.Version] {
	return c.repo.BulkGetVersions(ctx, gemNames, options)
}

// BulkGetDependencies implements the Repository interface
func (c *CachedRepository) BulkGetDependencies(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[[]*models.DependencyInfo] {
	return c.repo.BulkGetDependencies(ctx, gemNames, options)
}

// BulkGetReverseDependencies implements the Repository interface
func (c *CachedRepository) BulkGetReverseDependencies(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[[]string] {
	return c.repo.BulkGetReverseDependencies(ctx, gemNames, options)
}
