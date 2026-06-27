package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/models"
)

// 创建一个模拟的仓库实现用于测试
type mockRepository struct {
	mockPackages map[string]*models.PackageInformation
	mockVersions map[string][]*models.Version
	// 人为延迟，模拟网络请求延迟
	delay time.Duration
	// 人为错误，模拟请求失败
	failOn map[string]error
}

// 创建一个新的模拟仓库
func newMockRepository() *mockRepository {
	repo := &mockRepository{
		mockPackages: make(map[string]*models.PackageInformation),
		mockVersions: make(map[string][]*models.Version),
		delay:        10 * time.Millisecond, // 默认10ms延迟
		failOn:       make(map[string]error),
	}

	// 添加一些测试数据
	repo.mockPackages["rails"] = &models.PackageInformation{
		Name:        "rails",
		Version:     "7.0.5",
		Downloads:   1000000,
		HomepageURI: "https://rubyonrails.org",
		Info:        "Ruby on Rails",
	}

	repo.mockPackages["rack"] = &models.PackageInformation{
		Name:        "rack",
		Version:     "2.2.7",
		Downloads:   2000000,
		HomepageURI: "https://github.com/rack/rack",
		Info:        "Rack provides a minimal interface between webservers and Ruby frameworks",
	}

	// 添加一些版本信息
	repo.mockVersions["rails"] = []*models.Version{
		{Number: "7.0.5", CreatedAt: time.Now().Add(-24 * time.Hour)},
		{Number: "7.0.4", CreatedAt: time.Now().Add(-48 * time.Hour)},
	}

	repo.mockVersions["rack"] = []*models.Version{
		{Number: "2.2.7", CreatedAt: time.Now().Add(-24 * time.Hour)},
		{Number: "2.2.6", CreatedAt: time.Now().Add(-48 * time.Hour)},
	}

	return repo
}

// 设置特定gem会触发的错误
func (m *mockRepository) setFailOn(gemName string, err error) *mockRepository {
	m.failOn[gemName] = err
	return m
}

// 实现GetPackage方法
func (m *mockRepository) GetPackage(ctx context.Context, gemName string) (*models.PackageInformation, error) {
	// 检查是否应该失败
	if err, ok := m.failOn[gemName]; ok {
		return nil, err
	}

	// 模拟网络延迟
	time.Sleep(m.delay)

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 返回结果
	pkg, ok := m.mockPackages[gemName]
	if !ok {
		return nil, errors.New("gem not found")
	}
	return pkg, nil
}

// 实现GetGemVersions方法
func (m *mockRepository) GetGemVersions(ctx context.Context, gemName string) ([]*models.Version, error) {
	// 检查是否应该失败
	if err, ok := m.failOn[gemName]; ok {
		return nil, err
	}

	// 模拟网络延迟
	time.Sleep(m.delay)

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 返回结果
	versions, ok := m.mockVersions[gemName]
	if !ok {
		return nil, errors.New("gem not found")
	}
	return versions, nil
}

// 实现其他必要的接口方法（为简化测试，这些方法可以返回空值或错误）
func (m *mockRepository) Search(ctx context.Context, query string, page int) ([]*models.PackageInformation, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepository) GetGemLatestVersion(ctx context.Context, gemName string) (*models.LatestVersion, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepository) GetTimeFrameVersions(ctx context.Context, from, to time.Time) ([]*models.Version, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepository) Downloads(ctx context.Context) (*models.RepositoryDownloadCount, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepository) VersionDownloads(ctx context.Context, gemName, gemVersion string) (*models.VersionDownloadCount, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepository) GetDependencies(ctx context.Context, gemNames ...string) ([]*models.DependencyInfo, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepository) LatestGems(ctx context.Context) ([]*models.PackageInformation, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepository) GetReverseDependencies(ctx context.Context, gemName string) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepository) SearchAutocomplete(ctx context.Context, query string) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepository) GetGemVersionDetail(ctx context.Context, gemName, version string) (*models.VersionDetail, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepository) JustUpdatedGems(ctx context.Context) ([]*models.PackageInformation, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepository) TopDownloads(ctx context.Context) ([]*models.TopDownloadedGem, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepository) GetUserProfile(ctx context.Context, handleOrID string) (*models.UserProfile, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepository) GetOwnedGems(ctx context.Context) ([]*models.PackageInformation, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepository) GetGemsByOwner(ctx context.Context, handleOrID string) ([]*models.PackageInformation, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepository) GetGemOwners(ctx context.Context, gemName string) ([]*models.Owner, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepository) GetAttestations(ctx context.Context, gemName, version string) ([]*models.Attestation, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepository) GetGemVersionContents(ctx context.Context, gemName, version string) (*models.VersionContent, error) {
	return nil, errors.New("not implemented")
}

