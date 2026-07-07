package repository

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/models"
	"github.com/stretchr/testify/assert"
)

// ----- PushGem -----

func TestPushGem_Success(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems", 200, "uploaded")
	w := newStubbedWriteRepo(tr)
	w.options.SetToken("t1")
	out, err := w.PushGem(context.Background(), []byte("gem-bytes"))
	assert.NoError(t, err)
	assert.Equal(t, "uploaded", out)
	assert.Equal(t, http.MethodPost, tr.requests[0].Method)
	assert.Equal(t, "t1", tr.requests[0].Header.Get("Authorization"))
	assert.Contains(t, tr.requests[0].Header.Get("Content-Type"), "multipart/form-data")
}

func TestPushGem_ServerError(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems", 422, "bad gem")
	w := newStubbedWriteRepo(tr)
	w.options.SetToken("t1")
	_, err := w.PushGem(context.Background(), []byte("x"))
	assert.Error(t, err)
}

func TestPushGem_NoToken(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems", 200, "ok")
	w := newStubbedWriteRepo(tr)
	_, err := w.PushGem(context.Background(), []byte("x"))
	assert.NoError(t, err)
	assert.Empty(t, tr.requests[0].Header.Get("Authorization"))
}

func TestPushGem_Retry(t *testing.T) {
	tr := newFakeTransport()
	tr.stubSequence("/api/v1/gems",
		cannedResponse{err: errors.New("transient")},
		cannedResponse{statusCode: 200, body: []byte("ok"), header: http.Header{}})
	opts := NewOptions().
		SetHTTPClient(&http.Client{Transport: tr}).
		SetToken("t1").
		SetRetryOptions(NewDefaultRetryOptions())
	opts.RetryOptions.WaitTime = time.Millisecond
	opts.RetryOptions.MaxWaitTime = time.Millisecond
	w := NewWriteRepository(opts)
	out, err := w.PushGem(context.Background(), []byte("x"))
	assert.NoError(t, err)
	assert.Equal(t, "ok", out)
	assert.GreaterOrEqual(t, tr.callCount, 2)
}

// ----- YankGem / YankGemWithPlatform -----

func TestYankGem(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems/yank", 200, "yanked")
	w := newStubbedWriteRepo(tr)
	w.options.SetToken("t1")
	out, err := w.YankGem(context.Background(), "rails", "7.0.0")
	assert.NoError(t, err)
	assert.Equal(t, "yanked", out)
	assert.Equal(t, http.MethodDelete, tr.requests[0].Method)
	assert.Contains(t, string(tr.requests[0].Body), "gem_name=rails")
	assert.Contains(t, string(tr.requests[0].Body), "version=7.0.0")
}

func TestYankGemWithPlatform(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems/yank", 200, "yanked")
	w := newStubbedWriteRepo(tr)
	_, err := w.YankGemWithPlatform(context.Background(), "rails", "7.0.0", "ruby")
	assert.NoError(t, err)
	assert.Contains(t, string(tr.requests[0].Body), "platform=ruby")
}

func TestYankGem_Error(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems/yank", 500, "err")
	w := newStubbedWriteRepo(tr)
	_, err := w.YankGem(context.Background(), "rails", "7.0.0")
	assert.Error(t, err)
}

// ----- Owner management -----

func TestAddGemOwner(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems/rails/owners", 200, "ok")
	w := newStubbedWriteRepo(tr)
	w.options.SetToken("t1")
	err := w.AddGemOwner(context.Background(), "rails", "a@b.com", "owner")
	assert.NoError(t, err)
	assert.Equal(t, http.MethodPost, tr.requests[0].Method)
	assert.Contains(t, string(tr.requests[0].Body), "email=a%40b.com")
	assert.Contains(t, string(tr.requests[0].Body), "role=owner")
}

func TestRemoveGemOwner(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems/rails/owners", 200, "ok")
	w := newStubbedWriteRepo(tr)
	err := w.RemoveGemOwner(context.Background(), "rails", "a@b.com")
	assert.NoError(t, err)
	assert.Equal(t, http.MethodDelete, tr.requests[0].Method)
}

func TestUpdateGemOwnerRole(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems/rails/owners", 200, "ok")
	w := newStubbedWriteRepo(tr)
	err := w.UpdateGemOwnerRole(context.Background(), "rails", "a@b.com", "maintainer")
	assert.NoError(t, err)
	assert.Equal(t, http.MethodPatch, tr.requests[0].Method)
	assert.Contains(t, string(tr.requests[0].Body), "role=maintainer")
}

