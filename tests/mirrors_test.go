package tests

import (
	"context"
	"testing"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/repository"
	"github.com/stretchr/testify/assert"
)

// 测试所有镜像源是否正常工作
func TestAllMirrors(t *testing.T) {
	// Skip the test if running in short mode (CI environments)
	if testing.Short() {
		t.Skip("Skipping mirror tests in short mode")
	}

	// 创建上下文并设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 要测试的常见Gem包
	testGems := []string{
		"rails",
		"rack",
		"activesupport",
		"rake",
	}

	// 创建不同的镜像源仓库
	repos := map[string]repository.Repository{
		"Official":  repository.NewRepository(),
		"RubyChina": repository.NewRubyChinaRepository(),
		"TsingHua":  repository.NewTSingHuaRepository(),
		"AliYun":    repository.NewAliYunRepository(),
	}

	// 遍历所有镜像源
	for name, repo := range repos {
		t.Run(name, func(t *testing.T) {
			// 测试获取包信息
			for _, gemName := range testGems {
				t.Run("GetPackage-"+gemName, func(t *testing.T) {
					pkg, err := repo.GetPackage(ctx, gemName)
					assert.NoError(t, err, "获取包信息失败")
					assert.NotNil(t, pkg, "包信息为空")
					if pkg != nil {
						assert.Equal(t, gemName, pkg.Name, "包名不匹配")
						assert.NotEmpty(t, pkg.Version, "版本信息为空")
					} else {
						// Skip further assertions if pkg is nil
						t.SkipNow()
					}
				})
			}

			// 测试搜索
			t.Run("Search", func(t *testing.T) {
				results, err := repo.Search(ctx, "rails", 1)
				assert.NoError(t, err, "搜索失败")
				assert.NotEmpty(t, results, "搜索结果为空")
				// Skip further assertions if results is empty
				if len(results) == 0 {
					t.SkipNow()
				}
			})

			// 测试获取下载统计
			t.Run("Downloads", func(t *testing.T) {
				stats, err := repo.Downloads(ctx)
				assert.NoError(t, err, "获取下载统计失败")
				assert.NotNil(t, stats, "下载统计为空")
				// Skip further assertions if stats is nil
				if stats == nil {
					t.SkipNow()
				}
				assert.Greater(t, stats.TotalDownloads, 0, "下载量应大于0")
			})

			// 测试获取最新Gems
			t.Run("LatestGems", func(t *testing.T) {
				gems, err := repo.LatestGems(ctx)
				assert.NoError(t, err, "获取最新Gems失败")
				assert.NotEmpty(t, gems, "最新Gems列表为空")
				// Skip further assertions if gems is empty
				if len(gems) == 0 {
					t.SkipNow()
				}
			})
		})
	}
}