// 实现批量操作方法
func (m *mockRepository) BulkGetPackages(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[*models.PackageInformation] {
	// 只检查 options 是否为 nil，不再重新赋值
	if options == nil {
		options = NewBulkOptions()
	}

	results := make([]*BulkResult[*models.PackageInformation], 0, len(gemNames))
	for _, gemName := range gemNames {
		pkg, err := m.GetPackage(ctx, gemName)
		results = append(results, &BulkResult[*models.PackageInformation]{
			Key:   gemName,
			Value: pkg,
			Error: err,
		})
		if err != nil && !options.ContinueOnError {
			break
		}
	}
	return results
}

func (m *mockRepository) BulkGetVersions(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[[]*models.Version] {
	// 只检查 options 是否为 nil，不再重新赋值
	if options == nil {
		options = NewBulkOptions()
	}

	results := make([]*BulkResult[[]*models.Version], 0, len(gemNames))
	for _, gemName := range gemNames {
		versions, err := m.GetGemVersions(ctx, gemName)
		results = append(results, &BulkResult[[]*models.Version]{
			Key:   gemName,
			Value: versions,
			Error: err,
		})
		if err != nil && !options.ContinueOnError {
			break
		}
	}
	return results
}

func (m *mockRepository) BulkGetDependencies(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[[]*models.DependencyInfo] {
	return nil
}

func (m *mockRepository) BulkGetReverseDependencies(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[[]string] {
	return nil
}

// 测试批量获取包信息
func TestBulkGetPackages(t *testing.T) {
	// 创建模拟仓库
	mockRepo := newMockRepository()

	// 设置一个错误
	mockRepo.setFailOn("not-exist", errors.New("gem not found"))

	// 测试用例
	testCases := []struct {
		name        string
		gemNames    []string
		concurrency int
		timeout     time.Duration
		expectErr   bool
		expectCount int
	}{
		{
			name:        "获取有效包信息",
			gemNames:    []string{"rails", "rack"},
			concurrency: 2,
			timeout:     100 * time.Millisecond,
			expectErr:   false,
			expectCount: 2,
		},
		{
			name:        "包含一个不存在的包",
			gemNames:    []string{"rails", "rack", "not-exist"},
			concurrency: 2,
			timeout:     100 * time.Millisecond,
			expectErr:   true,
			expectCount: 3,
		},
		{
			name:        "超时测试",
			gemNames:    []string{"rails", "rack"},
			concurrency: 1,
			timeout:     5 * time.Millisecond, // 设置很短的超时时间
			expectErr:   true,
			expectCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 设置上下文和超时时间
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			// 设置并发数
			options := NewBulkOptions().WithMaxConcurrency(tc.concurrency)

			// 执行批量获取
			results := mockRepo.BulkGetPackages(ctx, tc.gemNames, options)

			// 验证结果数量
			if len(results) != tc.expectCount {
				t.Errorf("结果数量不符合预期，期望: %d, 实际: %d", tc.expectCount, len(results))
			}

			// 验证是否有错误
			hasError := false
			for _, result := range results {
				if result.Error != nil {
					hasError = true
					break
				}
			}

			if hasError != tc.expectErr {
				t.Errorf("错误状态不符合预期，期望有错误: %v, 实际: %v", tc.expectErr, hasError)
			}
		})
	}
}

// 测试批量获取版本信息
func TestBulkGetVersions(t *testing.T) {
	// 创建模拟仓库
	mockRepo := newMockRepository()

	// 设置一个错误
	mockRepo.setFailOn("not-exist", errors.New("gem not found"))

	// 测试用例
	testCases := []struct {
		name        string
		gemNames    []string
		concurrency int
		timeout     time.Duration
		expectErr   bool
		expectCount int
	}{
		{
			name:        "获取有效版本信息",
			gemNames:    []string{"rails", "rack"},
			concurrency: 2,
			timeout:     100 * time.Millisecond,
			expectErr:   false,
			expectCount: 2,
		},
		{
			name:        "包含一个不存在的包",
			gemNames:    []string{"rails", "rack", "not-exist"},
			concurrency: 2,
			timeout:     100 * time.Millisecond,
			expectErr:   true,
			expectCount: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 设置上下文和超时时间
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			// 设置并发数
			options := NewBulkOptions().WithMaxConcurrency(tc.concurrency)

			// 执行批量获取
			results := mockRepo.BulkGetVersions(ctx, tc.gemNames, options)

			// 验证结果数量
			if len(results) != tc.expectCount {
				t.Errorf("结果数量不符合预期，期望: %d, 实际: %d", tc.expectCount, len(results))
			}

			// 验证是否有错误
			hasError := false
			for _, result := range results {
				if result.Error != nil {
					hasError = true
					break
				}
			}

			if hasError != tc.expectErr {
				t.Errorf("错误状态不符合预期，期望有错误: %v, 实际: %v", tc.expectErr, hasError)
			}
		})
	}
}

// 测试批量操作选项
func TestBulkOptions(t *testing.T) {
	// 测试默认选项
	options := NewBulkOptions()
	if options.MaxConcurrency != 10 {
		t.Errorf("默认最大并发数不正确，期望: %d, 实际: %d", 10, options.MaxConcurrency)
	}
	if !options.ContinueOnError {
		t.Errorf("默认错误处理策略不正确，期望: %v, 实际: %v", true, options.ContinueOnError)
	}

	// 测试设置最大并发数
	options = NewBulkOptions().WithMaxConcurrency(5)
	if options.MaxConcurrency != 5 {
		t.Errorf("设置最大并发数后不正确，期望: %d, 实际: %d", 5, options.MaxConcurrency)
	}

	// 测试设置错误处理策略
	options = NewBulkOptions().WithContinueOnError(false)
	if options.ContinueOnError {
		t.Errorf("设置错误处理策略后不正确，期望: %v, 实际: %v", false, options.ContinueOnError)
	}
}
