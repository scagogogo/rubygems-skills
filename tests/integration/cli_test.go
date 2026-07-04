package integration

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// CLI integration test requires building the binary, so here we use calling external commands
func TestCLI(t *testing.T) {
	// Skip long running test
	if testing.Short() {
		t.Skip("skip CLI test in short mode")
	}

	// Try to get the compiled binary file path
	cmd := exec.Command("go", "build", "-o", "rubygems-cli", "../../cmd/rubygems/")
	err := cmd.Run()
	if err != nil {
		t.Fatalf("compile CLI failed: %v", err)
	}

	// Test help info
	t.Run("show help info", func(t *testing.T) {
		output, err := exec.Command("./rubygems-cli", "--help").CombinedOutput()
		assert.NoError(t, err, "execute help command failed")
		assert.Contains(t, string(output), "Get detailed package info", "help output should contain feature description")
		assert.Contains(t, string(output), "Search packages", "help output should contain feature description")
	})

	// Test get package info
	t.Run("get package info", func(t *testing.T) {
		output, err := exec.Command("./rubygems-cli", "get", "rails").CombinedOutput()
		assert.NoError(t, err, "get package info failed")
		assert.Contains(t, string(output), "rails", "output should contain package name")
	})

	// Test search function
	t.Run("search function", func(t *testing.T) {
		output, err := exec.Command("./rubygems-cli", "search", "rails", "--limit", "5").CombinedOutput()
		assert.NoError(t, err, "search packages failed")
		assert.Contains(t, string(output), "rails", "search result should contain rails")
	})

	// Test get version info
	t.Run("get version info", func(t *testing.T) {
		output, err := exec.Command("./rubygems-cli", "versions", "rails", "--limit", "5").CombinedOutput()
		assert.NoError(t, err, "get version info failed")
		assert.Contains(t, string(output), "rails", "version info should contain package name")
	})

	// Test get dependency info.
	// NOTE: the /api/v1/dependencies endpoint was deprecated and shut down by
	// RubyGems.org on 2023-02-22, so `deps` now returns a 404. We assert the
	// command runs and produces output (the error message), rather than the
	// deprecated dependency data.
	t.Run("get dependency info", func(t *testing.T) {
		output, _ := exec.Command("./rubygems-cli", "deps", "rails").CombinedOutput()
		assert.NotEmpty(t, output, "dependency command should produce output")
	})

	// Test version-detail (API v2) — the supported way to get dependency info.
	t.Run("version detail", func(t *testing.T) {
		output, err := exec.Command("./rubygems-cli", "version-detail", "rails", "8.1.3", "--json").CombinedOutput()
		assert.NoError(t, err, "get version detail failed")
		var result map[string]interface{}
		assert.NoError(t, json.Unmarshal(output, &result), "parse version-detail JSON failed")
		assert.Equal(t, "8.1.3", result["number"], "version detail should contain version number")
	})

	// Test get reverse dependency info
	t.Run("get reverse dependency info", func(t *testing.T) {
		output, err := exec.Command("./rubygems-cli", "rdeps", "rack", "--limit", "5").CombinedOutput()
		assert.NoError(t, err, "get reverse dependency info failed")
		assert.NotEmpty(t, output, "reverse dependency info should not be empty")
	})

	// Test JSON output
	t.Run("JSON output", func(t *testing.T) {
		output, err := exec.Command("./rubygems-cli", "get", "rails", "--json").CombinedOutput()
		assert.NoError(t, err, "get package info in JSON format failed")

		// Try to parse JSON
		var result map[string]interface{}
		err = json.Unmarshal(output, &result)
		assert.NoError(t, err, "parse JSON output failed")
		assert.Equal(t, "rails", result["name"], "JSON should contain correct package name")
	})

	// Test using cache
	t.Run("use cache", func(t *testing.T) {
		// First get
		start := time.Now()
		_, err := exec.Command("./rubygems-cli", "get", "rails").CombinedOutput()
		assert.NoError(t, err, "first get package info failed")
		firstDuration := time.Since(start)

		// Use cache to get again
		start = time.Now()
		_, err = exec.Command("./rubygems-cli", "get", "rails", "--cache").CombinedOutput()
		assert.NoError(t, err, "get package info using cache failed")
		secondDuration := time.Since(start)

		// Cache should be faster
		t.Logf("no cache duration: %v, using cache duration: %v", firstDuration, secondDuration)
	})

	// Test mirror selection.
	// Only default (official) and ruby-china provide the API; Tsinghua and
	// Alibaba Cloud mirrors only serve gem files and return 404 for API calls.
	t.Run("mirror selection", func(t *testing.T) {
		mirrors := []string{"default", "ruby-china"}

		for _, mirror := range mirrors {
			t.Run(mirror, func(t *testing.T) {
				output, err := exec.Command("./rubygems-cli", "get", "rake", "--mirror", mirror).CombinedOutput()
				assert.NoError(t, err, "get package info using mirror %s failed", mirror)
				assert.Contains(t, string(output), "rake", "output using mirror %s should contain package name", mirror)
			})
		}
	})

	// Test invalid command
	t.Run("invalid command", func(t *testing.T) {
		cmd := exec.Command("./rubygems-cli", "this-subcommand-does-not-exist")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		err := cmd.Run()

		// Command should exit with an error message
		assert.Error(t, err, "invalid command should return error")
	})

	// Test cleanup
	defer func() {
		err := exec.Command("rm", "-f", "rubygems-cli").Run()
		if err != nil {
			t.Logf("cleanup file failed: %v", err)
		}
	}()
}
