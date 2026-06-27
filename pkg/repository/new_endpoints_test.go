package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 测试搜索自动补全
func TestRepository_SearchAutocomplete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := NewRepository()

	suggestions, err := repo.SearchAutocomplete(ctx, "rails")
	assert.NoError(t, err, "获取搜索自动补全不应返回错误")
	assert.NotNil(t, suggestions, "自动补全结果不应为nil")

	if len(suggestions) > 0 {
		t.Logf("搜索 'rails' 的自动补全结果: %v", suggestions)
		for _, s := range suggestions {
			assert.NotEmpty(t, s, "自动补全建议不应为空字符串")
		}
	}
}

// 测试API v2获取版本详情
func TestRepository_GetGemVersionDetail(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := NewRepository()

	detail, err := repo.GetGemVersionDetail(ctx, "rails", "7.0.5")
	assert.NoError(t, err, "获取版本详情不应返回错误")
	assert.NotNil(t, detail, "版本详情不应为nil")

	if detail != nil {
		assert.Equal(t, "7.0.5", detail.Number, "版本号应该匹配")
		assert.NotEmpty(t, detail.Sha, "SHA不应为空")
		assert.False(t, detail.Yanked, "rails 7.0.5不应被yanked")
		assert.NotNil(t, detail.Dependencies.Runtime, "运行时依赖不应为nil")
	}
}

// 测试获取最近更新的gem包
func TestRepository_JustUpdatedGems(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := NewRepository()

	gems, err := repo.JustUpdatedGems(ctx)
	assert.NoError(t, err, "获取最近更新的gem包不应返回错误")
	assert.NotNil(t, gems, "返回结果不应为nil")

	if len(gems) > 0 {
		t.Logf("最近更新的gem包数量: %d", len(gems))
		assert.NotEmpty(t, gems[0].Name, "gem包名不应为空")
	}
}

// 测试获取下载量排名前50的gem包
func TestRepository_TopDownloads(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := NewRepository()

	gems, err := repo.TopDownloads(ctx)
	assert.NoError(t, err, "获取下载量排名不应返回错误")
	assert.NotNil(t, gems, "返回结果不应为nil")

	if len(gems) > 0 {
		t.Logf("下载量排名前50的gem包数量: %d", len(gems))
		assert.NotEmpty(t, gems[0].Name, "gem包名不应为空")
		assert.Greater(t, gems[0].Downloads, 0, "下载量应大于0")

		// 验证排名是按下载量降序排列
		for i := 1; i < len(gems); i++ {
			assert.GreaterOrEqual(t, gems[i-1].Downloads, gems[i].Downloads,
				"下载量排名应按降序排列")
		}
	}
}

// 测试获取用户资料
func TestRepository_GetUserProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := NewRepository()

	profile, err := repo.GetUserProfile(ctx, "qrush")
	assert.NoError(t, err, "获取用户资料不应返回错误")
	assert.NotNil(t, profile, "用户资料不应为nil")

	if profile != nil {
		assert.NotEmpty(t, profile.Handle, "用户handle不应为空")
		t.Logf("用户: %s (ID: %d)", profile.Handle, profile.ID)
	}
}

// 测试获取gem包的所有者
func TestRepository_GetGemOwners(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := NewRepository()

	owners, err := repo.GetGemOwners(ctx, "rails")
	assert.NoError(t, err, "获取gem包所有者不应返回错误")
	assert.NotNil(t, owners, "所有者列表不应为nil")

	if len(owners) > 0 {
		t.Logf("rails 的所有者数量: %d", len(owners))
		assert.NotEmpty(t, owners[0].Handle, "所有者handle不应为空")
	}
}

// 测试获取指定用户拥有的所有gem包
func TestRepository_GetGemsByOwner(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := NewRepository()

	gems, err := repo.GetGemsByOwner(ctx, "qrush")
	assert.NoError(t, err, "获取用户拥有的gem包不应返回错误")
	assert.NotNil(t, gems, "返回结果不应为nil")

	if len(gems) > 0 {
		t.Logf("qrush 拥有的gem包数量: %d", len(gems))
	}
}

// 测试获取sigstore证明
func TestRepository_GetAttestations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := NewRepository()

	attestations, err := repo.GetAttestations(ctx, "rails", "7.0.5")
	// 并非所有gem版本都有证明，404是正常的
	if err != nil {
		if !IsNotFound(err) {
			t.Errorf("获取证明返回了意外错误: %v", err)
		}
	} else {
		t.Logf("rails 7.0.5 的证明数量: %d", len(attestations))
	}
}

// 测试获取自定义仓库（私服支持）
func TestRepository_CustomRepository(t *testing.T) {
	// 这个测试不需要网络，可以在短模式下运行
	repo := NewCustomRepository("https://my-private-gems.example.com")
	assert.NotNil(t, repo, "自定义仓库不应为nil")

	repoImpl, ok := repo.(*RepositoryImpl)
	assert.True(t, ok, "应该能转换为*RepositoryImpl")
	assert.Equal(t, "https://my-private-gems.example.com", repoImpl.options.ServerURL,
		"自定义仓库URL应该匹配")
}
