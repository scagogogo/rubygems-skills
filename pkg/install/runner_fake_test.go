package install

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// ============================================================
// Fake commandRunner
// ============================================================

// fakeRunner is a programmable commandRunner for unit tests. It records every
// invocation and lets the test script the outcome of LookPath/Run/Output/IsRoot
// per command name.
type fakeRunner struct {
	mu sync.Mutex

	// lookPath maps command name -> (path, err). Missing name => ("", not found).
	lookPath map[string]string

	// lookPathSeq, if set for a name, returns results in order on successive
	// calls (and falls back to lookPath once exhausted). Lets a test make a
	// command "appear" only after some action (e.g. install) completes.
	lookPathSeq map[string][]struct {
		path string
		err  error
	}
	lookPathSeqIdx map[string]int

	// runResults maps "name arg0 arg1..." -> error to return from Run.
	// The special key "" is the fallback for any unscripted command.
	runResults map[string]error

	// outputResults maps command name -> (output, err).
	outputResults map[string]struct {
		out string
		err error
	}

	// isRootValue controls IsRoot().
	isRootValue bool

	// call log
	ranCalls     []string
	lookedUp     []string
	outputCalled []string
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{
		lookPath:       map[string]string{},
		lookPathSeq:    map[string][]struct{ path string; err error }{},
		lookPathSeqIdx: map[string]int{},
		runResults:     map[string]error{},
		outputResults: map[string]struct {
			out string
			err error
		}{},
		isRootValue: false,
	}
}

func (f *fakeRunner) withLookPath(name, path string) *fakeRunner {
	f.lookPath[name] = path
	return f
}

// withLookPathSeq scripts successive LookPath results for a command. The i-th
// call returns the i-th entry; once exhausted, falls back to lookPath.
func (f *fakeRunner) withLookPathSeq(name string, results ...lookPathResult) *fakeRunner {
	out := make([]struct {
		path string
		err  error
	}, len(results))
	for i, r := range results {
		out[i] = struct {
			path string
			err  error
		}{r.path, r.err}
	}
	f.lookPathSeq[name] = out
	return f
}

// lookPathResult is a (path, err) pair for withLookPathSeq scripting.
type lookPathResult struct {
	path string
	err  error
}

// lp builds a lookPathResult.
func lp(path string, err error) lookPathResult { return lookPathResult{path: path, err: err} }

// errNotFound is a sentinel "command not found" error for scripted LookPath.
var errNotFound = fmt.Errorf("command not found")

func (f *fakeRunner) withRun(name string, args []string, err error) *fakeRunner {
	f.runResults[joinCmd(name, args)] = err
	return f
}

func (f *fakeRunner) withRunFallback(err error) *fakeRunner {
	f.runResults[""] = err
	return f
}

func (f *fakeRunner) withOutput(name string, out string, err error) *fakeRunner {
	f.outputResults[name] = struct {
		out string
		err error
	}{out, err}
	return f
}

func (f *fakeRunner) withRoot(v bool) *fakeRunner {
	f.isRootValue = v
	return f
}

func (f *fakeRunner) LookPath(name string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lookedUp = append(f.lookedUp, name)
	if seq, ok := f.lookPathSeq[name]; ok {
		idx := f.lookPathSeqIdx[name]
		if idx < len(seq) {
			f.lookPathSeqIdx[name] = idx + 1
			r := seq[idx]
			if r.err != nil {
				return "", r.err
			}
			return r.path, nil
		}
	}
	if path, ok := f.lookPath[name]; ok {
		return path, nil
	}
	return "", fmt.Errorf("command not found: %s", name)
}

func (f *fakeRunner) Run(ctx context.Context, options *InstallOptions, name string, args ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := joinCmd(name, args)
	f.ranCalls = append(f.ranCalls, key)
	if err, ok := f.runResults[key]; ok {
		return err
	}
	if err, ok := f.runResults[""]; ok {
		return err
	}
	return nil
}

func (f *fakeRunner) Output(name string, args ...string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.outputCalled = append(f.outputCalled, name)
	if r, ok := f.outputResults[name]; ok {
		return r.out, r.err
	}
	return "", nil
}

func (f *fakeRunner) IsRoot() bool {
	return f.isRootValue
}

func joinCmd(name string, args []string) string {
	return name + " " + strings.Join(args, " ")
}

// swapRunner replaces the package-level runner and restores it on test cleanup.
func swapRunner(t *testing.T, r commandRunner) {
	t.Helper()
	prev := runner
	runner = r
	t.Cleanup(func() { runner = prev })
}

// swapFileReader replaces osReadFile and restores it on test cleanup.
func swapFileReader(t *testing.T, read func(string) ([]byte, error)) {
	t.Helper()
	prev := osReadFile
	osReadFile = read
	t.Cleanup(func() { osReadFile = prev })
}

// swapStat replaces osStat and restores it on test cleanup.
func swapStat(t *testing.T, stat func(string) (os.FileInfo, error)) {
	t.Helper()
	prev := osStat
	osStat = stat
	t.Cleanup(func() { osStat = prev })
}

// swapOS replaces detectOSFunc and restores it on test cleanup.
func swapOS(t *testing.T, osVal OperatingSystem) {
	t.Helper()
	prev := detectOSFunc
	detectOSFunc = func() OperatingSystem { return osVal }
	t.Cleanup(func() { detectOSFunc = prev })
}

// swapArch replaces detectArchFunc and restores it on test cleanup.
func swapArch(t *testing.T, arch Architecture) {
	t.Helper()
	prev := detectArchFunc
	detectArchFunc = func() Architecture { return arch }
	t.Cleanup(func() { detectArchFunc = prev })
}

// memoryFile builds an osReadFile that serves the given path->content map,
// returning os.ErrNotExist for anything else.
func memoryFile(files map[string]string) func(string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		if c, ok := files[path]; ok {
			return []byte(c), nil
		}
		return nil, os.ErrNotExist
	}
}

// statExists builds an osStat that reports a regular file for paths in the set
// and os.ErrNotExist otherwise.
func statExists(existing map[string]bool) func(string) (os.FileInfo, error) {
	return func(path string) (os.FileInfo, error) {
		if existing[path] {
			return fakeFileInfo{name: path}, nil
		}
		return nil, os.ErrNotExist
	}
}

// fakeFileInfo is a minimal os.FileInfo for statExists (a file, not a dir).
type fakeFileInfo struct {
	name string
}

func (f fakeFileInfo) Name() string       { return f.name }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return 0644 }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return false }
func (f fakeFileInfo) Sys() interface{}   { return nil }
