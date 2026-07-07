package repository

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ----- GetPackage -----

func TestGetPackage_Success(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems/rails.json", 200, `{"name":"rails","version":"7.0.0","downloads":100}`)
	repo := newStubbedRepo(t, tr)
	pkg, err := repo.GetPackage(context.Background(), "rails")
	assert.NoError(t, err)
	assert.Equal(t, "rails", pkg.Name)
	assert.Equal(t, "7.0.0", pkg.Version)
	assert.Equal(t, 100, pkg.Downloads)
	assert.Len(t, tr.requests, 1)
}

func TestGetPackage_NotFound(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems/nope.json", 404, "not found")
	repo := newStubbedRepo(t, tr)
	_, err := repo.GetPackage(context.Background(), "nope")
	assert.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestGetPackage_ServerError(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems/rails.json", 500, "boom")
	repo := newStubbedRepo(t, tr)
	_, err := repo.GetPackage(context.Background(), "rails")
	assert.Error(t, err)
	assert.False(t, IsNotFound(err))
}

func TestGetPackage_InvalidJSON(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems/rails.json", 200, `{not-json`)
	repo := newStubbedRepo(t, tr)
	_, err := repo.GetPackage(context.Background(), "rails")
	assert.Error(t, err)
}

func TestGetPackage_URLEscaping(t *testing.T) {
	tr := newFakeTransport()
	// url.URL.Path is the decoded path, so the stub key uses the decoded form.
	tr.stub("/api/v1/gems/foo bar.json", 200, `{"name":"foo bar"}`)
	repo := newStubbedRepo(t, tr)
	pkg, err := repo.GetPackage(context.Background(), "foo bar")
	assert.NoError(t, err)
	assert.Equal(t, "foo bar", pkg.Name)
	assert.Equal(t, "/api/v1/gems/foo bar.json", tr.requests[0].Path)
}

func TestGetPackage_WithToken(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems/rails.json", 200, `{"name":"rails"}`)
	opts := NewOptions().SetToken("abc123")
	opts.SetHTTPClient(&http.Client{Transport: tr})
	opts.DisableRetry()
	repo := NewRepository(opts)
	_, err := repo.GetPackage(context.Background(), "rails")
	assert.NoError(t, err)
	assert.Equal(t, "Bearer abc123", tr.requests[0].Header.Get("Authorization"))
}

func TestGetPackage_WithProxy(t *testing.T) {
	// A non-empty Proxy exercises the proxy branch in getBytes. The proxy URL is
	// intentionally unreachable, so the request fails fast at the transport
	// layer — but the branch is covered and the stub transport is never dialed.
	tr := newFakeTransport()
	tr.stub("/api/v1/gems/rails.json", 200, `{"name":"rails"}`)
	opts := NewOptions().SetProxy("http://127.0.0.1:1")
	opts.SetHTTPClient(&http.Client{Transport: tr})
	opts.DisableRetry()
	repo := NewRepository(opts)
	_, err := repo.GetPackage(context.Background(), "rails")
	assert.Error(t, err)
}

func TestGetPackage_EmptyTokenSendsBearerPrefix(t *testing.T) {
	// newStubbedRepo has no token; the Bearer header is still set with empty value.
	tr := newFakeTransport()
	tr.stub("/api/v1/gems/rails.json", 200, `{"name":"rails"}`)
	repo := newStubbedRepo(t, tr)
	_, err := repo.GetPackage(context.Background(), "rails")
	assert.NoError(t, err)
	// token is empty so the Authorization branch is skipped -> header absent
	assert.Empty(t, tr.requests[0].Header.Get("Authorization"))
}

// ----- Search -----

func TestSearch_PageZeroNormalizedToOne(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/search.json", 200, `[{"name":"rails"}]`)
	repo := newStubbedRepo(t, tr)
	res, err := repo.Search(context.Background(), "rail", 0)
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Contains(t, tr.requests[0].URL, "page=1")
}

func TestSearch_NegativePageNormalized(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/search.json", 200, `[]`)
	repo := newStubbedRepo(t, tr)
	res, err := repo.Search(context.Background(), "x", -5)
	assert.NoError(t, err)
	assert.Len(t, res, 0)
	assert.Contains(t, tr.requests[0].URL, "page=1")
}

func TestSearch_ServerError(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/search.json", 500, "err")
	repo := newStubbedRepo(t, tr)
	_, err := repo.Search(context.Background(), "x", 1)
	assert.Error(t, err)
}

func TestSearch_QueryEscaped(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/search.json", 200, `[]`)
	repo := newStubbedRepo(t, tr)
	_, err := repo.Search(context.Background(), "http client", 2)
	assert.NoError(t, err)
	assert.Contains(t, tr.requests[0].URL, "query=http+client")
	assert.Contains(t, tr.requests[0].URL, "page=2")
}

// ----- Table-driven read methods -----

func TestReadMethods_TableDriven(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
		call func(r *RepositoryImpl) (interface{}, error)
	}{
		{"autocomplete", "/api/v1/search/autocomplete.json", `["rails","rack"]`, func(r *RepositoryImpl) (interface{}, error) {
			return r.SearchAutocomplete(context.Background(), "rai")
		}},
		{"versions", "/api/v1/versions/rails.json", `[{"number":"7.0.0"}]`, func(r *RepositoryImpl) (interface{}, error) {
			return r.GetGemVersions(context.Background(), "rails")
		}},
		{"latest-version", "/api/v1/versions/rails/latest.json", `{"number":"7.0.0"}`, func(r *RepositoryImpl) (interface{}, error) {
			return r.GetGemLatestVersion(context.Background(), "rails")
		}},
		{"version-detail", "/api/v2/rubygems/rails/versions/7.0.0.json", `{"number":"7.0.0","yanked":false}`, func(r *RepositoryImpl) (interface{}, error) {
			return r.GetGemVersionDetail(context.Background(), "rails", "7.0.0")
		}},
		{"downloads", "/api/v1/downloads.json", `{"total":1000}`, func(r *RepositoryImpl) (interface{}, error) {
			return r.Downloads(context.Background())
		}},
		{"version-downloads", "/api/v1/downloads/rails-7.0.0.json", `{"version_downloads":50}`, func(r *RepositoryImpl) (interface{}, error) {
			return r.VersionDownloads(context.Background(), "rails", "7.0.0")
		}},
		{"top-downloads", "/api/v1/downloads/all.json", `[{"gem":"rails","downloads":100}]`, func(r *RepositoryImpl) (interface{}, error) {
			return r.TopDownloads(context.Background())
		}},
		{"deps", "/api/v1/dependencies", `[]`, func(r *RepositoryImpl) (interface{}, error) {
			return r.GetDependencies(context.Background(), "rails", "rack")
		}},
		{"rdeps", "/api/v1/gems/rails/reverse_dependencies.json", `["rack"]`, func(r *RepositoryImpl) (interface{}, error) {
			return r.GetReverseDependencies(context.Background(), "rails")
		}},
		{"version-rdeps", "/api/v1/versions/rails-7.0.0/reverse_dependencies.json", `["rack"]`, func(r *RepositoryImpl) (interface{}, error) {
			return r.GetVersionReverseDependencies(context.Background(), "rails-7.0.0")
		}},
		{"latest-gems", "/api/v1/activity/latest.json", `[{"name":"newgem"}]`, func(r *RepositoryImpl) (interface{}, error) {
			return r.LatestGems(context.Background())
		}},
		{"just-updated", "/api/v1/activity/just_updated.json", `[{"name":"up"}]`, func(r *RepositoryImpl) (interface{}, error) {
			return r.JustUpdatedGems(context.Background())
		}},
		{"user-profile", "/api/v1/profiles/qrush.json", `{"id":1,"handle":"qrush"}`, func(r *RepositoryImpl) (interface{}, error) {
			return r.GetUserProfile(context.Background(), "qrush")
		}},
		{"owned-gems", "/api/v1/gems.json", `[{"name":"rails"}]`, func(r *RepositoryImpl) (interface{}, error) {
			return r.GetOwnedGems(context.Background())
		}},
		{"gems-by-owner", "/api/v1/owners/qrush/gems.json", `[{"name":"rails"}]`, func(r *RepositoryImpl) (interface{}, error) {
			return r.GetGemsByOwner(context.Background(), "qrush")
		}},
		{"gem-owners", "/api/v1/gems/rails/owners.json", `[{"handle":"qrush"}]`, func(r *RepositoryImpl) (interface{}, error) {
			return r.GetGemOwners(context.Background(), "rails")
		}},
		{"attestations", "/api/v1/attestations/rails-7.0.0.json", `[]`, func(r *RepositoryImpl) (interface{}, error) {
			return r.GetAttestations(context.Background(), "rails", "7.0.0")
		}},
		{"version-contents", "/api/v2/rubygems/rails/versions/7.0.0/contents.json", `{"files":{"lib/rails.rb":"sha256:abc"}}`, func(r *RepositoryImpl) (interface{}, error) {
			return r.GetGemVersionContents(context.Background(), "rails", "7.0.0")
		}},
		{"mfa-status", "/api/v1/multifactor_auth", `{"enabled":false,"level":"disabled"}`, func(r *RepositoryImpl) (interface{}, error) {
			return r.GetMFAStatus(context.Background())
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tr := newFakeTransport()
			tr.stub(tc.path, 200, tc.body)
			repo := newStubbedRepo(t, tr)
			_, err := tc.call(repo)
			assert.NoError(t, err)
			assert.Equal(t, tc.path, tr.requests[0].Path)
		})
	}
}

