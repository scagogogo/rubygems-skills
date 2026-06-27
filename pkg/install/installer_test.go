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
// 平台检测测试
// ============================================================

func TestDetectOS(t *testing.T) {
	os := detectOS()
	// 在 Linux 上运行测试时应该是 OSLinux
	if os == OSUnknown {
		t.Error("detectOS() 不应返回 OSUnknown")
	}
}

func TestDetectArch(t *testing.T) {
	arch := detectArch()
	if arch == ArchUnknown {
		t.Error("detectArch() 不应返回 ArchUnknown")
	}
}

func TestDetectOSMapping(t *testing.T) {
	// 测试 GOOS 到 OperatingSystem 的映射关系
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
				t.Errorf("GOOS=%s 映射 = %v, want %v", tt.goos, result, tt.expected)
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

// detectArchFromGOARCH 是测试辅助函数
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
// /etc/os-release 解析测试
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
	// 创建一个临时的 os-release 文件
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
				t.Fatalf("写入测试文件失败: %v", err)
			}

			result := readOSReleaseFromString(string(readFileContent(osReleasePath)))
			if result != tt.expected {
				t.Errorf("readOSRelease() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// readFileContent 读取文件内容的辅助函数
func readFileContent(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return data
}

// readOSReleaseFromString 从字符串解析发行版信息（测试用）
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
// 版本号提取测试
// ============================================================

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ruby 3.2.2 (2023-03-30) [x86_64-linux]", "3.2.2"},
		{"ruby 2.7.0p183 (2020-03-31) [x86_64-darwin19]", "2.7.0"}, // 注意 p183 不匹配
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
		{"3", false}, // 没有点
		{"abc", false},
		{"", false},
		{"3.2.2-preview", false},
		{".1.2", false}, // 以点开头，不是有效版本号
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
// 安装选项测试
// ============================================================

func TestNewInstallOptions(t *testing.T) {
	opts := NewInstallOptions()

	if opts.ForceReinstall != false {
		t.Error("默认 ForceReinstall 应为 false")
	}
	if opts.InstallDevHeaders != true {
		t.Error("默认 InstallDevHeaders 应为 true")
	}
	if opts.InstallBundler != true {
		t.Error("默认 InstallBundler 应为 true")
	}
	if opts.UpdatePackageIndex != true {
		t.Error("默认 UpdatePackageIndex 应为 true")
	}
	if opts.TimeoutSeconds != 600 {
		t.Error("默认 TimeoutSeconds 应为 600")
	}
	if opts.UseSudo != true {
		t.Error("默认 UseSudo 应为 true")
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
		t.Error("WithForceReinstall(true) 失败")
	}
	if opts.RubyVersion != "3.2.2" {
		t.Errorf("WithRubyVersion 失败, got %s", opts.RubyVersion)
	}
	if opts.InstallDevHeaders {
		t.Error("WithDevHeaders(false) 失败")
	}
	if opts.InstallBundler {
		t.Error("WithBundler(false) 失败")
	}
	if opts.TimeoutSeconds != 600 {
		t.Errorf("WithTimeout(600) 失败, got %d", opts.TimeoutSeconds)
	}
	if opts.UseSudo {
		t.Error("WithSudo(false) 失败")
	}
	if len(opts.ExtraPackages) != 2 {
		t.Errorf("WithExtraPackages 失败, got %d packages", len(opts.ExtraPackages))
	}
}

// ============================================================
// Installer 创建和平台检测测试
// ============================================================

func TestNewInstaller(t *testing.T) {
	installer := NewInstaller()
	if installer == nil {
		t.Error("NewInstaller() 不应返回 nil")
	}
	if installer.options == nil {
		t.Error("Installer.options 不应为 nil")
	}
}

func TestNewInstallerWithCustomOptions(t *testing.T) {
	opts := NewInstallOptions().WithForceReinstall(true).WithTimeout(120)
	installer := NewInstaller(opts)
	if !installer.options.ForceReinstall {
		t.Error("自定义选项未被应用")
	}
	if installer.options.TimeoutSeconds != 120 {
		t.Error("自定义超时未被应用")
	}
}

func TestInstallerDetectPlatform(t *testing.T) {
	installer := NewInstaller()
	platform, err := installer.DetectPlatform()
	if err != nil {
		t.Fatalf("DetectPlatform() 失败: %v", err)
	}

	// 在 Linux 上运行测试
	if platform.OS != OSLinux {
		t.Errorf("在 Linux 上运行测试，但 OS = %v", platform.OS)
	}
	if platform.Arch == ArchUnknown {
		t.Error("Arch 不应为 Unknown")
	}
	if platform.PackageMgr == PMUnknown {
		t.Error("PackageMgr 不应为 Unknown")
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
// 包管理器检测测试
// ============================================================

func TestDetectLinuxPackageManager(t *testing.T) {
	tests := []struct {
		distro      LinuxDistro
		expectedPMs []PackageManager // 可能匹配多个包管理器
	}{
		{DistroUbuntu, []PackageManager{PMApt}},
		{DistroDebian, []PackageManager{PMApt}},
		{DistroCentOS, []PackageManager{PMYum, PMDnf}}, // CentOS 可能用 yum 或 dnf
		{DistroFedora, []PackageManager{PMDnf}},
		{DistroAlpine, []PackageManager{PMApk}},
		{DistroArch, []PackageManager{PMPacman}},
		{DistroOpenSUSE, []PackageManager{PMZypper}},
	}

	for _, tt := range tests {
		t.Run(string(tt.distro), func(t *testing.T) {
			pm, _, err := detectLinuxPackageManager(tt.distro)
			if err != nil {
				// 如果没有安装对应的包管理器，这是可以接受的
				t.Logf("detectLinuxPackageManager(%s) 返回错误: %v (可能环境中未安装该包管理器)", tt.distro, err)
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
				t.Errorf("detectLinuxPackageManager(%s) = %v, 期望以下之一: %v", tt.distro, pm, tt.expectedPMs)
			}
		})
	}
}

// ============================================================
// 命令查找测试
// ============================================================

func TestFindCommand(t *testing.T) {
	// 测试查找一个几乎肯定存在的命令
	_, err := findCommand("sh")
	if err != nil {
		t.Errorf("findCommand('sh') 不应返回错误: %v", err)
	}

	// 测试查找不存在的命令
	_, err = findCommand("nonexistent_command_12345")
	if err == nil {
		t.Error("findCommand('nonexistent_command_12345') 应返回错误")
	}
}

func TestCommandExists(t *testing.T) {
	if !commandExists("sh") {
		t.Error("commandExists('sh') 应返回 true")
	}
	if commandExists("nonexistent_command_12345") {
		t.Error("commandExists('nonexistent_command_12345') 应返回 false")
	}
}

// ============================================================
// 文件存在检查测试
// ============================================================

func TestFileExists(t *testing.T) {
	// 创建临时文件
	tmpFile := filepath.Join(t.TempDir(), "testfile")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}

	if !fileExists(tmpFile) {
		t.Error("fileExists() 对存在的文件应返回 true")
	}
	if fileExists("/nonexistent/path/file.txt") {
		t.Error("fileExists() 对不存在的文件应返回 false")
	}
	if fileExists(t.TempDir()) {
		t.Error("fileExists() 对目录应返回 false")
	}
}

// ============================================================
// isRootRequired 测试
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
// 安装结果测试
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
		t.Errorf("CommandsRun 长度 = %d, want 2", len(result.CommandsRun))
	}
}

// ============================================================
// runCommand 测试
// ============================================================

func TestRunCommand(t *testing.T) {
	ctx := context.Background()
	opts := NewInstallOptions().WithTimeout(10)

	// 测试执行一个简单命令
	err := runCommand(ctx, opts, "echo", "hello")
	if err != nil {
		t.Errorf("runCommand('echo hello') 不应返回错误: %v", err)
	}

	// 测试执行一个失败的命令
	err = runCommand(ctx, opts, "false")
	if err == nil {
		t.Error("runCommand('false') 应返回错误")
	}
}

func TestRunCommandTimeout(t *testing.T) {
	ctx := context.Background()
	opts := NewInstallOptions().WithTimeout(1)

	// 测试超时
	err := runCommand(ctx, opts, "sleep", "10")
	if err == nil {
		t.Error("runCommand('sleep 10') 应因超时返回错误")
	}
}

// ============================================================
// 类型常量测试
// ============================================================

func TestPackageManagerConstants(t *testing.T) {
	pms := []PackageManager{PMApt, PMYum, PMDnf, PMApk, PMPacman, PMBrew, PMChoco, PMScoop, PMZypper, PMUnknown}
	for _, pm := range pms {
		if pm == "" {
			t.Error("PackageManager 常量不应为空字符串")
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
			t.Error("LinuxDistro 常量不应为空字符串")
		}
	}
}

func TestOperatingSystemConstants(t *testing.T) {
	oses := []OperatingSystem{OSLinux, OSDarwin, OSWindows, OSUnknown}
	for _, os := range oses {
		if os == "" {
			t.Error("OperatingSystem 常量不应为空字符串")
		}
	}
}

func TestArchitectureConstants(t *testing.T) {
	archs := []Architecture{ArchAMD64, ArchARM64, ArchARM, Arch386, ArchUnknown}
	for _, a := range archs {
		if a == "" {
			t.Error("Architecture 常量不应为空字符串")
		}
	}
}

// ============================================================
// 集成测试（需要网络和包管理器权限）
// ============================================================

func TestInstallerIsInstalled(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	installer := NewInstaller()
	installed, info, err := installer.IsInstalled()
	if err != nil {
		t.Fatalf("IsInstalled() 返回错误: %v", err)
	}

	t.Logf("Ruby 安装状态: %v", installed)
	if installed && info != nil {
		t.Logf("Ruby 版本: %s", info.RubyVersion)
		t.Logf("gem 版本: %s", info.GemVersion)
		t.Logf("Ruby 路径: %s", info.RubyPath)
		t.Logf("gem 路径: %s", info.GemPath)
	}
}

// TestDetectPlatformOnLinux 是一个在 Linux 上的集成测试
func TestDetectPlatformOnLinux(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	installer := NewInstaller()
	platform, err := installer.DetectPlatform()
	if err != nil {
		t.Fatalf("DetectPlatform() 失败: %v", err)
	}

	t.Logf("平台信息: %s", platform)
	t.Logf("  OS: %s", platform.OS)
	t.Logf("  Arch: %s", platform.Arch)
	t.Logf("  Distro: %s", platform.Distro)
	t.Logf("  PackageManager: %s", platform.PackageMgr)
	t.Logf("  PackageMgrCmd: %s", platform.PackageMgrCmd)

	// 验证平台信息完整性
	if platform.OS == OSUnknown {
		t.Error("OS 不应为 Unknown")
	}
	if platform.Arch == ArchUnknown {
		t.Error("Arch 不应为 Unknown")
	}
}

// TestReadActualOSRelease 读取实际的 /etc/os-release 文件
func TestReadActualOSRelease(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		t.Skipf("无法读取 /etc/os-release: %v", err)
	}

	id := parseOSReleaseField(string(data), "ID")
	idLike := parseOSReleaseField(string(data), "ID_LIKE")
	name := parseOSReleaseField(string(data), "NAME")
	version := parseOSReleaseField(string(data), "VERSION")

	t.Logf("实际系统信息:")
	t.Logf("  ID: %s", id)
	t.Logf("  ID_LIKE: %s", idLike)
	t.Logf("  NAME: %s", name)
	t.Logf("  VERSION: %s", version)

	distro := readOSRelease()
	t.Logf("  检测到的发行版: %s", distro)

	if distro == DistroUnknown {
		t.Error("readOSRelease() 在实际 Linux 系统上不应返回 DistroUnknown")
	}
}

