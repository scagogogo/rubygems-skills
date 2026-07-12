package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/scagogogo/rubygems-skills/pkg/repository"
)

// ============================================================
// Test helpers
// ============================================================

// stubServer returns an httptest.Server whose handler maps URL path -> canned
// (status, body). It serves the JSON that the read/write commands expect.
func stubServer(routes map[string]struct {
	status int
	body   string
}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rt, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if rt.status == 0 {
			rt.status = 200
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(rt.status)
		_, _ = io.WriteString(w, rt.body)
	}))
}

// swapRepo replaces newRepoFunc and restores it on cleanup.
func swapRepo(t *testing.T, repo repository.Repository) {
	t.Helper()
	prev := newRepoFunc
	newRepoFunc = func() repository.Repository { return repo }
	t.Cleanup(func() { newRepoFunc = prev })
}

// swapWriteRepo replaces newWriteRepoFunc and restores it on cleanup.
func swapWriteRepo(t *testing.T, repo *repository.WriteRepositoryImpl) {
	t.Helper()
	prev := newWriteRepoFunc
	newWriteRepoFunc = func() *repository.WriteRepositoryImpl { return repo }
	t.Cleanup(func() { newWriteRepoFunc = prev })
}

// stubbedRepo builds a read Repository pointing at the given server URL, with
// retry disabled so 4xx/5xx stub responses fail fast instead of retrying 3×
// (which would make the test suite slow and flaky).
func stubbedRepo(url string) repository.Repository {
	opts := repository.NewOptions().SetServerURL(url).DisableRetry()
	return repository.NewRepository(opts)
}

// stubbedWriteRepo builds a WriteRepository pointing at the given server URL
// with a token (so write ops send Authorization) and retry disabled.
func stubbedWriteRepo(url string) *repository.WriteRepositoryImpl {
	opts := repository.NewOptions().SetToken("test-token").SetServerURL(url).DisableRetry()
	return repository.NewWriteRepository(opts)
}

// captureStdout runs fn with os.Stdout replaced by a pipe and returns whatever
// was written. Commands use fmt.Print* directly (not cobra's Out), so we must
// redirect the real os.Stdout.
func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = old })

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	fnErr := fn()
	_ = w.Close()
	<-done
	os.Stdout = old
	return buf.String(), fnErr
}

// runRoot executes the root command with the given args, capturing stdout.
// Returns (stdout, execErr). exitCode is reset before each run.
func runRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()
	exitCode = 0
	root := buildRootCmd()
	root.SetArgs(args)
	var out string
	var err error
	out, err = captureStdout(t, func() error {
		return root.Execute()
	})
	return out, err
}

// resetFlags zeroes the package-level flag vars so one test's flags don't leak
// into the next. Called at the top of each test.
func resetFlags() {
	flagMirror = "default"
	flagServer = ""
	flagToken = ""
	flagProxy = ""
	flagTimeout = 30
	flagJSON = false
	flagCache = false
	flagCacheTTL = 5
	flagRetry = false
	flagRetryAttempts = 3
	flagRetryWait = 1
	flagRetryBackoff = true
}

// keep imports referenced
var _ = context.Background
var _ = json.Marshal

// ============================================================
// buildRootCmd structure
// ============================================================

func TestBuildRootCmd_HasAllSubcommands(t *testing.T) {
	root := buildRootCmd()
	names := map[string]bool{}
	for _, c := range root.Commands() {
		names[c.Name()] = true
	}
	want := []string{
		"get", "search", "autocomplete", "versions", "latest-version",
		"version-detail", "version-contents", "downloads", "version-downloads",
		"top-downloads", "deps", "rdeps", "version-rdeps", "latest-gems",
		"just-updated", "user-profile", "owned-gems", "gems-by-owner",
		"gem-owners", "attestations", "mfa-status", "timeframe",
		"bulk-get", "bulk-versions", "bulk-deps", "bulk-rdeps",
		"push", "yank", "add-owner", "remove-owner", "update-owner",
		"list-webhooks", "create-webhook", "delete-webhook", "fire-webhook",
		"get-api-key", "create-api-key", "update-api-key", "my-profile",
		"install", "platform",
	}
	for _, w := range want {
		if !names[w] {
			t.Errorf("missing subcommand: %s", w)
		}
	}
}

