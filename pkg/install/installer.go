// Package install provides cross-platform auto-installation capability for the Ruby/RubyGems toolchain
//
// It supports auto-detection of OS and package managers, and installs Ruby and gem commands
// on different platforms. Supported platforms include: Ubuntu/Debian (apt), CentOS/RHEL/Fedora (yum/dnf),
// Alpine (apk), Arch (pacman), macOS (brew), Windows (choco/scoop), etc.
//
// Usage example:
//
//	installer := install.NewInstaller()
//	result, err := installer.Install(context.Background())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Ruby %s installed successfully!\n", result.RubyVersion)
package install

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ============================================================
// OS and architecture type definitions
// ============================================================

// OperatingSystem represents operating system type
type OperatingSystem string

const (
	OSLinux   OperatingSystem = "linux"
	OSDarwin  OperatingSystem = "darwin"
	OSWindows OperatingSystem = "windows"
	OSUnknown OperatingSystem = "unknown"
)

// LinuxDistro represents Linux distribution type
type LinuxDistro string

const (
	DistroUbuntu   LinuxDistro = "ubuntu"
	DistroDebian   LinuxDistro = "debian"
	DistroCentOS   LinuxDistro = "centos"
	DistroRHEL     LinuxDistro = "rhel"
	DistroFedora   LinuxDistro = "fedora"
	DistroRocky    LinuxDistro = "rocky"
	DistroAlma     LinuxDistro = "alma"
	DistroAlpine   LinuxDistro = "alpine"
	DistroArch     LinuxDistro = "arch"
	DistroManjaro  LinuxDistro = "manjaro"
	DistroAmazon   LinuxDistro = "amazon"
	DistroOpenSUSE LinuxDistro = "opensuse"
	DistroUnknown  LinuxDistro = "unknown"
)

// Architecture represents CPU architecture
type Architecture string

const (
	ArchAMD64   Architecture = "amd64"
	ArchARM64   Architecture = "arm64"
	ArchARM     Architecture = "arm"
	Arch386     Architecture = "386"
	ArchUnknown Architecture = "unknown"
)

// PackageManager represents package manager type
type PackageManager string

const (
	PMApt     PackageManager = "apt"
	PMYum     PackageManager = "yum"
	PMDnf     PackageManager = "dnf"
	PMApk     PackageManager = "apk"
	PMPacman  PackageManager = "pacman"
	PMBrew    PackageManager = "brew"
	PMChoco   PackageManager = "choco"
	PMScoop   PackageManager = "scoop"
	PMZypper  PackageManager = "zypper"
	PMUnknown PackageManager = "unknown"
)

// ============================================================
// Platform info
// ============================================================

// PlatformInfo contains detected platform info
type PlatformInfo struct {
	OS            OperatingSystem
	Arch          Architecture
	Distro        LinuxDistro
	PackageMgr    PackageManager
	PackageMgrCmd string // actual command path of the package manager
}

// String returns readable string of platform info
func (p *PlatformInfo) String() string {
	if p.OS == OSLinux {
		return fmt.Sprintf("%s/%s (%s, %s)", p.OS, p.Distro, p.Arch, p.PackageMgr)
	}
	return fmt.Sprintf("%s/%s (%s)", p.OS, p.Arch, p.PackageMgr)
}

// ============================================================
// Installation result
// ============================================================

// InstallResult contains installation result
type InstallResult struct {
	// Whether Ruby is already installed
	AlreadyInstalled bool

	// Ruby version after installation
	RubyVersion string

	// gem version after installation
	GemVersion string

	// Ruby executable path
	RubyPath string

	// gem executable path
	GemPath string

	// Package manager used
	PackageManager PackageManager

	// Commands executed during installation
	CommandsRun []string

	// Platform info
	Platform *PlatformInfo
}

// ============================================================
// Installation options
// ============================================================

