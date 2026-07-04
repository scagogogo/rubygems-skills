package install

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Docker cross-platform integration tests
// These tests require a Docker environment, enabled with the -docker flag

var dockerEnabled = os.Getenv("TEST_DOCKER") == "1"

func dockerAvailable() bool {
	if !dockerEnabled {
		return false
	}
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}

// DockerTestConfig defines the Docker test configuration
type DockerTestConfig struct {
	Image       string
	Distro      LinuxDistro
	PackageMgr  PackageManager
	InstallCmd  string // command that should exist after verifying installation
	SetupCmds   []string // additional container setup commands
}

// getDockerTestConfigs returns all Docker test configurations
func getDockerTestConfigs() []DockerTestConfig {
	return []DockerTestConfig{
		{
			Image:      "ubuntu:22.04",
			Distro:     DistroUbuntu,
			PackageMgr: PMApt,
			InstallCmd: "ruby --version",
			SetupCmds:  []string{"apt-get update"},
		},
		{
			Image:      "debian:12",
			Distro:     DistroDebian,
			PackageMgr: PMApt,
			InstallCmd: "ruby --version",
			SetupCmds:  []string{"apt-get update"},
		},
		{
			Image:      "alpine:3.18",
			Distro:     DistroAlpine,
			PackageMgr: PMApk,
			InstallCmd: "ruby --version",
			SetupCmds:  []string{"apk update"},
		},
		{
			Image:      "fedora:39",
			Distro:     DistroFedora,
			PackageMgr: PMDnf,
			InstallCmd: "ruby --version",
			SetupCmds:  []string{},
		},
		{
			Image:      "rockylinux:9",
			Distro:     DistroRocky,
			PackageMgr: PMDnf,
			InstallCmd: "ruby --version",
			SetupCmds:  []string{},
		},
	}
}

// TestDockerPlatformDetection tests platform detection in a Docker container
func TestDockerPlatformDetection(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker tests not enabled, set TEST_DOCKER=1 to enable")
	}

	configs := getDockerTestConfigs()

	for _, cfg := range configs {
		t.Run(string(cfg.Distro), func(t *testing.T) {
			testDockerPlatformDetection(t, cfg)
		})
	}
}

// TestDockerRubyInstallation tests Ruby installation in a Docker container
func TestDockerRubyInstallation(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker tests not enabled, set TEST_DOCKER=1 to enable")
	}

	configs := getDockerTestConfigs()

	for _, cfg := range configs {
		t.Run(string(cfg.Distro), func(t *testing.T) {
			testDockerRubyInstallation(t, cfg)
		})
	}
}

// testDockerPlatformDetection tests platform detection in a Docker container
func testDockerPlatformDetection(t *testing.T, cfg DockerTestConfig) {
	// Run a simple Go program in the container to detect the platform
	// But since we need to compile, first use a shell script to detect

	// 1. Detect /etc/os-release
	script := `
cat /etc/os-release 2>/dev/null || echo "NO_OS_RELEASE"
if [ -f /etc/debian_version ]; then echo "DEBIAN"; fi
if [ -f /etc/redhat-release ]; then echo "REDHAT"; cat /etc/redhat-release; fi
if [ -f /etc/alpine-release ]; then echo "ALPINE"; cat /etc/alpine-release; fi
if [ -f /etc/arch-release ]; then echo "ARCH"; fi
which apt-get 2>/dev/null && echo "HAS_APT_GET"
which apt 2>/dev/null && echo "HAS_APT"
which yum 2>/dev/null && echo "HAS_YUM"
which dnf 2>/dev/null && echo "HAS_DNF"
which apk 2>/dev/null && echo "HAS_APK"
which pacman 2>/dev/null && echo "HAS_PACMAN"
which zypper 2>/dev/null && echo "HAS_ZYPPER"
`

	output, err := runDockerCommand(cfg.Image, script, 60)
	if err != nil {
		t.Fatalf("Docker command execution failed: %v\nOutput: %s", err, output)
	}

	t.Logf("=== %s (%s) ===", cfg.Distro, cfg.Image)
	t.Logf("%s", output)

	// Verify that the expected package manager exists
	switch cfg.PackageMgr {
	case PMApt:
		if !strings.Contains(output, "HAS_APT_GET") && !strings.Contains(output, "HAS_APT") {
			t.Errorf("expected to find apt/apt-get, but not found")
		}
	case PMYum:
		if !strings.Contains(output, "HAS_YUM") {
			t.Errorf("expected to find yum, but not found")
		}
	case PMDnf:
		if !strings.Contains(output, "HAS_DNF") {
			t.Errorf("expected to find dnf, but not found")
		}
	case PMApk:
		if !strings.Contains(output, "HAS_APK") {
			t.Errorf("expected to find apk, but not found")
		}
	case PMPacman:
		if !strings.Contains(output, "HAS_PACMAN") {
			t.Errorf("expected to find pacman, but not found")
		}
	case PMZypper:
		if !strings.Contains(output, "HAS_ZYPPER") {
			t.Errorf("expected to find zypper, but not found")
		}
	}
}

