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

// Docker 跨平台集成测试
// 这些测试需要 Docker 环境，使用 -docker 标志启用

var dockerEnabled = os.Getenv("TEST_DOCKER") == "1"

func dockerAvailable() bool {
	if !dockerEnabled {
		return false
	}
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}

// DockerTestConfig 定义 Docker 测试配置
type DockerTestConfig struct {
	Image       string
	Distro      LinuxDistro
	PackageMgr  PackageManager
	InstallCmd  string // 验证安装后应存在的命令
	SetupCmds   []string // 额外的容器设置命令
}

// getDockerTestConfigs 返回所有 Docker 测试配置
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

// TestDockerPlatformDetection 测试在 Docker 容器中的平台检测
func TestDockerPlatformDetection(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker 测试未启用，设置 TEST_DOCKER=1 启用")
	}

	configs := getDockerTestConfigs()

	for _, cfg := range configs {
		t.Run(string(cfg.Distro), func(t *testing.T) {
			testDockerPlatformDetection(t, cfg)
		})
	}
}

// TestDockerRubyInstallation 测试在 Docker 容器中的 Ruby 安装
func TestDockerRubyInstallation(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker 测试未启用，设置 TEST_DOCKER=1 启用")
	}

	configs := getDockerTestConfigs()

	for _, cfg := range configs {
		t.Run(string(cfg.Distro), func(t *testing.T) {
			testDockerRubyInstallation(t, cfg)
		})
	}
}

// testDockerPlatformDetection 在 Docker 容器中测试平台检测
func testDockerPlatformDetection(t *testing.T, cfg DockerTestConfig) {
	// 在容器中运行一个简单的 Go 程序来检测平台
	// 但由于我们需要编译，先使用 shell 脚本来检测

	// 1. 检测 /etc/os-release
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
		t.Fatalf("Docker 命令执行失败: %v\n输出: %s", err, output)
	}

	t.Logf("=== %s (%s) ===", cfg.Distro, cfg.Image)
	t.Logf("%s", output)

	// 验证预期的包管理器存在
	switch cfg.PackageMgr {
	case PMApt:
		if !strings.Contains(output, "HAS_APT_GET") && !strings.Contains(output, "HAS_APT") {
			t.Errorf("期望找到 apt/apt-get，但未找到")
		}
	case PMYum:
		if !strings.Contains(output, "HAS_YUM") {
			t.Errorf("期望找到 yum，但未找到")
		}
	case PMDnf:
		if !strings.Contains(output, "HAS_DNF") {
			t.Errorf("期望找到 dnf，但未找到")
		}
	case PMApk:
		if !strings.Contains(output, "HAS_APK") {
			t.Errorf("期望找到 apk，但未找到")
		}
	case PMPacman:
		if !strings.Contains(output, "HAS_PACMAN") {
			t.Errorf("期望找到 pacman，但未找到")
		}
	case PMZypper:
		if !strings.Contains(output, "HAS_ZYPPER") {
			t.Errorf("期望找到 zypper，但未找到")
		}
	}
}

// testDockerRubyInstallation 在 Docker 容器中实际安装 Ruby
func testDockerRubyInstallation(t *testing.T, cfg DockerTestConfig) {
	// 构建安装命令
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
		t.Fatalf("不支持的包管理器: %s", cfg.PackageMgr)
	}

	// 添加验证命令
	verifyScript := installScript + `
echo "=== VERIFICATION ==="
command -v ruby && ruby --version || echo "RUBY_NOT_FOUND"
command -v gem && gem --version || echo "GEM_NOT_FOUND"
`

	output, err := runDockerCommand(cfg.Image, verifyScript, 600)
	if err != nil {
		t.Fatalf("Docker 安装测试失败: %v\n输出: %s", err, output)
	}

	t.Logf("=== %s (%s) 安装结果 ===", cfg.Distro, cfg.Image)
	t.Logf("%s", output)

	// 验证安装成功
	if strings.Contains(output, "RUBY_NOT_FOUND") {
		t.Errorf("Ruby 安装失败: ruby 命令未找到")
	}
	if strings.Contains(output, "GEM_NOT_FOUND") {
		t.Errorf("gem 安装失败: gem 命令未找到")
	}

	// 提取并显示版本信息
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ruby ") {
			t.Logf("  Ruby 版本: %s", line)
		}
		if isVersionString(line) && len(line) < 20 {
			// 可能是 gem 版本号
			t.Logf("  版本号: %s", line)
		}
	}
}

// runDockerCommand 在 Docker 容器中执行脚本
func runDockerCommand(image, script string, timeoutSec int) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// 使用 docker run --rm 执行命令
	args := []string{
		"run", "--rm",
		"--network", "host",
		image,
		"sh", "-c", script,
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return string(output), fmt.Errorf("Docker 命令超时 (%d秒)", timeoutSec)
	}

	return string(output), err
}

// TestDockerOSReleaseParsing 在 Docker 容器中测试 os-release 解析
func TestDockerOSReleaseParsing(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker 测试未启用，设置 TEST_DOCKER=1 启用")
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
				t.Fatalf("读取 /etc/os-release 失败: %v", err)
			}

			id := parseOSReleaseField(output, "ID")
			t.Logf("%s: ID=%s", tt.image, id)

			if id != tt.expectedID {
				t.Errorf("ID = %q, want %q", id, tt.expectedID)
			}
		})
	}
}

// TestDockerBundlerInstallation 测试在 Docker 中安装 bundler
func TestDockerBundlerInstallation(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker 测试未启用，设置 TEST_DOCKER=1 启用")
	}

	// 仅测试 Ubuntu，因为 bundler 安装需要网络
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
		t.Fatalf("Docker bundler 安装测试失败: %v\n输出: %s", err, output)
	}

	t.Logf("Ubuntu bundler 安装结果:\n%s", output)

	if strings.Contains(output, "BUNDLER_NOT_FOUND") {
		t.Error("bundler 安装失败")
	}
}