func TestBuildRootCmd_NoArgsPrintsHelp(t *testing.T) {
	resetFlags()
	root := buildRootCmd()
	root.SetArgs([]string{})
	// No subcommand -> cobra prints help to stdout (via root.Help), returns nil.
	out, err := runRoot(t)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "rubygems") {
		t.Errorf("help output missing 'rubygems': %q", out)
	}
}

// ============================================================
// Read commands against a stub server
// ============================================================

func TestGetCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems/rails.json": {200, `{"name":"rails","version":"7.0.0","downloads":100,"info":"full-stack","homepage_uri":"https://rubyonrails.org","metadata":{"source_code_uri":"https://github.com/rails/rails"}}`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "get", "rails")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "rails") || !strings.Contains(out, "7.0.0") {
		t.Errorf("output=%q", out)
	}
	if !strings.Contains(out, "Source code:") {
		t.Errorf("expected source code line: %q", out)
	}
}

func TestGetCmd_JSON(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems/rails.json": {200, `{"name":"rails","version":"7.0.0","downloads":1}`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "get", "rails", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, `"name":`) {
		t.Errorf("expected JSON output: %q", out)
	}
}

func TestGetCmd_NotFound(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems/nope.json": {404, "not found"},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	_, err := runRoot(t, "get", "nope")
	if err == nil || !strings.Contains(err.Error(), "not found (404)") {
		t.Errorf("err=%v", err)
	}
}

func TestGetCmd_ServerError(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems/rails.json": {500, "boom"},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	_, err := runRoot(t, "get", "rails")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetCmd_MissingArg(t *testing.T) {
	resetFlags()
	_, err := runRoot(t, "get")
	if err == nil {
		t.Fatal("expected error for missing arg")
	}
}

func TestSearchCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/search.json": {200, `[{"name":"rails","version":"7.0.0","downloads":100},{"name":"railtie","version":"6.1.0","downloads":50}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "search", "rail", "--limit", "1")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "rails") || strings.Contains(out, "railtie") {
		t.Errorf("limit not applied: %q", out)
	}
}

func TestSearchCmd_Error(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/search.json": {500, "err"},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	_, err := runRoot(t, "search", "x")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVersionsCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/versions/rails.json": {200, `[{"number":"7.0.0","downloads_count":10,"created_at":"2024-01-01T00:00:00Z"},{"number":"6.1.0","downloads_count":5,"created_at":"2023-01-01T00:00:00Z"}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "versions", "rails", "--limit", "1")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "7.0.0") || strings.Contains(out, "6.1.0") {
		t.Errorf("output=%q", out)
	}
}

func TestLatestVersionCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/versions/rails/latest.json": {200, `{"version":"7.0.0"}`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "latest-version", "rails", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "7.0.0") {
		t.Errorf("output=%q", out)
	}
}

func TestDownloadsCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/downloads.json": {200, `{"total":1000}`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	_, err := runRoot(t, "downloads", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestDepsCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/dependencies": {200, `[{"name":"rails","dependencies":{"railtie":"6.1.0"}},{"name":"rack","dependencies":{"rack-cache":"1.0"}}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "deps", "rails", "rack", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "rails") {
		t.Errorf("output=%q", out)
	}
}

func TestTimeframeCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/timeframe_versions.json": {200, `[]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	_, err := runRoot(t, "timeframe", "--from", "2024-01-01T00:00:00Z", "--to", "2024-12-31T00:00:00Z", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
}

// ============================================================
// Bulk commands
// ============================================================

func TestParseGems(t *testing.T) {
	cases := []struct {
		args []string
		want []string
	}{
		{[]string{"rails", "rack"}, []string{"rails", "rack"}},
		{[]string{"rails,rack"}, []string{"rails", "rack"}},
		{[]string{"rails, rack ,,"}, []string{"rails", "rack"}},
		{[]string{""}, []string{}},
	}
	for _, c := range cases {
		got := parseGems(c.args)
		if len(got) != len(c.want) {
			t.Errorf("parseGems(%v)=%v, want %v", c.args, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("parseGems(%v)[%d]=%q, want %q", c.args, i, got[i], c.want[i])
			}
		}
	}
}

func TestBulkGetCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems/rails.json": {200, `{"name":"rails","version":"7.0.0","downloads":100}`},
		"/api/v1/gems/rack.json":  {200, `{"name":"rack","version":"2.2.7","downloads":50}`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "bulk-get", "rails,rack")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "rails") || !strings.Contains(out, "rack") {
		t.Errorf("output=%q", out)
	}
}

func TestBulkGetCmd_JSON(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems/rails.json": {200, `{"name":"rails","version":"7.0.0","downloads":1}`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "bulk-get", "rails", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "rails") {
		t.Errorf("output=%q", out)
	}
}

func TestBulkGetCmd_PartialFailure(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems/rails.json": {200, `{"name":"rails","version":"7.0.0","downloads":1}`},
		"/api/v1/gems/fail.json":  {500, "err"},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "bulk-get", "rails,fail")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "FAILED") {
		t.Errorf("expected FAILED line: %q", out)
	}
}

func TestBulkVersionsCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/versions/rails.json": {200, `[{"number":"7.0.0"}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "bulk-versions", "rails")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "1 versions") {
		t.Errorf("output=%q", out)
	}
}

func TestBulkDepsCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/dependencies": {200, `[]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "bulk-deps", "rails")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "dependencies") {
		t.Errorf("output=%q", out)
	}
}

func TestBulkRdepsCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems/rails/reverse_dependencies.json": {200, `["rack"]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "bulk-rdeps", "rails")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "reverse dependencies") {
		t.Errorf("output=%q", out)
	}
}

// ============================================================
// Write commands against a stub server
// ============================================================

func TestYankCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems/yank": {200, `yanked`},
	})
	defer srv.Close()
	swapWriteRepo(t, stubbedWriteRepo(srv.URL))

	out, err := runRoot(t, "yank", "rails", "7.0.0")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "yanked") {
		t.Errorf("output=%q", out)
	}
}

func TestYankCmd_WithPlatform(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems/yank": {200, `yanked`},
	})
	defer srv.Close()
	swapWriteRepo(t, stubbedWriteRepo(srv.URL))

	_, err := runRoot(t, "yank", "rails", "7.0.0", "--platform", "x86_64-linux")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestAddOwnerCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems/rails/owners": {200, ``},
	})
	defer srv.Close()
	swapWriteRepo(t, stubbedWriteRepo(srv.URL))

	out, err := runRoot(t, "add-owner", "rails", "a@b.com")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "Added owner") {
		t.Errorf("output=%q", out)
	}
}

func TestRemoveOwnerCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems/rails/owners": {200, ``},
	})
	defer srv.Close()
	swapWriteRepo(t, stubbedWriteRepo(srv.URL))

	out, err := runRoot(t, "remove-owner", "rails", "a@b.com")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "Removed owner") {
		t.Errorf("output=%q", out)
	}
}

func TestUpdateOwnerCmd_MissingRole(t *testing.T) {
	resetFlags()
	// --role is required; omitting it must error before any HTTP call.
	_, err := runRoot(t, "update-owner", "rails", "a@b.com")
	if err == nil {
		t.Fatal("expected required-flag error")
	}
}

func TestListWebhooksCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/web_hooks.json": {200, `{"rails":[{"url":"https://example.com"}]}`},
	})
	defer srv.Close()
	swapWriteRepo(t, stubbedWriteRepo(srv.URL))

	out, err := runRoot(t, "list-webhooks", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "example.com") {
		t.Errorf("output=%q", out)
	}
}

func TestCreateWebhookCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/web_hooks": {200, ``},
	})
	defer srv.Close()
	swapWriteRepo(t, stubbedWriteRepo(srv.URL))

	out, err := runRoot(t, "create-webhook", "rails", "https://example.com")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "Created webhook") {
		t.Errorf("output=%q", out)
	}
}

func TestDeleteWebhookCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/web_hooks/remove": {200, ``},
	})
	defer srv.Close()
	swapWriteRepo(t, stubbedWriteRepo(srv.URL))

	out, err := runRoot(t, "delete-webhook", "rails", "https://example.com")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "Deleted webhook") {
		t.Errorf("output=%q", out)
	}
}

func TestFireWebhookCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/web_hooks/fire": {200, ``},
	})
	defer srv.Close()
	swapWriteRepo(t, stubbedWriteRepo(srv.URL))

	out, err := runRoot(t, "fire-webhook", "rails", "https://example.com")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "Fired webhook") {
		t.Errorf("output=%q", out)
	}
}

