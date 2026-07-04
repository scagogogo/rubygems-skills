package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/models"
)

// Create a mock repository implementation for testing
type mockRepository struct {
	mockPackages map[string]*models.PackageInformation
	mockVersions map[string][]*models.Version
	// artificial delay to simulate network request latency
	delay time.Duration
	// artificial error to simulate request failure
	failOn map[string]error
}

// Create a new mock repository
func newMockRepository() *mockRepository {
	repo := &mockRepository{
		mockPackages: make(map[string]*models.PackageInformation),
		mockVersions: make(map[string][]*models.Version),
		delay:        10 * time.Millisecond, // default 10ms delay
		failOn:       make(map[string]error),
	}

	// Add some test data
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

	// Add some version info
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

// Set the error that a specific gem will trigger
func (m *mockRepository) setFailOn(gemName string, err error) *mockRepository {
	m.failOn[gemName] = err
	return m
}

// Implement the GetPackage method
func (m *mockRepository) GetPackage(ctx context.Context, gemName string) (*models.PackageInformation, error) {
	// Check whether it should fail
	if err, ok := m.failOn[gemName]; ok {
		return nil, err
	}

	// Simulate network latency
	time.Sleep(m.delay)

	// Check whether the context has been cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Return the result
	pkg, ok := m.mockPackages[gemName]
	if !ok {
		return nil, errors.New("gem not found")
	}
	return pkg, nil
}

// Implement the GetGemVersions method
func (m *mockRepository) GetGemVersions(ctx context.Context, gemName string) ([]*models.Version, error) {
	// Check whether it should fail
	if err, ok := m.failOn[gemName]; ok {
		return nil, err
	}

	// Simulate network latency
	time.Sleep(m.delay)

	// Check whether the context has been cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Return the result
	versions, ok := m.mockVersions[gemName]
	if !ok {
		return nil, errors.New("gem not found")
	}
	return versions, nil
}

// Implement other necessary interface methods (to simplify tests, these methods can return empty values or errors)
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

// Implement bulk operation methods
func (m *mockRepository) BulkGetPackages(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[*models.PackageInformation] {
	// Only check whether options is nil, do not reassign
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
	// Only check whether options is nil, do not reassign
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

// Test bulk get package info
func TestBulkGetPackages(t *testing.T) {
	// Create a mock repository
	mockRepo := newMockRepository()

	// Set an error
	mockRepo.setFailOn("not-exist", errors.New("gem not found"))

	// Test cases
	testCases := []struct {
		name        string
		gemNames    []string
		concurrency int
		timeout     time.Duration
		expectErr   bool
		expectCount int
	}{
		{
			name:        "get valid package info",
			gemNames:    []string{"rails", "rack"},
			concurrency: 2,
			timeout:     100 * time.Millisecond,
			expectErr:   false,
			expectCount: 2,
		},
		{
			name:        "includes a non-existent package",
			gemNames:    []string{"rails", "rack", "not-exist"},
			concurrency: 2,
			timeout:     100 * time.Millisecond,
			expectErr:   true,
			expectCount: 3,
		},
		{
			name:        "timeout test",
			gemNames:    []string{"rails", "rack"},
			concurrency: 1,
			timeout:     5 * time.Millisecond, // set a very short timeout
			expectErr:   true,
			expectCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up context and timeout
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			// Set concurrency
			options := NewBulkOptions().WithMaxConcurrency(tc.concurrency)

			// Execute bulk get
			results := mockRepo.BulkGetPackages(ctx, tc.gemNames, options)

			// Verify the result count
			if len(results) != tc.expectCount {
				t.Errorf("result count does not match expectation, want: %d, got: %d", tc.expectCount, len(results))
			}

			// Verify whether there is an error
			hasError := false
			for _, result := range results {
				if result.Error != nil {
					hasError = true
					break
				}
			}

			if hasError != tc.expectErr {
				t.Errorf("error status does not match expectation, want error: %v, got: %v", tc.expectErr, hasError)
			}
		})
	}
}

// Test bulk get version info
func TestBulkGetVersions(t *testing.T) {
	// Create a mock repository
	mockRepo := newMockRepository()

	// Set an error
	mockRepo.setFailOn("not-exist", errors.New("gem not found"))

	// Test cases
	testCases := []struct {
		name        string
		gemNames    []string
		concurrency int
		timeout     time.Duration
		expectErr   bool
		expectCount int
	}{
		{
			name:        "get valid version info",
			gemNames:    []string{"rails", "rack"},
			concurrency: 2,
			timeout:     100 * time.Millisecond,
			expectErr:   false,
			expectCount: 2,
		},
		{
			name:        "includes a non-existent package",
			gemNames:    []string{"rails", "rack", "not-exist"},
			concurrency: 2,
			timeout:     100 * time.Millisecond,
			expectErr:   true,
			expectCount: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up context and timeout
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			// Set concurrency
			options := NewBulkOptions().WithMaxConcurrency(tc.concurrency)

			// Execute bulk get
			results := mockRepo.BulkGetVersions(ctx, tc.gemNames, options)

			// Verify the result count
			if len(results) != tc.expectCount {
				t.Errorf("result count does not match expectation, want: %d, got: %d", tc.expectCount, len(results))
			}

			// Verify whether there is an error
			hasError := false
			for _, result := range results {
				if result.Error != nil {
					hasError = true
					break
				}
			}

			if hasError != tc.expectErr {
				t.Errorf("error status does not match expectation, want error: %v, got: %v", tc.expectErr, hasError)
			}
		})
	}
}

// Test bulk operation options
func TestBulkOptions(t *testing.T) {
	// Test default options
	options := NewBulkOptions()
	if options.MaxConcurrency != 10 {
		t.Errorf("default max concurrency is incorrect, want: %d, got: %d", 10, options.MaxConcurrency)
	}
	if !options.ContinueOnError {
		t.Errorf("default error handling strategy is incorrect, want: %v, got: %v", true, options.ContinueOnError)
	}

	// Test setting max concurrency
	options = NewBulkOptions().WithMaxConcurrency(5)
	if options.MaxConcurrency != 5 {
		t.Errorf("max concurrency is incorrect after setting, want: %d, got: %d", 5, options.MaxConcurrency)
	}

	// Test setting error handling strategy
	options = NewBulkOptions().WithContinueOnError(false)
	if options.ContinueOnError {
		t.Errorf("error handling strategy is incorrect after setting, want: %v, got: %v", false, options.ContinueOnError)
	}
}
