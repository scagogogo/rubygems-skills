package repository

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	// ErrInvalidRequest invalid request parameters
	ErrInvalidRequest = errors.New("invalid request parameters")

	// ErrNotFound resource not found
	ErrNotFound = errors.New("resource not found")

	// ErrServerError server error
	ErrServerError = errors.New("server error")

	// ErrRateLimited request rate limited
	ErrRateLimited = errors.New("request rate limited")

	// ErrUnauthorized unauthorized
	ErrUnauthorized = errors.New("unauthorized")

	// ErrTimeout request timeout
	ErrTimeout = errors.New("request timeout")

	// ErrNetworkFailure network failure
	ErrNetworkFailure = errors.New("network failure")
)

// APIError represents error encountered during API call
type APIError struct {
	// Error cause
	Cause error

	// HTTP status code
	StatusCode int

	// Request URL
	URL string

	// Response content
	Response string
}

// implement Error interface
func (e *APIError) Error() string {
	return fmt.Sprintf("API error (status: %d, url: %s): %v", e.StatusCode, e.URL, e.Cause)
}

// NewAPIError create APIError from HTTP response
func NewAPIError(resp *http.Response, body []byte, cause error) *APIError {
	return &APIError{
		Cause:      cause,
		StatusCode: resp.StatusCode,
		URL:        resp.Request.URL.String(),
		Response:   string(body),
	}
}

// IsNotFound check if error is resource not found
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return errors.Is(err, ErrNotFound)
}

// IsRateLimited check if error is request rate limited
func IsRateLimited(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusTooManyRequests
	}
	return errors.Is(err, ErrRateLimited)
}

// IsUnauthorized check if error is unauthorized
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusUnauthorized
	}
	return errors.Is(err, ErrUnauthorized)
}