func TestGetAPIKeyCmd_MissingUser(t *testing.T) {
	resetFlags()
	_, err := runRoot(t, "get-api-key")
	if err == nil {
		t.Fatal("expected required --user error")
	}
}

func TestGetAPIKeyCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/api_key": {200, `{"key":"secret-key","name":"n"}`},
	})
	defer srv.Close()
	swapWriteRepo(t, stubbedWriteRepo(srv.URL))

	out, err := runRoot(t, "get-api-key", "--user", "u", "--password", "p", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "secret-key") {
		t.Errorf("output=%q", out)
	}
}

func TestCreateAPIKeyCmd_MissingName(t *testing.T) {
	resetFlags()
	_, err := runRoot(t, "create-api-key", "--user", "u")
	if err == nil {
		t.Fatal("expected required --name error")
	}
}

func TestCreateAPIKeyCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/api_key": {200, `new-key-plain-text`},
	})
	defer srv.Close()
	swapWriteRepo(t, stubbedWriteRepo(srv.URL))

	out, err := runRoot(t, "create-api-key", "--user", "u", "--password", "p", "--name", "ci", "--scopes", "push_rubygem", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	// CreateAPIKey returns {Name, Scopes} (the key value arrives as plain text and
	// isn't parsed by the SDK); assert the echoed name + scope instead.
	if !strings.Contains(out, "ci") || !strings.Contains(out, "push_rubygem") {
		t.Errorf("output=%q", out)
	}
}

func TestUpdateAPIKeyCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/api_key": {200, ``},
	})
	defer srv.Close()
	swapWriteRepo(t, stubbedWriteRepo(srv.URL))

	out, err := runRoot(t, "update-api-key", "--user", "u", "--password", "p", "--api-key", "k", "--scopes", "yank_rubygem", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	// UpdateAPIKey returns {Scopes}; assert the echoed scope.
	if !strings.Contains(out, "yank_rubygem") {
		t.Errorf("output=%q", out)
	}
}

func TestMyProfileCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/profiles/me.json": {200, `{"id":1,"handle":"u","email":"a@b.com"}`},
	})
	defer srv.Close()
	swapWriteRepo(t, stubbedWriteRepo(srv.URL))

	out, err := runRoot(t, "my-profile", "--user", "u", "--password", "p", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "u") {
		t.Errorf("output=%q", out)
	}
}

func TestPushCmd_FileMissing(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{})
	defer srv.Close()
	swapWriteRepo(t, stubbedWriteRepo(srv.URL))

	_, err := runRoot(t, "push", "/nonexistent/gem.gem")
	if err == nil {
		t.Fatal("expected file-read error")
	}
}

func TestPushCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems": {200, `pushed ok`},
	})
	defer srv.Close()
	swapWriteRepo(t, stubbedWriteRepo(srv.URL))

	tmp := t.TempDir() + "/test.gem"
	if err := os.WriteFile(tmp, []byte("fake gem bytes"), 0644); err != nil {
		t.Fatalf("write tmp: %v", err)
	}

	out, err := runRoot(t, "push", tmp)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "pushed ok") {
		t.Errorf("output=%q", out)
	}
}

// ============================================================
// handleErr mapping
// ============================================================

func TestHandleErr(t *testing.T) {
	if msg := handleErr(repository.ErrNotFound).Error(); msg != "not found (404)" {
		t.Errorf("not found: %s", msg)
	}
	if msg := handleErr(repository.ErrRateLimited).Error(); msg != "rate limited (429), retry later" {
		t.Errorf("rate limited: %s", msg)
	}
	if msg := handleErr(repository.ErrUnauthorized).Error(); msg != "unauthorized (401/403), check --token" {
		t.Errorf("unauthorized: %s", msg)
	}
	other := handleErr(context.DeadlineExceeded)
	if other.Error() != context.DeadlineExceeded.Error() {
		t.Errorf("passthrough: %v", other)
	}
}

// ============================================================
// buildOptions / newRepo flag branches
// ============================================================

