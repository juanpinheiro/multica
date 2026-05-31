package inplace_test

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/multica-ai/multica/server/internal/workspace/inplace"
)

func TestValidateDirAcceptsProjectDir(t *testing.T) {
	dir := t.TempDir()
	if err := inplace.ValidateDir(dir); err != nil {
		t.Fatalf("ValidateDir(%q) = %v, want nil", dir, err)
	}
}

// wantReason asserts ValidateDir(path) rejected with a *RejectedError carrying
// the expected reason.
func wantReason(t *testing.T, path string, reason inplace.Reason) {
	t.Helper()
	err := inplace.ValidateDir(path)
	if err == nil {
		t.Fatalf("ValidateDir(%q) = nil, want rejection %q", path, reason)
	}
	var rejected *inplace.RejectedError
	if !errors.As(err, &rejected) {
		t.Fatalf("ValidateDir(%q) error = %T, want *inplace.RejectedError", path, err)
	}
	if rejected.Reason != reason {
		t.Fatalf("ValidateDir(%q) reason = %q, want %q", path, rejected.Reason, reason)
	}
}

func TestValidateDirRejectsRelativePath(t *testing.T) {
	wantReason(t, filepath.Join("some", "project"), inplace.ReasonNotAbsolute)
}

func TestValidateDirRejectsNonExistent(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	wantReason(t, missing, inplace.ReasonNotExist)
}

func TestValidateDirRejectsFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "a-file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	wantReason(t, file, inplace.ReasonNotDir)
}

func TestValidateDirRejectsHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	wantReason(t, home, inplace.ReasonHomeDir)
}

func TestValidateDirRejectsRoots(t *testing.T) {
	var roots []string
	if runtime.GOOS == "windows" {
		wd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		vol := filepath.VolumeName(wd)
		roots = []string{`C:\`, vol + `\`, `\\server\share`}
	} else {
		roots = []string{"/", "/home", "/root", "/etc"}
	}
	for _, root := range roots {
		wantReason(t, root, inplace.ReasonSystemRoot)
	}
}

func TestValidateDirRejectsNonWritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory write permission is not enforced for the owner on Windows")
	}
	dir := filepath.Join(t.TempDir(), "readonly")
	if err := os.Mkdir(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	wantReason(t, dir, inplace.ReasonNotWritable)
}

func TestValidateDirRejectsSymlinkToBanned(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	link := filepath.Join(t.TempDir(), "link-to-home")
	// A true symlink (not a Windows junction, which EvalSymlinks does not
	// follow); needs privilege on Windows, so skip gracefully when denied.
	if err := os.Symlink(home, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	wantReason(t, link, inplace.ReasonHomeDir)
}
