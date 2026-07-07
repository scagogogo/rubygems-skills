package install

import (
	"context"
	"errors"
	"testing"
)

// ============================================================
// detectLinuxPackageManager — cover every distro branch
// ============================================================

func TestDetectLinuxPackageManager_AllDistros(t *testing.T) {
	cases := []struct {
		name    string
		distro  LinuxDistro
		cmds    []string // commands to make "exist"
		wantPM  PackageManager
		wantErr bool
	}{
		{"ubuntu", DistroUbuntu, []string{"apt-get"}, PMApt, false},
		{"debian", DistroDebian, []string{"apt"}, PMApt, false}, // apt fallback
		{"centos", DistroCentOS, []string{"yum"}, PMYum, false},
		{"rhel", DistroRHEL, []string{"yum"}, PMYum, false},
		{"amazon", DistroAmazon, []string{"yum"}, PMYum, false},
		{"fedora_dnf", DistroFedora, []string{"dnf"}, PMDnf, false},
		{"rocky_dnf", DistroRocky, []string{"dnf"}, PMDnf, false},
		{"alma_dnf", DistroAlma, []string{"dnf"}, PMDnf, false},
		{"alpine", DistroAlpine, []string{"apk"}, PMApk, false},
		{"arch", DistroArch, []string{"pacman"}, PMPacman, false},
		{"manjaro", DistroManjaro, []string{"pacman"}, PMPacman, false},
		{"opensuse", DistroOpenSUSE, []string{"zypper"}, PMZypper, false},
		{"missing_cmd", DistroUbuntu, []string{}, PMApt, true}, // apt-get/apt absent
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := newFakeRunner()
			for _, cmd := range c.cmds {
				r.withLookPath(cmd, "/usr/bin/"+cmd)
			}
			swapRunner(t, r)
			pm, _, err := detectLinuxPackageManager(c.distro)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got pm=%s", pm)
				}
				return
			}
			if err != nil || pm != c.wantPM {
				t.Fatalf("pm=%s err=%v, want %s", pm, err, c.wantPM)
			}
		})
	}
}

// ============================================================
// isRunningAsRoot (package func) — delegates to runner.IsRoot
// ============================================================

func TestIsRunningAsRoot_TrueAndFalse(t *testing.T) {
	r := newFakeRunner().withRoot(true)
	swapRunner(t, r)
	if !isRunningAsRoot() {
		t.Error("isRunningAsRoot() should be true when runner.IsRoot()=true")
	}

	r2 := newFakeRunner().withRoot(false)
	swapRunner(t, r2)
	if isRunningAsRoot() {
		t.Error("isRunningAsRoot() should be false when runner.IsRoot()=false")
	}
}

// ============================================================
// osRunner.Run — the sudo branch (needSudo=true)
// ============================================================

func TestOSRunner_Run_SudoBranch(t *testing.T) {
	// Swap runner to a fake that reports NOT root, so osRunner.Run computes
	// needSudo=true for apt-get (isRootRequired). osRunner.Run still executes
	// the real sudo command (which may fail if sudo is absent in CI); either
	// outcome covers the sudo-construction branch.
	swapRunner(t, newFakeRunner().withRoot(false))
	opts := NewInstallOptions().WithSudo(true).WithTimeout(5)
	// We don't assert on success/failure — only that the sudo branch ran.
	_ = osRunner{}.Run(context.Background(), opts, "apt-get", "--version")
}

// ============================================================
// DetectPlatform — the Linux distro-detection-error branch
// ============================================================

func TestDetectPlatform_LinuxDistroErrorBranch(t *testing.T) {
	// Linux, os-release missing, no distro files, no PM command ->
	// detectLinuxDistro returns err -> info.Distro=DistroUnknown, then
	// detectPackageManager -> detectLinuxPackageManager(Unknown) ->
	// detectPackageManagerByCommand -> no command -> error.
	swapOS(t, OSLinux)
	swapArch(t, ArchAMD64)
	swapFileReader(t, memoryFile(map[string]string{})) // /etc/os-release missing
	swapStat(t, statExists(map[string]bool{}))         // no distro files
	swapRunner(t, newFakeRunner())                     // no PM command
	_, err := NewInstaller().DetectPlatform()
	if err == nil {
		t.Fatal("expected error when distro detection and PM detection both fail")
	}
}

// ============================================================
// Installer.Install — every package manager end-to-end
// Each wires a Linux platform whose detected PM matches the installer path,
// scripts install success, and verifies the post-install check finds ruby.
// ============================================================

func installPlatformFor(pm PackageManager, distroID string) (string, []string) {
	// returns the os-release content + the install Run() command args
	switch pm {
	case PMApt:
		return "ID=ubuntu", []string{"install", "-y", "ruby", "ruby-dev"}
	case PMYum:
		return "ID=centos", []string{"install", "-y", "ruby", "ruby-devel"}
	case PMDnf:
		return "ID=fedora", []string{"install", "-y", "ruby", "ruby-devel"}
	case PMApk:
		return "ID=alpine", []string{"add", "ruby", "ruby-dev"}
	case PMPacman:
		return "ID=arch", []string{"-S", "--noconfirm", "ruby"}
	case PMZypper:
		return "ID=opensuse", []string{"install", "-y", "ruby", "ruby-devel"}
	}
	return "", nil
}

