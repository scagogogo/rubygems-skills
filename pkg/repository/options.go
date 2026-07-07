package repository

import "net/http"

// DefaultServerURL Default repository URL, connect directly to official repository
const DefaultServerURL = "https://rubygems.org"

type Options struct {

	// Repository server URL
	ServerURL string

	// Use proxy when sending requests to repository
	Proxy string

	// Token for API authentication
	// See: https://guides.rubygems.org/rubygems-org-api-v2/#rate-limits
	Token string

	// Request retry options
	RetryOptions *RetryOptions

	// HTTPClient is an optional custom *http.Client used for sending requests.
	// When non-nil, its Transport is applied to the request client (via a
	// RequestSetting), enabling HTTP stubbing in tests. When nil, the default
	// go-requests client behavior is used (proxy/token still applied).
	HTTPClient *http.Client
}

func NewOptions() *Options {
	return &Options{
		ServerURL:    DefaultServerURL,
		Proxy:        "",
		Token:        "",
		RetryOptions: NewDefaultRetryOptions(),
	}
}

func (x *Options) SetServerURL(serverUrl string) *Options {
	x.ServerURL = serverUrl
	return x
}

func (x *Options) SetProxy(proxy string) *Options {
	x.Proxy = proxy
	return x
}

func (x *Options) SetToken(token string) *Options {
	x.Token = token
	return x
}

func (x *Options) SetRetryOptions(retryOptions *RetryOptions) *Options {
	x.RetryOptions = retryOptions
	return x
}

// SetHTTPClient sets a custom HTTP client (used for test stubbing).
func (x *Options) SetHTTPClient(client *http.Client) *Options {
	x.HTTPClient = client
	return x
}

// DisableRetry disable retry functionality
func (x *Options) DisableRetry() *Options {
	x.RetryOptions = nil
	return x
}
