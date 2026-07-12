package install

import (
	"context"
	"errors"
	"testing"
)

// ============================================================
// DetectPlatform — custom PM, all OS, and detection-failure branches
// ============================================================

func TestDetectPlatform_CustomPackageManagerFound(t *testing.T) {
	swapOS(t, OSLinux)
	swapArch(t, ArchAMD64)
	r := newFakeRunner().withLookPath("apt", "/usr/bin/apt")
	swapRunner(t, r)
	i := NewInstaller(NewInstallOptions().WithCustomPackageManager(PMApt))
	info, err := i.DetectPlatform()
	if err != nil {
		t.Fatalf("DetectPlatform err=%v", err)
	}
	if info.PackageMgr != PMApt {
		t.Errorf("PackageMgr=%s want apt", info.PackageMgr)
	}
	if info.PackageMgrCmd != "/usr/bin/apt" {
		t.Errorf("PackageMgrCmd=%s", info.PackageMgrCmd)
	}
}

func TestDetectPlatform_CustomPackageManagerNotFound(t *testing.T) {
	swapOS(t, OSLinux)
	swapArch(t, ArchAMD64)
	swapRunner(t, newFakeRunner()) // custom PM command missing
	i := NewInstaller(NewInstallOptions().WithCustomPackageManager(PMApt))
	info, err := i.DetectPlatform()
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if info.PackageMgr != PMApt {
		t.Errorf("PackageMgr=%s want apt", info.PackageMgr)
	}
	if info.PackageMgrCmd != "" {
		t.Errorf("PackageMgrCmd should be empty, got %s", info.PackageMgrCmd)
	}
}

func TestDetectPlatform_LinuxAutoDetect(t *testing.T) {
	swapOS(t, OSLinux)
	swapArch(t, ArchARM64)
	swapFileReader(t, memoryFile(map[string]string{"/etc/os-release": "ID=ubuntu"}))
	swapStat(t, statExists(map[string]bool{}))
	r := newFakeRunner().withLookPath("apt-get", "/usr/bin/apt-get")
	swapRunner(t, r)
	i := NewInstaller()
	info, err := i.DetectPlatform()
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if info.OS != OSLinux || info.Arch != ArchARM64 || info.Distro != DistroUbuntu || info.PackageMgr != PMApt {
		t.Errorf("info=%+v", info)
	}
}

func TestDetectPlatform_Darwin(t *testing.T) {
	swapOS(t, OSDarwin)
	swapArch(t, ArchARM64)
	r := newFakeRunner().withLookPath("brew", "/usr/local/bin/brew")
	swapRunner(t, r)
	info, err := NewInstaller().DetectPlatform()
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if info.OS != OSDarwin || info.PackageMgr != PMBrew {
		t.Errorf("info=%+v", info)
	}
}

func TestDetectPlatform_Windows(t *testing.T) {
	swapOS(t, OSWindows)
	swapArch(t, ArchAMD64)
	r := newFakeRunner().withLookPath("choco", "/usr/bin/choco")
	swapRunner(t, r)
	info, err := NewInstaller().DetectPlatform()
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if info.OS != OSWindows || info.PackageMgr != PMChoco {
		t.Errorf("info=%+v", info)
	}
}

func TestDetectPlatform_PMDetectionFails(t *testing.T) {
	swapOS(t, OSUnknown)
	swapArch(t, ArchAMD64)
	swapRunner(t, newFakeRunner())
	_, err := NewInstaller().DetectPlatform()
	if err == nil {
		t.Fatal("expected error when PM detection fails on unknown OS")
	}
}

func TestDetectPlatform_LinuxDistroUnknownFallsToPMCommand(t *testing.T) {
	// os-release ID is unrecognised, no distro files, but apt-get exists ->
	// inferFromPackageManager returns DistroDebian, and PM is detected via command.
	swapOS(t, OSLinux)
	swapArch(t, ArchAMD64)
	swapFileReader(t, memoryFile(map[string]string{"/etc/os-release": "ID=gentoo"}))
	swapStat(t, statExists(map[string]bool{}))
	r := newFakeRunner().withLookPath("apt-get", "/usr/bin/apt-get")
	swapRunner(t, r)
	info, err := NewInstaller().DetectPlatform()
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	// readOSRelease returns Unknown for gentoo, but inferFromPackageManager finds
	// apt-get -> DistroDebian; PM via detectLinuxPackageManager(Debian)=apt.
	if info.Distro != DistroDebian {
		t.Errorf("Distro=%s want debian", info.Distro)
	}
	if info.PackageMgr != PMApt {
		t.Errorf("PackageMgr=%s want apt", info.PackageMgr)
	}
}