// InstallOptions defines installation configuration options
type InstallOptions struct {
	// Force reinstall even if Ruby is already installed
	ForceReinstall bool

	// Install a specific version of Ruby (if package manager supports it)
	// Empty string means install the latest version
	RubyVersion string

	// Whether to also install dev headers (ruby-dev / ruby-devel)
	// Some gems require dev headers for compilation
	InstallDevHeaders bool

	// Whether to install bundler
	InstallBundler bool

	// Custom package manager (override auto-detection)
	CustomPackageManager PackageManager

	// Whether to update package index before install
	UpdatePackageIndex bool

	// Timeout for install commands in seconds (default 10 minutes, each sub-command timed independently)
	TimeoutSeconds int

	// Whether to use sudo (Linux/macOS)
	// Default is true, auto-skip sudo if running as root
	UseSudo bool

	// Extra package names (installed along with Ruby)
	ExtraPackages []string
}

// NewInstallOptions creates install options with defaults
func NewInstallOptions() *InstallOptions {
	return &InstallOptions{
		ForceReinstall:     false,
		RubyVersion:        "",
		InstallDevHeaders:  true,
		InstallBundler:     true,
		UpdatePackageIndex: true,
		TimeoutSeconds:     600,
		UseSudo:            true,
		ExtraPackages:      []string{},
	}
}

// WithForceReinstall sets whether to force reinstall
func (o *InstallOptions) WithForceReinstall(force bool) *InstallOptions {
	o.ForceReinstall = force
	return o
}

// WithRubyVersion sets the Ruby version to install
func (o *InstallOptions) WithRubyVersion(version string) *InstallOptions {
	o.RubyVersion = version
	return o
}

// WithDevHeaders sets whether to install dev headers
func (o *InstallOptions) WithDevHeaders(install bool) *InstallOptions {
	o.InstallDevHeaders = install
	return o
}

// WithBundler sets whether to install bundler
func (o *InstallOptions) WithBundler(install bool) *InstallOptions {
	o.InstallBundler = install
	return o
}

// WithCustomPackageManager sets custom package manager
func (o *InstallOptions) WithCustomPackageManager(pm PackageManager) *InstallOptions {
	o.CustomPackageManager = pm
	return o
}

// WithUpdatePackageIndex sets whether to update package index
func (o *InstallOptions) WithUpdatePackageIndex(update bool) *InstallOptions {
	o.UpdatePackageIndex = update
	return o
}

// WithTimeout sets timeout in seconds
func (o *InstallOptions) WithTimeout(seconds int) *InstallOptions {
	o.TimeoutSeconds = seconds
	return o
}

// WithSudo sets whether to use sudo
func (o *InstallOptions) WithSudo(useSudo bool) *InstallOptions {
	o.UseSudo = useSudo
	return o
}

// WithExtraPackages adds extra packages
func (o *InstallOptions) WithExtraPackages(packages ...string) *InstallOptions {
	o.ExtraPackages = append(o.ExtraPackages, packages...)
	return o
}

// ============================================================
// Installer
// ============================================================

// RubyInfo contains installed Ruby info
type RubyInfo struct {
	RubyVersion string
	GemVersion  string
	RubyPath    string
	GemPath     string
}

// Installer is the Ruby/RubyGems auto-installer
type Installer struct {
	options *InstallOptions
}

// NewInstaller creates a new installer
func NewInstaller(options ...*InstallOptions) *Installer {
	if len(options) == 0 {
		options = append(options, NewInstallOptions())
	}
	return &Installer{
		options: options[0],
	}
}

