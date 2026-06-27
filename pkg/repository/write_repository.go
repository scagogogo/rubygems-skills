package repository

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/crawler-go-go-go/go-requests"
	"github.com/scagogogo/rubygems-skills/pkg/models"
)

// WriteRepository defines the RubyGems write operations interface that requires authentication
// These operations require API Token authentication, including gem publishing, owner management and Webhooks
type WriteRepository interface {
	Repository

	// ======== Gem publishing and management ========

	// PushGem publish (push) a gem to RubyGems.org
	// requires API Token authentication, request body is the binary content of .gem file
	// POST /api/v1/gems
	PushGem(ctx context.Context, gemFile []byte) (string, error)

	// YankGem remove a gem version from RubyGems.org index (yank)
	// requires API Token authentication
	// DELETE /api/v1/gems/yank
	YankGem(ctx context.Context, gemName, version string) (string, error)

	// YankGemWithPlatform remove a gem version for specified platform from RubyGems.org index
	// requires API Token authentication
	// DELETE /api/v1/gems/yank (with platform parameter)
	YankGemWithPlatform(ctx context.Context, gemName, version, platform string) (string, error)

	// ======== Owner management ========

	// AddGemOwner add an owner to a gem package
	// requires API Token authentication
	// POST /api/v1/gems/{gem}/owners
	AddGemOwner(ctx context.Context, gemName, email, role string) error

	// RemoveGemOwner remove an owner from a gem package
	// requires API Token authentication
	// DELETE /api/v1/gems/{gem}/owners
	RemoveGemOwner(ctx context.Context, gemName, email string) error

	// UpdateGemOwnerRole update the role of a gem package owner
	// requires API Token authentication, role can be "owner" or "maintainer"
	// PATCH /api/v1/gems/{gem}/owners
	UpdateGemOwnerRole(ctx context.Context, gemName, email, role string) error

	// ======== Webhook management ========

	// ListWebhooks list all Webhooks under the current account
	// requires API Token authentication
	// GET /api/v1/web_hooks.json
	ListWebhooks(ctx context.Context) (map[string][]*models.Webhook, error)

	// CreateWebhook create a new Webhook
	// requires API Token authentication
	// gemName of "*" means global Webhook (triggered for all gems)
	// POST /api/v1/web_hooks
	CreateWebhook(ctx context.Context, gemName, webhookURL string) error

	// DeleteWebhook delete a Webhook
	// requires API Token authentication
	// DELETE /api/v1/web_hooks/remove
	DeleteWebhook(ctx context.Context, gemName, webhookURL string) error

	// FireWebhook test trigger a Webhook
	// requires API Token authentication
	// POST /api/v1/web_hooks/fire
	FireWebhook(ctx context.Context, gemName, webhookURL string) error

	// ======== API Key management ========

	// GetAPIKey retrieve (or create) a legacy API key using HTTP Basic authentication
	// GET /api/v1/api_key
	GetAPIKey(ctx context.Context, username, password string) (*models.APIKey, error)

	// CreateAPIKey create a new API key with specified scopes
	// POST /api/v1/api_key
	CreateAPIKey(ctx context.Context, username, password string, req *models.CreateAPIKeyRequest) (*models.APIKey, error)

	// UpdateAPIKey update an existing API key's scopes
	// PATCH /api/v1/api_key
	UpdateAPIKey(ctx context.Context, username, password string, req *models.UpdateAPIKeyRequest) (*models.APIKey, error)

	// ======== User profile (authenticated) ========

	// GetMyProfile get the authenticated user's full profile (including private fields)
	// requires HTTP Basic authentication
	// GET /api/v1/profiles/me.json
	GetMyProfile(ctx context.Context, username, password string) (*models.UserProfile, error)
}

// Compile-time check that WriteRepositoryImpl implements the WriteRepository interface
var _ WriteRepository = (*WriteRepositoryImpl)(nil)

// WriteRepositoryImpl is the implementation of RubyGems write operations that require authentication
// It embeds RepositoryImpl, inheriting all read operations
type WriteRepositoryImpl struct {
	*RepositoryImpl
}

// NewWriteRepository create a repository client with write operations
// Must provide Options with Token
func NewWriteRepository(options *Options) *WriteRepositoryImpl {
	if options == nil {
		options = NewOptions()
	}
	return &WriteRepositoryImpl{
		RepositoryImpl: NewRepository(options),
	}
}