// ============================================================
// osRunner (real impl) — Output, IsRoot, Run sudo branch
// These exercise the default runner directly.
// ============================================================

func TestOSRunner_Output(t *testing.T) {
	out, err := osRunner{}.Output("echo", "hi")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if out != "hi\n" {
		t.Errorf("Output=%q want %q", out, "hi\n")
	}
}

func TestOSRunner_Output_Error(t *testing.T) {
	_, err := osRunner{}.Output("nonexistent_command_xyz_123")
	if err == nil {
		t.Fatal("expected error for missing command")
	}
}

func TestOSRunner_IsRoot(t *testing.T) {
	// Just ensure it returns a bool without panicking; value is env-dependent.
	_ = osRunner{}.IsRoot()
}

func TestOSRunner_Run_NonSudoPath(t *testing.T) {
	// Run "false" (no root required) to exercise the non-sudo path quickly.
	err := osRunner{}.Run(context.Background(), NewInstallOptions().WithTimeout(5), "false")
	if err == nil {
		t.Fatal("expected error from `false`")
	}
}

func TestRunCommand_SudoNotAppliedWhenRootRequiredButAlreadyRoot(t *testing.T) {
	// isRootRequired("apt-get")=true; when running AS root, sudo is skipped.
	// We simulate "root" via a fake runner whose IsRoot()=true, and Run records
	// the exact command (no "sudo" prefix expected).
	r := newFakeRunner().withRoot(true).withRunFallback(nil)
	swapRunner(t, r)
	opts := NewInstallOptions().WithSudo(true).WithTimeout(5)
	if err := runCommand(context.Background(), opts, "apt-get", "update"); err != nil {
		t.Fatalf("err=%v", err)
	}
	// The recorded call must be "apt-get update" (no sudo prefix), because root.
	if len(r.ranCalls) != 1 || r.ranCalls[0] != "apt-get update" {
		t.Errorf("ranCalls=%v, want [\"apt-get update\"]", r.ranCalls)
	}
}

func TestRunCommand_WithNilOptions(t *testing.T) {
	// options==nil branch: default timeout, UseSudo false -> no sudo.
	r := newFakeRunner().withRunFallback(nil)
	swapRunner(t, r)
	if err := runCommand(context.Background(), nil, "echo", "ok"); err != nil {
		t.Fatalf("err=%v", err)
	}
}

// ============================================================
// installVia* — every package manager, success + failure branches
// Each uses a fake runner that scripts Run outcomes.
// ============================================================

func newInstallerWith(opts *InstallOptions) *Installer {
	if opts == nil {
		opts = NewInstallOptions()
	}
	return NewInstaller(opts)
}