func TestBuildOptions_FlagsApplied(t *testing.T) {
	resetFlags()
	flagToken = "tok"
	flagProxy = "http://127.0.0.1:1"
	flagRetry = true
	opts := buildOptions()
	if opts.Token != "tok" {
		t.Errorf("token=%s", opts.Token)
	}
	if opts.Proxy != "http://127.0.0.1:1" {
		t.Errorf("proxy=%s", opts.Proxy)
	}
	if opts.RetryOptions == nil {
		t.Error("expected retry options set")
	}
}

func TestBuildOptions_EmptyFlags(t *testing.T) {
	resetFlags()
	opts := buildOptions()
	if opts.Token != "" {
		t.Errorf("token=%s", opts.Token)
	}
	// NewOptions() seeds a default RetryOptions; empty flags leave it intact.
	if opts.RetryOptions == nil {
		t.Error("expected default retry options from NewOptions")
	}
}

func TestNewRepo_MirrorBranches(t *testing.T) {
	cases := []struct {
		mirror string
		server string
	}{
		{"default", ""},
		{"ruby-china", ""},
		{"tsinghua", ""},
		{"aliyun", ""},
		{"", "https://custom.example.com"},
	}
	for _, c := range cases {
		t.Run(c.mirror+"/"+c.server, func(t *testing.T) {
			resetFlags()
			flagMirror = c.mirror
			flagServer = c.server
			if c.mirror == "" {
				flagMirror = "default"
			}
			r := newRepo()
			if r == nil {
				t.Error("newRepo returned nil")
			}
		})
	}
}

func TestNewRepo_CacheWrap(t *testing.T) {
	resetFlags()
	flagCache = true
	flagCacheTTL = 1
	r := newRepo()
	if r == nil {
		t.Error("newRepo returned nil")
	}
}

func TestNewWriteRepo_Factory(t *testing.T) {
	resetFlags()
	flagToken = "tok"
	w := newWriteRepo()
	if w == nil {
		t.Error("newWriteRepo returned nil")
	}
}

// ============================================================
// platform command (real detection on Linux CI)
// ============================================================

func TestPlatformCmd_Success(t *testing.T) {
	resetFlags()
	out, err := runRoot(t, "platform")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "linux") && !strings.Contains(out, "darwin") && !strings.Contains(out, "windows") {
		t.Errorf("output missing OS: %q", out)
	}
}

func TestPlatformCmd_JSON(t *testing.T) {
	resetFlags()
	out, err := runRoot(t, "platform", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, `"OS"`) {
		t.Errorf("expected JSON with OS field: %q", out)
	}
}

// ============================================================
// Read commands — RunE success paths (cover each subcommand)
// ============================================================

func TestAutocompleteCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/search/autocomplete.json": {200, `["rails","railties"]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "autocomplete", "rai", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "rails") {
		t.Errorf("output=%q", out)
	}
}

func TestVersionDetailCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v2/rubygems/rails/versions/7.0.0.json": {200, `{"name":"rails","number":"7.0.0"}`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "version-detail", "rails", "7.0.0", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "7.0.0") {
		t.Errorf("output=%q", out)
	}
}

func TestVersionContentsCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v2/rubygems/rails/versions/7.0.0/contents.json": {200, `{"files":{"lib/rails.rb":"sha256-abc"}}`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "version-contents", "rails", "7.0.0", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "lib/rails.rb") {
		t.Errorf("output=%q", out)
	}
}

func TestVersionDownloadsCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/downloads/rails-7.0.0.json": {200, `{"version_downloads":42,"total_downloads":100}`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "version-downloads", "rails", "7.0.0", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "42") {
		t.Errorf("output=%q", out)
	}
}

func TestTopDownloadsCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/downloads/all.json": {200, `[{"name":"rails","downloads":100}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "top-downloads", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "rails") {
		t.Errorf("output=%q", out)
	}
}

func TestTopDownloadsCmd_TextFormat(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/downloads/all.json": {200, `[{"name":"rails","downloads":100},{"name":"rack","downloads":50}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "top-downloads", "--limit", "1")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "Top") || !strings.Contains(out, "rails") {
		t.Errorf("output=%q", out)
	}
}

func TestDepsCmd_JSONFormat(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/dependencies": {200, `[{"name":"rack","version":"7.0.0","dependencies":{"rack":">= 2.2.4"}}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "deps", "rails", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "rack") {
		t.Errorf("output=%q", out)
	}
}

