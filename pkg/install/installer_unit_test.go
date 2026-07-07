package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

// ============================================================
// InstallOptions fluent setters (the two previously-uncovered ones)
// ============================================================

func TestWithCustomPackageManager_Fluent(t *testing.T) {
	o := NewInstallOptions()
	got := o.WithCustomPackageManager(PMApt)
	if o.CustomPackageManager != PMApt {
		t.Errorf("CustomPackageManager = %s, want apt", o.CustomPackageManager)
	}
	if got != o {
		t.Error("WithCustomPackageManager should return the same options pointer")
	}
}

func TestWithUpdatePackageIndex_Fluent(t *testing.T) {
	o := NewInstallOptions()
	got := o.WithUpdatePackageIndex(false)
	if o.UpdatePackageIndex {
		t.Error("UpdatePackageIndex should be false")
	}
	if got != o {
		t.Error("WithUpdatePackageIndex should return the same options pointer")
	}
}

// ============================================================
// detectOSFrom / detectArchFrom — every branch
// ============================================================

func TestDetectOSFrom_AllBranches(t *testing.T) {
	cases := []struct {
		goos string
		want OperatingSystem
	}{
		{"linux", OSLinux},
		{"darwin", OSDarwin},
		{"windows", OSWindows},
		{"freebsd", OSUnknown}, // default
		{"", OSUnknown},        // default
	}
	for _, c := range cases {
		if got := detectOSFrom(c.goos); got != c.want {
			t.Errorf("detectOSFrom(%q) = %v, want %v", c.goos, got, c.want)
		}
	}
}

func TestDetectArchFrom_AllBranches(t *testing.T) {
	cases := []struct {
		goarch string
		want   Architecture
	}{
		{"amd64", ArchAMD64},
		{"arm64", ArchARM64},
		{"arm", ArchARM},
		{"386", Arch386},
		{"mips", ArchUnknown}, // default
		{"", ArchUnknown},     // default
	}
	for _, c := range cases {
		if got := detectArchFrom(c.goarch); got != c.want {
			t.Errorf("detectArchFrom(%q) = %v, want %v", c.goarch, got, c.want)
		}
	}
}

// ============================================================
// readOSReleaseFrom — every distro branch (ID + ID_LIKE + unknown)
// ============================================================

func TestReadOSReleaseFrom_AllDistros(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    LinuxDistro
	}{
		{"ubuntu", `ID=ubuntu`, DistroUbuntu},
		{"debian", `ID=debian`, DistroDebian},
		{"centos", `ID=centos`, DistroCentOS},
		{"rhel", `ID=rhel`, DistroRHEL},
		{"fedora", `ID=fedora`, DistroFedora},
		{"rocky", `ID=rocky`, DistroRocky},
		{"alma", `ID=alma`, DistroAlma},
		{"almalinux", `ID=almalinux`, DistroAlma},
		{"alpine", `ID=alpine`, DistroAlpine},
		{"arch", `ID=arch`, DistroArch},
		{"manjaro", `ID=manjaro`, DistroManjaro},
		{"amzn", `ID=amzn`, DistroAmazon},
		{"amazon", `ID=amazon`, DistroAmazon},
		{"opensuse", `ID=opensuse`, DistroOpenSUSE},
		{"opensuse-leap", `ID=opensuse-leap`, DistroOpenSUSE},
		{"opensuse-tumbleweed", `ID=opensuse-tumbleweed`, DistroOpenSUSE},
		// ID_LIKE inference
		{"id_like_debian", `ID=unknown\nID_LIKE=debian`, DistroDebian},
		{"id_like_rhel", `ID=unknown\nID_LIKE=rhel`, DistroCentOS},
		{"id_like_fedora", `ID=unknown\nID_LIKE=fedora`, DistroCentOS},
		{"id_like_arch", `ID=unknown\nID_LIKE=arch`, DistroArch},
		{"id_like_suse", `ID=unknown\nID_LIKE=suse`, DistroOpenSUSE},
		// quoted variants
		{"quoted", `NAME="Ubuntu"\nID="ubuntu"`, DistroUbuntu},
		// unknown
		{"unknown", `ID=gentoo\nID_LIKE=`, DistroUnknown},
	}
	read := memoryFile(map[string]string{"/etc/os-release": ""})
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := memoryFile(map[string]string{"/etc/os-release": strings.ReplaceAll(c.content, `\n`, "\n")})
			if got := readOSReleaseFrom("/etc/os-release", r); got != c.want {
				t.Errorf("readOSReleaseFrom = %v, want %v (content=%q)", got, c.want, c.content)
			}
		})
	}
	_ = read // keep memoryFile referenced for the unused-first var
}