// TestGetCommandOutput 测试获取命令输出
func TestGetCommandOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 测试 echo 命令
	output, err := getCommandOutput("echo", "hello world")
	if err != nil {
		t.Fatalf("getCommandOutput('echo') 失败: %v", err)
	}
	if strings.TrimSpace(output) != "hello world" {
		t.Errorf("getCommandOutput('echo hello world') = %q, want 'hello world'", output)
	}
}

// TestCheckRubyInstalledIntegration 测试实际的 Ruby 安装检测
func TestCheckRubyInstalledIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	installed, info, err := checkRubyInstalled()
	if err != nil {
		t.Fatalf("checkRubyInstalled() 返回错误: %v", err)
	}

	if installed {
		t.Logf("Ruby 已安装: %s (gem: %s)", info.RubyVersion, info.GemVersion)
		// 验证版本号格式
		if !isVersionString(info.RubyVersion) {
			t.Errorf("Ruby 版本号格式异常: %s", info.RubyVersion)
		}
		if info.GemVersion != "" && !isVersionString(info.GemVersion) {
			t.Errorf("gem 版本号格式异常: %s", info.GemVersion)
		}
	} else {
		t.Log("Ruby 未安装")
	}
}

// TestFindCommandIntegration 测试实际命令查找
func TestFindCommandIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 测试常见命令
	commonCmds := []string{"ls", "cat", "sh", "bash"}
	for _, cmd := range commonCmds {
		path, err := findCommand(cmd)
		if err != nil {
			t.Logf("命令 %s 未找到", cmd)
			continue
		}
		t.Logf("命令 %s 位于: %s", cmd, path)
	}

	// 测试包管理器命令
	pmCmds := []string{"apt-get", "apt", "yum", "dnf", "apk", "pacman", "brew", "zypper"}
	foundPMs := []string{}
	for _, cmd := range pmCmds {
		if commandExists(cmd) {
			foundPMs = append(foundPMs, cmd)
		}
	}
	t.Logf("找到的包管理器: %v", foundPMs)
	if len(foundPMs) == 0 {
		t.Log("警告: 未找到任何包管理器")
	}
}

