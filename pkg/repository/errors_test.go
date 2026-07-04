package repository

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test creating an API error
func TestNewAPIError(t *testing.T) {
	// Create a test request and response
	req, _ := http.NewRequest("GET", "https://example.com/test", nil)
	resp := &http.Response{
		StatusCode: http.StatusNotFound,
		Request:    req,
	}

	// Create an API error
	cause := errors.New("test error")
	body := []byte("Not found")
	apiErr := NewAPIError(resp, body, cause)

	// Verify error fields
	assert.Equal(t, cause, apiErr.Cause, "original error should be preserved")
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode, "status code should be preserved")
	assert.Equal(t, "https://example.com/test", apiErr.URL, "URL should be preserved")
	assert.Equal(t, "Not found", apiErr.Response, "response body should be preserved")
}

// Test the string representation of an error
func TestAPIError_Error(t *testing.T) {
	apiErr := &APIError{
		Cause:      errors.New("test error"),
		StatusCode: http.StatusInternalServerError,
		URL:        "https://example.com/test",
		Response:   "Server error",
	}

	errorStr := apiErr.Error()
	assert.Contains(t, errorStr, "500", "error string should contain status code")
	assert.Contains(t, errorStr, "https://example.com/test", "error string should contain URL")
	assert.Contains(t, errorStr, "test error", "error string should contain original error message")
}

// Test NotFound error detection
func TestIsNotFound(t *testing.T) {
	// Test direct ErrNotFound
	assert.True(t, IsNotFound(ErrNotFound), "ErrNotFound should be identified as NotFound")

	// Test API error with 404 status code
	apiErr := &APIError{
		Cause:      errors.New("test error"),
		StatusCode: http.StatusNotFound,
		URL:        "https://example.com/test",
	}
	assert.True(t, IsNotFound(apiErr), "404 API error should be identified as NotFound")

	// Test other errors
	assert.False(t, IsNotFound(errors.New("random error")), "random error should not be identified as NotFound")

	// Test API error with other status codes
	apiErr = &APIError{
		Cause:      errors.New("test error"),
		StatusCode: http.StatusBadRequest,
		URL:        "https://example.com/test",
	}
	assert.False(t, IsNotFound(apiErr), "400 API error should not be identified as NotFound")
}

// Test RateLimited error detection
func TestIsRateLimited(t *testing.T) {
	// Test direct ErrRateLimited
	assert.True(t, IsRateLimited(ErrRateLimited), "ErrRateLimited should be identified as RateLimited")

	// Test API error with 429 status code
	apiErr := &APIError{
		Cause:      errors.New("test error"),
		StatusCode: http.StatusTooManyRequests,
		URL:        "https://example.com/test",
	}
	assert.True(t, IsRateLimited(apiErr), "429 API error should be identified as RateLimited")

	// Test other errors
	assert.False(t, IsRateLimited(errors.New("random error")), "random error should not be identified as RateLimited")

	// Test API error with other status codes
	apiErr = &APIError{
		Cause:      errors.New("test error"),
		StatusCode: http.StatusBadRequest,
		URL:        "https://example.com/test",
	}
	assert.False(t, IsRateLimited(apiErr), "400 API error should not be identified as RateLimited")
}

// Test Unauthorized error detection
func TestIsUnauthorized(t *testing.T) {
	// Test direct ErrUnauthorized
	assert.True(t, IsUnauthorized(ErrUnauthorized), "ErrUnauthorized should be identified as Unauthorized")

	// Test API error with 401 status code
	apiErr := &APIError{
		Cause:      errors.New("test error"),
		StatusCode: http.StatusUnauthorized,
		URL:        "https://example.com/test",
	}
	assert.True(t, IsUnauthorized(apiErr), "401 API error should be identified as Unauthorized")

	// Test other errors
	assert.False(t, IsUnauthorized(errors.New("random error")), "random error should not be identified as Unauthorized")

	// Test API error with other status codes
	apiErr = &APIError{
		Cause:      errors.New("test error"),
		StatusCode: http.StatusBadRequest,
		URL:        "https://example.com/test",
	}
	assert.False(t, IsUnauthorized(apiErr), "400 API error should not be identified as Unauthorized")
}

// Test type detection of wrapped errors
func TestErrorWrapping(t *testing.T) {
	// Create an API error
	apiErr := &APIError{
		Cause:      ErrNotFound,
		StatusCode: http.StatusNotFound,
		URL:        "https://example.com/test",
	}

	// Wrap it with another layer
	wrappedErr := errors.New("wrapped: " + apiErr.Error())

	// errors.Is should not find the original error, because it is not properly implemented
	assert.False(t, errors.Is(wrappedErr, ErrNotFound), "simple wrapping should not be able to identify the underlying error")

	// Use the correct wrapping approach
	wrappedErr2 := fmt.Errorf("wrapped: %w", apiErr)

	// errors.As should be able to extract the API error
	var extractedAPIErr *APIError
	assert.True(t, errors.As(wrappedErr2, &extractedAPIErr), "errors.As should be able to extract the API error")
	assert.Equal(t, http.StatusNotFound, extractedAPIErr.StatusCode, "extracted API error should preserve the status code")
}

// Test different error types
func TestErrorTypes(t *testing.T) {
	errorTypes := []error{
		ErrInvalidRequest,
		ErrNotFound,
		ErrServerError,
		ErrRateLimited,
		ErrUnauthorized,
		ErrTimeout,
		ErrNetworkFailure,
	}

	// Ensure all error types are different
	for i, err1 := range errorTypes {
		for j, err2 := range errorTypes {
			if i != j {
				assert.NotEqual(t, err1, err2, "error types should be different: %v and %v", err1, err2)
			}
		}
	}
}
