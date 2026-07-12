package repository

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/models"
	"github.com/stretchr/testify/assert"
)

// stubForBulk wires a stubbed RepositoryImpl whose read methods return success
// for known gems and 404 (error) for "fail".
func stubForBulk(t *testing.T) *RepositoryImpl {
	t.Helper()
	tr := newFakeTransport()
	// /api/v1/gems/{name}.json for GetPackage
	tr.stub("/api/v1/gems/rails.json", 200, `{"name":"rails"}`)
	tr.stub("/api/v1/gems/rack.json", 200, `{"name":"rack"}`)
	tr.stub("/api/v1/gems/fail.json", 404, "not found")
	// /api/v1/versions/{name}.json for GetGemVersions
	tr.stub("/api/v1/versions/rails.json", 200, `[{"number":"7.0.0"}]`)
	tr.stub("/api/v1/versions/rack.json", 200, `[{"number":"2.2.7"}]`)
	tr.stub("/api/v1/versions/fail.json", 404, "not found")
	// /api/v1/dependencies?gems=... for GetDependencies
	tr.stub("/api/v1/dependencies", 200, `[]`)
	// /api/v1/gems/{name}/reverse_dependencies.json for GetReverseDependencies
	tr.stub("/api/v1/gems/rails/reverse_dependencies.json", 200, `["rack"]`)
	tr.stub("/api/v1/gems/rack/reverse_dependencies.json", 200, `["rails"]`)
	tr.stub("/api/v1/gems/fail/reverse_dependencies.json", 404, "not found")
	return newStubbedRepo(t, tr)
}

func TestBulkGetPackages_Success(t *testing.T) {
	r := stubForBulk(t)
	results := r.BulkGetPackages(context.Background(), []string{"rails", "rack"}, nil)
	assert.Len(t, results, 2)
	for _, res := range results {
		assert.NoError(t, res.Error)
		assert.NotNil(t, res.Value)
	}
}

func TestBulkGetPackages_NilOptionsDefaults(t *testing.T) {
	r := stubForBulk(t)
	// nil options must default to NewBulkOptions (covers the options==nil branch).
	results := r.BulkGetPackages(context.Background(), []string{"rails"}, nil)
	assert.Len(t, results, 1)
}

func TestBulkGetPackages_StopOnError(t *testing.T) {
	r := stubForBulk(t)
	opts := NewBulkOptions().WithContinueOnError(false)
	// "fail" returns 404; with ContinueOnError=false the worker returns early.
	results := r.BulkGetPackages(context.Background(), []string{"fail"}, opts)
	assert.Len(t, results, 1)
	assert.Error(t, results[0].Error)
}

func TestBulkGetPackages_ContinueOnError(t *testing.T) {
	r := stubForBulk(t)
	opts := NewBulkOptions().WithContinueOnError(true).WithMaxConcurrency(1)
	results := r.BulkGetPackages(context.Background(), []string{"rails", "fail", "rack"}, opts)
	assert.Len(t, results, 3)
	assert.NoError(t, results[0].Error)
	assert.Error(t, results[1].Error)
	assert.NoError(t, results[2].Error)
}

