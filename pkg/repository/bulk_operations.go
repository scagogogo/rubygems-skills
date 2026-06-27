package repository

import (
	"context"
	"sync"

	"github.com/scagogogo/rubygems-skills/pkg/models"
)

// BulkResult represents single result of bulk operation
// it contains the request key (such as package name), returned value and possible error
type BulkResult[T any] struct {
	Key   string // Request key, usually gem name
	Value T      // Operation result value
	Error error  // Possible error during operation
}

// BulkOptions defines bulk operation configuration options
type BulkOptions struct {
	// MaxConcurrency defines max concurrent request count
	// reasonable value avoids excessive pressure on API server
	// default is 10
	MaxConcurrency int

	// ContinueOnError decides whether to continue other requests on error
	// if true, continue on error even if some requests fail
	// if false, stop on first error
	// default is true
	ContinueOnError bool
}

// NewBulkOptions create bulk options with defaults
// default config: max concurrency 10, continue on error
func NewBulkOptions() *BulkOptions {
	return &BulkOptions{
		MaxConcurrency:  10,
		ContinueOnError: true,
	}
}

// WithMaxConcurrency set max concurrent requests
// return options object itself, supports chaining
func (o *BulkOptions) WithMaxConcurrency(maxConcurrency int) *BulkOptions {
	if maxConcurrency > 0 {
		o.MaxConcurrency = maxConcurrency
	}
	return o
}

// WithContinueOnError set whether to continue on error
// return options object itself, supports chaining
func (o *BulkOptions) WithContinueOnError(continueOnError bool) *BulkOptions {
	o.ContinueOnError = continueOnError
	return o
}