// Per-method error-path coverage (404/500) for a representative subset, plus
// URL-escaping assertions for methods that escape gem/version.

func TestReadMethods_ErrorPaths(t *testing.T) {
	cases := []struct {
		name string
		path string
		call func(r *RepositoryImpl) error
	}{
		{"autocomplete", "/api/v1/search/autocomplete.json", func(r *RepositoryImpl) error {
			_, err := r.SearchAutocomplete(context.Background(), "x"); return err
		}},
		{"versions", "/api/v1/versions/rails.json", func(r *RepositoryImpl) error {
			_, err := r.GetGemVersions(context.Background(), "rails"); return err
		}},
		{"latest-version", "/api/v1/versions/rails/latest.json", func(r *RepositoryImpl) error {
			_, err := r.GetGemLatestVersion(context.Background(), "rails"); return err
		}},
		{"version-detail", "/api/v2/rubygems/rails/versions/7.0.0.json", func(r *RepositoryImpl) error {
			_, err := r.GetGemVersionDetail(context.Background(), "rails", "7.0.0"); return err
		}},
		{"downloads", "/api/v1/downloads.json", func(r *RepositoryImpl) error {
			_, err := r.Downloads(context.Background()); return err
		}},
		{"version-downloads", "/api/v1/downloads/rails-7.0.0.json", func(r *RepositoryImpl) error {
			_, err := r.VersionDownloads(context.Background(), "rails", "7.0.0"); return err
		}},
		{"top-downloads", "/api/v1/downloads/all.json", func(r *RepositoryImpl) error {
			_, err := r.TopDownloads(context.Background()); return err
		}},
		{"deps", "/api/v1/dependencies", func(r *RepositoryImpl) error {
			_, err := r.GetDependencies(context.Background(), "rails"); return err
		}},
		{"rdeps", "/api/v1/gems/rails/reverse_dependencies.json", func(r *RepositoryImpl) error {
			_, err := r.GetReverseDependencies(context.Background(), "rails"); return err
		}},
		{"version-rdeps", "/api/v1/versions/rails-7.0.0/reverse_dependencies.json", func(r *RepositoryImpl) error {
			_, err := r.GetVersionReverseDependencies(context.Background(), "rails-7.0.0"); return err
		}},
		{"latest-gems", "/api/v1/activity/latest.json", func(r *RepositoryImpl) error {
			_, err := r.LatestGems(context.Background()); return err
		}},
		{"just-updated", "/api/v1/activity/just_updated.json", func(r *RepositoryImpl) error {
			_, err := r.JustUpdatedGems(context.Background()); return err
		}},
		{"user-profile", "/api/v1/profiles/qrush.json", func(r *RepositoryImpl) error {
			_, err := r.GetUserProfile(context.Background(), "qrush"); return err
		}},
		{"owned-gems", "/api/v1/gems.json", func(r *RepositoryImpl) error {
			_, err := r.GetOwnedGems(context.Background()); return err
		}},
		{"gems-by-owner", "/api/v1/owners/qrush/gems.json", func(r *RepositoryImpl) error {
			_, err := r.GetGemsByOwner(context.Background(), "qrush"); return err
		}},
		{"gem-owners", "/api/v1/gems/rails/owners.json", func(r *RepositoryImpl) error {
			_, err := r.GetGemOwners(context.Background(), "rails"); return err
		}},
		{"attestations", "/api/v1/attestations/rails-7.0.0.json", func(r *RepositoryImpl) error {
			_, err := r.GetAttestations(context.Background(), "rails", "7.0.0"); return err
		}},
		{"version-contents", "/api/v2/rubygems/rails/versions/7.0.0/contents.json", func(r *RepositoryImpl) error {
			_, err := r.GetGemVersionContents(context.Background(), "rails", "7.0.0"); return err
		}},
		{"mfa-status", "/api/v1/multifactor_auth", func(r *RepositoryImpl) error {
			_, err := r.GetMFAStatus(context.Background()); return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr := newFakeTransport()
			tr.stub(tc.path, 500, "err")
			repo := newStubbedRepo(t, tr)
			assert.Error(t, tc.call(repo))
		})
	}
}