func TestReadOSReleaseFrom_ReadError(t *testing.T) {
	read := func(string) ([]byte, error) { return nil, os.ErrNotExist }
	if got := readOSReleaseFrom("/etc/os-release", read); got != DistroUnknown {
		t.Errorf("readOSReleaseFrom on read error = %v, want DistroUnknown", got)
	}
}

// readOSRelease (the thin wrapper) is covered end-to-end via osReadFile swap.
func TestReadOSRelease_Wrapper(t *testing.T) {
	swapFileReader(t, memoryFile(map[string]string{"/etc/os-release": "ID=ubuntu"}))
	if got := readOSRelease(); got != DistroUbuntu {
		t.Errorf("readOSRelease() = %v, want DistroUbuntu", got)
	}
}

// ============================================================
// checkDistroFilesAt — every branch
// ============================================================

func TestCheckDistroFilesAt_AllBranches(t *testing.T) {
	cases := []struct {
		name    string
		files   map[string]string
		exists  map[string]bool
		want    LinuxDistro
	}{
		{"debian_version", nil, map[string]bool{"/etc/debian_version": true}, DistroDebian},
		{"redhat_centos", map[string]string{"/etc/redhat-release": "CentOS Linux 8"}, map[string]bool{"/etc/redhat-release": true}, DistroCentOS},
		{"redhat_fedora", map[string]string{"/etc/redhat-release": "Fedora 39"}, map[string]bool{"/etc/redhat-release": true}, DistroFedora},
		{"redhat_rocky", map[string]string{"/etc/redhat-release": "Rocky Linux 9"}, map[string]bool{"/etc/redhat-release": true}, DistroRocky},
		{"redhat_alma", map[string]string{"/etc/redhat-release": "AlmaLinux 9"}, map[string]bool{"/etc/redhat-release": true}, DistroAlma},
		{"redhat_rhel", map[string]string{"/etc/redhat-release": "Red Hat Enterprise Linux 9"}, map[string]bool{"/etc/redhat-release": true}, DistroRHEL},
		{"redhat_default_centos", map[string]string{"/etc/redhat-release": "Something Else"}, map[string]bool{"/etc/redhat-release": true}, DistroCentOS},
		{"redhat_read_error", nil, map[string]bool{"/etc/redhat-release": true}, DistroCentOS}, // read fails -> default CentOS
		{"alpine", nil, map[string]bool{"/etc/alpine-release": true}, DistroAlpine},
		{"arch", nil, map[string]bool{"/etc/arch-release": true}, DistroArch},
		{"amazon", map[string]string{"/etc/system-release": "Amazon Linux 2"}, map[string]bool{"/etc/system-release": true}, DistroAmazon},
		{"system_release_not_amazon", map[string]string{"/etc/system-release": "CentOS Linux 8"}, map[string]bool{"/etc/system-release": true}, DistroUnknown},
		{"none", nil, map[string]bool{}, DistroUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			read := memoryFile(c.files)
			// make read fail for the "redhat_read_error" case where content absent
			if c.files == nil {
				read = func(string) ([]byte, error) { return nil, os.ErrNotExist }
			}
			exists := func(path string) bool { return c.exists[path] }
			if got := checkDistroFilesAt(read, exists); got != c.want {
				t.Errorf("checkDistroFilesAt = %v, want %v", got, c.want)
			}
		})
	}
}

// checkDistroFiles (wrapper) covered via osStat/osReadFile swap.
func TestCheckDistroFiles_Wrapper(t *testing.T) {
	swapStat(t, statExists(map[string]bool{"/etc/debian_version": true}))
	if got := checkDistroFiles(); got != DistroDebian {
		t.Errorf("checkDistroFiles() = %v, want DistroDebian", got)
	}
}

// ============================================================
// inferFromPackageManager — every branch
// ============================================================