func TestOwnerMethods_URLEscaping(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems/foo bar/owners", 200, "ok")
	w := newStubbedWriteRepo(tr)
	err := w.AddGemOwner(context.Background(), "foo bar", "a@b.com", "owner")
	assert.NoError(t, err)
	assert.Equal(t, "/api/v1/gems/foo bar/owners", tr.requests[0].Path)
}

func TestOwnerMethods_Error(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems/rails/owners", 403, "forbidden")
	w := newStubbedWriteRepo(tr)
	assert.Error(t, w.AddGemOwner(context.Background(), "rails", "a@b.com", "owner"))
	assert.Error(t, w.RemoveGemOwner(context.Background(), "rails", "a@b.com"))
	assert.Error(t, w.UpdateGemOwnerRole(context.Background(), "rails", "a@b.com", "owner"))
}

// ----- Webhooks -----

func TestListWebhooks(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/web_hooks.json", 200, `{"rails":[{"url":"http://h","failure_count":1}]}`)
	w := newStubbedWriteRepo(tr)
	res, err := w.ListWebhooks(context.Background())
	assert.NoError(t, err)
	assert.Contains(t, res, "rails")
	assert.Equal(t, "http://h", res["rails"][0].URL)
	assert.Equal(t, 1, res["rails"][0].FailureCount)
}

func TestListWebhooks_Error(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/web_hooks.json", 401, "unauth")
	w := newStubbedWriteRepo(tr)
	_, err := w.ListWebhooks(context.Background())
	assert.Error(t, err)
	assert.True(t, IsUnauthorized(err))
}

func TestCreateWebhook(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/web_hooks", 200, "ok")
	w := newStubbedWriteRepo(tr)
	w.options.SetToken("t1")
	err := w.CreateWebhook(context.Background(), "rails", "http://hook")
	assert.NoError(t, err)
	assert.Equal(t, http.MethodPost, tr.requests[0].Method)
	assert.Contains(t, string(tr.requests[0].Body), "gem_name=rails")
	assert.Contains(t, string(tr.requests[0].Body), "url=http%3A%2F%2Fhook")
}

func TestDeleteWebhook(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/web_hooks/remove", 200, "ok")
	w := newStubbedWriteRepo(tr)
	err := w.DeleteWebhook(context.Background(), "rails", "http://hook")
	assert.NoError(t, err)
	assert.Equal(t, http.MethodDelete, tr.requests[0].Method)
}

func TestFireWebhook(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/web_hooks/fire", 200, "ok")
	w := newStubbedWriteRepo(tr)
	err := w.FireWebhook(context.Background(), "*", "http://hook")
	assert.NoError(t, err)
	assert.Equal(t, http.MethodPost, tr.requests[0].Method)
	assert.Contains(t, string(tr.requests[0].Body), "gem_name=%2A")
}

func TestWebhookMethods_Error(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/web_hooks", 500, "err")
	tr.stub("/api/v1/web_hooks/remove", 500, "err")
	tr.stub("/api/v1/web_hooks/fire", 500, "err")
	w := newStubbedWriteRepo(tr)
	assert.Error(t, w.CreateWebhook(context.Background(), "rails", "http://hook"))
	assert.Error(t, w.DeleteWebhook(context.Background(), "rails", "http://hook"))
	assert.Error(t, w.FireWebhook(context.Background(), "rails", "http://hook"))
}

// ----- API Key (Basic Auth) -----

func TestGetAPIKey(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/api_key", 200, `{"id":1,"name":"key","scopes":["push_rubygem"]}`)
	w := newStubbedWriteRepo(tr)
	key, err := w.GetAPIKey(context.Background(), "user", "pass")
	assert.NoError(t, err)
	assert.Equal(t, "key", key.Name)
	assert.Contains(t, tr.requests[0].Header.Get("Authorization"), "Basic")
}

func TestGetAPIKey_Error(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/api_key", 401, "bad creds")
	w := newStubbedWriteRepo(tr)
	_, err := w.GetAPIKey(context.Background(), "user", "pass")
	assert.Error(t, err)
	assert.True(t, IsUnauthorized(err))
}