func TestDepsCmd_TextFormat(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/dependencies": {200, `[]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "deps", "rails")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "No dependencies") {
		t.Errorf("output=%q", out)
	}
}

func TestRdepsCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems/rails/reverse_dependencies.json": {200, `["actionmailer","actionpack"]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "rdeps", "rails", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "actionmailer") {
		t.Errorf("output=%q", out)
	}
}

func TestVersionRdepsCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/versions/rails-7.0.0/reverse_dependencies.json": {200, `["actionmailer"]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "version-rdeps", "rails-7.0.0", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "actionmailer") {
		t.Errorf("output=%q", out)
	}
}

func TestLatestGemsCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/activity/latest.json": {200, `[{"name":"newgem","version":"0.1.0"}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "latest-gems", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "newgem") {
		t.Errorf("output=%q", out)
	}
}

func TestJustUpdatedCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/activity/just_updated.json": {200, `[{"name":"upgem","version":"1.2.3"}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "just-updated", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "upgem") {
		t.Errorf("output=%q", out)
	}
}

func TestUserProfileCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/profiles/u.json": {200, `{"id":1,"handle":"u"}`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "user-profile", "u", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "u") {
		t.Errorf("output=%q", out)
	}
}

func TestOwnedGemsCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems.json": {200, `[{"name":"mygem","version":"1.0.0"}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "owned-gems", "--token", "t", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "mygem") {
		t.Errorf("output=%q", out)
	}
}

func TestGemsByOwnerCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/owners/u/gems.json": {200, `[{"name":"owngem","version":"1.0.0"}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "gems-by-owner", "u", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "owngem") {
		t.Errorf("output=%q", out)
	}
}

func TestGemOwnersCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems/rails/owners.json": {200, `[{"id":1,"handle":"dhh"}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "gem-owners", "rails", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "dhh") {
		t.Errorf("output=%q", out)
	}
}

func TestAttestationsCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/attestations/rails-7.0.0.json": {200, `[]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "attestations", "rails", "7.0.0", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "[") {
		t.Errorf("output=%q", out)
	}
}

func TestMfaStatusCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/multifactor_auth": {200, `{"enabled":true,"level":"enabled"}`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "mfa-status", "--token", "t", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "enabled") {
		t.Errorf("output=%q", out)
	}
}

func TestSearchCmd_TextFormat(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/search.json": {200, `[{"name":"rails","version":"7.0.0","downloads":100}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "search", "rails")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "Search results") || !strings.Contains(out, "rails") {
		t.Errorf("output=%q", out)
	}
}

func TestVersionsCmd_TextFormat(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/versions/rails.json": {200, `[{"number":"7.0.0","downloads_count":10,"created_at":"2024-01-01T00:00:00Z"}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "versions", "rails")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "Version list") || !strings.Contains(out, "7.0.0") {
		t.Errorf("output=%q", out)
	}
}

func TestUpdateOwnerCmd_Success(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems/rails/owners": {200, ``},
	})
	defer srv.Close()
	swapWriteRepo(t, stubbedWriteRepo(srv.URL))

	out, err := runRoot(t, "update-owner", "rails", "u", "--role", "pusher")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "Updated owner") {
		t.Errorf("output=%q", out)
	}
}

// ============================================================
// Error-path coverage for text-format branches
// ============================================================

func TestBulkVersionsCmd_TextWithFailure(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/versions/rails.json": {200, `[{"number":"7.0.0","downloads_count":1,"created_at":"2024-01-01T00:00:00Z"}]`},
		"/api/v1/versions/missing.json": {500, "boom"},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "bulk-versions", "rails,missing")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "FAILED") || !strings.Contains(out, "1 versions") {
		t.Errorf("output=%q", out)
	}
}

func TestBulkDepsCmd_TextWithFailure(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/dependencies": {200, `[{"name":"rack","dependencies":{"rack":">= 2.2.4"}}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "bulk-deps", "rails")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "1 dependencies") {
		t.Errorf("output=%q", out)
	}
}

func TestBulkRdepsCmd_TextFormat(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems/rails/reverse_dependencies.json": {200, `["actionmailer"]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "bulk-rdeps", "rails")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "1 reverse dependencies") {
		t.Errorf("output=%q", out)
	}
}