// DetectPlatform detects OS, arch and package manager of current platform
func (i *Installer) DetectPlatform() (*PlatformInfo, error) {
	info := &PlatformInfo{}

	// Detect OS
	info.OS = detectOS()

	// Detect architecture
	info.Arch = detectArch()

	// If Linux, detect distribution
	if info.OS == OSLinux {
		distro, err := detectLinuxDistro()
		if err != nil {
			info.Distro = DistroUnknown
		} else {
			info.Distro = distro
		}
	}

	// Detect package manager
	if i.options.CustomPackageManager != "" && i.options.CustomPackageManager != PMUnknown {
		info.PackageMgr = i.options.CustomPackageManager
		cmd, err := findCommand(string(i.options.CustomPackageManager))
		if err == nil {
			info.PackageMgrCmd = cmd
		}
	} else {
		pm, cmd, err := detectPackageManager(info)
		if err != nil {
			return nil, fmt.Errorf("unable to detect package manager: %w", err)
		}
		info.PackageMgr = pm
		info.PackageMgrCmd = cmd
	}

	return info, nil
}

// IsInstalled checks if Ruby is already installed
func (i *Installer) IsInstalled() (bool, *RubyInfo, error) {
	return checkRubyInstalled()
}

// Install performs Ruby/RubyGems auto-installation
func (i *Installer) Install(ctx context.Context) (*InstallResult, error) {
	// 1. Detect platform
	platform, err := i.DetectPlatform()
	if err != nil {
		return nil, fmt.Errorf("platform detection failed: %w", err)
	}

	// 2. Check if already installed
	installed, rubyInfo, checkErr := i.IsInstalled()
	if checkErr == nil && installed && !i.options.ForceReinstall {
		return &InstallResult{
			AlreadyInstalled: true,
			RubyVersion:      rubyInfo.RubyVersion,
			GemVersion:       rubyInfo.GemVersion,
			RubyPath:         rubyInfo.RubyPath,
			GemPath:          rubyInfo.GemPath,
			PackageManager:   platform.PackageMgr,
			Platform:         platform,
		}, nil
	}

	// 3. Install via the appropriate package manager
	commandsRun := []string{}
	var installErr error

	switch platform.PackageMgr {
	case PMApt:
		commandsRun, installErr = i.installViaApt(ctx, platform)
	case PMYum:
		commandsRun, installErr = i.installViaYum(ctx, platform)
	case PMDnf:
		commandsRun, installErr = i.installViaDnf(ctx, platform)
	case PMApk:
		commandsRun, installErr = i.installViaApk(ctx, platform)
	case PMPacman:
		commandsRun, installErr = i.installViaPacman(ctx, platform)
	case PMBrew:
		commandsRun, installErr = i.installViaBrew(ctx, platform)
	case PMChoco:
		commandsRun, installErr = i.installViaChoco(ctx, platform)
	case PMScoop:
		commandsRun, installErr = i.installViaScoop(ctx, platform)
	case PMZypper:
		commandsRun, installErr = i.installViaZypper(ctx, platform)
	default:
		return nil, fmt.Errorf("unsupported package manager: %s (platform: %s)", platform.PackageMgr, platform)
	}

	if installErr != nil {
		return nil, fmt.Errorf("installation failed: %w", installErr)
	}

	// 4. Verify installation
	installed, rubyInfo, verifyErr := i.IsInstalled()
	if verifyErr != nil {
		return nil, fmt.Errorf("post-install verification failed: %w", verifyErr)
	}
	if !installed {
		return nil, fmt.Errorf("installation completed but verification failed: Ruby not found")
	}

	// 5. Optional: install bundler
	if i.options.InstallBundler {
		cmd := "gem install bundler"
		commandsRun = append(commandsRun, cmd)
		if err := runCommand(ctx, i.options, "gem", "install", "bundler"); err != nil {
			// bundler install failure does not affect main flow
			commandsRun = append(commandsRun, fmt.Sprintf("# bundler install failed: %v", err))
		}
	}

	return &InstallResult{
		AlreadyInstalled: false,
		RubyVersion:      rubyInfo.RubyVersion,
		GemVersion:       rubyInfo.GemVersion,
		RubyPath:         rubyInfo.RubyPath,
		GemPath:          rubyInfo.GemPath,
		PackageManager:   platform.PackageMgr,
		CommandsRun:      commandsRun,
		Platform:         platform,
	}, nil
}