// testDockerRubyInstallation actually installs Ruby in a Docker container
func testDockerRubyInstallation(t *testing.T, cfg DockerTestConfig) {
	// Build the install command
	var installScript string

	switch cfg.PackageMgr {
	case PMApt:
		installScript = `
apt-get update -qq
apt-get install -y -qq ruby ruby-dev 2>&1
`
	case PMYum:
		installScript = `
yum install -y ruby ruby-devel 2>&1
`
	case PMDnf:
		installScript = `
dnf install -y ruby ruby-devel 2>&1
`
	case PMApk:
		installScript = `
apk update
apk add ruby ruby-dev 2>&1
`
	case PMPacman:
		installScript = `
pacman -Sy --noconfirm ruby 2>&1
`
	case PMZypper:
		installScript = `
zypper install -y ruby ruby-devel 2>&1
`
	default:
		t.Fatalf("unsupported package manager: %s", cfg.PackageMgr)
	}

	// Add verification commands
	verifyScript := installScript + `
echo "=== VERIFICATION ==="
command -v ruby && ruby --version || echo "RUBY_NOT_FOUND"
command -v gem && gem --version || echo "GEM_NOT_FOUND"
`

	output, err := runDockerCommand(cfg.Image, verifyScript, 600)
	if err != nil {
		t.Fatalf("Docker installation test failed: %v\nOutput: %s", err, output)
	}

	t.Logf("=== %s (%s) installation result ===", cfg.Distro, cfg.Image)
	t.Logf("%s", output)

	// Verify the installation succeeded
	if strings.Contains(output, "RUBY_NOT_FOUND") {
		t.Errorf("Ruby installation failed: ruby command not found")
	}
	if strings.Contains(output, "GEM_NOT_FOUND") {
		t.Errorf("gem installation failed: gem command not found")
	}

	// Extract and display version info
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ruby ") {
			t.Logf("  Ruby version: %s", line)
		}
		if isVersionString(line) && len(line) < 20 {
			// Possibly a gem version number
			t.Logf("  Version number: %s", line)
		}
	}
}

// runDockerCommand executes a script in a Docker container
func runDockerCommand(image, script string, timeoutSec int) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// Use docker run --rm to execute the command
	args := []string{
		"run", "--rm",
		"--network", "host",
		image,
		"sh", "-c", script,
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return string(output), fmt.Errorf("Docker command timed out (%d seconds)", timeoutSec)
	}

	return string(output), err
}

// TestDockerOSReleaseParsing tests os-release parsing in a Docker container
func TestDockerOSReleaseParsing(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker tests not enabled, set TEST_DOCKER=1 to enable")
	}

	tests := []struct {
		image      string
		expectedID string
	}{
		{"ubuntu:22.04", "ubuntu"},
		{"debian:12", "debian"},
		{"alpine:3.18", "alpine"},
		{"fedora:39", "fedora"},
		{"rockylinux:9", "rocky"},
	}

	for _, tt := range tests {
		t.Run(tt.image, func(t *testing.T) {
			output, err := runDockerCommand(tt.image, "cat /etc/os-release", 30)
			if err != nil {
				t.Fatalf("failed to read /etc/os-release: %v", err)
			}

			id := parseOSReleaseField(output, "ID")
			t.Logf("%s: ID=%s", tt.image, id)

			if id != tt.expectedID {
				t.Errorf("ID = %q, want %q", id, tt.expectedID)
			}
		})
	}
}

// TestDockerBundlerInstallation tests installing bundler in Docker
func TestDockerBundlerInstallation(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker tests not enabled, set TEST_DOCKER=1 to enable")
	}

	// Only test Ubuntu, because bundler installation requires network
	script := `
apt-get update -qq
apt-get install -y -qq ruby ruby-dev 2>&1
echo "=== RUBY INSTALLED ==="
gem install bundler 2>&1
echo "=== BUNDLER INSTALLED ==="
which bundler && bundler --version || echo "BUNDLER_NOT_FOUND"
`

	output, err := runDockerCommand("ubuntu:22.04", script, 300)
	if err != nil {
		t.Fatalf("Docker bundler installation test failed: %v\nOutput: %s", err, output)
	}

	t.Logf("Ubuntu bundler installation result:\n%s", output)

	if strings.Contains(output, "BUNDLER_NOT_FOUND") {
		t.Error("bundler installation failed")
	}
}