func TestCreateAPIKey_FullForm(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/api_key", 200, "plain-key-text")
	w := newStubbedWriteRepo(tr)
	req := &models.CreateAPIKeyRequest{
		Name:        "ci",
		Scopes:      []string{"push_rubygem", "yank_rubygem"},
		MFA:         "enabled",
		RubygemName: "rails",
		ExpiresAt:   "2025-01-01",
	}
	key, err := w.CreateAPIKey(context.Background(), "user", "pass", req)
	assert.NoError(t, err)
	assert.Equal(t, "ci", key.Name)
	assert.Equal(t, req.Scopes, key.Scopes)
	body := string(tr.requests[0].Body)
	assert.Contains(t, body, "name=ci")
	assert.Contains(t, body, "scopes%5B%5D=push_rubygem")
	assert.Contains(t, body, "scopes%5B%5D=yank_rubygem")
	assert.Contains(t, body, "mfa=enabled")
	assert.Contains(t, body, "rubygem_name=rails")
	assert.Contains(t, body, "expires_at=2025-01-01")
	assert.Equal(t, http.MethodPost, tr.requests[0].Method)
}

func TestCreateAPIKey_MinimalForm(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/api_key", 200, "ok")
	w := newStubbedWriteRepo(tr)
	req := &models.CreateAPIKeyRequest{Name: "ci", Scopes: []string{"push_rubygem"}}
	_, err := w.CreateAPIKey(context.Background(), "user", "pass", req)
	assert.NoError(t, err)
	body := string(tr.requests[0].Body)
	assert.NotContains(t, body, "mfa=")
	assert.NotContains(t, body, "rubygem_name=")
	assert.NotContains(t, body, "expires_at=")
}

func TestCreateAPIKey_Error(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/api_key", 422, "bad")
	w := newStubbedWriteRepo(tr)
	_, err := w.CreateAPIKey(context.Background(), "user", "pass", &models.CreateAPIKeyRequest{Name: "ci"})
	assert.Error(t, err)
}

func TestUpdateAPIKey(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/api_key", 200, "ok")
	w := newStubbedWriteRepo(tr)
	req := &models.UpdateAPIKeyRequest{
		APIKey: "abc123",
		Scopes: []string{"push_rubygem"},
		MFA:    "disabled",
	}
	key, err := w.UpdateAPIKey(context.Background(), "user", "pass", req)
	assert.NoError(t, err)
	assert.Equal(t, req.Scopes, key.Scopes)
	body := string(tr.requests[0].Body)
	assert.Contains(t, body, "api_key=abc123")
	assert.Contains(t, body, "mfa=disabled")
	assert.Equal(t, http.MethodPatch, tr.requests[0].Method)
}

func TestUpdateAPIKey_Error(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/api_key", 401, "bad")
	w := newStubbedWriteRepo(tr)
	_, err := w.UpdateAPIKey(context.Background(), "user", "pass", &models.UpdateAPIKeyRequest{APIKey: "x"})
	assert.Error(t, err)
}

// ----- GetMyProfile (Basic Auth) -----

func TestGetMyProfile(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/profiles/me.json", 200, `{"id":1,"handle":"qrush","email":"q@x.com"}`)
	w := newStubbedWriteRepo(tr)
	prof, err := w.GetMyProfile(context.Background(), "user", "pass")
	assert.NoError(t, err)
	assert.Equal(t, "qrush", prof.Handle)
	assert.Equal(t, "q@x.com", prof.Email)
}

func TestGetMyProfile_Error(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/profiles/me.json", 401, "bad")
	w := newStubbedWriteRepo(tr)
	_, err := w.GetMyProfile(context.Background(), "user", "pass")
	assert.Error(t, err)
}

// ----- NewWriteRepository edge -----

func TestNewWriteRepository_NilOptions(t *testing.T) {
	w := NewWriteRepository(nil)
	assert.NotNil(t, w)
	assert.Equal(t, DefaultServerURL, w.options.ServerURL)
}

// ----- form request with retry (covers RetryOptions != nil branch) -----

func TestSendFormRequest_Retry(t *testing.T) {
	tr := newFakeTransport()
	tr.stubSequence("/api/v1/gems/yank",
		cannedResponse{err: errors.New("transient")},
		cannedResponse{statusCode: 200, body: []byte("ok"), header: http.Header{}})
	opts := NewOptions().
		SetHTTPClient(&http.Client{Transport: tr}).
		SetToken("t1").
		SetRetryOptions(NewDefaultRetryOptions())
	opts.RetryOptions.WaitTime = time.Millisecond
	opts.RetryOptions.MaxWaitTime = time.Millisecond
	w := NewWriteRepository(opts)
	out, err := w.YankGem(context.Background(), "rails", "7.0.0")
	assert.NoError(t, err)
	assert.Equal(t, "ok", out)
	assert.GreaterOrEqual(t, tr.callCount, 2)
}