// ============================================================
// Platform detection implementation
// ============================================================

// detectOS detects current OS
func detectOS() OperatingSystem {
	switch runtime.GOOS {
	case "linux":
		return OSLinux
	case "darwin":
		return OSDarwin
	case "windows":
		return OSWindows
	default:
		return OSUnknown
	}
}

// detectArch detects current CPU architecture
func detectArch() Architecture {
	switch runtime.GOARCH {
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

// detectLinuxDistro detects Linux distribution
func detectLinuxDistro() (LinuxDistro, error) {
	// Method 1: read /etc/os-release
	if distro := readOSRelease(); distro != DistroUnknown {
		return distro, nil
	}

	// Method 2: check distro-specific files
	if distro := checkDistroFiles(); distro != DistroUnknown {
		return distro, nil
	}

	// Method 3: infer from package manager
	if distro := inferFromPackageManager(); distro != DistroUnknown {
		return distro, nil
	}

	return DistroUnknown, fmt.Errorf("unable to detect Linux distribution")
}

// readOSRelease reads distribution info from /etc/os-release
func readOSRelease() LinuxDistro {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return DistroUnknown
	}

	id := parseOSReleaseField(string(data), "ID")
	idLike := parseOSReleaseField(string(data), "ID_LIKE")

	// First, exact match on ID
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

	// Then infer from ID_LIKE
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

// parseOSReleaseField parses specified field from os-release content
func parseOSReleaseField(data, field string) string {
	lines := strings.Split(data, "\n")
	prefix := field + "="
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			value := strings.TrimPrefix(line, prefix)
			// Strip quotes
			value = strings.Trim(value, `"'`)
			return value
		}
	}
	return ""
}

// checkDistroFiles identifies distribution by checking distro-specific files
func checkDistroFiles() LinuxDistro {
	// Debian
	if fileExists("/etc/debian_version") {
		return DistroDebian
	}
	// Red Hat family
	if fileExists("/etc/redhat-release") {
		content, err := os.ReadFile("/etc/redhat-release")
		if err == nil {
			contentLower := strings.ToLower(string(content))
			switch {
			case strings.Contains(contentLower, "centos"):
				return DistroCentOS
			case strings.Contains(contentLower, "fedora"):
				return DistroFedora
			case strings.Contains(contentLower, "rocky"):
				return DistroRocky
			case strings.Contains(contentLower, "alma"):
				return DistroAlma
			case strings.Contains(contentLower, "red hat"):
				return DistroRHEL
			}
		}
		return DistroCentOS // default to CentOS
	}
	// Alpine
	if fileExists("/etc/alpine-release") {
		return DistroAlpine
	}
	// Arch
	if fileExists("/etc/arch-release") {
		return DistroArch
	}
	// Amazon Linux
	if fileExists("/etc/system-release") {
		content, err := os.ReadFile("/etc/system-release")
		if err == nil && strings.Contains(strings.ToLower(string(content)), "amazon") {
			return DistroAmazon
		}
	}
	return DistroUnknown
}

// inferFromPackageManager infers distribution from installed package manager
func inferFromPackageManager() LinuxDistro {
	if commandExists("apt-get") || commandExists("apt") {
		return DistroDebian
	}
	if commandExists("dnf") {
		return DistroFedora
	}
	if commandExists("yum") {
		return DistroCentOS
	}
	if commandExists("apk") {
		return DistroAlpine
	}
	if commandExists("pacman") {
		return DistroArch
	}
	if commandExists("zypper") {
		return DistroOpenSUSE
	}
	return DistroUnknown
}

// detectPackageManager detects package manager of current platform
func detectPackageManager(info *PlatformInfo) (PackageManager, string, error) {
	switch info.OS {
	case OSLinux:
		return detectLinuxPackageManager(info.Distro)
	case OSDarwin:
		return detectDarwinPackageManager()
	case OSWindows:
		return detectWindowsPackageManager()
	default:
		return PMUnknown, "", fmt.Errorf("unsupported OS: %s", info.OS)
	}
}

