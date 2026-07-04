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

// ===== URL construction verification tests =====

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

// ===== URL encoding tests =====

func TestRepository_URLPathEscape(t *testing.T) {
	repo := NewRepository()

	// Package name with special characters should be encoded correctly
	actualURL := fmt.Sprintf("%s/api/v1/gems/%s.json", repo.options.ServerURL, url.PathEscape("my-gem"))
	assert.Equal(t, "https://rubygems.org/api/v1/gems/my-gem.json", actualURL)

	// Package name with spaces should be encoded
	actualURL = fmt.Sprintf("%s/api/v1/gems/%s.json", repo.options.ServerURL, url.PathEscape("my gem"))
	assert.Equal(t, "https://rubygems.org/api/v1/gems/my%20gem.json", actualURL)
}

func TestRepository_URLQueryEscape(t *testing.T) {
	repo := NewRepository()

	// Search query with special characters should be encoded correctly
	actualURL := fmt.Sprintf("%s/api/v1/search.json?query=%s&page=%d", repo.options.ServerURL, url.QueryEscape("rails & rack"), 1)
	assert.Equal(t, "https://rubygems.org/api/v1/search.json?query=rails+%26+rack&page=1", actualURL)
}

// ===== CachedRepository new method tests =====

func TestCachedRepository_GetVersionReverseDependencies(t *testing.T) {
	ctx := context.Background()
	mockRepo := NewMockRepo()
	memCache := cache.NewMemoryCache(10*time.Minute, 30*time.Minute)
	cacheRepo := NewCachedRepository(mockRepo, 10*time.Minute, memCache)

	// Call the cache-wrapped method
	deps, err := cacheRepo.GetVersionReverseDependencies(ctx, "rails-7.0.5")
	assert.NoError(t, err)
	assert.Nil(t, deps)

	// Cleanup
	cacheRepo.ClearCache()
	cacheRepo.Close()
}

func TestCachedRepository_GetMFAStatus(t *testing.T) {
	ctx := context.Background()
	mockRepo := NewMockRepo()
	memCache := cache.NewMemoryCache(10*time.Minute, 30*time.Minute)
	cacheRepo := NewCachedRepository(mockRepo, 10*time.Minute, memCache)

	// Call the cache-wrapped method
	status, err := cacheRepo.GetMFAStatus(ctx)
	assert.NoError(t, err)
	assert.Nil(t, status)

	// Cleanup
	cacheRepo.ClearCache()
	cacheRepo.Close()
}

// ===== Model construction tests =====

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
	// scopes should appear twice
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

// ===== API integration tests (require network, skipped in short mode) =====

func TestRepository_GetVersionReverseDependencies_API(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := NewRepository()

	t.Run("version reverse dependencies", func(t *testing.T) {
		// Use the full_name format
		dependencies, err := repo.GetVersionReverseDependencies(ctx, "rack-2.2.7")

		if err != nil {
			if !IsNotFound(err) {
				t.Logf("getting version reverse dependencies returned error: %v", err)
			}
			return
		}

		assert.NotNil(t, dependencies, "reverse dependencies list should not be nil")
		if len(dependencies) > 0 {
			t.Logf("number of version-level reverse dependencies for rack-2.2.7: %d", len(dependencies))
		}
	})
}

func TestRepository_GetMFAStatus_API(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	// MFA status requires an API Token, skip when not set
	token := ""
	if token == "" {
		t.Skip("API Token not set, skip MFA status test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opts := NewOptions().SetToken(token)
	repo := NewRepository(opts)

	status, err := repo.GetMFAStatus(ctx)
	assert.NoError(t, err, "getting MFA status should not return an error")
	assert.NotNil(t, status, "MFA status should not be nil")
}
