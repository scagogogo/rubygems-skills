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
	cmd := exec.Command("go", "build", "-o", "rubygems-cli", "../../cmd/rubygems/main.go")
	err := cmd.Run()
	if err != nil {
		t.Fatalf("compile CLI failed: %v", err)
	}

	// Test help info
	t.Run("show help info", func(t *testing.T) {
		output, err := exec.Command("./rubygems-cli", "-help").CombinedOutput()
		assert.NoError(t, err, "execute help command failed")
		assert.Contains(t, string(output), "get package info", "help output should contain feature description")
		assert.Contains(t, string(output), "search packages", "help output should contain feature description")
	})

	// Test get package info
	t.Run("get package info", func(t *testing.T) {
		output, err := exec.Command("./rubygems-cli", "-get", "-gem", "rails").CombinedOutput()
		assert.NoError(t, err, "get package info failed")
		assert.Contains(t, string(output), "rails", "output should contain package name")
	})

	// Test search function
	t.Run("search function", func(t *testing.T) {
		output, err := exec.Command("./rubygems-cli", "-search", "-query", "rails", "-limit", "5").CombinedOutput()
		assert.NoError(t, err, "search packages failed")
		assert.Contains(t, string(output), "rails", "search result should contain rails")
	})

	// Test get version info
	t.Run("get version info", func(t *testing.T) {
		output, err := exec.Command("./rubygems-cli", "-versions", "-gem", "rails", "-limit", "5").CombinedOutput()
		assert.NoError(t, err, "get version info failed")
		assert.Contains(t, string(output), "rails", "version info should contain package name")
	})

	// Test get dependency info
	t.Run("get dependency info", func(t *testing.T) {
		output, err := exec.Command("./rubygems-cli", "-deps", "-gem", "rails").CombinedOutput()
		assert.NoError(t, err, "get dependency info failed")
		assert.NotEmpty(t, output, "dependency info should not be empty")
	})

	// Test get reverse dependency info
	t.Run("get reverse dependency info", func(t *testing.T) {
		output, err := exec.Command("./rubygems-cli", "-rdeps", "-gem", "rack", "-limit", "5").CombinedOutput()
		assert.NoError(t, err, "get reverse dependency info failed")
		assert.NotEmpty(t, output, "reverse dependency info should not be empty")
	})

	// Test JSON output
	t.Run("JSON output", func(t *testing.T) {
		output, err := exec.Command("./rubygems-cli", "-get", "-gem", "rails", "-json").CombinedOutput()
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
		_, err := exec.Command("./rubygems-cli", "-get", "-gem", "rails").CombinedOutput()
		assert.NoError(t, err, "first get package info failed")
		firstDuration := time.Since(start)

		// Use cache to get again
		start = time.Now()
		_, err = exec.Command("./rubygems-cli", "-get", "-gem", "rails", "-cache").CombinedOutput()
		assert.NoError(t, err, "get package info using cache failed")
		secondDuration := time.Since(start)

		// Cache should be faster
		t.Logf("no cache duration: %v, using cache duration: %v", firstDuration, secondDuration)
	})

	// Test mirror selection
	t.Run("mirror selection", func(t *testing.T) {
		mirrors := []string{"default", "ruby-china", "tsinghua", "aliyun"}

		for _, mirror := range mirrors {
			t.Run(mirror, func(t *testing.T) {
				output, err := exec.Command("./rubygems-cli", "-get", "-gem", "rake", "-mirror", mirror).CombinedOutput()
				assert.NoError(t, err, "get package info using mirror %s failed", mirror)
				assert.Contains(t, string(output), "rake", "output using mirror %s should contain package name", mirror)
			})
		}
	})

	// Test invalid command
	t.Run("invalid command", func(t *testing.T) {
		cmd := exec.Command("./rubygems-cli", "-invalid", "-gem", "rails")
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