// detectLinuxPackageManager detects Linux package manager
func detectLinuxPackageManager(distro LinuxDistro) (PackageManager, string, error) {
	switch distro {
	case DistroUbuntu, DistroDebian:
		return findPmWithFallback(PMApt, []string{"apt-get", "apt"})
	case DistroCentOS, DistroRHEL, DistroAmazon:
		return findPmWithFallback(PMYum, []string{"yum"})
	case DistroFedora, DistroRocky, DistroAlma:
		// Fedora and newer Rocky/Alma prefer dnf
		if cmd, err := findCommand("dnf"); err == nil {
			return PMDnf, cmd, nil
		}
		return findPmWithFallback(PMYum, []string{"yum"})
	case DistroAlpine:
		return findPmWithFallback(PMApk, []string{"apk"})
	case DistroArch, DistroManjaro:
		return findPmWithFallback(PMPacman, []string{"pacman"})
	case DistroOpenSUSE:
		return findPmWithFallback(PMZypper, []string{"zypper"})
	default:
		// Try detecting by package manager command
		return detectPackageManagerByCommand()
	}
}

// detectDarwinPackageManager detects macOS package manager
func detectDarwinPackageManager() (PackageManager, string, error) {
	if cmd, err := findCommand("brew"); err == nil {
		return PMBrew, cmd, nil
	}
	return PMUnknown, "", fmt.Errorf("Homebrew not found on macOS, please install it first: https://brew.sh")
}

// detectWindowsPackageManager detects Windows package manager
func detectWindowsPackageManager() (PackageManager, string, error) {
	if cmd, err := findCommand("choco"); err == nil {
		return PMChoco, cmd, nil
	}
	if cmd, err := findCommand("scoop"); err == nil {
		return PMScoop, cmd, nil
	}
	return PMUnknown, "", fmt.Errorf("Chocolatey or Scoop not found on Windows, please install one of them first")
}

// detectPackageManagerByCommand detects package manager by command
func detectPackageManagerByCommand() (PackageManager, string, error) {
	type pmCandidate struct {
		pm  PackageManager
		cmd string
	}
	candidates := []pmCandidate{
		{PMApt, "apt-get"},
		{PMApt, "apt"},
		{PMDnf, "dnf"},
		{PMYum, "yum"},
		{PMApk, "apk"},
		{PMPacman, "pacman"},
		{PMZypper, "zypper"},
	}

	for _, c := range candidates {
		if cmd, err := findCommand(c.cmd); err == nil {
			return c.pm, cmd, nil
		}
	}

	return PMUnknown, "", fmt.Errorf("no known package manager found")
}

// findPmWithFallback finds package manager command
func findPmWithFallback(pm PackageManager, cmds []string) (PackageManager, string, error) {
	for _, cmd := range cmds {
		if path, err := findCommand(cmd); err == nil {
			return pm, path, nil
		}
	}
	return PMUnknown, "", fmt.Errorf("package manager command not found: %v", cmds)
}

// ============================================================
// Ruby installation detection
// ============================================================

// checkRubyInstalled checks if Ruby is installed
func checkRubyInstalled() (bool, *RubyInfo, error) {
	rubyPath, err := findCommand("ruby")
	if err != nil {
		return false, nil, nil
	}

	// Get Ruby version
	rubyVersion, err := getCommandOutput("ruby", "--version")
	if err != nil {
		return false, nil, nil
	}
	// ruby --version output format: "ruby 3.2.2 (2023-03-30) [x86_64-linux]"
	rubyVersion = extractVersion(rubyVersion)

	// Get gem version
	gemPath, _ := findCommand("gem")
	gemVersion := ""
	if gemPath != "" {
		if ver, err := getCommandOutput("gem", "--version"); err == nil {
			gemVersion = strings.TrimSpace(ver)
		}
	}

	return true, &RubyInfo{
		RubyVersion: rubyVersion,
		GemVersion:  gemVersion,
		RubyPath:    rubyPath,
		GemPath:     gemPath,
	}, nil
}

