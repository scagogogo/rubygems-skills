package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultRetryOptions(t *testing.T) {
	options := NewDefaultRetryOptions()

	// Check default values
	assert.Equal(t, DefaultRetryAttempts, options.MaxAttempts)
	assert.Equal(t, DefaultRetryWaitTime, options.WaitTime)
	assert.Equal(t, DefaultRetryMaxWaitTime, options.MaxWaitTime)
	assert.True(t, options.UseExponentialBackoff)
	assert.NotNil(t, options.ShouldRetry)

	// Test the default shouldRetry function: should retry when there's an error
	assert.True(t, options.ShouldRetry(assert.AnError))

	// Test the default shouldRetry function: should NOT retry when there's no error
	assert.False(t, options.ShouldRetry(nil))
}

func TestRetryOptions_WithMaxAttempts(t *testing.T) {
	options := NewDefaultRetryOptions()

	// Test fluent interface
	result := options.WithMaxAttempts(10)
	assert.Same(t, options, result)

	// Verify value was set
	assert.Equal(t, 10, options.MaxAttempts)
}

func TestRetryOptions_WithWaitTime(t *testing.T) {
	options := NewDefaultRetryOptions()

	// Test fluent interface
	waitTime := 5 * time.Second
	result := options.WithWaitTime(waitTime)
	assert.Same(t, options, result)

	// Verify value was set
	assert.Equal(t, waitTime, options.WaitTime)
}

func TestRetryOptions_WithMaxWaitTime(t *testing.T) {
	options := NewDefaultRetryOptions()

	// Test fluent interface
	maxWaitTime := 60 * time.Second
	result := options.WithMaxWaitTime(maxWaitTime)
	assert.Same(t, options, result)

	// Verify value was set
	assert.Equal(t, maxWaitTime, options.MaxWaitTime)
}

func TestRetryOptions_WithExponentialBackoff(t *testing.T) {
	options := NewDefaultRetryOptions()

	// Test fluent interface with disabling exponential backoff
	result := options.WithExponentialBackoff(false)
	assert.Same(t, options, result)

	// Verify value was set
	assert.False(t, options.UseExponentialBackoff)

	// Test enabling it again
	options.WithExponentialBackoff(true)
	assert.True(t, options.UseExponentialBackoff)
}

func TestRetryOptions_WithShouldRetry(t *testing.T) {
	options := NewDefaultRetryOptions()

	// Create a custom retry function that only retries on specific errors
	customShouldRetry := func(err error) bool {
		if err == nil {
			return false
		}
		// Only retry on network-related errors, not on "not found" errors
		return !errors.Is(err, ErrNotFound)
	}

	// Test fluent interface
	result := options.WithShouldRetry(customShouldRetry)
	assert.Same(t, options, result)

	// Verify function was set by testing it
	assert.True(t, options.ShouldRetry(assert.AnError))
	assert.False(t, options.ShouldRetry(nil))
	assert.False(t, options.ShouldRetry(ErrNotFound))
}
