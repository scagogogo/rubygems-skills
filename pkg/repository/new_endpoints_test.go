package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test search autocomplete
func TestRepository_SearchAutocomplete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := NewRepository()

	suggestions, err := repo.SearchAutocomplete(ctx, "rails")
	assert.NoError(t, err, "getting search autocomplete should not return an error")
	assert.NotNil(t, suggestions, "autocomplete results should not be nil")

	if len(suggestions) > 0 {
		t.Logf("autocomplete results for 'rails': %v", suggestions)
		for _, s := range suggestions {
			assert.NotEmpty(t, s, "autocomplete suggestion should not be an empty string")
		}
	}
}

// Test API v2 get version detail
func TestRepository_GetGemVersionDetail(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := NewRepository()

	detail, err := repo.GetGemVersionDetail(ctx, "rails", "7.0.5")
	assert.NoError(t, err, "getting version detail should not return an error")
	assert.NotNil(t, detail, "version detail should not be nil")

	if detail != nil {
		assert.Equal(t, "7.0.5", detail.Number, "version number should match")
		assert.NotEmpty(t, detail.Sha, "SHA should not be empty")
		assert.False(t, detail.Yanked, "rails 7.0.5 should not be yanked")
		assert.NotNil(t, detail.Dependencies.Runtime, "runtime dependencies should not be nil")
	}
}

// Test getting recently updated gems
func TestRepository_JustUpdatedGems(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := NewRepository()

	gems, err := repo.JustUpdatedGems(ctx)
	assert.NoError(t, err, "getting recently updated gems should not return an error")
	assert.NotNil(t, gems, "returned result should not be nil")

	if len(gems) > 0 {
		t.Logf("number of recently updated gems: %d", len(gems))
		assert.NotEmpty(t, gems[0].Name, "gem name should not be empty")
	}
}

// Test getting the top 50 gems by downloads
func TestRepository_TopDownloads(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := NewRepository()

	gems, err := repo.TopDownloads(ctx)
	assert.NoError(t, err, "getting download ranking should not return an error")
	assert.NotNil(t, gems, "returned result should not be nil")

	if len(gems) > 0 {
		t.Logf("number of top 50 gems by downloads: %d", len(gems))
		assert.NotEmpty(t, gems[0].Name, "gem name should not be empty")
		assert.Greater(t, gems[0].Downloads, 0, "downloads should be greater than 0")

		// Verify that the ranking is sorted by downloads in descending order
		for i := 1; i < len(gems); i++ {
			assert.GreaterOrEqual(t, gems[i-1].Downloads, gems[i].Downloads,
				"download ranking should be in descending order")
		}
	}
}

// Test getting user profile
func TestRepository_GetUserProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := NewRepository()

	profile, err := repo.GetUserProfile(ctx, "qrush")
	assert.NoError(t, err, "getting user profile should not return an error")
	assert.NotNil(t, profile, "user profile should not be nil")

	if profile != nil {
		assert.NotEmpty(t, profile.Handle, "user handle should not be empty")
		t.Logf("user: %s (ID: %d)", profile.Handle, profile.ID)
	}
}

// Test getting the owners of a gem
func TestRepository_GetGemOwners(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := NewRepository()

	owners, err := repo.GetGemOwners(ctx, "rails")
	assert.NoError(t, err, "getting gem owners should not return an error")
	assert.NotNil(t, owners, "owner list should not be nil")

	if len(owners) > 0 {
		t.Logf("number of owners of rails: %d", len(owners))
		assert.NotEmpty(t, owners[0].Handle, "owner handle should not be empty")
	}
}

// Test getting all gems owned by a specified user
func TestRepository_GetGemsByOwner(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := NewRepository()

	gems, err := repo.GetGemsByOwner(ctx, "qrush")
	assert.NoError(t, err, "getting gems owned by a user should not return an error")
	assert.NotNil(t, gems, "returned result should not be nil")

	if len(gems) > 0 {
		t.Logf("number of gems owned by qrush: %d", len(gems))
	}
}

// Test getting sigstore attestations
func TestRepository_GetAttestations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := NewRepository()

	attestations, err := repo.GetAttestations(ctx, "rails", "7.0.5")
	// Not all gem versions have attestations, 404 is normal
	if err != nil {
		if !IsNotFound(err) {
			t.Errorf("getting attestations returned an unexpected error: %v", err)
		}
	} else {
		t.Logf("number of attestations for rails 7.0.5: %d", len(attestations))
	}
}

// Test getting a custom repository (private gem server support)
func TestRepository_CustomRepository(t *testing.T) {
	// This test does not require network and can run in short mode
	repo := NewCustomRepository("https://my-private-gems.example.com")
	assert.NotNil(t, repo, "custom repository should not be nil")

	repoImpl, ok := repo.(*RepositoryImpl)
	assert.True(t, ok, "should be able to convert to *RepositoryImpl")
	assert.Equal(t, "https://my-private-gems.example.com", repoImpl.options.ServerURL,
		"custom repository URL should match")
}