// PushGem publish (push) a gem to RubyGems.org
// POST /api/v1/gems
func (w *WriteRepositoryImpl) PushGem(ctx context.Context, gemFile []byte) (string, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/gems", w.options.ServerURL)

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "gem.gem")
	if err != nil {
		return "", fmt.Errorf("creating multipart form: %w", err)
	}

	if _, err := part.Write(gemFile); err != nil {
		return "", fmt.Errorf("writing gem file to form: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("closing multipart writer: %w", err)
	}

	// Store the multipart body bytes for retry support
	multipartBytes := body.Bytes()
	contentType := writer.FormDataContentType()

	responseHandler := func(httpResponse *http.Response) (string, error) {
		respBody, err := io.ReadAll(httpResponse.Body)
		if err != nil {
			return "", fmt.Errorf("reading response: %w", err)
		}

		if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
			return string(respBody), NewAPIError(httpResponse, respBody, fmt.Errorf("push gem failed with status %d", httpResponse.StatusCode))
		}

		return string(respBody), nil
	}

	options := requests.NewOptions[any, string](targetUrl, responseHandler)

	// Set multipart body
	options.AppendRequestSetting(func(client *http.Client, request *http.Request) error {
		request.Method = http.MethodPost
		request.Body = io.NopCloser(bytes.NewReader(multipartBytes))
		request.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(multipartBytes)), nil
		}
		request.ContentLength = int64(len(multipartBytes))
		request.Header.Set("Content-Type", contentType)
		return nil
	})

	// Set proxy
	if w.options.Proxy != "" {
		options.AppendRequestSetting(requests.RequestSettingProxy(w.options.Proxy))
	}

	// Set Token authentication
	if w.options.Token != "" {
		options.AppendRequestSetting(func(client *http.Client, request *http.Request) error {
			request.Header.Set("Authorization", w.options.Token)
			return nil
		})
	}

	// If retry enabled, use request with retry
	if w.options.RetryOptions != nil {
		return SendRequestWithRetry(ctx, options, w.options.RetryOptions)
	}

	return requests.SendRequest[any, string](ctx, options)
}

// YankGem remove a gem version from RubyGems.org index
// DELETE /api/v1/gems/yank
func (w *WriteRepositoryImpl) YankGem(ctx context.Context, gemName, version string) (string, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/gems/yank", w.options.ServerURL)

	form := url.Values{}
	form.Set("gem_name", gemName)
	form.Set("version", version)

	return w.sendFormRequest(ctx, http.MethodDelete, targetUrl, form)
}

// YankGemWithPlatform remove a gem version for specified platform from RubyGems.org index
// DELETE /api/v1/gems/yank (with platform parameter)
func (w *WriteRepositoryImpl) YankGemWithPlatform(ctx context.Context, gemName, version, platform string) (string, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/gems/yank", w.options.ServerURL)

	form := url.Values{}
	form.Set("gem_name", gemName)
	form.Set("version", version)
	form.Set("platform", platform)

	return w.sendFormRequest(ctx, http.MethodDelete, targetUrl, form)
}

// AddGemOwner add an owner to a gem package
// POST /api/v1/gems/{gem}/owners
func (w *WriteRepositoryImpl) AddGemOwner(ctx context.Context, gemName, email, role string) error {
	targetUrl := fmt.Sprintf("%s/api/v1/gems/%s/owners", w.options.ServerURL, url.PathEscape(gemName))

	form := url.Values{}
	form.Set("email", email)
	form.Set("role", role)

	_, err := w.sendFormRequest(ctx, http.MethodPost, targetUrl, form)
	return err
}

// RemoveGemOwner remove an owner from a gem package
// DELETE /api/v1/gems/{gem}/owners
func (w *WriteRepositoryImpl) RemoveGemOwner(ctx context.Context, gemName, email string) error {
	targetUrl := fmt.Sprintf("%s/api/v1/gems/%s/owners", w.options.ServerURL, url.PathEscape(gemName))

	form := url.Values{}
	form.Set("email", email)

	_, err := w.sendFormRequest(ctx, http.MethodDelete, targetUrl, form)
	return err
}

