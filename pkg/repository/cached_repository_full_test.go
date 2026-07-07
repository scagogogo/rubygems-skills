package repository

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/cache"
	"github.com/scagogogo/rubygems-skills/pkg/models"
	"github.com/stretchr/testify/assert"
)

// scriptedRepo is a fully programmable Repository fake: every method returns a
// configurable (value, error) pair and records how many times it was called.
// It is used to drive CachedRepository coverage (cache miss / hit / error /
// wrong-type) for all 22 cached methods without hitting the network.
type scriptedRepo struct {
	mu sync.Mutex

	pkg       *models.PackageInformation
	pkgErr    error
	pkgCalls  int

	search       []*models.PackageInformation
	searchErr    error
	searchCalls  int

	versions       []*models.Version
	versionsErr    error
	versionsCalls  int

	latest       *models.LatestVersion
	latestErr    error
	latestCalls  int

	timeframe       []*models.Version
	timeframeErr    error
	timeframeCalls  int

	downloads       *models.RepositoryDownloadCount
	downloadsErr    error
	downloadsCalls  int

	vd       *models.VersionDownloadCount
	vdErr    error
	vdCalls  int

	deps       []*models.DependencyInfo
	depsErr    error
	depsCalls  int

	latestGems       []*models.PackageInformation
	latestGemsErr    error
	latestGemsCalls  int

	rdeps       []string
	rdepsErr    error
	rdepsCalls  int

	vrdeps       []string
	vrdepsErr    error
	vrdepsCalls  int

	autocomplete       []string
	autocompleteErr    error
	autocompleteCalls  int

	detail       *models.VersionDetail
	detailErr    error
	detailCalls  int

	justUpdated       []*models.PackageInformation
	justUpdatedErr    error
	justUpdatedCalls  int

	topDownloads       []*models.TopDownloadedGem
	topDownloadsErr    error
	topDownloadsCalls  int

	userProfile       *models.UserProfile
	userProfileErr    error
	userProfileCalls  int

	ownedGems       []*models.PackageInformation
	ownedGemsErr    error
	ownedGemsCalls  int

	gemsByOwner       []*models.PackageInformation
	gemsByOwnerErr    error
	gemsByOwnerCalls  int

	gemOwners       []*models.Owner
	gemOwnersErr    error
	gemOwnersCalls  int

	attestations       []*models.Attestation
	attestationsErr    error
	attestationsCalls  int

	versionContents       *models.VersionContent
	versionContentsErr    error
	versionContentsCalls  int

	mfa       *models.MFAStatus
	mfaErr    error
	mfaCalls  int
}

func newScriptedRepo() *scriptedRepo {
	return &scriptedRepo{
		pkg:            &models.PackageInformation{Name: "rails"},
		search:         []*models.PackageInformation{{Name: "rails"}},
		versions:       []*models.Version{{Number: "7.0.0"}},
		latest:         &models.LatestVersion{Version: "7.0.0"},
		timeframe:      []*models.Version{{Number: "1.0.0"}},
		downloads:      &models.RepositoryDownloadCount{TotalDownloads: 10},
		vd:             &models.VersionDownloadCount{VersionDownloads: 5},
		deps:           []*models.DependencyInfo{{Name: "rack"}},
		latestGems:     []*models.PackageInformation{{Name: "newgem"}},
		rdeps:          []string{"rack"},
		vrdeps:         []string{"rack"},
		autocomplete:   []string{"rails"},
		detail:         &models.VersionDetail{Number: "7.0.0"},
		justUpdated:    []*models.PackageInformation{{Name: "up"}},
		topDownloads:   []*models.TopDownloadedGem{{Name: "rails", Downloads: 100}},
		userProfile:    &models.UserProfile{ID: 1, Handle: "qrush"},
		ownedGems:      []*models.PackageInformation{{Name: "rails"}},
		gemsByOwner:    []*models.PackageInformation{{Name: "rails"}},
		gemOwners:      []*models.Owner{{Handle: "qrush"}},
		attestations:   []*models.Attestation{{Body: "sig"}},
		versionContents: &models.VersionContent{Files: map[string]string{"lib/x.rb": "sha"}},
		mfa:            &models.MFAStatus{Enabled: false, Level: "disabled"},
	}
}