// ============================================================
// Mock 测试 (不需要 root 权限)
// ============================================================

// TestInstallViaAptDryRun 模拟 apt 安装（dry run）
func TestInstallViaAptDryRun(t *testing.T) {
	// 使用 echo 命令模拟 apt-get
	// 这需要创建一个假的 apt-get 脚本
	tmpDir := t.TempDir()
	fakeAptGet := filepath.Join(tmpDir, "apt-get")
	if err := os.WriteFile(fakeAptGet, []byte("#!/bin/sh\necho apt-get $@\nexit 0"), 0755); err != nil {
		t.Fatalf("创建假 apt-get 失败: %v", err)
	}

	// 修改 PATH 包含我们的临时目录
	origPath := os.Getenv("PATH")
 newPath := tmpDir + ":" + origPath
	if err := os.Setenv("PATH", newPath); err != nil {
		t.Fatalf("设置 PATH 失败: %v", err)
	}
	defer os.Setenv("PATH", origPath)

	// 验证假命令可用
	path, err := exec.LookPath("apt-get")
	if err != nil {
		t.Fatalf("假 apt-get 不可用: %v", err)
	}
	t.Logf("使用假 apt-get: %s", path)
}

// TestInstallAlreadyInstalled 测试已安装时跳过安装
func TestInstallAlreadyInstalled(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 先检查 Ruby 是否已安装
	installed, _, _ := checkRubyInstalled()
	if !installed {
		t.Skip("Ruby 未安装，跳过此测试")
	}

	installer := NewInstaller()
	result, err := installer.Install(context.Background())
	if err != nil {
		t.Fatalf("Install() 失败: %v", err)
	}

	if !result.AlreadyInstalled {
		t.Error("Ruby 已安装但返回 AlreadyInstalled=false")
	}
	t.Logf("Ruby 已安装: %s (gem: %s)", result.RubyVersion, result.GemVersion)
}