// UpdateGemOwnerRole update the role of a gem package owner
// PATCH /api/v1/gems/{gem}/owners
func (w *WriteRepositoryImpl) UpdateGemOwnerRole(ctx context.Context, gemName, email, role string) error {
	targetUrl := fmt.Sprintf("%s/api/v1/gems/%s/owners", w.options.ServerURL, url.PathEscape(gemName))

	form := url.Values{}
	form.Set("email", email)
	form.Set("role", role)

	_, err := w.sendFormRequest(ctx, http.MethodPatch, targetUrl, form)
	return err
}

// ListWebhooks list all Webhooks under the current account
// GET /api/v1/web_hooks.json
func (w *WriteRepositoryImpl) ListWebhooks(ctx context.Context) (map[string][]*models.Webhook, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/web_hooks.json", w.options.ServerURL)
	return getJson[map[string][]*models.Webhook](ctx, w.RepositoryImpl, targetUrl)
}

// CreateWebhook create a new Webhook
// POST /api/v1/web_hooks
func (w *WriteRepositoryImpl) CreateWebhook(ctx context.Context, gemName, webhookURL string) error {
	targetUrl := fmt.Sprintf("%s/api/v1/web_hooks", w.options.ServerURL)

	form := url.Values{}
	form.Set("gem_name", gemName)
	form.Set("url", webhookURL)

	_, err := w.sendFormRequest(ctx, http.MethodPost, targetUrl, form)
	return err
}

// DeleteWebhook delete a Webhook
// DELETE /api/v1/web_hooks/remove
func (w *WriteRepositoryImpl) DeleteWebhook(ctx context.Context, gemName, webhookURL string) error {
	targetUrl := fmt.Sprintf("%s/api/v1/web_hooks/remove", w.options.ServerURL)

	form := url.Values{}
	form.Set("gem_name", gemName)
	form.Set("url", webhookURL)

	_, err := w.sendFormRequest(ctx, http.MethodDelete, targetUrl, form)
	return err
}

// FireWebhook test trigger a Webhook
// POST /api/v1/web_hooks/fire
func (w *WriteRepositoryImpl) FireWebhook(ctx context.Context, gemName, webhookURL string) error {
	targetUrl := fmt.Sprintf("%s/api/v1/web_hooks/fire", w.options.ServerURL)

	form := url.Values{}
	form.Set("gem_name", gemName)
	form.Set("url", webhookURL)

	_, err := w.sendFormRequest(ctx, http.MethodPost, targetUrl, form)
	return err
}

// GetAPIKey retrieve (or create) a legacy API key using HTTP Basic authentication
// GET /api/v1/api_key
func (w *WriteRepositoryImpl) GetAPIKey(ctx context.Context, username, password string) (*models.APIKey, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/api_key", w.options.ServerURL)
	return getJsonWithBasicAuth[*models.APIKey](ctx, w.RepositoryImpl, targetUrl, username, password)
}

// CreateAPIKey create a new API key with specified scopes
// POST /api/v1/api_key
func (w *WriteRepositoryImpl) CreateAPIKey(ctx context.Context, username, password string, req *models.CreateAPIKeyRequest) (*models.APIKey, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/api_key", w.options.ServerURL)

	form := url.Values{}
	form.Set("name", req.Name)
	for _, scope := range req.Scopes {
		form.Add("scopes[]", scope)
	}
	if req.MFA != "" {
		form.Set("mfa", req.MFA)
	}
	if req.RubygemName != "" {
		form.Set("rubygem_name", req.RubygemName)
	}
	if req.ExpiresAt != "" {
		form.Set("expires_at", req.ExpiresAt)
	}

	_, err := w.sendFormRequestWithBasicAuth(ctx, http.MethodPost, targetUrl, form, username, password)
	if err != nil {
		return nil, err
	}

	// RubyGems returns the API key as plain text, not JSON
	// Return an empty APIKey since the actual key value is in the response body
	return &models.APIKey{Name: req.Name, Scopes: req.Scopes}, nil
}