// extractVersion extracts version number from version string
// Input: "ruby 3.2.2 (2023-03-30) [x86_64-linux]"
// Input: "ruby 2.7.0p183 (2020-03-31) [x86_64-darwin19]"
// Output: "3.2.2" / "2.7.0"
func extractVersion(versionStr string) string {
	parts := strings.Fields(versionStr)
	for _, part := range parts {
		// Find the first field that looks like a version number (x.y.z format)
		if isVersionString(part) {
			return part
		}
		// Handle Ruby's p revision format: "2.7.0p183" -> "2.7.0"
		if idx := strings.IndexByte(part, 'p'); idx > 0 {
			prefix := part[:idx]
			if isVersionString(prefix) {
				return prefix
			}
		}
	}
	// If no standard format found, return cleaned string
	return strings.TrimSpace(versionStr)
}

// isVersionString checks if string looks like a version number
// Valid version number format: x.y.z, starts with digit, contains at least one dot
func isVersionString(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Version number must start with a digit
	if s[0] < '0' || s[0] > '9' {
		return false
	}
	hasDigit := false
	hasDot := false
	for _, c := range s {
		if c >= '0' && c <= '9' {
			hasDigit = true
		} else if c == '.' {
			hasDot = true
		} else {
			return false
		}
	}
	return hasDigit && hasDot
}

// ============================================================
// Per-platform installation implementation
// ============================================================

// installViaApt installs Ruby via apt (Ubuntu/Debian)
func (i *Installer) installViaApt(ctx context.Context, platform *PlatformInfo) ([]string, error) {
	var commands []string

	// Update package index
	if i.options.UpdatePackageIndex {
		cmd := "apt-get update"
		commands = append(commands, cmd)
		if err := runCommand(ctx, i.options, "apt-get", "update"); err != nil {
			return commands, fmt.Errorf("apt-get update failed: %w", err)
		}
	}

	// Build package list
	packages := []string{"ruby"}
	if i.options.InstallDevHeaders {
		packages = append(packages, "ruby-dev")
	}
	packages = append(packages, i.options.ExtraPackages...)

	// Install Ruby
	installCmd := fmt.Sprintf("apt-get install -y %s", strings.Join(packages, " "))
	commands = append(commands, installCmd)
	args := append([]string{"install", "-y"}, packages...)
	if err := runCommand(ctx, i.options, "apt-get", args...); err != nil {
		return commands, fmt.Errorf("apt-get install failed: %w", err)
	}

	return commands, nil
}

// installViaYum installs Ruby via yum (CentOS/RHEL)
func (i *Installer) installViaYum(ctx context.Context, platform *PlatformInfo) ([]string, error) {
	var commands []string

	// Update package index
	if i.options.UpdatePackageIndex {
		cmd := "yum makecache"
		commands = append(commands, cmd)
		if err := runCommand(ctx, i.options, "yum", "makecache"); err != nil {
			commands = append(commands, "# yum makecache failed, continue installing")
		}
	}

	// Build package list
	packages := []string{"ruby"}
	if i.options.InstallDevHeaders {
		packages = append(packages, "ruby-devel")
	}
	packages = append(packages, i.options.ExtraPackages...)

	// Install Ruby
	installCmd := fmt.Sprintf("yum install -y %s", strings.Join(packages, " "))
	commands = append(commands, installCmd)
	args := append([]string{"install", "-y"}, packages...)
	if err := runCommand(ctx, i.options, "yum", args...); err != nil {
		return commands, fmt.Errorf("yum install failed: %w", err)
	}

	return commands, nil
}

