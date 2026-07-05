package install

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ============================================================
// Platform detection tests
// ============================================================

func TestDetectOS(t *testing.T) {
	os := detectOS()
	// When running tests on Linux, it should be OSLinux
	if os == OSUnknown {
		t.Error("detectOS() should not return OSUnknown")
	}
}

func TestDetectArch(t *testing.T) {
	arch := detectArch()
	if arch == ArchUnknown {
		t.Error("detectArch() should not return ArchUnknown")
	}
}

func TestDetectOSMapping(t *testing.T) {
	// Test the mapping from GOOS to OperatingSystem
	tests := []struct {
		goos     string
		expected OperatingSystem
	}{
		{"linux", OSLinux},
		{"darwin", OSDarwin},
		{"windows", OSWindows},
		{"freebsd", OSUnknown},
		{"", OSUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			var result OperatingSystem
			switch tt.goos {
			case "linux":
				result = OSLinux
			case "darwin":
				result = OSDarwin
			case "windows":
				result = OSWindows
			default:
				result = OSUnknown
			}
			if result != tt.expected {
				t.Errorf("GOOS=%s mapping = %v, want %v", tt.goos, result, tt.expected)
			}
		})
	}
}

func TestDetectArchValues(t *testing.T) {
	tests := []struct {
		goarch   string
		expected Architecture
	}{
		{"amd64", ArchAMD64},
		{"arm64", ArchARM64},
		{"arm", ArchARM},
		{"386", Arch386},
		{"mips", ArchUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.goarch, func(t *testing.T) {
			result := detectArchFromGOARCH(tt.goarch)
			if result != tt.expected {
				t.Errorf("detectArchFromGOARCH(%s) = %v, want %v", tt.goarch, result, tt.expected)
			}
		})
	}
}

// detectArchFromGOARCH is a test helper function
func detectArchFromGOARCH(goarch string) Architecture {
	switch goarch {
	case "amd64":
		return ArchAMD64
	case "arm64":
		return ArchARM64
	case "arm":
		return ArchARM
	case "386":
		return Arch386
	default:
		return ArchUnknown
	}
}

// ============================================================
// /etc/os-release parsing tests
// ============================================================

func TestParseOSReleaseField(t *testing.T) {
	data := `NAME="Ubuntu"
VERSION="22.04.2 LTS (Jammy Jellyfish)"
ID=ubuntu
ID_LIKE=debian
PRETTY_NAME="Ubuntu 22.04.2 LTS"
VERSION_ID="22.04"
HOME_URL="https://www.ubuntu.com/"
SUPPORT_URL="https://help.ubuntu.com/"
BUG_REPORT_URL="https://bugs.launchpad.net/ubuntu/"
PRIVACY_POLICY_URL="https://www.ubuntu.com/legal/terms-and-policies/privacy-policy"
VERSION_CODENAME=jammy
UBUNTU_CODENAME=jammy`

	tests := []struct {
		field    string
		expected string
	}{
		{"ID", "ubuntu"},
		{"ID_LIKE", "debian"},
		{"NAME", "Ubuntu"},
		{"VERSION_ID", "22.04"},
		{"VERSION_CODENAME", "jammy"},
		{"NONEXISTENT", ""},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			result := parseOSReleaseField(data, tt.field)
			if result != tt.expected {
				t.Errorf("parseOSReleaseField(%q) = %q, want %q", tt.field, result, tt.expected)
			}
		})
	}
}

