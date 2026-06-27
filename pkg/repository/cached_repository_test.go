package repository

import (
	"context"
	"testing"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/cache"
	"github.com/scagogogo/rubygems-skills/pkg/models"
	"github.com/stretchr/testify/assert"
)

// 模拟Repository用于测试
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

// 实现Repository接口的必要方法
func (m *MockRepo) GetPackage(ctx context.Context, gemName string) (*models.PackageInformation, error) {
	m.calledTimes++
	return m.testPkg, nil
}

// 为了满足Repository接口，需要实现的其他方法
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

// 实现批量操作方法
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

	// 创建一个内存缓存
	memCache := cache.NewMemoryCache(10*time.Minute, 30*time.Minute)

	// 创建一个测试包装器
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

	// 测试不使用缓存的情况
	for i := 0; i < 3; i++ {
		pkg, err := wrapper.repo.GetPackage(ctx, "test-gem")
		assert.NoError(t, err)
		assert.Equal(t, "test-gem", pkg.Name)
	}

	// 应该被调用3次
	assert.Equal(t, 3, wrapper.getCalled())

	// 创建新的mock和缓存仓库
	mockRepo2 := NewMockRepo()
	// 使用我们的Mock作为底层仓库
	cacheRepo := NewCachedRepository(mockRepo2, 10*time.Minute, memCache)

	// 首次调用，应该会调用底层仓库
	pkg, err := cacheRepo.GetPackage(ctx, "test-gem")
	assert.NoError(t, err)
	assert.Equal(t, "test-gem", pkg.Name)
	assert.Equal(t, 1, mockRepo2.calledTimes)

	// 第二次调用，应该从缓存获取
	cachedPkg, err := cacheRepo.GetPackage(ctx, "test-gem")
	assert.NoError(t, err)
	assert.Equal(t, "test-gem", cachedPkg.Name)

	// mock仍然只被调用了一次
	assert.Equal(t, 1, mockRepo2.calledTimes)

	// 清理
	cacheRepo.ClearCache()
	cacheRepo.Close()
}