func TestInstallViaApt_UpdateAndInstallSuccess(t *testing.T) {
	r := newFakeRunner().
		withRun("apt-get", []string{"update"}, nil).
		withRun("apt-get", []string{"install", "-y", "ruby", "ruby-dev"}, nil)
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithDevHeaders(true).WithUpdatePackageIndex(true))
	cmds, err := i.installViaApt(context.Background(), &PlatformInfo{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(cmds) != 2 {
		t.Errorf("cmds=%v", cmds)
	}
}

func TestInstallViaApt_NoUpdateNoDevHeaders(t *testing.T) {
	r := newFakeRunner().
		withRun("apt-get", []string{"install", "-y", "ruby"}, nil)
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithDevHeaders(false).WithUpdatePackageIndex(false))
	cmds, err := i.installViaApt(context.Background(), &PlatformInfo{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(cmds) != 1 {
		t.Errorf("cmds=%v", cmds)
	}
}

func TestInstallViaApt_UpdateFails(t *testing.T) {
	r := newFakeRunner().
		withRun("apt-get", []string{"update"}, errors.New("update failed"))
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithUpdatePackageIndex(true))
	_, err := i.installViaApt(context.Background(), &PlatformInfo{})
	if err == nil || err.Error() == "" {
		t.Fatalf("expected non-empty error, got %v", err)
	}
}

func TestInstallViaApt_InstallFails(t *testing.T) {
	r := newFakeRunner().
		withRun("apt-get", []string{"update"}, nil).
		withRun("apt-get", []string{"install", "-y", "ruby", "ruby-dev"}, errors.New("install failed"))
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithDevHeaders(true).WithUpdatePackageIndex(true))
	_, err := i.installViaApt(context.Background(), &PlatformInfo{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInstallViaApt_ExtraPackages(t *testing.T) {
	r := newFakeRunner().
		withRun("apt-get", []string{"install", "-y", "ruby", "ruby-dev", "libssl-dev"}, nil)
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithDevHeaders(true).WithUpdatePackageIndex(false).WithExtraPackages("libssl-dev"))
	cmds, err := i.installViaApt(context.Background(), &PlatformInfo{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(cmds) != 1 {
		t.Errorf("cmds=%v", cmds)
	}
}

func TestInstallViaYum_Success(t *testing.T) {
	r := newFakeRunner().
		withRun("yum", []string{"makecache"}, nil).
		withRun("yum", []string{"install", "-y", "ruby", "ruby-devel"}, nil)
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithDevHeaders(true).WithUpdatePackageIndex(true))
	cmds, err := i.installViaYum(context.Background(), &PlatformInfo{})
	if err != nil || len(cmds) != 2 {
		t.Fatalf("err=%v cmds=%v", err, cmds)
	}
}

func TestInstallViaYum_MakecacheFailsContinues(t *testing.T) {
	r := newFakeRunner().
		withRun("yum", []string{"makecache"}, errors.New("makecache failed")).
		withRun("yum", []string{"install", "-y", "ruby", "ruby-devel"}, nil)
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithDevHeaders(true).WithUpdatePackageIndex(true))
	cmds, err := i.installViaYum(context.Background(), &PlatformInfo{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	// makecache failure is appended as a comment, install still proceeds.
	if len(cmds) != 3 {
		t.Errorf("cmds=%v", cmds)
	}
}

func TestInstallViaYum_InstallFails(t *testing.T) {
	r := newFakeRunner().
		withRun("yum", []string{"install", "-y", "ruby", "ruby-devel"}, errors.New("install failed"))
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithDevHeaders(true).WithUpdatePackageIndex(false))
	_, err := i.installViaYum(context.Background(), &PlatformInfo{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInstallViaDnf_Success(t *testing.T) {
	r := newFakeRunner().
		withRun("dnf", []string{"makecache"}, nil).
		withRun("dnf", []string{"install", "-y", "ruby", "ruby-devel"}, nil)
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithDevHeaders(true).WithUpdatePackageIndex(true))
	cmds, err := i.installViaDnf(context.Background(), &PlatformInfo{})
	if err != nil || len(cmds) != 2 {
		t.Fatalf("err=%v cmds=%v", err, cmds)
	}
}

func TestInstallViaDnf_MakecacheFailsContinues(t *testing.T) {
	r := newFakeRunner().
		withRun("dnf", []string{"makecache"}, errors.New("makecache failed")).
		withRun("dnf", []string{"install", "-y", "ruby", "ruby-devel"}, nil)
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithDevHeaders(true).WithUpdatePackageIndex(true))
	if _, err := i.installViaDnf(context.Background(), &PlatformInfo{}); err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestInstallViaDnf_InstallFails(t *testing.T) {
	r := newFakeRunner().
		withRun("dnf", []string{"install", "-y", "ruby", "ruby-devel"}, errors.New("install failed"))
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithDevHeaders(true).WithUpdatePackageIndex(false))
	_, err := i.installViaDnf(context.Background(), &PlatformInfo{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInstallViaApk_Success(t *testing.T) {
	r := newFakeRunner().
		withRun("apk", []string{"update"}, nil).
		withRun("apk", []string{"add", "ruby", "ruby-dev"}, nil)
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithDevHeaders(true).WithUpdatePackageIndex(true))
	cmds, err := i.installViaApk(context.Background(), &PlatformInfo{})
	if err != nil || len(cmds) != 2 {
		t.Fatalf("err=%v cmds=%v", err, cmds)
	}
}

func TestInstallViaApk_UpdateFails(t *testing.T) {
	r := newFakeRunner().
		withRun("apk", []string{"update"}, errors.New("update failed"))
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithUpdatePackageIndex(true))
	_, err := i.installViaApk(context.Background(), &PlatformInfo{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInstallViaApk_InstallFails(t *testing.T) {
	r := newFakeRunner().
		withRun("apk", []string{"add", "ruby", "ruby-dev"}, errors.New("add failed"))
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithDevHeaders(true).WithUpdatePackageIndex(false))
	_, err := i.installViaApk(context.Background(), &PlatformInfo{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInstallViaPacman_Success(t *testing.T) {
	r := newFakeRunner().
		withRun("pacman", []string{"-Sy", "--noconfirm"}, nil).
		withRun("pacman", []string{"-S", "--noconfirm", "ruby", "extra-pkg"}, nil)
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithUpdatePackageIndex(true).WithExtraPackages("extra-pkg"))
	cmds, err := i.installViaPacman(context.Background(), &PlatformInfo{})
	if err != nil || len(cmds) != 2 {
		t.Fatalf("err=%v cmds=%v", err, cmds)
	}
}

func TestInstallViaPacman_SyFails(t *testing.T) {
	r := newFakeRunner().
		withRun("pacman", []string{"-Sy", "--noconfirm"}, errors.New("sy failed"))
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithUpdatePackageIndex(true))
	_, err := i.installViaPacman(context.Background(), &PlatformInfo{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInstallViaPacman_InstallFails(t *testing.T) {
	r := newFakeRunner().
		withRun("pacman", []string{"-S", "--noconfirm", "ruby"}, errors.New("install failed"))
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithUpdatePackageIndex(false))
	_, err := i.installViaPacman(context.Background(), &PlatformInfo{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInstallViaBrew_Success(t *testing.T) {
	r := newFakeRunner().
		withRun("brew", []string{"update"}, nil).
		withRun("brew", []string{"install", "ruby"}, nil)
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithUpdatePackageIndex(true))
	cmds, err := i.installViaBrew(context.Background(), &PlatformInfo{})
	if err != nil || len(cmds) != 2 {
		t.Fatalf("err=%v cmds=%v", err, cmds)
	}
}

func TestInstallViaBrew_UpdateFailsContinues(t *testing.T) {
	r := newFakeRunner().
		withRun("brew", []string{"update"}, errors.New("update failed")).
		withRun("brew", []string{"install", "ruby"}, nil)
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithUpdatePackageIndex(true))
	if _, err := i.installViaBrew(context.Background(), &PlatformInfo{}); err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestInstallViaBrew_InstallFails(t *testing.T) {
	r := newFakeRunner().
		withRun("brew", []string{"install", "ruby"}, errors.New("install failed"))
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithUpdatePackageIndex(false))
	_, err := i.installViaBrew(context.Background(), &PlatformInfo{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInstallViaChoco_Success(t *testing.T) {
	r := newFakeRunner().
		withRun("choco", []string{"install", "-y", "ruby"}, nil)
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithUpdatePackageIndex(false))
	cmds, err := i.installViaChoco(context.Background(), &PlatformInfo{})
	if err != nil || len(cmds) != 1 {
		t.Fatalf("err=%v cmds=%v", err, cmds)
	}
}

func TestInstallViaChoco_WithVersion(t *testing.T) {
	r := newFakeRunner().
		withRun("choco", []string{"install", "-y", "ruby", "--version=3.2.2"}, nil)
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithRubyVersion("3.2.2"))
	cmds, err := i.installViaChoco(context.Background(), &PlatformInfo{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(cmds) != 1 || cmds[0] != "choco install -y ruby --version=3.2.2" {
		t.Errorf("cmds=%v", cmds)
	}
}

func TestInstallViaChoco_InstallFails(t *testing.T) {
	r := newFakeRunner().
		withRun("choco", []string{"install", "-y", "ruby"}, errors.New("install failed"))
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions())
	_, err := i.installViaChoco(context.Background(), &PlatformInfo{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInstallViaScoop_Success(t *testing.T) {
	r := newFakeRunner().
		withRun("scoop", []string{"install", "ruby"}, nil)
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions())
	cmds, err := i.installViaScoop(context.Background(), &PlatformInfo{})
	if err != nil || len(cmds) != 1 {
		t.Fatalf("err=%v cmds=%v", err, cmds)
	}
}

func TestInstallViaScoop_Fails(t *testing.T) {
	r := newFakeRunner().
		withRun("scoop", []string{"install", "ruby"}, errors.New("install failed"))
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions())
	_, err := i.installViaScoop(context.Background(), &PlatformInfo{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInstallViaZypper_Success(t *testing.T) {
	r := newFakeRunner().
		withRun("zypper", []string{"refresh"}, nil).
		withRun("zypper", []string{"install", "-y", "ruby", "ruby-devel"}, nil)
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithDevHeaders(true).WithUpdatePackageIndex(true))
	cmds, err := i.installViaZypper(context.Background(), &PlatformInfo{})
	if err != nil || len(cmds) != 2 {
		t.Fatalf("err=%v cmds=%v", err, cmds)
	}
}

func TestInstallViaZypper_RefreshFailsContinues(t *testing.T) {
	r := newFakeRunner().
		withRun("zypper", []string{"refresh"}, errors.New("refresh failed")).
		withRun("zypper", []string{"install", "-y", "ruby", "ruby-devel"}, nil)
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithDevHeaders(true).WithUpdatePackageIndex(true))
	if _, err := i.installViaZypper(context.Background(), &PlatformInfo{}); err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestInstallViaZypper_InstallFails(t *testing.T) {
	r := newFakeRunner().
		withRun("zypper", []string{"install", "-y", "ruby", "ruby-devel"}, errors.New("install failed"))
	swapRunner(t, r)
	i := newInstallerWith(NewInstallOptions().WithDevHeaders(true).WithUpdatePackageIndex(false))
	_, err := i.installViaZypper(context.Background(), &PlatformInfo{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ============================================================
// Installer.Install — end-to-end via fake runner
// ============================================================

func TestInstall_AlreadyInstalled_Skips(t *testing.T) {
	// Ruby already installed pre-install; ForceReinstall=false -> skip.
	swapOS(t, OSLinux)
	swapArch(t, ArchAMD64)
	swapFileReader(t, memoryFile(map[string]string{"/etc/os-release": "ID=ubuntu"}))
	swapStat(t, statExists(map[string]bool{}))
	r := newFakeRunner().
		withLookPath("apt-get", "/usr/bin/apt-get").
		withLookPath("ruby", "/usr/bin/ruby").
		withLookPath("gem", "/usr/bin/gem").
		withOutput("ruby", "ruby 3.2.2 ()\n", nil).
		withOutput("gem", "3.4.10\n", nil)
	swapRunner(t, r)
	i := NewInstaller(NewInstallOptions().WithBundler(false))
	res, err := i.Install(context.Background())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !res.AlreadyInstalled || res.RubyVersion != "3.2.2" {
		t.Errorf("res=%+v", res)
	}
}

func TestInstall_AptSuccess_WithBundler(t *testing.T) {
	swapOS(t, OSLinux)
	swapArch(t, ArchAMD64)
	swapFileReader(t, memoryFile(map[string]string{"/etc/os-release": "ID=ubuntu"}))
	swapStat(t, statExists(map[string]bool{}))
	r := newFakeRunner().
		withLookPath("apt-get", "/usr/bin/apt-get").
		// ruby is absent during the pre-check (1st LookPath fails) but present
		// during post-install verification (2nd LookPath succeeds).
		withLookPathSeq("ruby", lp("", errNotFound), lp("/usr/bin/ruby", nil)).
		withLookPath("gem", "/usr/bin/gem").
		withOutput("ruby", "ruby 3.2.2 ()\n", nil).
		withOutput("gem", "3.4.10\n", nil).
		withRun("apt-get", []string{"update"}, nil).
		withRun("apt-get", []string{"install", "-y", "ruby", "ruby-dev"}, nil).
		withRun("gem", []string{"install", "bundler"}, nil)
	swapRunner(t, r)
	i := NewInstaller(NewInstallOptions().WithBundler(true).WithDevHeaders(true))
	res, err := i.Install(context.Background())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if res.AlreadyInstalled {
		t.Error("should not be AlreadyInstalled")
	}
	if res.RubyVersion != "3.2.2" {
		t.Errorf("RubyVersion=%s", res.RubyVersion)
	}
	if len(res.CommandsRun) < 2 {
		t.Errorf("CommandsRun=%v", res.CommandsRun)
	}
}

func TestInstall_AptSuccess_BundlerFailsIgnored(t *testing.T) {
	swapOS(t, OSLinux)
	swapArch(t, ArchAMD64)
	swapFileReader(t, memoryFile(map[string]string{"/etc/os-release": "ID=ubuntu"}))
	swapStat(t, statExists(map[string]bool{}))
	r := newFakeRunner().
		withLookPath("apt-get", "/usr/bin/apt-get").
		withLookPathSeq("ruby", lp("", errNotFound), lp("/usr/bin/ruby", nil)).
		withLookPath("gem", "/usr/bin/gem").
		withOutput("ruby", "ruby 3.2.2 ()\n", nil).
		withOutput("gem", "3.4.10\n", nil).
		withRun("apt-get", []string{"update"}, nil).
		withRun("apt-get", []string{"install", "-y", "ruby", "ruby-dev"}, nil).
		withRun("gem", []string{"install", "bundler"}, errors.New("bundler failed"))
	swapRunner(t, r)
	i := NewInstaller(NewInstallOptions().WithBundler(true).WithDevHeaders(true))
	res, err := i.Install(context.Background())
	if err != nil {
		t.Fatalf("bundler failure must not fail Install: %v", err)
	}
	hasComment := false
	for _, c := range res.CommandsRun {
		if len(c) > 0 && c[0] == '#' {
			hasComment = true
		}
	}
	if !hasComment {
		t.Errorf("expected a # comment for bundler failure, CommandsRun=%v", res.CommandsRun)
	}
}

func TestInstall_AptInstallFails(t *testing.T) {
	swapOS(t, OSLinux)
	swapArch(t, ArchAMD64)
	swapFileReader(t, memoryFile(map[string]string{"/etc/os-release": "ID=ubuntu"}))
	swapStat(t, statExists(map[string]bool{}))
	r := newFakeRunner().
		withLookPath("apt-get", "/usr/bin/apt-get").
		withRun("apt-get", []string{"update"}, nil).
		withRun("apt-get", []string{"install", "-y", "ruby", "ruby-dev"}, errors.New("install failed"))
	swapRunner(t, r)
	i := NewInstaller(NewInstallOptions().WithDevHeaders(true))
	_, err := i.Install(context.Background())
	if err == nil {
		t.Fatal("expected install error")
	}
}

func TestInstall_VerificationFails(t *testing.T) {
	// install succeeds but post-install IsInstalled finds no ruby (LookPath fails).
	swapOS(t, OSLinux)
	swapArch(t, ArchAMD64)
	swapFileReader(t, memoryFile(map[string]string{"/etc/os-release": "ID=ubuntu"}))
	swapStat(t, statExists(map[string]bool{}))
	r := newFakeRunner().
		withLookPath("apt-get", "/usr/bin/apt-get").
		withRun("apt-get", []string{"update"}, nil).
		withRun("apt-get", []string{"install", "-y", "ruby", "ruby-dev"}, nil)
	swapRunner(t, r)
	i := NewInstaller(NewInstallOptions().WithBundler(false).WithDevHeaders(true))
	_, err := i.Install(context.Background())
	if err == nil {
		t.Fatal("expected verification error")
	}
}

func TestInstall_PlatformDetectionFails(t *testing.T) {
	swapOS(t, OSUnknown)
	swapArch(t, ArchAMD64)
	swapRunner(t, newFakeRunner())
	i := NewInstaller()
	_, err := i.Install(context.Background())
	if err == nil {
		t.Fatal("expected platform detection error")
	}
}

func TestInstall_UnsupportedPackageManager(t *testing.T) {
	swapOS(t, OSLinux)
	swapArch(t, ArchAMD64)
	swapFileReader(t, memoryFile(map[string]string{"/etc/os-release": "ID=ubuntu"}))
	swapStat(t, statExists(map[string]bool{}))
	r := newFakeRunner()
	swapRunner(t, r)
	// CustomPackageManager set to a value that matches no installVia* case.
	i := NewInstaller(NewInstallOptions().WithCustomPackageManager(PackageManager("weird-pm")))
	_, err := i.Install(context.Background())
	if err == nil {
		t.Fatal("expected unsupported package manager error")
	}
}

func TestInstall_ForceReinstallWhenAlreadyInstalled(t *testing.T) {
	// Already installed, but ForceReinstall=true -> proceeds to install.
	swapOS(t, OSLinux)
	swapArch(t, ArchAMD64)
	swapFileReader(t, memoryFile(map[string]string{"/etc/os-release": "ID=ubuntu"}))
	swapStat(t, statExists(map[string]bool{}))
	r := newFakeRunner().
		withLookPath("apt-get", "/usr/bin/apt-get").
		withLookPath("ruby", "/usr/bin/ruby").
		withLookPath("gem", "/usr/bin/gem").
		withOutput("ruby", "ruby 3.2.2 ()\n", nil).
		withOutput("gem", "3.4.10\n", nil).
		withRun("apt-get", []string{"update"}, nil).
		withRun("apt-get", []string{"install", "-y", "ruby", "ruby-dev"}, nil)
	swapRunner(t, r)
	i := NewInstaller(NewInstallOptions().WithBundler(false).WithDevHeaders(true).WithForceReinstall(true))
	res, err := i.Install(context.Background())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if res.AlreadyInstalled {
		t.Error("ForceReinstall should re-run install, AlreadyInstalled must be false")
	}
}