func TestBulkGetPackages_ContextCancelled(t *testing.T) {
	r := stubForBulk(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled so workers hit ctx.Done() quickly
	opts := NewBulkOptions().WithMaxConcurrency(1)
	results := r.BulkGetPackages(ctx, []string{"rails", "rack"}, opts)
	assert.Len(t, results, 2)
	// At least one result carries an error (covers ctx.Done branch); some
	// results may be nil if the worker stopped before processing them.
	hasErr := false
	for _, res := range results {
		if res != nil && res.Error != nil {
			hasErr = true
		}
	}
	assert.True(t, hasErr)
}

func TestBulkGetVersions_Success(t *testing.T) {
	r := stubForBulk(t)
	results := r.BulkGetVersions(context.Background(), []string{"rails", "rack"}, nil)
	assert.Len(t, results, 2)
	for _, res := range results {
		assert.NoError(t, res.Error)
		assert.Len(t, res.Value, 1)
	}
}

func TestBulkGetVersions_StopOnError(t *testing.T) {
	r := stubForBulk(t)
	opts := NewBulkOptions().WithContinueOnError(false)
	results := r.BulkGetVersions(context.Background(), []string{"fail"}, opts)
	assert.Len(t, results, 1)
	assert.Error(t, results[0].Error)
}

func TestBulkGetVersions_ContextCancelled(t *testing.T) {
	r := stubForBulk(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	opts := NewBulkOptions().WithMaxConcurrency(1)
	results := r.BulkGetVersions(ctx, []string{"rails", "rack"}, opts)
	assert.Len(t, results, 2)
	hasCtxErr := false
	for _, res := range results {
		if res != nil && res.Error != nil {
			hasCtxErr = true
		}
	}
	assert.True(t, hasCtxErr)
}

func TestBulkGetDependencies_Success(t *testing.T) {
	r := stubForBulk(t)
	results := r.BulkGetDependencies(context.Background(), []string{"rails", "rack"}, nil)
	assert.Len(t, results, 2)
	for _, res := range results {
		assert.NoError(t, res.Error)
	}
}

func TestBulkGetDependencies_ContextCancelled(t *testing.T) {
	r := stubForBulk(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	opts := NewBulkOptions().WithMaxConcurrency(1)
	results := r.BulkGetDependencies(ctx, []string{"rails", "rack"}, opts)
	assert.Len(t, results, 2)
	hasCtxErr := false
	for _, res := range results {
		if res != nil && res.Error != nil {
			hasCtxErr = true
		}
	}
	assert.True(t, hasCtxErr)
}

func TestBulkGetReverseDependencies_Success(t *testing.T) {
	r := stubForBulk(t)
	results := r.BulkGetReverseDependencies(context.Background(), []string{"rails", "rack"}, nil)
	assert.Len(t, results, 2)
	for _, res := range results {
		assert.NoError(t, res.Error)
	}
}

func TestBulkGetReverseDependencies_StopOnError(t *testing.T) {
	r := stubForBulk(t)
	opts := NewBulkOptions().WithContinueOnError(false)
	results := r.BulkGetReverseDependencies(context.Background(), []string{"fail"}, opts)
	assert.Len(t, results, 1)
	assert.Error(t, results[0].Error)
}

func TestBulkGetReverseDependencies_ContextCancelled(t *testing.T) {
	r := stubForBulk(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	opts := NewBulkOptions().WithMaxConcurrency(1)
	results := r.BulkGetReverseDependencies(ctx, []string{"rails", "rack"}, opts)
	assert.Len(t, results, 2)
	hasCtxErr := false
	for _, res := range results {
		if res != nil && res.Error != nil {
			hasCtxErr = true
		}
	}
	assert.True(t, hasCtxErr)
}

func TestBulkGetDependencies_StopOnError(t *testing.T) {
	// Separate stub: /api/v1/dependencies returns 500 so the worker hits the
	// !ContinueOnError && err != nil branch.
	tr := newFakeTransport()
	tr.stub("/api/v1/dependencies", 500, "err")
	r := newStubbedRepo(t, tr)
	opts := NewBulkOptions().WithContinueOnError(false)
	results := r.BulkGetDependencies(context.Background(), []string{"rails"}, opts)
	assert.Len(t, results, 1)
	assert.Error(t, results[0].Error)
}

// TestRunWorkerPool_WorkersCappedToJobs covers the numWorkers > numJobs branch
// of runWorkerPool, and the empty-jobs case (numJobs == 0).
func TestRunWorkerPool_WorkersCappedToJobs(t *testing.T) {
	// More workers than jobs -> numWorkers reduced to numJobs.
	var mu sync.Mutex
	ran := 0
	worker := func(wg *sync.WaitGroup, jobs <-chan int, results []*BulkResult[int]) {
		defer wg.Done()
		for i := range jobs {
			mu.Lock()
			ran++
			mu.Unlock()
			results[i] = &BulkResult[int]{Key: "k", Value: i}
		}
	}
	results := make([]*BulkResult[int], 3)
	runWorkerPool(100, 3, results, worker)
	assert.Equal(t, 3, ran)
	for i := 0; i < 3; i++ {
		assert.NotNil(t, results[i])
	}
}

func TestRunWorkerPool_ZeroJobs(t *testing.T) {
	// Zero jobs -> no workers started, returns immediately without deadlock.
	worker := func(wg *sync.WaitGroup, jobs <-chan int, results []*BulkResult[int]) {
		defer wg.Done()
		for range jobs {
		}
	}
	results := make([]*BulkResult[int], 0)
	runWorkerPool(4, 0, results, worker) // must not hang
}

// TestBulkOptions_Fluent covers the WithMaxConcurrency <=0 guard and chaining.
func TestBulkOptions_Fluent(t *testing.T) {
	o := NewBulkOptions()
	// WithMaxConcurrency(0) must NOT apply (keeps default 10).
	o.WithMaxConcurrency(0)
	assert.Equal(t, 10, o.MaxConcurrency)
	// Negative also ignored.
	o.WithMaxConcurrency(-1)
	assert.Equal(t, 10, o.MaxConcurrency)
	// Positive applies.
	o.WithMaxConcurrency(3)
	assert.Equal(t, 3, o.MaxConcurrency)
	// WithContinueOnError toggles.
	o.WithContinueOnError(false)
	assert.False(t, o.ContinueOnError)
}

// keep imports referenced
var _ = errors.New
var _ time.Duration
var _ = models.PackageInformation{}