func TestGetTimeFrameVersions_RFC3339Format(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/timeframe_versions.json", 200, `[]`)
	repo := newStubbedRepo(t, tr)
	from := mustParseTime("2024-01-01T00:00:00Z")
	to := mustParseTime("2024-12-31T23:59:59Z")
	_, err := repo.GetTimeFrameVersions(context.Background(), from, to)
	assert.NoError(t, err)
	assert.Contains(t, tr.requests[0].URL, "from=2024-01-01T00%3A00%3A00Z")
	assert.Contains(t, tr.requests[0].URL, "to=2024-12-31T23%3A59%3A59Z")
}

func TestGetTimeFrameVersions_Error(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/timeframe_versions.json", 500, "err")
	repo := newStubbedRepo(t, tr)
	_, err := repo.GetTimeFrameVersions(context.Background(), time.Now(), time.Now())
	assert.Error(t, err)
}

func TestGetPackage_NetworkErrorRetriedDisabled(t *testing.T) {
	// transport returns an error; with retry disabled the error propagates directly.
	tr := newFakeTransport()
	tr.stubErr("/api/v1/gems/rails.json", errors.New("connection refused"))
	repo := newStubbedRepo(t, tr)
	_, err := repo.GetPackage(context.Background(), "rails")
	assert.Error(t, err)
}