func TestInstall_AllPackageManagers(t *testing.T) {
	// apt/yum/dnf/apk/pacman/zypper each derive from a Linux distro. brew/choco/
	// scoop require non-Linux OS, covered separately below.
	cases := []struct {
		name string
		pm   PackageManager
		cmd  string // package-manager command name
	}{
		{"apt", PMApt, "apt-get"},
		{"yum", PMYum, "yum"},
		{"dnf", PMDnf, "dnf"},
		{"apk", PMApk, "apk"},
		{"pacman", PMPacman, "pacman"},
		{"zypper", PMZypper, "zypper"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			swapOS(t, OSLinux)
			swapArch(t, ArchAMD64)
			osRelease, installArgs := installPlatformFor(c.pm, "")
			swapFileReader(t, memoryFile(map[string]string{"/etc/os-release": osRelease}))
			swapStat(t, statExists(map[string]bool{}))
			r := newFakeRunner().
				withLookPath(c.cmd, "/usr/bin/"+c.cmd).
				// ruby absent pre-check, present post-check
				withLookPathSeq("ruby", lp("", errNotFound), lp("/usr/bin/ruby", nil)).
				withLookPath("gem", "/usr/bin/gem").
				withOutput("ruby", "ruby 3.2.2 ()\n", nil).
				withOutput("gem", "3.4.10\n", nil).
				// the install Run; use UpdatePackageIndex=false so only one Run per PM
				withRun(c.cmd, installArgs, nil)
			// yum/dnf/apk/pacman/zypper with UpdatePackageIndex=false skip makecache
			swapRunner(t, r)
			opts := NewInstallOptions().WithBundler(false).WithDevHeaders(true).WithUpdatePackageIndex(false)
			res, err := NewInstaller(opts).Install(context.Background())
			if err != nil {
				t.Fatalf("err=%v", err)
			}
			if res.AlreadyInstalled {
				t.Error("should not be AlreadyInstalled")
			}
			if res.PackageManager != c.pm {
				t.Errorf("PackageManager=%s want %s", res.PackageManager, c.pm)
			}
		})
	}
}

func TestInstall_Brew(t *testing.T) {
	swapOS(t, OSDarwin)
	swapArch(t, ArchARM64)
	r := newFakeRunner().
		withLookPath("brew", "/usr/local/bin/brew").
		withLookPathSeq("ruby", lp("", errNotFound), lp("/usr/bin/ruby", nil)).
		withLookPath("gem", "/usr/bin/gem").
		withOutput("ruby", "ruby 3.2.2 ()\n", nil).
		withOutput("gem", "3.4.10\n", nil).
		withRun("brew", []string{"install", "ruby"}, nil)
	swapRunner(t, r)
	opts := NewInstallOptions().WithBundler(false).WithUpdatePackageIndex(false)
	res, err := NewInstaller(opts).Install(context.Background())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if res.PackageManager != PMBrew {
		t.Errorf("PackageManager=%s want brew", res.PackageManager)
	}
}

func TestInstall_Choco(t *testing.T) {
	swapOS(t, OSWindows)
	swapArch(t, ArchAMD64)
	r := newFakeRunner().
		withLookPath("choco", "/usr/bin/choco").
		withLookPathSeq("ruby", lp("", errNotFound), lp("/usr/bin/ruby", nil)).
		withLookPath("gem", "/usr/bin/gem").
		withOutput("ruby", "ruby 3.2.2 ()\n", nil).
		withOutput("gem", "3.4.10\n", nil).
		withRun("choco", []string{"install", "-y", "ruby"}, nil)
	swapRunner(t, r)
	opts := NewInstallOptions().WithBundler(false)
	res, err := NewInstaller(opts).Install(context.Background())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if res.PackageManager != PMChoco {
		t.Errorf("PackageManager=%s want choco", res.PackageManager)
	}
}

func TestInstall_Scoop(t *testing.T) {
	swapOS(t, OSWindows)
	swapArch(t, ArchAMD64)
	r := newFakeRunner().
		withLookPath("scoop", "/usr/bin/scoop").
		withLookPathSeq("ruby", lp("", errNotFound), lp("/usr/bin/ruby", nil)).
		withLookPath("gem", "/usr/bin/gem").
		withOutput("ruby", "ruby 3.2.2 ()\n", nil).
		withOutput("gem", "3.4.10\n", nil).
		withRun("scoop", []string{"install", "ruby"}, nil)
	swapRunner(t, r)
	opts := NewInstallOptions().WithBundler(false)
	res, err := NewInstaller(opts).Install(context.Background())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if res.PackageManager != PMScoop {
		t.Errorf("PackageManager=%s want scoop", res.PackageManager)
	}
}

// keep imports referenced
var _ = errors.New

// TestInstall_PostInstallVerificationError covers the verifyErr != nil branch
// of Install: install succeeds, but the post-install checkRubyInstalled returns
// an error (ruby found on PATH but `ruby --version` fails).
func TestInstall_PostInstallVerificationError(t *testing.T) {
	swapOS(t, OSLinux)
	swapArch(t, ArchAMD64)
	swapFileReader(t, memoryFile(map[string]string{"/etc/os-release": "ID=ubuntu"}))
	swapStat(t, statExists(map[string]bool{}))
	r := newFakeRunner().
		withLookPath("apt-get", "/usr/bin/apt-get").
		// pre-check: ruby not found -> (false, nil, nil) -> proceed to install.
		withLookPathSeq("ruby", lp("", errNotFound), lp("/usr/bin/ruby", nil)).
		// post-install verification: ruby found, but `--version` fails -> error.
		// (pre-check never calls getCommandOutput since LookPath failed first,
		// so this single failing result is consumed by the verification call.)
		withOutput("ruby", "", errors.New("version boom")).
		withRun("apt-get", []string{"update"}, nil).
		withRun("apt-get", []string{"install", "-y", "ruby", "ruby-dev"}, nil)
	swapRunner(t, r)
	opts := NewInstallOptions().WithBundler(false).WithDevHeaders(true)
	_, err := NewInstaller(opts).Install(context.Background())
	if err == nil {
		t.Fatal("expected post-install verification error")
	}
}
