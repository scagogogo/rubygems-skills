package repository

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/crawler-go-go-go/go-requests"
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

// ----- SendRequestWithRetry boundary coverage (calls the real function) ------

// newRetryOptions builds an Options[any, []byte] wired to a stub transport that
// returns the given sequence of responses for "/x".
func newRetryOptions(t *testing.T, tr *fakeRoundTripper) *requests.Options[any, []byte] {
	t.Helper()
	rh := func(resp *http.Response) ([]byte, error) {
		return requests.BytesResponseHandler()(resp)
	}
	opts := requests.NewOptions[any, []byte]("https://example.com/x", rh)
	opts.AppendRequestSetting(func(client *http.Client, request *http.Request) error {
		client.Transport = tr
		return nil
	})
	return opts
}

// TestSendRequestWithRetry_NilRetryOptions covers the retryOptions==nil branch
// (defaults are applied) and a successful first attempt.
func TestSendRequestWithRetry_NilRetryOptions(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/x", 200, "ok")
	opts := newRetryOptions(t, tr)
	out, err := SendRequestWithRetry[any, []byte](context.Background(), opts, nil)
	assert.NoError(t, err)
	assert.Equal(t, []byte("ok"), out)
}

// TestSendRequestWithRetry_ShouldRetryNilSuccess covers the err==nil branch
// that is only reachable when ShouldRetry is nil (otherwise the ShouldRetry
// check returns first).
func TestSendRequestWithRetry_ShouldRetryNilSuccess(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/x", 200, "ok")
	opts := newRetryOptions(t, tr)
	ro := NewDefaultRetryOptions().WithMaxAttempts(3)
	ro.WithShouldRetry(nil) // nil ShouldRetry
	ro.WaitTime = time.Millisecond
	ro.MaxWaitTime = time.Millisecond
	out, err := SendRequestWithRetry[any, []byte](context.Background(), opts, ro)
	assert.NoError(t, err)
	assert.Equal(t, []byte("ok"), out)
}

// TestSendRequestWithRetry_ZeroMaxAttempts covers the loop-skipped path that
// returns (lastResp, nil) when MaxAttempts == 0.
func TestSendRequestWithRetry_ZeroMaxAttempts(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/x", 200, "ok")
	opts := newRetryOptions(t, tr)
	ro := &RetryOptions{MaxAttempts: 0, ShouldRetry: func(error) bool { return true }}
	out, err := SendRequestWithRetry[any, []byte](context.Background(), opts, ro)
	assert.NoError(t, err)
	assert.Nil(t, out)
}

// TestSendRequestWithRetry_ContextCancelledDuringWait covers the
// case <-ctx.Done() branch inside the retry wait.
func TestSendRequestWithRetry_ContextCancelledDuringWait(t *testing.T) {
	tr := newFakeTransport()
	tr.stubErr("/x", errors.New("transient"))
	opts := newRetryOptions(t, tr)
	ctx, cancel := context.WithCancel(context.Background())
	ro := NewDefaultRetryOptions().WithMaxAttempts(5)
	ro.WaitTime = 200 * time.Millisecond
	ro.WithExponentialBackoff(false)
	// Cancel while the first retry wait is in flight.
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	_, err := SendRequestWithRetry[any, []byte](ctx, opts, ro)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// TestSendRequestWithRetry_BackoffCappedToMaxWait covers the
// waitTime > MaxWaitTime branch of exponential backoff.
func TestSendRequestWithRetry_BackoffCappedToMaxWait(t *testing.T) {
	tr := newFakeTransport()
	tr.stubErr("/x", errors.New("transient"))
	opts := newRetryOptions(t, tr)
	ro := NewDefaultRetryOptions().WithMaxAttempts(3)
	ro.WaitTime = 1 * time.Second        // large base
	ro.MaxWaitTime = 5 * time.Millisecond // capped tiny
	ro.WithExponentialBackoff(true)
	start := time.Now()
	_, err := SendRequestWithRetry[any, []byte](context.Background(), opts, ro)
	elapsed := time.Since(start)
	assert.Error(t, err)
	// Two waits (attempt 1->2 and 2->3) each capped at ~5ms; total well under 1s.
	assert.Less(t, elapsed, 500*time.Millisecond, "backoff should be capped to MaxWaitTime")
}
