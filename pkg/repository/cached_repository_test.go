package repository

import (
	"context"
	"testing"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/cache"
	"github.com/scagogogo/rubygems-skills/pkg/models"
	"github.com/stretchr/testify/assert"
)

// Mock Repository for testing
type MockRepo struct {
	calledTimes int
	testPkg     *models.PackageInformation
}

func NewMockRepo() *MockRepo {
	return &MockRepo{
		calledTimes: 0,
		testPkg: &models.PackageInformation{
			Name:    "test-gem",
			Version: "1.0.0",
			Authors: "Test Author",
		},
	}
}

// Implement the necessary methods of the Repository interface
func (m *MockRepo) GetPackage(ctx context.Context, gemName string) (*models.PackageInformation, error) {
	m.calledTimes++
	return m.testPkg, nil
}

// Other methods that need to be implemented to satisfy the Repository interface
func (m *MockRepo) Search(ctx context.Context, query string, page int) ([]*models.PackageInformation, error) {
	return nil, nil
}

func (m *MockRepo) GetGemVersions(ctx context.Context, gemName string) ([]*models.Version, error) {
	return nil, nil
}

func (m *MockRepo) GetGemLatestVersion(ctx context.Context, gemName string) (*models.LatestVersion, error) {
	return nil, nil
}

func (m *MockRepo) GetTimeFrameVersions(ctx context.Context, from, to time.Time) ([]*models.Version, error) {
	return nil, nil
}

func (m *MockRepo) Downloads(ctx context.Context) (*models.RepositoryDownloadCount, error) {
	return nil, nil
}

func (m *MockRepo) VersionDownloads(ctx context.Context, gemName, gemVersion string) (*models.VersionDownloadCount, error) {
	return nil, nil
}

func (m *MockRepo) GetDependencies(ctx context.Context, gemsNames ...string) ([]*models.DependencyInfo, error) {
	return nil, nil
}

func (m *MockRepo) LatestGems(ctx context.Context) ([]*models.PackageInformation, error) {
	return nil, nil
}

func (m *MockRepo) GetReverseDependencies(ctx context.Context, gemName string) ([]string, error) {
	return nil, nil
}

func (m *MockRepo) GetVersionReverseDependencies(ctx context.Context, fullName string) ([]string, error) {
	return nil, nil
}

func (m *MockRepo) SearchAutocomplete(ctx context.Context, query string) ([]string, error) {
	return nil, nil
}

func (m *MockRepo) GetGemVersionDetail(ctx context.Context, gemName, version string) (*models.VersionDetail, error) {
	return nil, nil
}

func (m *MockRepo) JustUpdatedGems(ctx context.Context) ([]*models.PackageInformation, error) {
	return nil, nil
}

func (m *MockRepo) TopDownloads(ctx context.Context) ([]*models.TopDownloadedGem, error) {
	return nil, nil
}

func (m *MockRepo) GetUserProfile(ctx context.Context, handleOrID string) (*models.UserProfile, error) {
	return nil, nil
}

func (m *MockRepo) GetOwnedGems(ctx context.Context) ([]*models.PackageInformation, error) {
	return nil, nil
}

func (m *MockRepo) GetGemsByOwner(ctx context.Context, handleOrID string) ([]*models.PackageInformation, error) {
	return nil, nil
}

func (m *MockRepo) GetGemOwners(ctx context.Context, gemName string) ([]*models.Owner, error) {
	return nil, nil
}

func (m *MockRepo) GetAttestations(ctx context.Context, gemName, version string) ([]*models.Attestation, error) {
	return nil, nil
}

func (m *MockRepo) GetGemVersionContents(ctx context.Context, gemName, version string) (*models.VersionContent, error) {
	return nil, nil
}

func (m *MockRepo) GetMFAStatus(ctx context.Context) (*models.MFAStatus, error) {
	return nil, nil
}

// Implement bulk operation methods
func (m *MockRepo) BulkGetPackages(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[*models.PackageInformation] {
	return nil
}

func (m *MockRepo) BulkGetVersions(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[[]*models.Version] {
	return nil
}

func (m *MockRepo) BulkGetDependencies(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[[]*models.DependencyInfo] {
	return nil
}

func (m *MockRepo) BulkGetReverseDependencies(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[[]string] {
	return nil
}

func TestCachedRepository(t *testing.T) {
	ctx := context.Background()
	mockRepo := NewMockRepo()

	// Create an in-memory cache
	memCache := cache.NewMemoryCache(10*time.Minute, 30*time.Minute)

	// Create a test wrapper
	type testWrapper struct {
		repo      *MockRepo
		cache     cache.Cache
		getCalled func() int
	}

	wrapper := &testWrapper{
		repo:  mockRepo,
		cache: memCache,
		getCalled: func() int {
			return mockRepo.calledTimes
		},
	}

	// Test the case without using cache
	for i := 0; i < 3; i++ {
		pkg, err := wrapper.repo.GetPackage(ctx, "test-gem")
		assert.NoError(t, err)
		assert.Equal(t, "test-gem", pkg.Name)
	}

	// Should be called 3 times
	assert.Equal(t, 3, wrapper.getCalled())

	// Create a new mock and cached repository
	mockRepo2 := NewMockRepo()
	// Use our Mock as the underlying repository
	cacheRepo := NewCachedRepository(mockRepo2, 10*time.Minute, memCache)

	// First call should invoke the underlying repository
	pkg, err := cacheRepo.GetPackage(ctx, "test-gem")
	assert.NoError(t, err)
	assert.Equal(t, "test-gem", pkg.Name)
	assert.Equal(t, 1, mockRepo2.calledTimes)

	// Second call should get from cache
	cachedPkg, err := cacheRepo.GetPackage(ctx, "test-gem")
	assert.NoError(t, err)
	assert.Equal(t, "test-gem", cachedPkg.Name)

	// mock should still only be called once
	assert.Equal(t, 1, mockRepo2.calledTimes)

	// Cleanup
	cacheRepo.ClearCache()
	cacheRepo.Close()
}