// UpdateAPIKey update an existing API key's scopes
// PATCH /api/v1/api_key
func (w *WriteRepositoryImpl) UpdateAPIKey(ctx context.Context, username, password string, req *models.UpdateAPIKeyRequest) (*models.APIKey, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/api_key", w.options.ServerURL)

	form := url.Values{}
	form.Set("api_key", req.APIKey)
	for _, scope := range req.Scopes {
		form.Add("scopes[]", scope)
	}
	if req.MFA != "" {
		form.Set("mfa", req.MFA)
	}

	_, err := w.sendFormRequestWithBasicAuth(ctx, http.MethodPatch, targetUrl, form, username, password)
	if err != nil {
		return nil, err
	}

	return &models.APIKey{Scopes: req.Scopes}, nil
}

// GetMyProfile get the authenticated user's full profile (including private fields)
// requires HTTP Basic authentication
// GET /api/v1/profiles/me.json
func (w *WriteRepositoryImpl) GetMyProfile(ctx context.Context, username, password string) (*models.UserProfile, error) {
	targetUrl := fmt.Sprintf("%s/api/v1/profiles/me.json", w.options.ServerURL)
	return getJsonWithBasicAuth[*models.UserProfile](ctx, w.RepositoryImpl, targetUrl, username, password)
}

// sendFormRequest send HTTP request with form data, supports retry
func (w *WriteRepositoryImpl) sendFormRequest(ctx context.Context, method, targetUrl string, form url.Values) (string, error) {
	encodedForm := form.Encode()

	responseHandler := func(httpResponse *http.Response) (string, error) {
		body, err := io.ReadAll(httpResponse.Body)
		if err != nil {
			return "", fmt.Errorf("reading response: %w", err)
		}

		if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
			return string(body), NewAPIError(httpResponse, body, fmt.Errorf("%s request failed with status %d", method, httpResponse.StatusCode))
		}

		return string(body), nil
	}

	options := requests.NewOptions[any, string](targetUrl, responseHandler)

	// Set request body and Content-Type
	options.AppendRequestSetting(func(client *http.Client, request *http.Request) error {
		request.Method = method
		request.Body = io.NopCloser(strings.NewReader(encodedForm))
		request.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(encodedForm)), nil
		}
		request.ContentLength = int64(len(encodedForm))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return nil
	})

	// Set proxy
	if w.options.Proxy != "" {
		options.AppendRequestSetting(requests.RequestSettingProxy(w.options.Proxy))
	}

	// Set Token authentication
	if w.options.Token != "" {
		options.AppendRequestSetting(func(client *http.Client, request *http.Request) error {
			request.Header.Set("Authorization", w.options.Token)
			return nil
		})
	}

	// If retry enabled, use request with retry
	if w.options.RetryOptions != nil {
		return SendRequestWithRetry(ctx, options, w.options.RetryOptions)
	}

	return requests.SendRequest[any, string](ctx, options)
}

// sendFormRequestWithBasicAuth send HTTP request with form data using HTTP Basic authentication, supports retry
func (w *WriteRepositoryImpl) sendFormRequestWithBasicAuth(ctx context.Context, method, targetUrl string, form url.Values, username, password string) (string, error) {
	encodedForm := form.Encode()

	responseHandler := func(httpResponse *http.Response) (string, error) {
		body, err := io.ReadAll(httpResponse.Body)
		if err != nil {
			return "", fmt.Errorf("reading response: %w", err)
		}

		if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
			return string(body), NewAPIError(httpResponse, body, fmt.Errorf("%s request failed with status %d", method, httpResponse.StatusCode))
		}

		return string(body), nil
	}

	options := requests.NewOptions[any, string](targetUrl, responseHandler)

	// Set request body and Content-Type
	options.AppendRequestSetting(func(client *http.Client, request *http.Request) error {
		request.Method = method
		request.Body = io.NopCloser(strings.NewReader(encodedForm))
		request.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(encodedForm)), nil
		}
		request.ContentLength = int64(len(encodedForm))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return nil
	})

	// Set proxy
	if w.options.Proxy != "" {
		options.AppendRequestSetting(requests.RequestSettingProxy(w.options.Proxy))
	}

	// Set HTTP Basic authentication
	options.AppendRequestSetting(func(client *http.Client, request *http.Request) error {
		request.SetBasicAuth(username, password)
		return nil
	})

	// If retry enabled, use request with retry
	if w.options.RetryOptions != nil {
		return SendRequestWithRetry(ctx, options, w.options.RetryOptions)
	}

	return requests.SendRequest[any, string](ctx, options)
}