// installViaDnf installs Ruby via dnf (Fedora/Rocky/Alma)
func (i *Installer) installViaDnf(ctx context.Context, platform *PlatformInfo) ([]string, error) {
	var commands []string

	// Update package index
	if i.options.UpdatePackageIndex {
		cmd := "dnf makecache"
		commands = append(commands, cmd)
		if err := runCommand(ctx, i.options, "dnf", "makecache"); err != nil {
			commands = append(commands, "# dnf makecache failed, continue installing")
		}
	}

	// Build package list
	packages := []string{"ruby"}
	if i.options.InstallDevHeaders {
		packages = append(packages, "ruby-devel")
	}
	packages = append(packages, i.options.ExtraPackages...)

	// Install Ruby
	installCmd := fmt.Sprintf("dnf install -y %s", strings.Join(packages, " "))
	commands = append(commands, installCmd)
	args := append([]string{"install", "-y"}, packages...)
	if err := runCommand(ctx, i.options, "dnf", args...); err != nil {
		return commands, fmt.Errorf("dnf install failed: %w", err)
	}

	return commands, nil
}

// installViaApk installs Ruby via apk (Alpine)
func (i *Installer) installViaApk(ctx context.Context, platform *PlatformInfo) ([]string, error) {
	var commands []string

	// Update package index
	if i.options.UpdatePackageIndex {
		cmd := "apk update"
		commands = append(commands, cmd)
		if err := runCommand(ctx, i.options, "apk", "update"); err != nil {
			return commands, fmt.Errorf("apk update failed: %w", err)
		}
	}

	// Build package list
	packages := []string{"ruby"}
	if i.options.InstallDevHeaders {
		packages = append(packages, "ruby-dev")
	}
	packages = append(packages, i.options.ExtraPackages...)

	// Install Ruby
	installCmd := fmt.Sprintf("apk add %s", strings.Join(packages, " "))
	commands = append(commands, installCmd)
	args := append([]string{"add"}, packages...)
	if err := runCommand(ctx, i.options, "apk", args...); err != nil {
		return commands, fmt.Errorf("apk add failed: %w", err)
	}

	return commands, nil
}

// installViaPacman installs Ruby via pacman (Arch/Manjaro)
func (i *Installer) installViaPacman(ctx context.Context, platform *PlatformInfo) ([]string, error) {
	var commands []string

	// Update package index
	if i.options.UpdatePackageIndex {
		cmd := "pacman -Sy"
		commands = append(commands, cmd)
		if err := runCommand(ctx, i.options, "pacman", "-Sy", "--noconfirm"); err != nil {
			return commands, fmt.Errorf("pacman -Sy failed: %w", err)
		}
	}

	// Build package list
	packages := []string{"ruby"}
	// Arch's ruby package already includes dev headers
	packages = append(packages, i.options.ExtraPackages...)

	// Install Ruby
	installCmd := fmt.Sprintf("pacman -S --noconfirm %s", strings.Join(packages, " "))
	commands = append(commands, installCmd)
	args := append([]string{"-S", "--noconfirm"}, packages...)
	if err := runCommand(ctx, i.options, "pacman", args...); err != nil {
		return commands, fmt.Errorf("pacman -S failed: %w", err)
	}

	return commands, nil
}

// installViaBrew installs Ruby via Homebrew (macOS)
func (i *Installer) installViaBrew(ctx context.Context, platform *PlatformInfo) ([]string, error) {
	var commands []string

	// Update Homebrew
	if i.options.UpdatePackageIndex {
		cmd := "brew update"
		commands = append(commands, cmd)
		if err := runCommand(ctx, i.options, "brew", "update"); err != nil {
			commands = append(commands, "# brew update failed, continue installing")
		}
	}

	// Install Ruby
	installCmd := "brew install ruby"
	commands = append(commands, installCmd)
	if err := runCommand(ctx, i.options, "brew", "install", "ruby"); err != nil {
		return commands, fmt.Errorf("brew install ruby failed: %w", err)
	}

	return commands, nil
}

