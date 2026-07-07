package repository

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewOptions(t *testing.T) {
	options := NewOptions()

	// Verify default values
	assert.Equal(t, DefaultServerURL, options.ServerURL)
	assert.Equal(t, "", options.Proxy)
	assert.Equal(t, "", options.Token)
	assert.NotNil(t, options.RetryOptions)
}

func TestOptions_SetServerURL(t *testing.T) {
	options := NewOptions()

	// Test fluent interface
	result := options.SetServerURL("https://custom-rubygems.org")
	assert.Same(t, options, result)

	// Verify value was set
	assert.Equal(t, "https://custom-rubygems.org", options.ServerURL)
}

func TestOptions_SetProxy(t *testing.T) {
	options := NewOptions()

	// Test fluent interface
	result := options.SetProxy("http://proxy.example.com:8080")
	assert.Same(t, options, result)

	// Verify value was set
	assert.Equal(t, "http://proxy.example.com:8080", options.Proxy)
}

func TestOptions_SetToken(t *testing.T) {
	options := NewOptions()

	// Test fluent interface
	result := options.SetToken("my-api-token")
	assert.Same(t, options, result)

	// Verify value was set
	assert.Equal(t, "my-api-token", options.Token)
}

func TestOptions_SetRetryOptions(t *testing.T) {
	options := NewOptions()

	// Create custom retry options
	customRetryOptions := &RetryOptions{
		MaxAttempts:           10,
		WaitTime:              5 * time.Second,
		MaxWaitTime:           60 * time.Second,
		UseExponentialBackoff: false,
	}

	// Test fluent interface
	result := options.SetRetryOptions(customRetryOptions)
	assert.Same(t, options, result)

	// Verify value was set
	assert.Same(t, customRetryOptions, options.RetryOptions)
	assert.Equal(t, 10, options.RetryOptions.MaxAttempts)
	assert.Equal(t, 5*time.Second, options.RetryOptions.WaitTime)
	assert.Equal(t, 60*time.Second, options.RetryOptions.MaxWaitTime)
	assert.Equal(t, false, options.RetryOptions.UseExponentialBackoff)
}

func TestOptions_DisableRetry(t *testing.T) {
	options := NewOptions()
	assert.NotNil(t, options.RetryOptions)

	// Test fluent interface
	result := options.DisableRetry()
	assert.Same(t, options, result)

	// Verify retry was disabled
	assert.Nil(t, options.RetryOptions)
}

func TestNewOptions_HTTPClientNilByDefault(t *testing.T) {
	options := NewOptions()
	assert.Nil(t, options.HTTPClient)
}

func TestOptions_SetHTTPClient(t *testing.T) {
	options := NewOptions()
	c := &http.Client{}
	result := options.SetHTTPClient(c)
	assert.Same(t, options, result)
	assert.Same(t, c, options.HTTPClient)
}

func TestOptions_DisableRetryAfterSet(t *testing.T) {
	options := NewOptions().SetRetryOptions(&RetryOptions{MaxAttempts: 5})
	assert.NotNil(t, options.RetryOptions)
	options.DisableRetry()
	assert.Nil(t, options.RetryOptions)
}

func TestNewOptions_DefaultRetryValues(t *testing.T) {
	options := NewOptions()
	assert.Equal(t, DefaultRetryAttempts, options.RetryOptions.MaxAttempts)
	assert.Equal(t, DefaultRetryWaitTime, options.RetryOptions.WaitTime)
	assert.Equal(t, DefaultRetryMaxWaitTime, options.RetryOptions.MaxWaitTime)
	assert.True(t, options.RetryOptions.UseExponentialBackoff)
	assert.NotNil(t, options.RetryOptions.ShouldRetry)
}

// keep time referenced (avoids unused import if other assertions drop it)
var _ = time.Second