func TestSendFormRequestWithBasicAuth_Retry(t *testing.T) {
	tr := newFakeTransport()
	tr.stubSequence("/api/v1/api_key",
		cannedResponse{err: errors.New("transient")},
		cannedResponse{statusCode: 200, body: []byte(`{"id":1,"name":"k"}`), header: http.Header{}})
	opts := NewOptions().
		SetHTTPClient(&http.Client{Transport: tr}).
		SetRetryOptions(NewDefaultRetryOptions())
	opts.RetryOptions.WaitTime = time.Millisecond
	opts.RetryOptions.MaxWaitTime = time.Millisecond
	w := NewWriteRepository(opts)
	key, err := w.GetAPIKey(context.Background(), "user", "pass")
	assert.NoError(t, err)
	assert.Equal(t, "k", key.Name)
	assert.GreaterOrEqual(t, tr.callCount, 2)
}

// TestCreateAPIKey_Retry covers the RetryOptions != nil branch of
// sendFormRequestWithBasicAuth via a POST form basic-auth call.
func TestCreateAPIKey_Retry(t *testing.T) {
	tr := newFakeTransport()
	tr.stubSequence("/api/v1/api_key",
		cannedResponse{err: errors.New("transient")},
		cannedResponse{statusCode: 200, body: []byte("ok"), header: http.Header{}})
	opts := NewOptions().
		SetHTTPClient(&http.Client{Transport: tr}).
		SetRetryOptions(NewDefaultRetryOptions())
	opts.RetryOptions.WaitTime = time.Millisecond
	opts.RetryOptions.MaxWaitTime = time.Millisecond
	w := NewWriteRepository(opts)
	_, err := w.CreateAPIKey(context.Background(), "u", "p", &models.CreateAPIKeyRequest{Name: "k"})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, tr.callCount, 2)
}

// TestCreateAPIKey_WithProxy covers the Proxy branch of sendFormRequestWithBasicAuth.
func TestCreateAPIKey_WithProxy(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/api_key", 200, "ok")
	opts := NewOptions().SetProxy("http://127.0.0.1:1")
	opts.SetHTTPClient(&http.Client{Transport: tr})
	opts.DisableRetry()
	w := NewWriteRepository(opts)
	_, err := w.CreateAPIKey(context.Background(), "u", "p", &models.CreateAPIKeyRequest{Name: "k"})
	assert.Error(t, err)
}

// ----- multipart builder error branches -----

// failWriter always returns an error on Write, exercising the CreateFormFile
// error branch of buildGemMultipart.
type failWriter struct{}

func (failWriter) Write(p []byte) (int, error) { return 0, errors.New("write failed") }

// failAfterWriter succeeds for the first `ok` bytes then fails, exercising the
// part.Write / writer.Close error branches (which run after the form header
// has been written).
type failAfterWriter struct {
	ok      int
	written int
}

func (f *failAfterWriter) Write(p []byte) (int, error) {
	if f.written >= f.ok {
		return 0, errors.New("write failed")
	}
	n := copy(p, make([]byte, f.ok-f.written))
	if n < len(p) {
		f.written = f.ok
		return n, errors.New("write failed")
	}
	f.written += n
	return n, nil
}

func TestBuildGemMultipart_CreateFormFileError(t *testing.T) {
	_, err := buildGemMultipart([]byte("x"), failWriter{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "creating multipart form")
}

func TestBuildGemMultipart_PartWriteError(t *testing.T) {
	// First, learn the byte length of the form header (everything written before
	// the gem payload) by running a successful build with an empty payload.
	gem := make([]byte, 500)
	headerLen := len(multipartBytesFor(t, nil))
	// Allow exactly the header through, then fail on the first gem byte.
	_, err := buildGemMultipart(gem, &failAfterWriter{ok: headerLen})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "writing gem file to form")
}