func (s *scriptedRepo) GetPackage(ctx context.Context, gemName string) (*models.PackageInformation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pkgCalls++
	return s.pkg, s.pkgErr
}
func (s *scriptedRepo) Search(ctx context.Context, query string, page int) ([]*models.PackageInformation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.searchCalls++
	return s.search, s.searchErr
}
func (s *scriptedRepo) GetGemVersions(ctx context.Context, gemName string) ([]*models.Version, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.versionsCalls++
	return s.versions, s.versionsErr
}
func (s *scriptedRepo) GetGemLatestVersion(ctx context.Context, gemName string) (*models.LatestVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latestCalls++
	return s.latest, s.latestErr
}
func (s *scriptedRepo) GetTimeFrameVersions(ctx context.Context, from, to time.Time) ([]*models.Version, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.timeframeCalls++
	return s.timeframe, s.timeframeErr
}
func (s *scriptedRepo) Downloads(ctx context.Context) (*models.RepositoryDownloadCount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.downloadsCalls++
	return s.downloads, s.downloadsErr
}
func (s *scriptedRepo) VersionDownloads(ctx context.Context, gemName, gemVersion string) (*models.VersionDownloadCount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vdCalls++
	return s.vd, s.vdErr
}
func (s *scriptedRepo) GetDependencies(ctx context.Context, gemsNames ...string) ([]*models.DependencyInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.depsCalls++
	return s.deps, s.depsErr
}
func (s *scriptedRepo) LatestGems(ctx context.Context) ([]*models.PackageInformation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latestGemsCalls++
	return s.latestGems, s.latestGemsErr
}
func (s *scriptedRepo) GetReverseDependencies(ctx context.Context, gemName string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rdepsCalls++
	return s.rdeps, s.rdepsErr
}
func (s *scriptedRepo) GetVersionReverseDependencies(ctx context.Context, fullName string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vrdepsCalls++
	return s.vrdeps, s.vrdepsErr
}
func (s *scriptedRepo) SearchAutocomplete(ctx context.Context, query string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.autocompleteCalls++
	return s.autocomplete, s.autocompleteErr
}
func (s *scriptedRepo) GetGemVersionDetail(ctx context.Context, gemName, version string) (*models.VersionDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.detailCalls++
	return s.detail, s.detailErr
}
func (s *scriptedRepo) JustUpdatedGems(ctx context.Context) ([]*models.PackageInformation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.justUpdatedCalls++
	return s.justUpdated, s.justUpdatedErr
}
func (s *scriptedRepo) TopDownloads(ctx context.Context) ([]*models.TopDownloadedGem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.topDownloadsCalls++
	return s.topDownloads, s.topDownloadsErr
}
func (s *scriptedRepo) GetUserProfile(ctx context.Context, handleOrID string) (*models.UserProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userProfileCalls++
	return s.userProfile, s.userProfileErr
}
func (s *scriptedRepo) GetOwnedGems(ctx context.Context) ([]*models.PackageInformation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ownedGemsCalls++
	return s.ownedGems, s.ownedGemsErr
}
func (s *scriptedRepo) GetGemsByOwner(ctx context.Context, handleOrID string) ([]*models.PackageInformation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gemsByOwnerCalls++
	return s.gemsByOwner, s.gemsByOwnerErr
}
func (s *scriptedRepo) GetGemOwners(ctx context.Context, gemName string) ([]*models.Owner, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gemOwnersCalls++
	return s.gemOwners, s.gemOwnersErr
}
func (s *scriptedRepo) GetAttestations(ctx context.Context, gemName, version string) ([]*models.Attestation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attestationsCalls++
	return s.attestations, s.attestationsErr
}
func (s *scriptedRepo) GetGemVersionContents(ctx context.Context, gemName, version string) (*models.VersionContent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.versionContentsCalls++
	return s.versionContents, s.versionContentsErr
}
func (s *scriptedRepo) GetMFAStatus(ctx context.Context) (*models.MFAStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mfaCalls++
	return s.mfa, s.mfaErr
}
func (s *scriptedRepo) BulkGetPackages(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[*models.PackageInformation] {
	return nil
}
func (s *scriptedRepo) BulkGetVersions(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[[]*models.Version] {
	return nil
}
func (s *scriptedRepo) BulkGetDependencies(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[[]*models.DependencyInfo] {
	return nil
}
func (s *scriptedRepo) BulkGetReverseDependencies(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[[]string] {
	return nil
}

// newCachedRepoWith builds a CachedRepository backed by a fresh scripted repo
// and a fresh memory cache.
func newCachedRepoWith(t *testing.T) (*CachedRepository, *scriptedRepo) {
	t.Helper()
	repo := newScriptedRepo()
	c := NewCachedRepository(repo, 10*time.Minute, cache.NewMemoryCache(10*time.Minute, 30*time.Minute))
	return c, repo
}

// TestCachedRepository_AllMethods_MissThenHit verifies that every cached method
// calls the underlying repo on a miss and serves from cache on a hit (covering
// the cache-miss + cache-hit branches and the success type-assertion branch).
func TestCachedRepository_AllMethods_MissThenHit(t *testing.T) {
	ctx := context.Background()

	t.Run("GetPackage", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.GetPackage(ctx, "rails")
		_, _ = c.GetPackage(ctx, "rails")
		assert.Equal(t, 1, r.pkgCalls)
	})
	t.Run("Search", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.Search(ctx, "q", 1)
		_, _ = c.Search(ctx, "q", 1)
		assert.Equal(t, 1, r.searchCalls)
	})
	t.Run("GetGemVersions", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.GetGemVersions(ctx, "rails")
		_, _ = c.GetGemVersions(ctx, "rails")
		assert.Equal(t, 1, r.versionsCalls)
	})
	t.Run("GetGemLatestVersion", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.GetGemLatestVersion(ctx, "rails")
		_, _ = c.GetGemLatestVersion(ctx, "rails")
		assert.Equal(t, 1, r.latestCalls)
	})
	t.Run("GetTimeFrameVersions", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
		_, _ = c.GetTimeFrameVersions(ctx, from, to)
		_, _ = c.GetTimeFrameVersions(ctx, from, to)
		assert.Equal(t, 1, r.timeframeCalls)
	})
	t.Run("Downloads", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.Downloads(ctx)
		_, _ = c.Downloads(ctx)
		assert.Equal(t, 1, r.downloadsCalls)
	})
	t.Run("VersionDownloads", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.VersionDownloads(ctx, "rails", "7.0.0")
		_, _ = c.VersionDownloads(ctx, "rails", "7.0.0")
		assert.Equal(t, 1, r.vdCalls)
	})
	t.Run("GetDependencies", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.GetDependencies(ctx, "rails", "rack")
		_, _ = c.GetDependencies(ctx, "rails", "rack")
		assert.Equal(t, 1, r.depsCalls)
	})
	t.Run("LatestGems", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.LatestGems(ctx)
		_, _ = c.LatestGems(ctx)
		assert.Equal(t, 1, r.latestGemsCalls)
	})
	t.Run("GetReverseDependencies", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.GetReverseDependencies(ctx, "rails")
		_, _ = c.GetReverseDependencies(ctx, "rails")
		assert.Equal(t, 1, r.rdepsCalls)
	})
	t.Run("GetVersionReverseDependencies", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.GetVersionReverseDependencies(ctx, "rails-7.0.0")
		_, _ = c.GetVersionReverseDependencies(ctx, "rails-7.0.0")
		assert.Equal(t, 1, r.vrdepsCalls)
	})
	t.Run("SearchAutocomplete", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.SearchAutocomplete(ctx, "rai")
		_, _ = c.SearchAutocomplete(ctx, "rai")
		assert.Equal(t, 1, r.autocompleteCalls)
	})
	t.Run("GetGemVersionDetail", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.GetGemVersionDetail(ctx, "rails", "7.0.0")
		_, _ = c.GetGemVersionDetail(ctx, "rails", "7.0.0")
		assert.Equal(t, 1, r.detailCalls)
	})
	t.Run("JustUpdatedGems", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.JustUpdatedGems(ctx)
		_, _ = c.JustUpdatedGems(ctx)
		assert.Equal(t, 1, r.justUpdatedCalls)
	})
	t.Run("TopDownloads", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.TopDownloads(ctx)
		_, _ = c.TopDownloads(ctx)
		assert.Equal(t, 1, r.topDownloadsCalls)
	})
	t.Run("GetUserProfile", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.GetUserProfile(ctx, "qrush")
		_, _ = c.GetUserProfile(ctx, "qrush")
		assert.Equal(t, 1, r.userProfileCalls)
	})
	t.Run("GetOwnedGems", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.GetOwnedGems(ctx)
		_, _ = c.GetOwnedGems(ctx)
		assert.Equal(t, 1, r.ownedGemsCalls)
	})
	t.Run("GetGemsByOwner", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.GetGemsByOwner(ctx, "qrush")
		_, _ = c.GetGemsByOwner(ctx, "qrush")
		assert.Equal(t, 1, r.gemsByOwnerCalls)
	})
	t.Run("GetGemOwners", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.GetGemOwners(ctx, "rails")
		_, _ = c.GetGemOwners(ctx, "rails")
		assert.Equal(t, 1, r.gemOwnersCalls)
	})
	t.Run("GetAttestations", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.GetAttestations(ctx, "rails", "7.0.0")
		_, _ = c.GetAttestations(ctx, "rails", "7.0.0")
		assert.Equal(t, 1, r.attestationsCalls)
	})
	t.Run("GetGemVersionContents", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.GetGemVersionContents(ctx, "rails", "7.0.0")
		_, _ = c.GetGemVersionContents(ctx, "rails", "7.0.0")
		assert.Equal(t, 1, r.versionContentsCalls)
	})
	t.Run("GetMFAStatus", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		_, _ = c.GetMFAStatus(ctx)
		_, _ = c.GetMFAStatus(ctx)
		assert.Equal(t, 1, r.mfaCalls)
	})
}

