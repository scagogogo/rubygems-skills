package integration

import (
	"context"
	"testing"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/cache"
	"github.com/scagogogo/rubygems-skills/pkg/repository"
	"github.com/stretchr/testify/assert"
)

// 测试仓库批量操作与缓存结合使用
func TestBulkOperationsWithCache(t *testing.T) {
	// 创建一个较长超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 创建内存缓存和仓库
	memCache := cache.NewMemoryCache(5*time.Minute, 30*time.Minute)
	baseRepo := repository.NewRepository()
	cachedRepo := repository.NewCachedRepository(baseRepo, 5*time.Minute, memCache)
	defer cachedRepo.Close()

	// 准备测试数据
	gems := []string{
		"rails",
		"rack",
		"activesupport",
		"rake",
		"nokogiri",
	}

	// 创建批量操作选项
	options := repository.NewBulkOptions().WithMaxConcurrency(3)

	// 测试先进行批量操作，然后从缓存获取
	t.Run("批量操作后缓存获取", func(t *testing.T) {
		// 清空缓存
		cachedRepo.ClearCache()

		// 首次使用基础仓库批量获取
		startTime := time.Now()
		results1 := baseRepo.BulkGetPackages(ctx, gems, options)
		duration1 := time.Since(startTime)

		// 校验结果
		assert.Equal(t, len(gems), len(results1), "结果数量应匹配请求数量")
		for _, result := range results1 {
			assert.NoError(t, result.Error, "获取包 %s 不应返回错误", result.Key)
			assert.NotNil(t, result.Value, "获取包 %s 返回的包信息不应为nil", result.Key)

			// 手动缓存结果
			if result.Error == nil {
				cacheKey := "package:" + result.Key
				memCache.SetWithExpiration(cacheKey, result.Value, 5*time.Minute)
			}
		}

		// 等待一会，确保不是网络波动导致的速度差异
		time.Sleep(500 * time.Millisecond)

		// 使用缓存仓库逐个获取
		startTime = time.Now()
		for _, gemName := range gems {
			pkg, err := cachedRepo.GetPackage(ctx, gemName)
			assert.NoError(t, err, "从缓存获取包 %s 不应返回错误", gemName)
			assert.NotNil(t, pkg, "从缓存获取包 %s 返回的包信息不应为nil", gemName)
		}
		duration2 := time.Since(startTime)

		// 从缓存获取应该明显快于API获取
		t.Logf("批量API获取耗时: %v, 缓存逐个获取耗时: %v", duration1, duration2)
		assert.True(t, duration2 < duration1, "从缓存逐个获取应该快于API批量获取")
	})

	// 测试缓存仓库与版本信息
	t.Run("版本信息缓存", func(t *testing.T) {
		// 清空缓存
		cachedRepo.ClearCache()

		// 使用普通仓库获取数据
		startTime := time.Now()
		results1 := baseRepo.BulkGetVersions(ctx, gems[:3], options)
		duration1 := time.Since(startTime)

		// 校验结果
		assert.Equal(t, 3, len(results1), "结果数量应匹配请求数量")
		for _, result := range results1 {
			assert.NoError(t, result.Error, "获取包 %s 的版本不应返回错误", result.Key)
			assert.NotNil(t, result.Value, "获取包 %s 返回的版本不应为nil", result.Key)

			// 手动缓存结果
			if result.Error == nil {
				cacheKey := "versions:" + result.Key
				memCache.SetWithExpiration(cacheKey, result.Value, 5*time.Minute)
			}
		}

		// 等待一会，确保不是网络波动导致的速度差异
		time.Sleep(500 * time.Millisecond)

		// 从缓存获取数据
		startTime = time.Now()
		for _, gemName := range gems[:3] {
			versions, err := cachedRepo.GetGemVersions(ctx, gemName)
			assert.NoError(t, err, "从缓存获取包 %s 的版本不应返回错误", gemName)
			assert.NotNil(t, versions, "从缓存获取包 %s 返回的版本不应为nil", gemName)
		}
		duration2 := time.Since(startTime)

		// 缓存应该更快
		t.Logf("批量获取版本耗时: %v, 缓存获取版本耗时: %v", duration1, duration2)
		assert.True(t, duration2 < duration1, "从缓存获取版本应该快于批量获取")
	})

	// 测试反向依赖获取和缓存
	t.Run("反向依赖缓存", func(t *testing.T) {
		// 清空缓存
		cachedRepo.ClearCache()

		// 批量获取反向依赖
		startTime := time.Now()
		results := baseRepo.BulkGetReverseDependencies(ctx, gems[:2], options)
		duration1 := time.Since(startTime)

		// 手动缓存结果
		for _, result := range results {
			if result.Error == nil {
				cacheKey := "reverse_dependencies:" + result.Key
				memCache.SetWithExpiration(cacheKey, result.Value, 5*time.Minute)
			}
		}

		// 等待一会
		time.Sleep(500 * time.Millisecond)

		// 从缓存获取
		startTime = time.Now()
		for _, gemName := range gems[:2] {
			deps, err := cachedRepo.GetReverseDependencies(ctx, gemName)
			assert.NoError(t, err, "从缓存获取包 %s 的反向依赖不应返回错误", gemName)
			assert.NotNil(t, deps, "从缓存获取包 %s 返回的反向依赖不应为nil", gemName)
		}
		duration2 := time.Since(startTime)

		// 缓存应该更快
		t.Logf("批量获取反向依赖耗时: %v, 缓存获取反向依赖耗时: %v", duration1, duration2)
		assert.True(t, duration2 < duration1, "从缓存获取反向依赖应该快于批量获取")
	})
}

// 测试同时使用缓存和批量查询
func TestCacheStatsAndExpiration(t *testing.T) {
	// 跳过长时间运行的测试
	if testing.Short() {
		t.Skip("在短模式下跳过缓存过期测试")
	}

	// 创建一个短缓存周期的缓存
	shortCache := cache.NewMemoryCache(2*time.Second, 1*time.Second)
	baseRepo := repository.NewRepository()
	cachedRepo := repository.NewCachedRepository(baseRepo, 2*time.Second, shortCache)
	defer cachedRepo.Close()

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 测试数据
	gemName := "rails"

	// 首次获取包信息，手动缓存
	pkg, err := baseRepo.GetPackage(ctx, gemName)
	assert.NoError(t, err, "获取包信息不应返回错误")
	assert.NotNil(t, pkg, "包信息不应为nil")

	// 手动缓存
	cacheKey := "package:" + gemName
	shortCache.SetWithExpiration(cacheKey, pkg, 2*time.Second)

	// 验证缓存状态
	cacheStats1 := cachedRepo.GetCacheStats()
	assert.Equal(t, 1, cacheStats1, "缓存中应该有一个项目")

	// 从缓存获取
	cachedPkg, err := cachedRepo.GetPackage(ctx, gemName)
	assert.NoError(t, err, "从缓存获取包信息不应返回错误")
	assert.Equal(t, pkg.Name, cachedPkg.Name, "缓存的包名应该与原始包名相同")

	// 等待缓存过期
	time.Sleep(3 * time.Second)

	// 缓存应该已经清除
	cacheStats2 := cachedRepo.GetCacheStats()
	assert.Equal(t, 0, cacheStats2, "缓存应该为空")

	// 再次获取，应该重新从API获取
	pkg2, err := cachedRepo.GetPackage(ctx, gemName)
	assert.NoError(t, err, "重新获取包信息不应返回错误")
	assert.NotNil(t, pkg2, "包信息不应为nil")
}