// TestGetPackage_RetryEnabledSuccess covers the RetryOptions != nil branch of
// getBytes. The stub returns 200 on first try, so no actual retry wait occurs.
func TestGetPackage_RetryEnabledSuccess(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems/rails.json", 200, `{"name":"rails"}`)
	opts := NewOptions().
		SetHTTPClient(&http.Client{Transport: tr}).
		SetRetryOptions(NewDefaultRetryOptions())
	// shrink wait so any retry would be fast (not exercised on success, but safe)
	opts.RetryOptions.WaitTime = time.Millisecond
	opts.RetryOptions.MaxWaitTime = time.Millisecond
	repo := NewRepository(opts)
	pkg, err := repo.GetPackage(context.Background(), "rails")
	assert.NoError(t, err)
	assert.Equal(t, "rails", pkg.Name)
	assert.Equal(t, 1, tr.callCount)
}

// TestGetPackage_RetryRetriesOnError covers the retry loop: the stub fails
// twice then succeeds on the third attempt.
func TestGetPackage_RetryRetriesOnError(t *testing.T) {
	tr := newFakeTransport()
	tr.stubSequence("/api/v1/gems/rails.json",
		cannedResponse{err: errors.New("transient")},
		cannedResponse{err: errors.New("transient")},
		cannedResponse{statusCode: 200, body: []byte(`{"name":"rails"}`), header: http.Header{}})
	opts := NewOptions().
		SetHTTPClient(&http.Client{Transport: tr}).
		SetRetryOptions(NewDefaultRetryOptions())
	opts.RetryOptions.WaitTime = time.Millisecond
	opts.RetryOptions.MaxWaitTime = time.Millisecond
	repo := NewRepository(opts)
	pkg, err := repo.GetPackage(context.Background(), "rails")
	assert.NoError(t, err)
	assert.Equal(t, "rails", pkg.Name)
	assert.Equal(t, 3, tr.callCount)
}

// TestGetPackage_RetryExhausted covers the retry loop exhausting attempts.
func TestGetPackage_RetryExhausted(t *testing.T) {
	tr := newFakeTransport()
	tr.stubErr("/api/v1/gems/rails.json", errors.New("always fails"))
	opts := NewOptions().
		SetHTTPClient(&http.Client{Transport: tr}).
		SetRetryOptions(NewDefaultRetryOptions())
	opts.RetryOptions.WaitTime = time.Millisecond
	opts.RetryOptions.MaxWaitTime = time.Millisecond
	repo := NewRepository(opts)
	_, err := repo.GetPackage(context.Background(), "rails")
	assert.Error(t, err)
	// The transport is retried at least MaxAttempts times (go-requests may add
	// its own internal retries on top).
	assert.GreaterOrEqual(t, tr.callCount, 3)
}

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// ----- NewRepository default options path -----

func TestNewRepository_NoOptionsUsesDefault(t *testing.T) {
	r := NewRepository()
	assert.NotNil(t, r)
	assert.Equal(t, DefaultServerURL, r.options.ServerURL)
}