// BulkGetPackages bulk get info of multiple packages
// concurrent execution of GetPackage requests, improve large-scale data fetching efficiency
// Parameters:
//   - ctx: context for request timeout and cancellation
//   - gemNames: list of package names to fetch
//   - options: bulk options, controlling concurrency etc.
//
// Returns:
//   - slice containing each package request result, same order as input package names
func (r *RepositoryImpl) BulkGetPackages(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[*models.PackageInformation] {
	if options == nil {
		options = NewBulkOptions()
	}

	results := make([]*BulkResult[*models.PackageInformation], len(gemNames))

	// Create worker pool
	worker := func(wg *sync.WaitGroup, jobs <-chan int, results []*BulkResult[*models.PackageInformation]) {
		defer wg.Done()

		for i := range jobs {
			select {
			case <-ctx.Done():
				// Context cancelled, stop processing
				results[i] = &BulkResult[*models.PackageInformation]{
					Key:   gemNames[i],
					Error: ctx.Err(),
				}
				return
			default:
				// Get package info/versions/dependencies
				pkg, err := r.GetPackage(ctx, gemNames[i])
				results[i] = &BulkResult[*models.PackageInformation]{
					Key:   gemNames[i],
					Value: pkg,
					Error: err,
				}

				// If set to stop on error, and error occurred
				if !options.ContinueOnError && err != nil {
					return
				}
			}
		}
	}

	// Run worker pool
	runWorkerPool(options.MaxConcurrency, len(gemNames), results, worker)

	return results
}

// BulkGetVersions bulk get version info of multiple packages
// concurrent execution of GetGemVersions requests, improve large-scale data fetching efficiency
// Parameters:
//   - ctx: context for request timeout and cancellation
//   - gemNames: list of package names to fetch
//   - options: bulk options, controlling concurrency etc.
//
// Returns:
//   - slice containing each package version request result, same order as input package names
func (r *RepositoryImpl) BulkGetVersions(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[[]*models.Version] {
	if options == nil {
		options = NewBulkOptions()
	}

	results := make([]*BulkResult[[]*models.Version], len(gemNames))

	// Create worker pool
	worker := func(wg *sync.WaitGroup, jobs <-chan int, results []*BulkResult[[]*models.Version]) {
		defer wg.Done()

		for i := range jobs {
			select {
			case <-ctx.Done():
				// Context cancelled, stop processing
				results[i] = &BulkResult[[]*models.Version]{
					Key:   gemNames[i],
					Error: ctx.Err(),
				}
				return
			default:
				// Get version info
				versions, err := r.GetGemVersions(ctx, gemNames[i])
				results[i] = &BulkResult[[]*models.Version]{
					Key:   gemNames[i],
					Value: versions,
					Error: err,
				}

				// If set to stop on error, and error occurred
				if !options.ContinueOnError && err != nil {
					return
				}
			}
		}
	}

	// Run worker pool
	runWorkerPool(options.MaxConcurrency, len(gemNames), results, worker)

	return results
}

// BulkGetDependencies bulk get dependency info of multiple packages
// concurrent execution of GetDependencies requests, improve large-scale data fetching efficiency
// Parameters:
//   - ctx: context for request timeout and cancellation
//   - gemNames: list of package names to fetch
//   - options: bulk options, controlling concurrency etc.
//
// Returns:
//   - slice containing each package dependency request result, same order as input package names
func (r *RepositoryImpl) BulkGetDependencies(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[[]*models.DependencyInfo] {
	if options == nil {
		options = NewBulkOptions()
	}

	results := make([]*BulkResult[[]*models.DependencyInfo], len(gemNames))

	// Create worker pool
	worker := func(wg *sync.WaitGroup, jobs <-chan int, results []*BulkResult[[]*models.DependencyInfo]) {
		defer wg.Done()

		for i := range jobs {
			select {
			case <-ctx.Done():
				// Context cancelled, stop processing
				results[i] = &BulkResult[[]*models.DependencyInfo]{
					Key:   gemNames[i],
					Error: ctx.Err(),
				}
				return
			default:
				// Get dependency info
				deps, err := r.GetDependencies(ctx, gemNames[i])
				results[i] = &BulkResult[[]*models.DependencyInfo]{
					Key:   gemNames[i],
					Value: deps,
					Error: err,
				}

				// If set to stop on error, and error occurred
				if !options.ContinueOnError && err != nil {
					return
				}
			}
		}
	}

	// Run worker pool
	runWorkerPool(options.MaxConcurrency, len(gemNames), results, worker)

	return results
}

// BulkGetReverseDependencies bulk get reverse dependency info of multiple packages
// concurrent execution of GetReverseDependencies requests, improve large-scale data fetching efficiency
// Parameters:
//   - ctx: context for request timeout and cancellation
//   - gemNames: list of package names to fetch
//   - options: bulk options, controlling concurrency etc.
//
// Returns:
//   - slice containing each package reverse dependency request result, same order as input package names
func (r *RepositoryImpl) BulkGetReverseDependencies(ctx context.Context, gemNames []string, options *BulkOptions) []*BulkResult[[]string] {
	if options == nil {
		options = NewBulkOptions()
	}

	results := make([]*BulkResult[[]string], len(gemNames))

	// Create worker pool
	worker := func(wg *sync.WaitGroup, jobs <-chan int, results []*BulkResult[[]string]) {
		defer wg.Done()

		for i := range jobs {
			select {
			case <-ctx.Done():
				// Context cancelled, stop processing
				results[i] = &BulkResult[[]string]{
					Key:   gemNames[i],
					Error: ctx.Err(),
				}
				return
			default:
				// Get reverse dependency info
				deps, err := r.GetReverseDependencies(ctx, gemNames[i])
				results[i] = &BulkResult[[]string]{
					Key:   gemNames[i],
					Value: deps,
					Error: err,
				}

				// If set to stop on error, and error occurred
				if !options.ContinueOnError && err != nil {
					return
				}
			}
		}
	}

	// Run worker pool
	runWorkerPool(options.MaxConcurrency, len(gemNames), results, worker)

	return results
}

// runWorkerPool is a generic worker pool implementation for concurrent task processing
// Parameters:
//   - numWorkers: worker goroutine count
//   - numJobs: total task count
//   - results: slice for storing results
//   - workerFunc: worker function, defines behavior of each worker goroutine
func runWorkerPool[T any](numWorkers, numJobs int, results []*BulkResult[T], workerFunc func(*sync.WaitGroup, <-chan int, []*BulkResult[T])) {
	// Ensure worker count doesn't exceed task count
	if numWorkers > numJobs {
		numWorkers = numJobs
	}

	// Create wait group and job channel
	var wg sync.WaitGroup
	jobs := make(chan int, numJobs)

	// Start worker goroutines
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go workerFunc(&wg, jobs, results)
	}

	// Dispatch jobs
	for i := 0; i < numJobs; i++ {
		jobs <- i
	}
	close(jobs)

	// Wait for all worker goroutines to finish
	wg.Wait()
}