// TestCachedRepository_AllMethods_ErrorPropagation verifies the error branch
// (cache miss -> underlying returns error -> return error without caching).
func TestCachedRepository_AllMethods_ErrorPropagation(t *testing.T) {
	ctx := context.Background()
	errAny := errors.New("boom")

	t.Run("GetPackage", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.pkgErr = errAny
		_, err := c.GetPackage(ctx, "rails")
		assert.Error(t, err)
	})
	t.Run("Search", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.searchErr = errAny
		_, err := c.Search(ctx, "q", 1)
		assert.Error(t, err)
	})
	t.Run("GetGemVersions", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.versionsErr = errAny
		_, err := c.GetGemVersions(ctx, "rails")
		assert.Error(t, err)
	})
	t.Run("GetGemLatestVersion", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.latestErr = errAny
		_, err := c.GetGemLatestVersion(ctx, "rails")
		assert.Error(t, err)
	})
	t.Run("GetTimeFrameVersions", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.timeframeErr = errAny
		_, err := c.GetTimeFrameVersions(ctx, time.Now(), time.Now())
		assert.Error(t, err)
	})
	t.Run("Downloads", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.downloadsErr = errAny
		_, err := c.Downloads(ctx)
		assert.Error(t, err)
	})
	t.Run("VersionDownloads", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.vdErr = errAny
		_, err := c.VersionDownloads(ctx, "rails", "7.0.0")
		assert.Error(t, err)
	})
	t.Run("GetDependencies", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.depsErr = errAny
		_, err := c.GetDependencies(ctx, "rails")
		assert.Error(t, err)
	})
	t.Run("LatestGems", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.latestGemsErr = errAny
		_, err := c.LatestGems(ctx)
		assert.Error(t, err)
	})
	t.Run("GetReverseDependencies", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.rdepsErr = errAny
		_, err := c.GetReverseDependencies(ctx, "rails")
		assert.Error(t, err)
	})
	t.Run("GetVersionReverseDependencies", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.vrdepsErr = errAny
		_, err := c.GetVersionReverseDependencies(ctx, "rails-7.0.0")
		assert.Error(t, err)
	})
	t.Run("SearchAutocomplete", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.autocompleteErr = errAny
		_, err := c.SearchAutocomplete(ctx, "rai")
		assert.Error(t, err)
	})
	t.Run("GetGemVersionDetail", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.detailErr = errAny
		_, err := c.GetGemVersionDetail(ctx, "rails", "7.0.0")
		assert.Error(t, err)
	})
	t.Run("JustUpdatedGems", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.justUpdatedErr = errAny
		_, err := c.JustUpdatedGems(ctx)
		assert.Error(t, err)
	})
	t.Run("TopDownloads", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.topDownloadsErr = errAny
		_, err := c.TopDownloads(ctx)
		assert.Error(t, err)
	})
	t.Run("GetUserProfile", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.userProfileErr = errAny
		_, err := c.GetUserProfile(ctx, "qrush")
		assert.Error(t, err)
	})
	t.Run("GetOwnedGems", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.ownedGemsErr = errAny
		_, err := c.GetOwnedGems(ctx)
		assert.Error(t, err)
	})
	t.Run("GetGemsByOwner", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.gemsByOwnerErr = errAny
		_, err := c.GetGemsByOwner(ctx, "qrush")
		assert.Error(t, err)
	})
	t.Run("GetGemOwners", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.gemOwnersErr = errAny
		_, err := c.GetGemOwners(ctx, "rails")
		assert.Error(t, err)
	})
	t.Run("GetAttestations", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.attestationsErr = errAny
		_, err := c.GetAttestations(ctx, "rails", "7.0.0")
		assert.Error(t, err)
	})
	t.Run("GetGemVersionContents", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.versionContentsErr = errAny
		_, err := c.GetGemVersionContents(ctx, "rails", "7.0.0")
		assert.Error(t, err)
	})
	t.Run("GetMFAStatus", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		r.mfaErr = errAny
		_, err := c.GetMFAStatus(ctx)
		assert.Error(t, err)
	})
}

