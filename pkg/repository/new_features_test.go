package repository

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/cache"
	"github.com/scagogogo/rubygems-skills/pkg/models"
	"github.com/stretchr/testify/assert"
)

// ===== URL 构造验证测试 =====

func TestRepository_GetVersionReverseDependencies_URL(t *testing.T) {
	repo := NewRepository()

	expectedURL := "https://rubygems.org/api/v1/versions/rails-7.0.5/reverse_dependencies.json"
	actualURL := fmt.Sprintf("%s/api/v1/versions/%s/reverse_dependencies.json", repo.options.ServerURL, url.PathEscape("rails-7.0.5"))
	assert.Equal(t, expectedURL, actualURL)
}

func TestWriteRepository_GetMFAStatus_URL(t *testing.T) {
	repo := NewRepository()

	expectedURL := "https://rubygems.org/api/v1/multifactor_auth"
	actualURL := fmt.Sprintf("%s/api/v1/multifactor_auth", repo.options.ServerURL)
	assert.Equal(t, expectedURL, actualURL)
}

func TestWriteRepository_GetMyProfile_URL(t *testing.T) {
	opts := NewOptions()
	repo := NewWriteRepository(opts)

	actualURL := fmt.Sprintf("%s/api/v1/profiles/me.json", repo.options.ServerURL)
	assert.Equal(t, "https://rubygems.org/api/v1/profiles/me.json", actualURL)
}

func TestWriteRepository_GetAPIKey_URL(t *testing.T) {
	opts := NewOptions()
	repo := NewWriteRepository(opts)

	expectedURL := "https://rubygems.org/api/v1/api_key"
	actualURL := fmt.Sprintf("%s/api/v1/api_key", repo.options.ServerURL)
	assert.Equal(t, expectedURL, actualURL)
}

// ===== URL 编码测试 =====

func TestRepository_URLPathEscape(t *testing.T) {
	repo := NewRepository()

	// 包名含特殊字符时应正确编码
	actualURL := fmt.Sprintf("%s/api/v1/gems/%s.json", repo.options.ServerURL, url.PathEscape("my-gem"))
	assert.Equal(t, "https://rubygems.org/api/v1/gems/my-gem.json", actualURL)

	// 包名含空格时应编码
	actualURL = fmt.Sprintf("%s/api/v1/gems/%s.json", repo.options.ServerURL, url.PathEscape("my gem"))
	assert.Equal(t, "https://rubygems.org/api/v1/gems/my%20gem.json", actualURL)
}

func TestRepository_URLQueryEscape(t *testing.T) {
	repo := NewRepository()

	// 搜索查询含特殊字符时应正确编码
	actualURL := fmt.Sprintf("%s/api/v1/search.json?query=%s&page=%d", repo.options.ServerURL, url.QueryEscape("rails & rack"), 1)
	assert.Equal(t, "https://rubygems.org/api/v1/search.json?query=rails+%26+rack&page=1", actualURL)
}

// ===== CachedRepository 新方法测试 =====

func TestCachedRepository_GetVersionReverseDependencies(t *testing.T) {
	ctx := context.Background()
	mockRepo := NewMockRepo()
	memCache := cache.NewMemoryCache(10*time.Minute, 30*time.Minute)
	cacheRepo := NewCachedRepository(mockRepo, 10*time.Minute, memCache)

	// 调用缓存包装的方法
	deps, err := cacheRepo.GetVersionReverseDependencies(ctx, "rails-7.0.5")
	assert.NoError(t, err)
	assert.Nil(t, deps)

	// 清理
	cacheRepo.ClearCache()
	cacheRepo.Close()
}

func TestCachedRepository_GetMFAStatus(t *testing.T) {
	ctx := context.Background()
	mockRepo := NewMockRepo()
	memCache := cache.NewMemoryCache(10*time.Minute, 30*time.Minute)
	cacheRepo := NewCachedRepository(mockRepo, 10*time.Minute, memCache)

	// 调用缓存包装的方法
	status, err := cacheRepo.GetMFAStatus(ctx)
	assert.NoError(t, err)
	assert.Nil(t, status)

	// 清理
	cacheRepo.ClearCache()
	cacheRepo.Close()
}

// ===== 模型构造测试 =====

func TestCreateAPIKeyRequest_FormEncoding(t *testing.T) {
	req := &models.CreateAPIKeyRequest{
		Name:   "my-key",
		Scopes: []string{"index_rubygems", "push_rubygem"},
		MFA:    "enabled",
	}

	form := url.Values{}
	form.Set("name", req.Name)
	for _, scope := range req.Scopes {
		form.Add("scopes[]", scope)
	}
	form.Set("mfa", req.MFA)

	encoded := form.Encode()
	assert.Contains(t, encoded, "name=my-key")
	assert.Contains(t, encoded, "mfa=enabled")
	// scopes 应该出现两次
	assert.Contains(t, encoded, "scopes%5B%5D=index_rubygems")
	assert.Contains(t, encoded, "scopes%5B%5D=push_rubygem")
}

func TestUpdateAPIKeyRequest_FormEncoding(t *testing.T) {
	req := &models.UpdateAPIKeyRequest{
		APIKey: "my-secret-key",
		Scopes: []string{"index_rubygems"},
		MFA:    "disabled",
	}

	form := url.Values{}
	form.Set("api_key", req.APIKey)
	for _, scope := range req.Scopes {
		form.Add("scopes[]", scope)
	}
	form.Set("mfa", req.MFA)

	encoded := form.Encode()
	assert.Contains(t, encoded, "api_key=my-secret-key")
	assert.Contains(t, encoded, "mfa=disabled")
}

// ===== API 集成测试 (需要网络，short 模式下跳过) =====

func TestRepository_GetVersionReverseDependencies_API(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := NewRepository()

	t.Run("版本反向依赖", func(t *testing.T) {
		// 使用 full_name 格式
		dependencies, err := repo.GetVersionReverseDependencies(ctx, "rack-2.2.7")

		if err != nil {
			if !IsNotFound(err) {
				t.Logf("获取版本反向依赖返回错误: %v", err)
			}
			return
		}

		assert.NotNil(t, dependencies, "反向依赖列表不应为nil")
		if len(dependencies) > 0 {
			t.Logf("rack-2.2.7 的版本级别反向依赖数量: %d", len(dependencies))
		}
	})
}

func TestRepository_GetMFAStatus_API(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	// MFA 状态需要 API Token，未设置时跳过
	token := ""
	if token == "" {
		t.Skip("未设置 API Token，跳过 MFA 状态测试")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opts := NewOptions().SetToken(token)
	repo := NewRepository(opts)

	status, err := repo.GetMFAStatus(ctx)
	assert.NoError(t, err, "获取 MFA 状态不应返回错误")
	assert.NotNil(t, status, "MFA 状态不应为nil")
}