func TestBulkRdepsCmd_TextWithFailure(t *testing.T) {
	resetFlags()
	// Reverse-deps bulk returns []string per gem; a 500 on one gem produces a
	// FAILED line, covering the r.Error != nil branch.
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems/rails/reverse_dependencies.json":  {200, `["actionmailer"]`},
		"/api/v1/gems/boom/reverse_dependencies.json":   {500, "err"},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "bulk-rdeps", "rails,boom")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "FAILED") || !strings.Contains(out, "1 reverse dependencies") {
		t.Errorf("output=%q", out)
	}
}

func TestDepsCmd_TextWithRuntimeAndDev(t *testing.T) {
	resetFlags()
	// DependencyInfo uses DependentType runtime/development to split sections.
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/dependencies": {200, `[{"name":"rack","dependencies":{"rack":">= 2.2.4"},"dependent_type":"runtime"},{"name":"rspec","dependencies":{"rspec":">= 3.0"},"dependent_type":"development"}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "deps", "rails")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "Runtime dependencies") || !strings.Contains(out, "Development dependencies") {
		t.Errorf("output=%q", out)
	}
}

func TestRdepsCmd_TextFormat(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems/rails/reverse_dependencies.json": {200, `["actionmailer","actionpack"]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "rdeps", "rails")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "Packages depending") || !strings.Contains(out, "actionmailer") {
		t.Errorf("output=%q", out)
	}
}

func TestAutocompleteCmd_TextFormat(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/search/autocomplete.json": {200, `["rails","railties"]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "autocomplete", "rai")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "rails") {
		t.Errorf("output=%q", out)
	}
}

func TestSearchCmd_Limit(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/search.json": {200, `[{"name":"rails","version":"7.0.0","downloads":100},{"name":"railties","version":"7.0.0","downloads":50}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "search", "rai", "--limit", "1")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "rails") || strings.Contains(out, "railties") {
		t.Errorf("output=%q", out)
	}
}

func TestVersionsCmd_Limit(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/versions/rails.json": {200, `[{"number":"7.0.0","downloads_count":10,"created_at":"2024-01-01T00:00:00Z"},{"number":"6.1.0","downloads_count":5,"created_at":"2023-01-01T00:00:00Z"}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "versions", "rails", "--limit", "1")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "7.0.0") || strings.Contains(out, "6.1.0") {
		t.Errorf("output=%q", out)
	}
}

func TestTopDownloadsCmd_Limit(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/downloads/all.json": {200, `[{"name":"rails","downloads":100},{"name":"rack","downloads":50}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "top-downloads", "--limit", "1")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "rails") || strings.Contains(out, "rack") {
		t.Errorf("output=%q", out)
	}
}

func TestRdepsCmd_Limit(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/gems/rails/reverse_dependencies.json": {200, `["actionmailer","actionpack"]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "rdeps", "rails", "--limit", "1")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "actionmailer") || strings.Contains(out, "actionpack") {
		t.Errorf("output=%q", out)
	}
}

func TestLatestGemsCmd_Limit(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/activity/latest.json": {200, `[{"name":"a","version":"1.0.0"},{"name":"b","version":"2.0.0"}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "latest-gems", "--limit", "1", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "\"a\"") || strings.Contains(out, "\"b\"") {
		t.Errorf("output=%q", out)
	}
}

func TestJustUpdatedCmd_Limit(t *testing.T) {
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{
		"/api/v1/activity/just_updated.json": {200, `[{"name":"a","version":"1.0.0"},{"name":"b","version":"2.0.0"}]`},
	})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))

	out, err := runRoot(t, "just-updated", "--limit", "1", "--json")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(out, "\"a\"") || strings.Contains(out, "\"b\"") {
		t.Errorf("output=%q", out)
	}
}

func TestTimeframeCmd_InvalidFrom(t *testing.T) {
	resetFlags()
	_, err := runRoot(t, "timeframe", "--from", "bad", "--to", "2024-12-31T00:00:00Z")
	if err == nil || !strings.Contains(err.Error(), "invalid --from") {
		t.Errorf("err=%v", err)
	}
}