// TestCachedRepository_WrongTypeInCache covers the failed type-assertion branch:
// a value of the wrong type is stored under the cache key, so the cached method
// falls through to the underlying repo.
func TestCachedRepository_WrongTypeInCache(t *testing.T) {
	ctx := context.Background()

	t.Run("GetPackage", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("package:rails", "not-a-pkg")
		_, _ = c.GetPackage(ctx, "rails")
		assert.Equal(t, 1, r.pkgCalls) // fell through to repo
	})
	t.Run("Search", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("search:q:1", "not-a-slice")
		_, _ = c.Search(ctx, "q", 1)
		assert.Equal(t, 1, r.searchCalls)
	})
	t.Run("GetGemVersions", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("versions:rails", 123)
		_, _ = c.GetGemVersions(ctx, "rails")
		assert.Equal(t, 1, r.versionsCalls)
	})
	t.Run("GetGemLatestVersion", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("latest_version:rails", 123)
		_, _ = c.GetGemLatestVersion(ctx, "rails")
		assert.Equal(t, 1, r.latestCalls)
	})
	t.Run("GetTimeFrameVersions", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
		key := "timeframe:" + from.Format(time.RFC3339) + ":" + to.Format(time.RFC3339)
		c.cache.Set(key, 123)
		_, _ = c.GetTimeFrameVersions(ctx, from, to)
		assert.Equal(t, 1, r.timeframeCalls)
	})
	t.Run("Downloads", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("downloads", 123)
		_, _ = c.Downloads(ctx)
		assert.Equal(t, 1, r.downloadsCalls)
	})
	t.Run("VersionDownloads", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("version_downloads:rails:7.0.0", 123)
		_, _ = c.VersionDownloads(ctx, "rails", "7.0.0")
		assert.Equal(t, 1, r.vdCalls)
	})
	t.Run("GetDependencies", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("dependencies:rails,rack", 123)
		_, _ = c.GetDependencies(ctx, "rails", "rack")
		assert.Equal(t, 1, r.depsCalls)
	})
	t.Run("LatestGems", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("latest_gems", 123)
		_, _ = c.LatestGems(ctx)
		assert.Equal(t, 1, r.latestGemsCalls)
	})
	t.Run("GetReverseDependencies", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("reverse_dependencies:rails", 123)
		_, _ = c.GetReverseDependencies(ctx, "rails")
		assert.Equal(t, 1, r.rdepsCalls)
	})
	t.Run("GetVersionReverseDependencies", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("version_reverse_dependencies:rails-7.0.0", 123)
		_, _ = c.GetVersionReverseDependencies(ctx, "rails-7.0.0")
		assert.Equal(t, 1, r.vrdepsCalls)
	})
	t.Run("SearchAutocomplete", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("autocomplete:rai", 123)
		_, _ = c.SearchAutocomplete(ctx, "rai")
		assert.Equal(t, 1, r.autocompleteCalls)
	})
	t.Run("GetGemVersionDetail", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("version_detail:rails:7.0.0", 123)
		_, _ = c.GetGemVersionDetail(ctx, "rails", "7.0.0")
		assert.Equal(t, 1, r.detailCalls)
	})
	t.Run("JustUpdatedGems", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("just_updated_gems", 123)
		_, _ = c.JustUpdatedGems(ctx)
		assert.Equal(t, 1, r.justUpdatedCalls)
	})
	t.Run("TopDownloads", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("top_downloads", 123)
		_, _ = c.TopDownloads(ctx)
		assert.Equal(t, 1, r.topDownloadsCalls)
	})
	t.Run("GetUserProfile", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("user_profile:qrush", 123)
		_, _ = c.GetUserProfile(ctx, "qrush")
		assert.Equal(t, 1, r.userProfileCalls)
	})
	t.Run("GetOwnedGems", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("owned_gems", 123)
		_, _ = c.GetOwnedGems(ctx)
		assert.Equal(t, 1, r.ownedGemsCalls)
	})
	t.Run("GetGemsByOwner", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("gems_by_owner:qrush", 123)
		_, _ = c.GetGemsByOwner(ctx, "qrush")
		assert.Equal(t, 1, r.gemsByOwnerCalls)
	})
	t.Run("GetGemOwners", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("gem_owners:rails", 123)
		_, _ = c.GetGemOwners(ctx, "rails")
		assert.Equal(t, 1, r.gemOwnersCalls)
	})
	t.Run("GetAttestations", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("attestations:rails:7.0.0", 123)
		_, _ = c.GetAttestations(ctx, "rails", "7.0.0")
		assert.Equal(t, 1, r.attestationsCalls)
	})
	t.Run("GetGemVersionContents", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("version_contents:rails:7.0.0", 123)
		_, _ = c.GetGemVersionContents(ctx, "rails", "7.0.0")
		assert.Equal(t, 1, r.versionContentsCalls)
	})
	t.Run("GetMFAStatus", func(t *testing.T) {
		c, r := newCachedRepoWith(t)
		c.cache.Set("mfa_status", 123)
		_, _ = c.GetMFAStatus(ctx)
		assert.Equal(t, 1, r.mfaCalls)
	})
}