func TestBuildGemMultipart_CloseError(t *testing.T) {
	// Allow header + gem payload through, then fail on the closing boundary
	// written by writer.Close().
	gem := []byte("g")
	full := len(multipartBytesFor(t, gem))
	// ok just below the full length so the final Close-boundary write fails.
	_, err := buildGemMultipart(gem, &failAfterWriter{ok: full - 1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closing multipart writer")
}

// multipartBytesFor runs a successful build into a buffer and returns the bytes
// written, so tests can size failAfterWriter precisely.
func multipartBytesFor(t *testing.T, gem []byte) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	_, err := buildGemMultipart(gem, buf)
	assert.NoError(t, err)
	return buf.Bytes()
}

func TestBuildGemMultipart_Success(t *testing.T) {
	buf := &bytes.Buffer{}
	ct, err := buildGemMultipart([]byte("gem"), buf)
	assert.NoError(t, err)
	assert.Contains(t, ct, "multipart/form-data")
	assert.Greater(t, buf.Len(), 0)
}

// ----- proxy branch coverage (write methods + basic-auth GET) -----

func TestPushGem_WithProxy(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems", 200, "ok")
	opts := NewOptions().SetProxy("http://127.0.0.1:1").SetToken("t1")
	opts.SetHTTPClient(&http.Client{Transport: tr})
	opts.DisableRetry()
	w := NewWriteRepository(opts)
	_, err := w.PushGem(context.Background(), []byte("x"))
	assert.Error(t, err) // unreachable proxy fails fast
}

func TestSendFormRequest_WithProxy(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems/yank", 200, "ok")
	opts := NewOptions().SetProxy("http://127.0.0.1:1").SetToken("t1")
	opts.SetHTTPClient(&http.Client{Transport: tr})
	opts.DisableRetry()
	w := NewWriteRepository(opts)
	_, err := w.YankGem(context.Background(), "rails", "7.0.0")
	assert.Error(t, err)
}

func TestGetBytesWithBasicAuth_WithProxy(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/profiles/me.json", 200, `{"id":1,"handle":"q"}`)
	opts := NewOptions().SetProxy("http://127.0.0.1:1")
	opts.SetHTTPClient(&http.Client{Transport: tr})
	opts.DisableRetry()
	w := NewWriteRepository(opts)
	_, err := w.GetMyProfile(context.Background(), "user", "pass")
	assert.Error(t, err)
}

// ----- response-body read-error branch coverage -----
// A body that fails on read exercises the "reading response" error path in
// PushGem, sendFormRequest, sendFormRequestWithBasicAuth, getBytes and
// getBytesWithBasicAuth response handlers.

func TestPushGem_BodyReadError(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems", 200, "ignored")
	tr.overrideBody("/api/v1/gems", brokenBody{})
	w := newStubbedWriteRepo(tr)
	w.options.SetToken("t1")
	_, err := w.PushGem(context.Background(), []byte("x"))
	assert.Error(t, err)
}

func TestSendFormRequest_BodyReadError(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems/yank", 200, "ignored")
	tr.overrideBody("/api/v1/gems/yank", brokenBody{})
	w := newStubbedWriteRepo(tr)
	_, err := w.YankGem(context.Background(), "rails", "7.0.0")
	assert.Error(t, err)
}

func TestSendFormRequestWithBasicAuth_BodyReadError(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/api_key", 200, "ignored")
	tr.overrideBody("/api/v1/api_key", brokenBody{})
	w := newStubbedWriteRepo(tr)
	_, err := w.CreateAPIKey(context.Background(), "u", "p", &models.CreateAPIKeyRequest{Name: "k"})
	assert.Error(t, err)
}

func TestGetBytes_BodyReadError(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/gems/rails.json", 200, "ignored")
	tr.overrideBody("/api/v1/gems/rails.json", brokenBody{})
	repo := newStubbedRepo(t, tr)
	_, err := repo.GetPackage(context.Background(), "rails")
	assert.Error(t, err)
}

func TestGetBytesWithBasicAuth_BodyReadError(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/profiles/me.json", 200, "ignored")
	tr.overrideBody("/api/v1/profiles/me.json", brokenBody{})
	w := newStubbedWriteRepo(tr)
	_, err := w.GetMyProfile(context.Background(), "u", "p")
	assert.Error(t, err)
}

// getBytesWithBasicAuth non-2xx with a successful body read covers the
// readErr==nil branch of the error path.
func TestGetBytesWithBasicAuth_ServerErrorBodyRead(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/profiles/me.json", 500, "server boom")
	w := newStubbedWriteRepo(tr)
	_, err := w.GetMyProfile(context.Background(), "u", "p")
	assert.Error(t, err)
}

// getBytesWithBasicAuth 404 (mirrors the getBytes NotFound test) to cover the
// readErr==nil branch with a small, successfully-readable body.
func TestGetBytesWithBasicAuth_NotFoundBodyRead(t *testing.T) {
	tr := newFakeTransport()
	tr.stub("/api/v1/profiles/me.json", 404, "not found")
	w := newStubbedWriteRepo(tr)
	_, err := w.GetMyProfile(context.Background(), "u", "p")
	assert.Error(t, err)
	assert.True(t, IsNotFound(err))
}