func TestTimeframeCmd_InvalidTo(t *testing.T) {
	resetFlags()
	_, err := runRoot(t, "timeframe", "--from", "2024-01-01T00:00:00Z", "--to", "bad")
	if err == nil || !strings.Contains(err.Error(), "invalid --to") {
		t.Errorf("err=%v", err)
	}
}

func TestTimeframeCmd_MissingFlags(t *testing.T) {
	resetFlags()
	_, err := runRoot(t, "timeframe")
	if err == nil {
		t.Fatal("expected required flags error")
	}
}

func TestPrintJSON_Error(t *testing.T) {
	resetFlags()
	// A channel cannot be JSON-marshaled, forcing the error branch of printJSON.
	out, _ := runRootWithJSONBadValue(t)
	if !strings.Contains(out, "JSON serialization failed") {
		t.Errorf("output=%q", out)
	}
}

// runRootWithJSONBadValue triggers printJSON's marshal-error branch by feeding
// it a value that encoding/json cannot serialize (a channel) via prettyPrint's
// default case (which just prints %+v, NOT json). To actually exercise printJSON
// failure we call printJSON directly.
func runRootWithJSONBadValue(t *testing.T) (string, error) {
	t.Helper()
	return captureStdout(t, func() error {
		printJSON(make(chan int))
		return nil
	})
}

func TestMarkRequired_PanicsOnUnknownFlag(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unknown flag")
		}
	}()
	c := &cobra.Command{}
	c.Flags().String("known", "", "")
	// "nonexistent" isn't registered -> MarkFlagRequired errors -> panic.
	markRequired(c, "nonexistent")
}

// ============================================================
// Error-path (404) coverage for remaining read commands
// ============================================================

func runNotFound(t *testing.T, route string, args ...string) {
	t.Helper()
	resetFlags()
	srv := stubServer(map[string]struct {
		status int
		body   string
	}{route: {404, "not found"}})
	defer srv.Close()
	swapRepo(t, stubbedRepo(srv.URL))
	_, err := runRoot(t, args...)
	if err == nil || !strings.Contains(err.Error(), "not found (404)") {
		t.Fatalf("expected 404 err, got: %v", err)
	}
}

func TestAutocompleteCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v1/search/autocomplete.json", "autocomplete", "x", "--json")
}
func TestVersionDetailCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v2/rubygems/rails/versions/9.9.9.json", "version-detail", "rails", "9.9.9", "--json")
}
func TestVersionContentsCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v2/rubygems/rails/versions/9.9.9/contents.json", "version-contents", "rails", "9.9.9", "--json")
}
func TestVersionDownloadsCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v1/downloads/rails-9.9.9.json", "version-downloads", "rails", "9.9.9", "--json")
}
func TestTopDownloadsCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v1/downloads/all.json", "top-downloads", "--json")
}
func TestDepsCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v1/dependencies", "deps", "nope", "--json")
}
func TestRdepsCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v1/gems/nope/reverse_dependencies.json", "rdeps", "nope", "--json")
}
func TestVersionRdepsCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v1/versions/nope-1.0.0/reverse_dependencies.json", "version-rdeps", "nope-1.0.0", "--json")
}
func TestLatestGemsCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v1/activity/latest.json", "latest-gems", "--json")
}
func TestJustUpdatedCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v1/activity/just_updated.json", "just-updated", "--json")
}
func TestUserProfileCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v1/profiles/nope.json", "user-profile", "nope", "--json")
}
func TestOwnedGemsCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v1/gems.json", "owned-gems", "--token", "t", "--json")
}
func TestGemsByOwnerCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v1/owners/nope/gems.json", "gems-by-owner", "nope", "--json")
}
func TestGemOwnersCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v1/gems/nope/owners.json", "gem-owners", "nope", "--json")
}
func TestAttestationsCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v1/attestations/nope-1.0.0.json", "attestations", "nope", "1.0.0", "--json")
}
func TestMfaStatusCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v1/multifactor_auth", "mfa-status", "--token", "t", "--json")
}
func TestVersionsCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v1/versions/nope.json", "versions", "nope", "--json")
}
func TestLatestVersionCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v1/versions/nope/latest.json", "latest-version", "nope", "--json")
}
func TestDownloadsCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v1/downloads.json", "downloads", "--json")
}
func TestSearchCmd_NotFound(t *testing.T) {
	runNotFound(t, "/api/v1/search.json", "search", "nope", "--json")
}
