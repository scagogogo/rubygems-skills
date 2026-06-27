package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/crawler-go-go-go/go-requests"
)

const (
	// DefaultRetryAttempts default retry attempts
	DefaultRetryAttempts = 3

	// DefaultRetryWaitTime default retry wait time
	DefaultRetryWaitTime = 1 * time.Second

	// DefaultRetryMaxWaitTime default max retry wait time
	DefaultRetryMaxWaitTime = 30 * time.Second
)

// RetryOptions retry options
type RetryOptions struct {
	// Retry count
	MaxAttempts int

	// Initial wait time
	WaitTime time.Duration

	// Max wait time
	MaxWaitTime time.Duration

	// Whether to use exponential backoff
	UseExponentialBackoff bool

	// Custom retry condition
	// Note: Due to generic constraints, ShouldRetry can only determine whether to retry based on error.
	// HTTP status code related retry logic is now handled by the custom ResponseHandler in the getBytes method,
	// which returns errors for non-2xx status codes, thus triggering retries.
	ShouldRetry func(err error) bool
}

// NewDefaultRetryOptions create default retry options
func NewDefaultRetryOptions() *RetryOptions {
	return &RetryOptions{
		MaxAttempts:           DefaultRetryAttempts,
		WaitTime:              DefaultRetryWaitTime,
		MaxWaitTime:           DefaultRetryMaxWaitTime,
		UseExponentialBackoff: true,
		ShouldRetry: func(err error) bool {
			// If there is an error, always retry
			// HTTP status code errors (such as 429, 500, etc.) are now converted to errors by the
			// custom ResponseHandler in getBytes, so they will trigger retries
			return err != nil
		},
	}
}

// WithMaxAttempts set max retry attempts
func (o *RetryOptions) WithMaxAttempts(attempts int) *RetryOptions {
	o.MaxAttempts = attempts
	return o
}

// WithWaitTime set initial wait time
func (o *RetryOptions) WithWaitTime(waitTime time.Duration) *RetryOptions {
	o.WaitTime = waitTime
	return o
}

// WithMaxWaitTime set max wait time
func (o *RetryOptions) WithMaxWaitTime(maxWaitTime time.Duration) *RetryOptions {
	o.MaxWaitTime = maxWaitTime
	return o
}

// WithExponentialBackoff set whether to use exponential backoff
func (o *RetryOptions) WithExponentialBackoff(use bool) *RetryOptions {
	o.UseExponentialBackoff = use
	return o
}

// WithShouldRetry set custom retry condition
func (o *RetryOptions) WithShouldRetry(shouldRetry func(err error) bool) *RetryOptions {
	o.ShouldRetry = shouldRetry
	return o
}

// SendRequestWithRetry send request with retry
func SendRequestWithRetry[Request any, Response any](
	ctx context.Context,
	options *requests.Options[Request, Response],
	retryOptions *RetryOptions,
) (Response, error) {
	var lastErr error
	var lastResp Response

	// If retry options not provided, use defaults
	if retryOptions == nil {
		retryOptions = NewDefaultRetryOptions()
	}

	for attempt := 0; attempt < retryOptions.MaxAttempts; attempt++ {
		// If not first attempt, wait for a while
		if attempt > 0 {
			waitTime := retryOptions.WaitTime

			// If using exponential backoff, exponentially increase wait time
			if retryOptions.UseExponentialBackoff {
				factor := 1 << uint(attempt-1)
				waitTime = time.Duration(float64(waitTime) * float64(factor))
				if waitTime > retryOptions.MaxWaitTime {
					waitTime = retryOptions.MaxWaitTime
				}
			}

			// Wait then retry
			select {
			case <-time.After(waitTime):
				// Continue execution
			case <-ctx.Done():
				// Context cancelled, stop retrying
				var zero Response
				return zero, ctx.Err()
			}
		}

		// Execute request
		resp, err := requests.SendRequest[Request, Response](ctx, options)

		// Save last result
		if err != nil {
			lastErr = err
		}
		lastResp = resp

		// Call user-provided retry condition function
		// HTTP status code errors (such as 429, 500, etc.) are now converted to errors by the
		// custom ResponseHandler in getBytes, so only need to check error here.
		if retryOptions.ShouldRetry != nil && !retryOptions.ShouldRetry(err) {
			// User custom ShouldRetry returns false, don't retry
			return resp, err
		}

		// If no error (err == nil), request succeeded, return directly
		if err == nil {
			return resp, nil
		}

		// err != nil, continue retry loop
	}

	// Max retry attempts reached, return last error
	if lastErr != nil {
		return lastResp, fmt.Errorf("max retry attempts reached (%d attempts): %w", retryOptions.MaxAttempts, lastErr)
	}

	return lastResp, nil
}
