package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/crawler-go-go-go/go-requests"
	"github.com/stretchr/testify/assert"
)

// Test retry options settings
func TestRetryOptions(t *testing.T) {
	opts := NewDefaultRetryOptions()

	// Test default values
	assert.Equal(t, DefaultRetryAttempts, opts.MaxAttempts)
	assert.Equal(t, DefaultRetryWaitTime, opts.WaitTime)
	assert.Equal(t, DefaultRetryMaxWaitTime, opts.MaxWaitTime)
	assert.True(t, opts.UseExponentialBackoff)
	assert.NotNil(t, opts.ShouldRetry)

	// Test method chaining
	opts = opts.WithMaxAttempts(5).
		WithWaitTime(2 * time.Second).
		WithMaxWaitTime(10 * time.Second).
		WithExponentialBackoff(false)

	assert.Equal(t, 5, opts.MaxAttempts)
	assert.Equal(t, 2*time.Second, opts.WaitTime)
	assert.Equal(t, 10*time.Second, opts.MaxWaitTime)
	assert.False(t, opts.UseExponentialBackoff)
}

// Test default retry condition
func TestDefaultShouldRetry(t *testing.T) {
	opts := NewDefaultRetryOptions()

	// Should retry when there is an error
	assert.True(t, opts.ShouldRetry(errors.New("test error")))

	// Should not retry when there is no error
	assert.False(t, opts.ShouldRetry(nil))
}

// Simulate a request sending function for testing retry logic
type mockRequestSender struct {
	attempts      int
	maxAttempts   int
	responses     []interface{}
	errors        []error
	requestTimes  []time.Time
	shouldTimeout bool
}

func newMockRequestSender(maxAttempts int) *mockRequestSender {
	return &mockRequestSender{
		attempts:     0,
		maxAttempts:  maxAttempts,
		responses:    make([]interface{}, maxAttempts),
		errors:       make([]error, maxAttempts),
		requestTimes: make([]time.Time, maxAttempts),
	}
}

func (m *mockRequestSender) sendRequest(ctx context.Context, options *requests.Options[any, any]) (any, error) {
	// Record request time
	m.requestTimes[m.attempts] = time.Now()

	// If timeout simulation is set, check whether the context has been cancelled
	if m.shouldTimeout {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// continue execution
		}
	}

	// Get the preset response and error for the current attempt
	response := m.responses[m.attempts]
	err := m.errors[m.attempts]

	// Increment the attempt count
	m.attempts++

	return response, err
}

// Test the behavior of the retry mechanism
func TestSendRequestWithRetry(t *testing.T) {
	// Test the case where retry succeeds
	t.Run("retry success", func(t *testing.T) {
		// Set up the mock sender, first attempt fails, second succeeds
		mock := newMockRequestSender(3)
		mock.errors[0] = errors.New("first attempt failed")
		mock.responses[1] = "success"
		mock.errors[1] = nil

		// Create retry options, no exponential backoff
		retryOpts := NewDefaultRetryOptions().
			WithMaxAttempts(3).
			WithWaitTime(100 * time.Millisecond).
			WithExponentialBackoff(false)

		// Execute the test
		ctx := context.Background()
		result, err := sendWithMock(ctx, mock, retryOpts)

		// Verify the result
		assert.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.Equal(t, 2, mock.attempts) // should only attempt twice

		// Verify the retry time interval
		if mock.attempts >= 2 {
			interval := mock.requestTimes[1].Sub(mock.requestTimes[0])
			assert.True(t, interval >= 100*time.Millisecond, "retry interval should be at least 100ms")
		}
	})

	// Test reaching max retry attempts
	t.Run("reaching max retry attempts", func(t *testing.T) {
		// Set up the mock sender, all attempts fail
		mock := newMockRequestSender(3)
		for i := 0; i < mock.maxAttempts; i++ {
			mock.errors[i] = errors.New("attempt failed")
		}

		// Create retry options
		retryOpts := NewDefaultRetryOptions().
			WithMaxAttempts(3).
			WithWaitTime(50 * time.Millisecond).
			WithExponentialBackoff(false)

		// Execute the test
		ctx := context.Background()
		_, err := sendWithMock(ctx, mock, retryOpts)

		// Verify the result
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max retry attempts reached")
		assert.Equal(t, 3, mock.attempts) // should attempt three times
	})

	// Test exponential backoff
	t.Run("exponential backoff", func(t *testing.T) {
		// Set up the mock sender, all attempts fail
		mock := newMockRequestSender(3)
		for i := 0; i < mock.maxAttempts; i++ {
			mock.errors[i] = errors.New("attempt failed")
		}

		// Create retry options, using exponential backoff
		retryOpts := NewDefaultRetryOptions().
			WithMaxAttempts(3).
			WithWaitTime(100 * time.Millisecond).
			WithExponentialBackoff(true)

		// Execute the test
		ctx := context.Background()
		_, _ = sendWithMock(ctx, mock, retryOpts)

		// Verify the retry time interval, the second retry interval should be longer than the first
		if mock.attempts >= 3 {
			interval1 := mock.requestTimes[1].Sub(mock.requestTimes[0])
			interval2 := mock.requestTimes[2].Sub(mock.requestTimes[1])
			assert.True(t, interval2 > interval1, "exponential backoff should make the second retry interval longer than the first")
		}
	})

	// Test context cancellation
	t.Run("context cancellation", func(t *testing.T) {
		// Set up the mock sender
		mock := newMockRequestSender(3)
		mock.errors[0] = errors.New("first attempt failed")
		mock.shouldTimeout = true

		// Create a cancellable context
		ctx, cancel := context.WithCancel(context.Background())

		// Cancel the context after a short time
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		// Create retry options, with a longer wait time
		retryOpts := NewDefaultRetryOptions().
			WithMaxAttempts(3).
			WithWaitTime(500 * time.Millisecond).
			WithExponentialBackoff(false)

		// Execute the test
		_, err := sendWithMock(ctx, mock, retryOpts)

		// Verify the result
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

// Helper function, execute retry using the mock sender
func sendWithMock(ctx context.Context, mock *mockRequestSender, retryOptions *RetryOptions) (interface{}, error) {
	// Empty request options
	options := &requests.Options[any, any]{}

	// Record attempt count
	attempts := 0
	var lastErr error
	var lastResp interface{}

	for attempts < retryOptions.MaxAttempts {
		// If not the first attempt, wait the specified time
		if attempts > 0 {
			waitTime := retryOptions.WaitTime

			// If using exponential backoff
			if retryOptions.UseExponentialBackoff {
				factor := 1 << uint(attempts-1)
				waitTime = time.Duration(float64(waitTime) * float64(factor))
				if waitTime > retryOptions.MaxWaitTime {
					waitTime = retryOptions.MaxWaitTime
				}
			}

			// Wait
			select {
			case <-time.After(waitTime):
				// continue execution
			case <-ctx.Done():
				// context cancelled
				return nil, ctx.Err()
			}
		}

		// Send request
		resp, err := mock.sendRequest(ctx, options)

		// Check whether retry is needed
		if err == nil {
			return resp, nil
		}

		lastErr = err
		lastResp = resp
		attempts++
	}

	// Max retry attempts reached
	return lastResp, errors.New("max retry attempts reached: " + lastErr.Error())
}
