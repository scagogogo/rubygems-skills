// Package repository provides a Go client implementation for the RubyGems API
// It supports the official source and multiple domestic mirror sources, with error handling, retry mechanism and caching support
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/crawler-go-go-go/go-requests"
	"github.com/scagogogo/rubygems-skills/pkg/models"
)

// Compile-time check that RepositoryImpl implements the Repository interface
var _ Repository = (*RepositoryImpl)(nil)

// https://guides.rubygems.org/api/v2/rubygems/[GEM%20NAME]/versions/[VERSION%20NUMBER].(json%7Cyaml)

// Repository defines the interface for RubyGems API operations
// it includes all core methods for interacting with RubyGems
type Repository interface {
	// ======== Package info queries ========

	// GetPackage get detailed package info by name
	// package info includes name, version, authors, downloads, homepage URL, etc.
	// returns NotFound error if package doesn't exist
	// GET /api/v1/gems/{gem}.json
	GetPackage(ctx context.Context, gemName string) (*models.PackageInformation, error)

	// Search search packages by query string
	// query parameter can be part of the package name
	// results sorted by relevance and popularity
	// returns empty slice instead of error if no matches
	// GET /api/v1/search.json?query={query}&page={page}
	Search(ctx context.Context, query string, page int) ([]*models.PackageInformation, error)

	// SearchAutocomplete return search autocomplete suggestions
	// return matching package name list
	// GET /api/v1/search/autocomplete.json?query={query}
	SearchAutocomplete(ctx context.Context, query string) ([]string, error)

	// ======== Version info queries ========

	// GetGemVersions get all version info for specified package
	// versions sorted by release time descending (latest version first)
	// GET /api/v1/versions/{gem}.json
	GetGemVersions(ctx context.Context, gemName string) ([]*models.Version, error)

	// GetGemLatestVersion get latest version of given package
	// GET /api/v1/versions/{gem}/latest.json
	GetGemLatestVersion(ctx context.Context, gemName string) (*models.LatestVersion, error)

	// GetGemVersionDetail get detailed info of specific package version (API v2)
	// more detailed than V1 version info, includes spec_sha, yanked status and full dependency info
	// GET /api/v2/rubygems/{gem}/versions/{version}.json
	GetGemVersionDetail(ctx context.Context, gemName, version string) (*models.VersionDetail, error)

	// GetTimeFrameVersions get version info within specific time range
	// GET /api/v1/timeframe_versions.json
	GetTimeFrameVersions(ctx context.Context, from, to time.Time) ([]*models.Version, error)

	// ======== Download statistics ========

	// Downloads get total download count for this repository
	// GET /api/v1/downloads.json
	Downloads(ctx context.Context) (*models.RepositoryDownloadCount, error)

	// VersionDownloads get download count for specific package version
	// GET /api/v1/downloads/{gem}-{version}.json
	VersionDownloads(ctx context.Context, gemName, gemVersion string) (*models.VersionDownloadCount, error)

	// TopDownloads get top 50 most downloaded gems
	// GET /api/v1/downloads/all.json
	TopDownloads(ctx context.Context) ([]*models.TopDownloadedGem, error)

	// ======== Dependencies ========

	// GetDependencies get dependencies of specified gem
	// GET /api/v1/dependencies?gems={comma-separated-names}
	GetDependencies(ctx context.Context, gemsNames ...string) ([]*models.DependencyInfo, error)

	// GetReverseDependencies get all packages depending on specified gem
	// GET /api/v1/gems/{gem}/reverse_dependencies.json
	GetReverseDependencies(ctx context.Context, gemName string) ([]string, error)

	// GetVersionReverseDependencies get packages that depend on a specific version of a gem
	// The fullName parameter should be in the format "gemname-version" (e.g., "rails-7.0.5")
	// GET /api/v1/versions/{fullName}/reverse_dependencies.json
	GetVersionReverseDependencies(ctx context.Context, fullName string) ([]string, error)

	// ======== Activity info ========

	// LatestGems get latest published gems
	// GET /api/v1/activity/latest.json
	LatestGems(ctx context.Context) ([]*models.PackageInformation, error)

	// JustUpdatedGems get 50 most recently updated gems (gems with new versions)
	// GET /api/v1/activity/just_updated.json
	JustUpdatedGems(ctx context.Context) ([]*models.PackageInformation, error)

	// ======== Users and owners ========

	// GetUserProfile get public profile of specified user
	// GET /api/v1/profiles/{handle_or_id}.json
	GetUserProfile(ctx context.Context, handleOrID string) (*models.UserProfile, error)

	// GetOwnedGems get all gems owned by current authenticated user
	// requires API Token authentication
	// GET /api/v1/gems.json
	GetOwnedGems(ctx context.Context) ([]*models.PackageInformation, error)

	// GetGemsByOwner get all gems owned by specified user
	// GET /api/v1/owners/{handle_or_id}/gems.json
	GetGemsByOwner(ctx context.Context, handleOrID string) ([]*models.PackageInformation, error)

	// GetGemOwners get all owners of specified gem
	// GET /api/v1/gems/{gem}/owners.json
	GetGemOwners(ctx context.Context, gemName string) ([]*models.Owner, error)

	// ======== Attestations and verification ========

	// GetAttestations get sigstore attestations for specified gem version
	// GET /api/v1/attestations/{gem}-{version}.json
	GetAttestations(ctx context.Context, gemName, version string) ([]*models.Attestation, error)

	// GetGemVersionContents get file checksums/manifest for specified gem version
	// GET /api/v2/rubygems/{gem}/versions/{version}/contents.json
	GetGemVersionContents(ctx context.Context, gemName, version string) (*models.VersionContent, error)

	// ======== MFA status ========

	// GetMFAStatus check MFA status for the authenticated user
	// requires API Token authentication
	// GET /api/v1/multifactor_auth
	GetMFAStatus(ctx context.Context) (*models.MFAStatus, error)

	// ======== Bulk operations ========

	// BulkGetPackages bulk get info of multiple packages
	BulkGetPackages(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[*models.PackageInformation]

	// BulkGetVersions bulk get version info of multiple packages
	BulkGetVersions(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[[]*models.Version]

	// BulkGetDependencies bulk get dependency info of multiple packages
	BulkGetDependencies(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[[]*models.DependencyInfo]

	// BulkGetReverseDependencies bulk get reverse dependency info of multiple packages
	BulkGetReverseDependencies(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[[]string]
}

type RepositoryImpl struct {
	options *Options
}

// NewRepository create a repository, gems are stored in repositories
func NewRepository(options ...*Options) *RepositoryImpl {
	if len(options) == 0 {
		options = append(options, NewOptions())
	}
	return &RepositoryImpl{
		options: options[0],
	}
}

// GetPackage get basic info of a gem package
// GetPackage GET - /api/v1/gems/[GEM NAME].(json|yaml)
func (x *RepositoryImpl) GetPackage(ctx context.Context, gemName string) (*models.PackageInformation, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/gems/%s.json", x.options.ServerURL, url.PathEscape(gemName))
	return getJson[*models.PackageInformation](ctx, x, targetUrl)
}

// Search search packages in the repository matching the criteria, use page parameter for pagination, empty response list means last page
// GET - /api/v1/search.(json|yaml)?query=[YOUR QUERY]
func (x *RepositoryImpl) Search(ctx context.Context, query string, page int) ([]*models.PackageInformation, error) {
	if page <= 0 {
		page = 1
	}
	targetUrl := fmt.Sprintf("%s/api/v1/search.json?query=%s&page=%d", x.options.ServerURL, url.QueryEscape(query), page)
	return getJson[[]*models.PackageInformation](ctx, x, targetUrl)
}

// GetGemVersions get all versions of the specified gem package
// GET - /api/v1/versions/[GEM NAME].(json|yaml)
func (x *RepositoryImpl) GetGemVersions(ctx context.Context, gemName string) ([]*models.Version, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/versions/%s.json", x.options.ServerURL, url.PathEscape(gemName))
	return getJson[[]*models.Version](ctx, x, targetUrl)
}

// GetGemLatestVersion get latest version of given package
// GET - /api/v1/versions/[GEM NAME]/latest.json
func (x *RepositoryImpl) GetGemLatestVersion(ctx context.Context, gemName string) (*models.LatestVersion, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/versions/%s/latest.json", x.options.ServerURL, url.PathEscape(gemName))
	return getJson[*models.LatestVersion](ctx, x, targetUrl)
}

// GetTimeFrameVersions get version info within specific time range
// GET - /api/v1/timeframe_versions.json
// Time format example: 2019-01-18T21:24:29Z
func (x *RepositoryImpl) GetTimeFrameVersions(ctx context.Context, from, to time.Time) ([]*models.Version, error) {
	// Format time in RFC3339 format
	fromStr := from.Format(time.RFC3339)
	toStr := to.Format(time.RFC3339)
	targetUrl := fmt.Sprintf("%s/api/v1/timeframe_versions.json?from=%s&to=%s", x.options.ServerURL, url.QueryEscape(fromStr), url.QueryEscape(toStr))
	return getJson[[]*models.Version](ctx, x, targetUrl)
}

// Downloads get total download count for this repository
// GET - /api/v1/downloads.(json|yaml)
// Returns an object containing the total number of downloads on RubyGems.
func (x *RepositoryImpl) Downloads(ctx context.Context) (*models.RepositoryDownloadCount, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/downloads.json", x.options.ServerURL)
	return getJson[*models.RepositoryDownloadCount](ctx, x, targetUrl)
}

// VersionDownloads get download count for specific package version
// GET - /api/v1/downloads/[GEM NAME]-[GEM VERSION].(json|yaml)
func (x *RepositoryImpl) VersionDownloads(ctx context.Context, gemName, gemVersion string) (*models.VersionDownloadCount, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/downloads/%s-%s.json", x.options.ServerURL, url.PathEscape(gemName), url.PathEscape(gemVersion))
	return getJson[*models.VersionDownloadCount](ctx, x, targetUrl)
}

// GetDependencies get dependencies of specified gem
// GET - /api/v1/dependencies?gems=[COMMA DELIMITED GEM NAMES]
func (x *RepositoryImpl) GetDependencies(ctx context.Context, gemsNames ...string) ([]*models.DependencyInfo, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/dependencies?gems=%s", x.options.ServerURL, url.QueryEscape(strings.Join(gemsNames, ",")))
	return getJson[[]*models.DependencyInfo](ctx, x, targetUrl)
}

// LatestGems get latest published gems
// GET - /api/v1/activity/latest.json
func (x *RepositoryImpl) LatestGems(ctx context.Context) ([]*models.PackageInformation, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/activity/latest.json", x.options.ServerURL)
	return getJson[[]*models.PackageInformation](ctx, x, targetUrl)
}

// GetReverseDependencies get all packages depending on specified gem
// GET /api/v1/gems/{gem}/reverse_dependencies.json
func (x *RepositoryImpl) GetReverseDependencies(ctx context.Context, gemName string) ([]string, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/gems/%s/reverse_dependencies.json", x.options.ServerURL, url.PathEscape(gemName))
	return getJson[[]string](ctx, x, targetUrl)
}

// GetVersionReverseDependencies get packages that depend on a specific version of a gem
// The fullName parameter should be in the format "gemname-version" (e.g., "rails-7.0.5")
// GET /api/v1/versions/{fullName}/reverse_dependencies.json
func (x *RepositoryImpl) GetVersionReverseDependencies(ctx context.Context, fullName string) ([]string, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/versions/%s/reverse_dependencies.json", x.options.ServerURL, url.PathEscape(fullName))
	return getJson[[]string](ctx, x, targetUrl)
}

// SearchAutocomplete return search autocomplete suggestions
// GET /api/v1/search/autocomplete.json?query={query}
func (x *RepositoryImpl) SearchAutocomplete(ctx context.Context, query string) ([]string, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/search/autocomplete.json?query=%s", x.options.ServerURL, query)
	return getJson[[]string](ctx, x, targetUrl)
}

// GetGemVersionDetail get detailed info of specific package version (API v2)
// more detailed than V1 version info, includes spec_sha, yanked status and full dependency info
// GET /api/v2/rubygems/{gem}/versions/{version}.json
func (x *RepositoryImpl) GetGemVersionDetail(ctx context.Context, gemName, version string) (*models.VersionDetail, error) {
	targetUrl := fmt.Sprintf("%s/api/v2/rubygems/%s/versions/%s.json", x.options.ServerURL, url.PathEscape(gemName), url.PathEscape(version))
	return getJson[*models.VersionDetail](ctx, x, targetUrl)
}

// JustUpdatedGems get 50 most recently updated gems (gems with new versions)
// GET /api/v1/activity/just_updated.json
func (x *RepositoryImpl) JustUpdatedGems(ctx context.Context) ([]*models.PackageInformation, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/activity/just_updated.json", x.options.ServerURL)
	return getJson[[]*models.PackageInformation](ctx, x, targetUrl)
}

// TopDownloads get top 50 most downloaded gems
// GET /api/v1/downloads/all.json
func (x *RepositoryImpl) TopDownloads(ctx context.Context) ([]*models.TopDownloadedGem, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/downloads/all.json", x.options.ServerURL)
	return getJson[[]*models.TopDownloadedGem](ctx, x, targetUrl)
}

// GetUserProfile get public profile of specified user
// GET /api/v1/profiles/{handle_or_id}.json
func (x *RepositoryImpl) GetUserProfile(ctx context.Context, handleOrID string) (*models.UserProfile, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/profiles/%s.json", x.options.ServerURL, url.PathEscape(handleOrID))
	return getJson[*models.UserProfile](ctx, x, targetUrl)
}

// GetOwnedGems get all gems owned by current authenticated user
// requires API Token authentication
// GET /api/v1/gems.json
func (x *RepositoryImpl) GetOwnedGems(ctx context.Context) ([]*models.PackageInformation, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/gems.json", x.options.ServerURL)
	return getJson[[]*models.PackageInformation](ctx, x, targetUrl)
}

// GetGemsByOwner get all gems owned by specified user
// GET /api/v1/owners/{handle_or_id}/gems.json
func (x *RepositoryImpl) GetGemsByOwner(ctx context.Context, handleOrID string) ([]*models.PackageInformation, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/owners/%s/gems.json", x.options.ServerURL, url.PathEscape(handleOrID))
	return getJson[[]*models.PackageInformation](ctx, x, targetUrl)
}

// GetGemOwners get all owners of specified gem
// GET /api/v1/gems/{gem}/owners.json
func (x *RepositoryImpl) GetGemOwners(ctx context.Context, gemName string) ([]*models.Owner, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/gems/%s/owners.json", x.options.ServerURL, url.PathEscape(gemName))
	return getJson[[]*models.Owner](ctx, x, targetUrl)
}

// GetAttestations get sigstore attestations for specified gem version
// GET /api/v1/attestations/{gem}-{version}.json
func (x *RepositoryImpl) GetAttestations(ctx context.Context, gemName, version string) ([]*models.Attestation, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/attestations/%s-%s.json", x.options.ServerURL, url.PathEscape(gemName), url.PathEscape(version))
	return getJson[[]*models.Attestation](ctx, x, targetUrl)
}

// GetGemVersionContents get file checksums/manifest for specified gem version
// GET /api/v2/rubygems/{gem}/versions/{version}/contents.json
func (x *RepositoryImpl) GetGemVersionContents(ctx context.Context, gemName, version string) (*models.VersionContent, error) {
	targetUrl := fmt.Sprintf("%s/api/v2/rubygems/%s/versions/%s/contents.json", x.options.ServerURL, url.PathEscape(gemName), url.PathEscape(version))
	return getJson[*models.VersionContent](ctx, x, targetUrl)
}

// GetMFAStatus check MFA status for the authenticated user
// requires API Token authentication
// GET /api/v1/multifactor_auth
func (x *RepositoryImpl) GetMFAStatus(ctx context.Context) (*models.MFAStatus, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/multifactor_auth", x.options.ServerURL)
	return getJson[*models.MFAStatus](ctx, x, targetUrl)
}

func getJson[T any](ctx context.Context, repository *RepositoryImpl, targetUrl string) (T, error) {
	bytes, err := repository.getBytes(ctx, targetUrl)
	if err != nil {
		var zero T
		return zero, err
	}
	return unmarshalJson[T](bytes)
}

func unmarshalJson[T any](bytes []byte) (T, error) {
	var r T
	err := json.Unmarshal(bytes, &r)
	if err != nil {
		var zero T
		return zero, err
	}
	return r, nil
}

// internal unified request method
func (x *RepositoryImpl) getBytes(ctx context.Context, targetUrl string) ([]byte, error) {
	// Use custom ResponseHandler, only accept 2xx status codes
	responseHandler := func(httpResponse *http.Response) ([]byte, error) {
		// Check HTTP status code
		if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
			// Read response body
			body, readErr := requests.BytesResponseHandler()(httpResponse)
			bodyStr := ""
			if readErr == nil {
				bodyStr = string(body)
			}
			return nil, NewAPIError(httpResponse, []byte(bodyStr), fmt.Errorf("unexpected status code: %d", httpResponse.StatusCode))
		}
		return requests.BytesResponseHandler()(httpResponse)
	}

	options := requests.NewOptions[any, []byte](targetUrl, responseHandler)

	// Set proxy
	if x.options.Proxy != "" {
		options.AppendRequestSetting(requests.RequestSettingProxy(x.options.Proxy))
	}

	// Set Token authentication
	if x.options.Token != "" {
		// Use anonymous function to set HTTP header
		options.AppendRequestSetting(func(client *http.Client, request *http.Request) error {
			request.Header.Set("Authorization", "Bearer "+x.options.Token)
			return nil
		})
	}

	// If retry enabled, use request with retry
	if x.options.RetryOptions != nil {
		return SendRequestWithRetry(ctx, options, x.options.RetryOptions)
	}

	// Otherwise send request directly
	return requests.SendRequest[any, []byte](ctx, options)
}

// getJsonWithBasicAuth sends a GET request with HTTP Basic authentication and returns JSON-decoded result
func getJsonWithBasicAuth[T any](ctx context.Context, repository *RepositoryImpl, targetUrl, username, password string) (T, error) {
	bytes, err := repository.getBytesWithBasicAuth(ctx, targetUrl, username, password)
	if err != nil {
		var zero T
		return zero, err
	}
	return unmarshalJson[T](bytes)
}

// getBytesWithBasicAuth sends a GET request with HTTP Basic authentication
func (x *RepositoryImpl) getBytesWithBasicAuth(ctx context.Context, targetUrl, username, password string) ([]byte, error) {
	responseHandler := func(httpResponse *http.Response) ([]byte, error) {
		if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
			body, readErr := requests.BytesResponseHandler()(httpResponse)
			bodyStr := ""
			if readErr == nil {
				bodyStr = string(body)
			}
			return nil, NewAPIError(httpResponse, []byte(bodyStr), fmt.Errorf("unexpected status code: %d", httpResponse.StatusCode))
		}
		return requests.BytesResponseHandler()(httpResponse)
	}

	options := requests.NewOptions[any, []byte](targetUrl, responseHandler)

	// Set proxy
	if x.options.Proxy != "" {
		options.AppendRequestSetting(requests.RequestSettingProxy(x.options.Proxy))
	}

	// Set HTTP Basic authentication
	options.AppendRequestSetting(func(client *http.Client, request *http.Request) error {
		request.SetBasicAuth(username, password)
		return nil
	})

	// If retry enabled, use request with retry
	if x.options.RetryOptions != nil {
		return SendRequestWithRetry(ctx, options, x.options.RetryOptions)
	}

	return requests.SendRequest[any, []byte](ctx, options)
}