// TestCachedRepository_NewWithNilCache covers the cacheImpl==nil branch of
// NewCachedRepository (creates its own memory cache).
func TestCachedRepository_NewWithNilCache(t *testing.T) {
	repo := newScriptedRepo()
	c := NewCachedRepository(repo, 10*time.Minute, nil)
	assert.NotNil(t, c)
	assert.NotNil(t, c.cache)
	// Should still work end-to-end.
	pkg, err := c.GetPackage(context.Background(), "rails")
	assert.NoError(t, err)
	assert.Equal(t, "rails", pkg.Name)
}

// TestCachedRepository_CloseClearStats covers Close / ClearCache / GetCacheStats.
func TestCachedRepository_CloseClearStats(t *testing.T) {
	ctx := context.Background()
	c, _ := newCachedRepoWith(t)
	_, _ = c.GetPackage(ctx, "rails")
	_, _ = c.Search(ctx, "q", 1)
	assert.Equal(t, 2, c.GetCacheStats())

	c.ClearCache()
	assert.Equal(t, 0, c.GetCacheStats())

	c.Close()
}

// TestCachedRepository_BulkDelegation covers the four bulk delegation methods.
func TestCachedRepository_BulkDelegation(t *testing.T) {
	ctx := context.Background()
	c, _ := newCachedRepoWith(t)
	// The scripted repo returns nil for all bulk methods; the cached wrapper
	// must delegate transparently.
	assert.Nil(t, c.BulkGetPackages(ctx, []string{"a", "b"}, nil))
	assert.Nil(t, c.BulkGetVersions(ctx, []string{"a"}, nil))
	assert.Nil(t, c.BulkGetDependencies(ctx, []string{"a"}, nil))
	assert.Nil(t, c.BulkGetReverseDependencies(ctx, []string{"a"}, nil))
}