func TestInferFromPackageManager_AllBranches(t *testing.T) {
	cases := []struct {
		name    string
		cmds    []string // commands that "exist"
		want    LinuxDistro
	}{
		{"apt-get", []string{"apt-get"}, DistroDebian},
		{"apt", []string{"apt"}, DistroDebian},
		{"dnf", []string{"dnf"}, DistroFedora},
		{"yum", []string{"yum"}, DistroCentOS},
		{"apk", []string{"apk"}, DistroAlpine},
		{"pacman", []string{"pacman"}, DistroArch},
		{"zypper", []string{"zypper"}, DistroOpenSUSE},
		{"none", []string{}, DistroUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := newFakeRunner()
			for _, cmd := range c.cmds {
				r.withLookPath(cmd, "/usr/bin/"+cmd)
			}
			swapRunner(t, r)
			if got := inferFromPackageManager(); got != c.want {
				t.Errorf("inferFromPackageManager = %v, want %v", got, c.want)
			}
		})
	}
}

// ============================================================
// detectLinuxDistro — three methods + failure
// ============================================================

func TestDetectLinuxDistro_FromOSRelease(t *testing.T) {
	swapFileReader(t, memoryFile(map[string]string{"/etc/os-release": "ID=ubuntu"}))
	swapStat(t, statExists(map[string]bool{}))
	if d, err := detectLinuxDistro(); d != DistroUbuntu || err != nil {
		t.Fatalf("detectLinuxDistro = %v, %v, want ubuntu nil", d, err)
	}
}

func TestDetectLinuxDistro_FromDistroFiles(t *testing.T) {
	// os-release absent/unparseable, fall to distro files
	swapFileReader(t, memoryFile(map[string]string{})) // /etc/os-release missing -> Unknown
	swapStat(t, statExists(map[string]bool{"/etc/debian_version": true}))
	if d, err := detectLinuxDistro(); d != DistroDebian || err != nil {
		t.Fatalf("detectLinuxDistro = %v, %v, want debian nil", d, err)
	}
}

func TestDetectLinuxDistro_FromPackageManager(t *testing.T) {
	// os-release missing, no distro files, fall to package manager inference
	swapFileReader(t, memoryFile(map[string]string{}))
	swapStat(t, statExists(map[string]bool{}))
	r := newFakeRunner().withLookPath("apt-get", "/usr/bin/apt-get")
	swapRunner(t, r)
	if d, err := detectLinuxDistro(); d != DistroDebian || err != nil {
		t.Fatalf("detectLinuxDistro = %v, %v, want debian nil", d, err)
	}
}

func TestDetectLinuxDistro_Failure(t *testing.T) {
	swapFileReader(t, memoryFile(map[string]string{}))
	swapStat(t, statExists(map[string]bool{}))
	swapRunner(t, newFakeRunner()) // nothing exists
	d, err := detectLinuxDistro()
	if d != DistroUnknown || err == nil {
		t.Fatalf("detectLinuxDistro = %v, %v, want DistroUnknown + error", d, err)
	}
}

// ============================================================
// detectPackageManager (OS dispatch) + darwin/windows/bycommand
// ============================================================

func TestDetectPackageManager_AllOSBranches(t *testing.T) {
	// Linux
	r := newFakeRunner().withLookPath("apt-get", "/usr/bin/apt-get")
	swapRunner(t, r)
	pm, _, err := detectPackageManager(&PlatformInfo{OS: OSLinux, Distro: DistroUbuntu})
	if err != nil || pm != PMApt {
		t.Fatalf("linux: pm=%s err=%v", pm, err)
	}

	// Darwin
	r = newFakeRunner().withLookPath("brew", "/usr/local/bin/brew")
	swapRunner(t, r)
	pm, _, err = detectPackageManager(&PlatformInfo{OS: OSDarwin})
	if err != nil || pm != PMBrew {
		t.Fatalf("darwin: pm=%s err=%v", pm, err)
	}

	// Windows (choco)
	r = newFakeRunner().withLookPath("choco", "/usr/bin/choco")
	swapRunner(t, r)
	pm, _, err = detectPackageManager(&PlatformInfo{OS: OSWindows})
	if err != nil || pm != PMChoco {
		t.Fatalf("windows choco: pm=%s err=%v", pm, err)
	}

	// Windows (scoop)
	r = newFakeRunner().withLookPath("scoop", "/usr/bin/scoop")
	swapRunner(t, r)
	pm, _, err = detectPackageManager(&PlatformInfo{OS: OSWindows})
	if err != nil || pm != PMScoop {
		t.Fatalf("windows scoop: pm=%s err=%v", pm, err)
	}

	// Unknown OS
	_, _, err = detectPackageManager(&PlatformInfo{OS: OSUnknown})
	if err == nil {
		t.Fatal("unknown OS should return error")
	}
}