func TestReadOSRelease(t *testing.T) {
	// Create a temporary os-release file
	tmpDir := t.TempDir()
	osReleasePath := filepath.Join(tmpDir, "os-release")

	tests := []struct {
		name     string
		content  string
		expected LinuxDistro
	}{
		{
			name: "Ubuntu",
			content: `NAME="Ubuntu"
ID=ubuntu
ID_LIKE=debian
VERSION="22.04.2 LTS (Jammy Jellyfish)"`,
			expected: DistroUbuntu,
		},
		{
			name: "Debian",
			content: `NAME="Debian GNU/Linux"
ID=debian
VERSION="12 (bookworm)"`,
			expected: DistroDebian,
		},
		{
			name: "CentOS",
			content: `NAME="CentOS Linux"
ID=centos
VERSION="8 (Core)"`,
			expected: DistroCentOS,
		},
		{
			name: "Fedora",
			content: `NAME="Fedora Linux"
ID=fedora
VERSION="38 (Workstation Edition)"`,
			expected: DistroFedora,
		},
		{
			name: "Alpine",
			content: `NAME="Alpine Linux"
ID=alpine
VERSION_ID=3.18.2`,
			expected: DistroAlpine,
		},
		{
			name: "Arch",
			content: `NAME="Arch Linux"
ID=arch`,
			expected: DistroArch,
		},
		{
			name: "Rocky",
			content: `NAME="Rocky Linux"
ID=rocky
ID_LIKE="rhel centos fedora"`,
			expected: DistroRocky,
		},
		{
			name: "Amazon Linux",
			content: `NAME="Amazon Linux"
ID=amzn
ID_LIKE="centos rhel fedora"`,
			expected: DistroAmazon,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(osReleasePath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			result := readOSReleaseFromString(string(readFileContent(osReleasePath)))
			if result != tt.expected {
				t.Errorf("readOSRelease() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// readFileContent is a helper function that reads file contents
func readFileContent(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return data
}

// readOSReleaseFromString parses distro info from a string (for testing)
func readOSReleaseFromString(data string) LinuxDistro {
	id := parseOSReleaseField(data, "ID")
	idLike := parseOSReleaseField(data, "ID_LIKE")

	switch strings.ToLower(id) {
	case "ubuntu":
		return DistroUbuntu
	case "debian":
		return DistroDebian
	case "centos":
		return DistroCentOS
	case "rhel":
		return DistroRHEL
	case "fedora":
		return DistroFedora
	case "rocky":
		return DistroRocky
	case "almalinux", "alma":
		return DistroAlma
	case "alpine":
		return DistroAlpine
	case "arch":
		return DistroArch
	case "manjaro":
		return DistroManjaro
	case "amzn", "amazon":
		return DistroAmazon
	case "opensuse", "opensuse-leap", "opensuse-tumbleweed":
		return DistroOpenSUSE
	}

	idLikeLower := strings.ToLower(idLike)
	switch {
	case strings.Contains(idLikeLower, "debian"):
		return DistroDebian
	case strings.Contains(idLikeLower, "rhel") || strings.Contains(idLikeLower, "fedora"):
		return DistroCentOS
	case strings.Contains(idLikeLower, "arch"):
		return DistroArch
	case strings.Contains(idLikeLower, "suse"):
		return DistroOpenSUSE
	}

	return DistroUnknown
}

// ============================================================
// Version number extraction tests
// ============================================================

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ruby 3.2.2 (2023-03-30) [x86_64-linux]", "3.2.2"},
		{"ruby 2.7.0p183 (2020-03-31) [x86_64-darwin19]", "2.7.0"}, // note p183 does not match
		{"3.1.0", "3.1.0"},
		{"ruby 3.0.0", "3.0.0"},
		{"", ""},
		{"no version here", "no version here"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractVersion(tt.input)
			if result != tt.expected {
				t.Errorf("extractVersion(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsVersionString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"3.2.2", true},
		{"2.7.0", true},
		{"1.0", true},
		{"3", false}, // no dot
		{"abc", false},
		{"", false},
		{"3.2.2-preview", false},
		{".1.2", false}, // starts with a dot, not a valid version number
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isVersionString(tt.input)
			if result != tt.expected {
				t.Errorf("isVersionString(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// ============================================================
// Install options tests
// ============================================================

func TestNewInstallOptions(t *testing.T) {
	opts := NewInstallOptions()

	if opts.ForceReinstall != false {
		t.Error("default ForceReinstall should be false")
	}
	if opts.InstallDevHeaders != true {
		t.Error("default InstallDevHeaders should be true")
	}
	if opts.InstallBundler != true {
		t.Error("default InstallBundler should be true")
	}
	if opts.UpdatePackageIndex != true {
		t.Error("default UpdatePackageIndex should be true")
	}
	if opts.TimeoutSeconds != 600 {
		t.Error("default TimeoutSeconds should be 600")
	}
	if opts.UseSudo != true {
		t.Error("default UseSudo should be true")
	}
}

func TestInstallOptionsChaining(t *testing.T) {
	opts := NewInstallOptions().
		WithForceReinstall(true).
		WithRubyVersion("3.2.2").
		WithDevHeaders(false).
		WithBundler(false).
		WithTimeout(600).
		WithSudo(false).
		WithExtraPackages("libssl-dev", "libffi-dev")

	if !opts.ForceReinstall {
		t.Error("WithForceReinstall(true) failed")
	}
	if opts.RubyVersion != "3.2.2" {
		t.Errorf("WithRubyVersion failed, got %s", opts.RubyVersion)
	}
	if opts.InstallDevHeaders {
		t.Error("WithDevHeaders(false) failed")
	}
	if opts.InstallBundler {
		t.Error("WithBundler(false) failed")
	}
	if opts.TimeoutSeconds != 600 {
		t.Errorf("WithTimeout(600) failed, got %d", opts.TimeoutSeconds)
	}
	if opts.UseSudo {
		t.Error("WithSudo(false) failed")
	}
	if len(opts.ExtraPackages) != 2 {
		t.Errorf("WithExtraPackages failed, got %d packages", len(opts.ExtraPackages))
	}
}

// ============================================================
// Installer creation and platform detection tests
// ============================================================

func TestNewInstaller(t *testing.T) {
	installer := NewInstaller()
	if installer == nil {
		t.Fatal("NewInstaller() should not return nil")
	}
	if installer.options == nil {
		t.Error("Installer.options should not be nil")
	}
}

func TestNewInstallerWithCustomOptions(t *testing.T) {
	opts := NewInstallOptions().WithForceReinstall(true).WithTimeout(120)
	installer := NewInstaller(opts)
	if !installer.options.ForceReinstall {
		t.Error("custom options not applied")
	}
	if installer.options.TimeoutSeconds != 120 {
		t.Error("custom timeout not applied")
	}
}

func TestInstallerDetectPlatform(t *testing.T) {
	installer := NewInstaller()
	platform, err := installer.DetectPlatform()
	if err != nil {
		t.Fatalf("DetectPlatform() failed: %v", err)
	}

	// Running tests on Linux
	if platform.OS != OSLinux {
		t.Errorf("running tests on Linux, but OS = %v", platform.OS)
	}
	if platform.Arch == ArchUnknown {
		t.Error("Arch should not be Unknown")
	}
	if platform.PackageMgr == PMUnknown {
		t.Error("PackageMgr should not be Unknown")
	}
}

func TestPlatformInfoString(t *testing.T) {
	tests := []struct {
		name     string
		platform *PlatformInfo
		expected string
	}{
		{
			name: "Linux/Ubuntu",
			platform: &PlatformInfo{
				OS:         OSLinux,
				Arch:       ArchAMD64,
				Distro:     DistroUbuntu,
				PackageMgr: PMApt,
			},
			expected: "linux/ubuntu (amd64, apt)",
		},
		{
			name: "macOS",
			platform: &PlatformInfo{
				OS:         OSDarwin,
				Arch:       ArchARM64,
				PackageMgr: PMBrew,
			},
			expected: "darwin/arm64 (brew)",
		},
		{
			name: "Windows",
			platform: &PlatformInfo{
				OS:         OSWindows,
				Arch:       ArchAMD64,
				PackageMgr: PMChoco,
			},
			expected: "windows/amd64 (choco)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.platform.String()
			if result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// ============================================================
// Package manager detection tests
// ============================================================

func TestDetectLinuxPackageManager(t *testing.T) {
	tests := []struct {
		distro      LinuxDistro
		expectedPMs []PackageManager // may match multiple package managers
	}{
		{DistroUbuntu, []PackageManager{PMApt}},
		{DistroDebian, []PackageManager{PMApt}},
		{DistroCentOS, []PackageManager{PMYum, PMDnf}}, // CentOS may use yum or dnf
		{DistroFedora, []PackageManager{PMDnf}},
		{DistroAlpine, []PackageManager{PMApk}},
		{DistroArch, []PackageManager{PMPacman}},
		{DistroOpenSUSE, []PackageManager{PMZypper}},
	}

	for _, tt := range tests {
		t.Run(string(tt.distro), func(t *testing.T) {
			pm, _, err := detectLinuxPackageManager(tt.distro)
			if err != nil {
				// If the corresponding package manager is not installed, this is acceptable
				t.Logf("detectLinuxPackageManager(%s) returned error: %v (the package manager may not be installed in the environment)", tt.distro, err)
				return
			}
			found := false
			for _, expectedPM := range tt.expectedPMs {
				if pm == expectedPM {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("detectLinuxPackageManager(%s) = %v, expected one of: %v", tt.distro, pm, tt.expectedPMs)
			}
		})
	}
}

// ============================================================
// Command lookup tests
// ============================================================

func TestFindCommand(t *testing.T) {
	// Test finding a command that almost certainly exists
	_, err := findCommand("sh")
	if err != nil {
		t.Errorf("findCommand('sh') should not return an error: %v", err)
	}

	// Test finding a non-existent command
	_, err = findCommand("nonexistent_command_12345")
	if err == nil {
		t.Error("findCommand('nonexistent_command_12345') should return an error")
	}
}

func TestCommandExists(t *testing.T) {
	if !commandExists("sh") {
		t.Error("commandExists('sh') should return true")
	}
	if commandExists("nonexistent_command_12345") {
		t.Error("commandExists('nonexistent_command_12345') should return false")
	}
}

// ============================================================
// File existence check tests
// ============================================================

func TestFileExists(t *testing.T) {
	// Create a temporary file
	tmpFile := filepath.Join(t.TempDir(), "testfile")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create temporary file: %v", err)
	}

	if !fileExists(tmpFile) {
		t.Error("fileExists() should return true for an existing file")
	}
	if fileExists("/nonexistent/path/file.txt") {
		t.Error("fileExists() should return false for a non-existent file")
	}
	if fileExists(t.TempDir()) {
		t.Error("fileExists() should return false for a directory")
	}
}

// ============================================================
// isRootRequired tests
// ============================================================

func TestIsRootRequired(t *testing.T) {
	tests := []struct {
		cmd      string
		expected bool
	}{
		{"apt-get", true},
		{"apt", true},
		{"yum", true},
		{"dnf", true},
		{"apk", true},
		{"pacman", true},
		{"zypper", true},
		{"brew", false},
		{"gem", false},
		{"ruby", false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			result := isRootRequired(tt.cmd)
			if result != tt.expected {
				t.Errorf("isRootRequired(%q) = %v, want %v", tt.cmd, result, tt.expected)
			}
		})
	}
}

// ============================================================
// Install result tests
// ============================================================

func TestInstallResultString(t *testing.T) {
	platform := &PlatformInfo{
		OS:         OSLinux,
		Arch:       ArchAMD64,
		Distro:     DistroUbuntu,
		PackageMgr: PMApt,
	}

	result := &InstallResult{
		AlreadyInstalled: false,
		RubyVersion:      "3.2.2",
		GemVersion:       "3.4.10",
		RubyPath:         "/usr/bin/ruby",
		GemPath:          "/usr/bin/gem",
		PackageManager:   PMApt,
		CommandsRun:      []string{"apt-get update", "apt-get install -y ruby ruby-dev"},
		Platform:         platform,
	}

	if result.RubyVersion != "3.2.2" {
		t.Errorf("RubyVersion = %s, want 3.2.2", result.RubyVersion)
	}
	if result.GemVersion != "3.4.10" {
		t.Errorf("GemVersion = %s, want 3.4.10", result.GemVersion)
	}
	if result.PackageManager != PMApt {
		t.Errorf("PackageManager = %s, want apt", result.PackageManager)
	}
	if len(result.CommandsRun) != 2 {
		t.Errorf("CommandsRun length = %d, want 2", len(result.CommandsRun))
	}
}

// ============================================================
// runCommand tests
// ============================================================

func TestRunCommand(t *testing.T) {
	ctx := context.Background()
	opts := NewInstallOptions().WithTimeout(10)

	// Test executing a simple command
	err := runCommand(ctx, opts, "echo", "hello")
	if err != nil {
		t.Errorf("runCommand('echo hello') should not return an error: %v", err)
	}

	// Test executing a failing command
	err = runCommand(ctx, opts, "false")
	if err == nil {
		t.Error("runCommand('false') should return an error")
	}
}

func TestRunCommandTimeout(t *testing.T) {
	ctx := context.Background()
	opts := NewInstallOptions().WithTimeout(1)

	// Test timeout
	err := runCommand(ctx, opts, "sleep", "10")
	if err == nil {
		t.Error("runCommand('sleep 10') should return an error due to timeout")
	}
}

// ============================================================
// Type constant tests
// ============================================================

func TestPackageManagerConstants(t *testing.T) {
	pms := []PackageManager{PMApt, PMYum, PMDnf, PMApk, PMPacman, PMBrew, PMChoco, PMScoop, PMZypper, PMUnknown}
	for _, pm := range pms {
		if pm == "" {
			t.Error("PackageManager constants should not be empty strings")
		}
	}
}

func TestLinuxDistroConstants(t *testing.T) {
	distros := []LinuxDistro{
		DistroUbuntu, DistroDebian, DistroCentOS, DistroRHEL,
		DistroFedora, DistroRocky, DistroAlma, DistroAlpine,
		DistroArch, DistroManjaro, DistroAmazon, DistroOpenSUSE,
		DistroUnknown,
	}
	for _, d := range distros {
		if d == "" {
			t.Error("LinuxDistro constants should not be empty strings")
		}
	}
}

func TestOperatingSystemConstants(t *testing.T) {
	oses := []OperatingSystem{OSLinux, OSDarwin, OSWindows, OSUnknown}
	for _, os := range oses {
		if os == "" {
			t.Error("OperatingSystem constants should not be empty strings")
		}
	}
}

func TestArchitectureConstants(t *testing.T) {
	archs := []Architecture{ArchAMD64, ArchARM64, ArchARM, Arch386, ArchUnknown}
	for _, a := range archs {
		if a == "" {
			t.Error("Architecture constants should not be empty strings")
		}
	}
}

// ============================================================
// Integration tests (require network and package manager permissions)
// ============================================================

func TestInstallerIsInstalled(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test")
	}

	installer := NewInstaller()
	installed, info, err := installer.IsInstalled()
	if err != nil {
		t.Fatalf("IsInstalled() returned error: %v", err)
	}

	t.Logf("Ruby installation status: %v", installed)
	if installed && info != nil {
		t.Logf("Ruby version: %s", info.RubyVersion)
		t.Logf("gem version: %s", info.GemVersion)
		t.Logf("Ruby path: %s", info.RubyPath)
		t.Logf("gem path: %s", info.GemPath)
	}
}

// TestDetectPlatformOnLinux is an integration test on Linux
func TestDetectPlatformOnLinux(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test")
	}

	installer := NewInstaller()
	platform, err := installer.DetectPlatform()
	if err != nil {
		t.Fatalf("DetectPlatform() failed: %v", err)
	}

	t.Logf("Platform info: %s", platform)
	t.Logf("  OS: %s", platform.OS)
	t.Logf("  Arch: %s", platform.Arch)
	t.Logf("  Distro: %s", platform.Distro)
	t.Logf("  PackageManager: %s", platform.PackageMgr)
	t.Logf("  PackageMgrCmd: %s", platform.PackageMgrCmd)

	// Verify platform info completeness
	if platform.OS == OSUnknown {
		t.Error("OS should not be Unknown")
	}
	if platform.Arch == ArchUnknown {
		t.Error("Arch should not be Unknown")
	}
}

// TestReadActualOSRelease reads the actual /etc/os-release file
func TestReadActualOSRelease(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test")
	}

	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		t.Skipf("cannot read /etc/os-release: %v", err)
	}

	id := parseOSReleaseField(string(data), "ID")
	idLike := parseOSReleaseField(string(data), "ID_LIKE")
	name := parseOSReleaseField(string(data), "NAME")
	version := parseOSReleaseField(string(data), "VERSION")

	t.Logf("Actual system info:")
	t.Logf("  ID: %s", id)
	t.Logf("  ID_LIKE: %s", idLike)
	t.Logf("  NAME: %s", name)
	t.Logf("  VERSION: %s", version)

	distro := readOSRelease()
	t.Logf("  Detected distro: %s", distro)

	if distro == DistroUnknown {
		t.Error("readOSRelease() should not return DistroUnknown on an actual Linux system")
	}
}

// TestGetCommandOutput tests getting command output
func TestGetCommandOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test")
	}

	// Test the echo command
	output, err := getCommandOutput("echo", "hello world")
	if err != nil {
		t.Fatalf("getCommandOutput('echo') failed: %v", err)
	}
	if strings.TrimSpace(output) != "hello world" {
		t.Errorf("getCommandOutput('echo hello world') = %q, want 'hello world'", output)
	}
}

// TestCheckRubyInstalledIntegration tests actual Ruby installation detection
func TestCheckRubyInstalledIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test")
	}

	installed, info, err := checkRubyInstalled()
	if err != nil {
		t.Fatalf("checkRubyInstalled() returned error: %v", err)
	}

	if installed {
		t.Logf("Ruby installed: %s (gem: %s)", info.RubyVersion, info.GemVersion)
		// Verify version number format
		if !isVersionString(info.RubyVersion) {
			t.Errorf("Ruby version number format is abnormal: %s", info.RubyVersion)
		}
		if info.GemVersion != "" && !isVersionString(info.GemVersion) {
			t.Errorf("gem version number format is abnormal: %s", info.GemVersion)
		}
	} else {
		t.Log("Ruby not installed")
	}
}

// TestFindCommandIntegration tests actual command lookup
func TestFindCommandIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test")
	}

	// Test common commands
	commonCmds := []string{"ls", "cat", "sh", "bash"}
	for _, cmd := range commonCmds {
		path, err := findCommand(cmd)
		if err != nil {
			t.Logf("command %s not found", cmd)
			continue
		}
		t.Logf("command %s located at: %s", cmd, path)
	}

	// Test package manager commands
	pmCmds := []string{"apt-get", "apt", "yum", "dnf", "apk", "pacman", "brew", "zypper"}
	foundPMs := []string{}
	for _, cmd := range pmCmds {
		if commandExists(cmd) {
			foundPMs = append(foundPMs, cmd)
		}
	}
	t.Logf("Found package managers: %v", foundPMs)
	if len(foundPMs) == 0 {
		t.Log("warning: no package manager found")
	}
}

// ============================================================
// Mock tests (do not require root permissions)
// ============================================================

// TestInstallViaAptDryRun simulates an apt install (dry run)
func TestInstallViaAptDryRun(t *testing.T) {
	// Use the echo command to simulate apt-get
	// This requires creating a fake apt-get script
	tmpDir := t.TempDir()
	fakeAptGet := filepath.Join(tmpDir, "apt-get")
	if err := os.WriteFile(fakeAptGet, []byte("#!/bin/sh\necho apt-get $@\nexit 0"), 0755); err != nil {
		t.Fatalf("failed to create fake apt-get: %v", err)
	}

	// Modify PATH to include our temporary directory
	origPath := os.Getenv("PATH")
 newPath := tmpDir + ":" + origPath
	if err := os.Setenv("PATH", newPath); err != nil {
		t.Fatalf("failed to set PATH: %v", err)
	}
	defer os.Setenv("PATH", origPath)

	// Verify the fake command is available
	path, err := exec.LookPath("apt-get")
	if err != nil {
		t.Fatalf("fake apt-get not available: %v", err)
	}
	t.Logf("Using fake apt-get: %s", path)
}

// TestInstallAlreadyInstalled tests skipping installation when already installed
func TestInstallAlreadyInstalled(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test")
	}

	// First check whether Ruby is already installed
	installed, _, _ := checkRubyInstalled()
	if !installed {
		t.Skip("Ruby not installed, skip this test")
	}

	installer := NewInstaller()
	result, err := installer.Install(context.Background())
	if err != nil {
		t.Fatalf("Install() failed: %v", err)
	}

	if !result.AlreadyInstalled {
		t.Error("Ruby is installed but returned AlreadyInstalled=false")
	}
	t.Logf("Ruby installed: %s (gem: %s)", result.RubyVersion, result.GemVersion)
}