// installViaChoco installs Ruby via Chocolatey (Windows)
func (i *Installer) installViaChoco(ctx context.Context, platform *PlatformInfo) ([]string, error) {
	var commands []string

	// Install Ruby
	installCmd := "choco install -y ruby"
	if i.options.RubyVersion != "" {
		installCmd = fmt.Sprintf("choco install -y ruby --version=%s", i.options.RubyVersion)
	}
	commands = append(commands, installCmd)

	args := []string{"install", "-y", "ruby"}
	if i.options.RubyVersion != "" {
		args = append(args, fmt.Sprintf("--version=%s", i.options.RubyVersion))
	}
	if err := runCommand(ctx, i.options, "choco", args...); err != nil {
		return commands, fmt.Errorf("choco install failed: %w", err)
	}

	return commands, nil
}

// installViaScoop installs Ruby via Scoop (Windows)
func (i *Installer) installViaScoop(ctx context.Context, platform *PlatformInfo) ([]string, error) {
	var commands []string

	installCmd := "scoop install ruby"
	commands = append(commands, installCmd)
	if err := runCommand(ctx, i.options, "scoop", "install", "ruby"); err != nil {
		return commands, fmt.Errorf("scoop install ruby failed: %w", err)
	}

	return commands, nil
}

// installViaZypper installs Ruby via zypper (openSUSE)
func (i *Installer) installViaZypper(ctx context.Context, platform *PlatformInfo) ([]string, error) {
	var commands []string

	// Update package index
	if i.options.UpdatePackageIndex {
		cmd := "zypper refresh"
		commands = append(commands, cmd)
		if err := runCommand(ctx, i.options, "zypper", "refresh"); err != nil {
			commands = append(commands, "# zypper refresh failed, continue installing")
		}
	}

	// Build package list
	packages := []string{"ruby"}
	if i.options.InstallDevHeaders {
		packages = append(packages, "ruby-devel")
	}
	packages = append(packages, i.options.ExtraPackages...)

	// Install Ruby
	installCmd := fmt.Sprintf("zypper install -y %s", strings.Join(packages, " "))
	commands = append(commands, installCmd)
	args := append([]string{"install", "-y"}, packages...)
	if err := runCommand(ctx, i.options, "zypper", args...); err != nil {
		return commands, fmt.Errorf("zypper install failed: %w", err)
	}

	return commands, nil
}

// ============================================================
// Utility functions
// ============================================================

// runCommand executes a system command
func runCommand(ctx context.Context, options *InstallOptions, name string, args ...string) error {
	timeout := 600
	if options != nil && options.TimeoutSeconds > 0 {
		timeout = options.TimeoutSeconds
	}

	cmdCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// Check if sudo is needed
	needSudo := options != nil && options.UseSudo && isRootRequired(name) && !isRunningAsRoot()

	var cmd *exec.Cmd
	if needSudo {
		allArgs := append([]string{name}, args...)
		cmd = exec.CommandContext(cmdCtx, "sudo", allArgs...)
	} else {
		cmd = exec.CommandContext(cmdCtx, name, args...)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command execution failed [%s %v]: %w\nOutput: %s", name, args, err, string(output))
	}

	return nil
}

// findCommand finds executable path of a command
func findCommand(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("command not found: %s", name)
	}
	return path, nil
}

// commandExists checks if command exists
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// getCommandOutput executes command and returns output
func getCommandOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// isRunningAsRoot checks if running as root
func isRunningAsRoot() bool {
	return os.Getuid() == 0
}

// isRootRequired checks if command requires root
func isRootRequired(cmd string) bool {
	rootRequired := map[string]bool{
		"apt-get": true,
		"apt":     true,
		"yum":     true,
		"dnf":     true,
		"apk":     true,
		"pacman":  true,
		"zypper":  true,
	}
	return rootRequired[cmd]
}

// fileExists checks if file exists
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