func TestDetectDarwinPackageManager_NotFound(t *testing.T) {
	swapRunner(t, newFakeRunner()) // no brew
	_, _, err := detectDarwinPackageManager()
	if err == nil {
		t.Fatal("expected error when brew missing")
	}
}

func TestDetectWindowsPackageManager_NotFound(t *testing.T) {
	swapRunner(t, newFakeRunner()) // no choco/scoop
	_, _, err := detectWindowsPackageManager()
	if err == nil {
		t.Fatal("expected error when choco/scoop missing")
	}
}

func TestDetectPackageManagerByCommand_AllCandidates(t *testing.T) {
	// First match wins in candidate order: apt-get, apt, dnf, yum, apk, pacman, zypper.
	r := newFakeRunner().
		withLookPath("apt-get", "/usr/bin/apt-get")
	swapRunner(t, r)
	pm, cmd, err := detectPackageManagerByCommand()
	if err != nil || pm != PMApt || cmd == "" {
		t.Fatalf("apt-get: pm=%s cmd=%s err=%v", pm, cmd, err)
	}

	// apt only
	r = newFakeRunner().withLookPath("apt", "/usr/bin/apt")
	swapRunner(t, r)
	pm, _, err = detectPackageManagerByCommand()
	if err != nil || pm != PMApt {
		t.Fatalf("apt: pm=%s err=%v", pm, err)
	}

	// dnf
	r = newFakeRunner().withLookPath("dnf", "/usr/bin/dnf")
	swapRunner(t, r)
	pm, _, _ = detectPackageManagerByCommand()
	if pm != PMDnf {
		t.Fatalf("dnf: pm=%s", pm)
	}

	// yum
	r = newFakeRunner().withLookPath("yum", "/usr/bin/yum")
	swapRunner(t, r)
	pm, _, _ = detectPackageManagerByCommand()
	if pm != PMYum {
		t.Fatalf("yum: pm=%s", pm)
	}

	// apk
	r = newFakeRunner().withLookPath("apk", "/usr/bin/apk")
	swapRunner(t, r)
	pm, _, _ = detectPackageManagerByCommand()
	if pm != PMApk {
		t.Fatalf("apk: pm=%s", pm)
	}

	// pacman
	r = newFakeRunner().withLookPath("pacman", "/usr/bin/pacman")
	swapRunner(t, r)
	pm, _, _ = detectPackageManagerByCommand()
	if pm != PMPacman {
		t.Fatalf("pacman: pm=%s", pm)
	}

	// zypper
	r = newFakeRunner().withLookPath("zypper", "/usr/bin/zypper")
	swapRunner(t, r)
	pm, _, _ = detectPackageManagerByCommand()
	if pm != PMZypper {
		t.Fatalf("zypper: pm=%s", pm)
	}

	// none
	swapRunner(t, newFakeRunner())
	_, _, err = detectPackageManagerByCommand()
	if err == nil {
		t.Fatal("expected error when no PM command found")
	}
}

// ============================================================
// detectLinuxPackageManager — the default (unknown) branch routes to
// detectPackageManagerByCommand.
// ============================================================

func TestDetectLinuxPackageManager_UnknownDistroByCommand(t *testing.T) {
	r := newFakeRunner().withLookPath("apt-get", "/usr/bin/apt-get")
	swapRunner(t, r)
	pm, _, err := detectLinuxPackageManager(DistroUnknown)
	if err != nil || pm != PMApt {
		t.Fatalf("unknown distro: pm=%s err=%v", pm, err)
	}
}

func TestDetectLinuxPackageManager_CommandNotFound(t *testing.T) {
	// Known distro but its command is missing -> error.
	swapRunner(t, newFakeRunner())
	_, _, err := detectLinuxPackageManager(DistroUbuntu)
	if err == nil {
		t.Fatal("expected error when apt-get/apt missing for ubuntu")
	}
}

func TestDetectLinuxPackageManager_FedoraFallsBackToYum(t *testing.T) {
	// Fedora prefers dnf; when dnf missing, falls back to yum.
	r := newFakeRunner().withLookPath("yum", "/usr/bin/yum")
	swapRunner(t, r)
	pm, _, err := detectLinuxPackageManager(DistroFedora)
	if err != nil || pm != PMYum {
		t.Fatalf("fedora fallback: pm=%s err=%v", pm, err)
	}
}

// ============================================================
// checkRubyInstalled / IsInstalled
// ============================================================

func TestCheckRubyInstalled_SuccessWithGem(t *testing.T) {
	r := newFakeRunner().
		withLookPath("ruby", "/usr/bin/ruby").
		withLookPath("gem", "/usr/bin/gem").
		withOutput("ruby", "ruby 3.2.2 (2023-03-30) [x86_64-linux]\n", nil).
		withOutput("gem", "3.4.10\n", nil)
	swapRunner(t, r)
	installed, info, err := checkRubyInstalled()
	if err != nil || !installed || info == nil {
		t.Fatalf("installed=%v info=%v err=%v", installed, info, err)
	}
	if info.RubyVersion != "3.2.2" {
		t.Errorf("RubyVersion=%s want 3.2.2", info.RubyVersion)
	}
	if info.GemVersion != "3.4.10" {
		t.Errorf("GemVersion=%s want 3.4.10", info.GemVersion)
	}
	if info.RubyPath != "/usr/bin/ruby" || info.GemPath != "/usr/bin/gem" {
		t.Errorf("paths: ruby=%s gem=%s", info.RubyPath, info.GemPath)
	}
}

func TestCheckRubyInstalled_SuccessNoGem(t *testing.T) {
	r := newFakeRunner().
		withLookPath("ruby", "/usr/bin/ruby").
		// gem not registered -> LookPath fails -> gemPath "" -> gemVersion ""
		withOutput("ruby", "ruby 3.1.0 ()\n", nil)
	swapRunner(t, r)
	installed, info, err := checkRubyInstalled()
	if err != nil || !installed {
		t.Fatalf("installed=%v err=%v", installed, err)
	}
	if info.GemPath != "" || info.GemVersion != "" {
		t.Errorf("expected empty gem info, got path=%s ver=%s", info.GemPath, info.GemVersion)
	}
	if info.RubyVersion != "3.1.0" {
		t.Errorf("RubyVersion=%s want 3.1.0", info.RubyVersion)
	}
}

func TestCheckRubyInstalled_RubyNotFound(t *testing.T) {
	swapRunner(t, newFakeRunner()) // no ruby
	installed, info, err := checkRubyInstalled()
	if err != nil || installed || info != nil {
		t.Fatalf("installed=%v info=%v err=%v", installed, info, err)
	}
}

func TestCheckRubyInstalled_RubyVersionFails(t *testing.T) {
	// ruby is on PATH but `ruby --version` fails -> returns (false, nil, err).
	r := newFakeRunner().
		withLookPath("ruby", "/usr/bin/ruby").
		withOutput("ruby", "", errors.New("version error"))
	swapRunner(t, r)
	installed, info, err := checkRubyInstalled()
	if err == nil || installed || info != nil {
		t.Fatalf("installed=%v info=%v err=%v", installed, info, err)
	}
}

func TestInstallerIsInstalled_Delegates(t *testing.T) {
	r := newFakeRunner().
		withLookPath("ruby", "/usr/bin/ruby").
		withLookPath("gem", "/usr/bin/gem").
		withOutput("ruby", "ruby 3.2.2\n", nil).
		withOutput("gem", "3.4.10\n", nil)
	swapRunner(t, r)
	i := NewInstaller()
	installed, info, err := i.IsInstalled()
	if err != nil || !installed || info == nil {
		t.Fatalf("IsInstalled: installed=%v info=%v err=%v", installed, info, err)
	}
}

// keep imports referenced
var _ = context.Background
var _ = fmt.Sprintf
