package manifest

import (
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// FS is the minimal filesystem interface for the resolver.
// Only ReadFile is needed; path manipulation uses filepath from stdlib.
type FS interface {
	ReadFile(path string) ([]byte, error)
}

// RepoEntry describes a single repository in the manifest.
type RepoEntry struct {
	Name   string `toml:"name"`
	Path   string `toml:"path"`
	Remote string `toml:"remote"`
}

// Manifest is the decoded content of a .multica/workspace.toml file.
//
// Mode holds the raw execution-mode string as written in the file (empty when
// absent). It is normalized to a known value via execmode.Normalize by the
// layer that projects it onto the server, so the unknown-value warning is
// raised once, where a logger is available.
type Manifest struct {
	Workspace string      `toml:"workspace"`
	Mode      string      `toml:"mode"`
	Repos     []RepoEntry `toml:"repo"`
}

const (
	dotMultica   = ".multica"
	manifestFile = "workspace.toml"
)

// Find walks up from startDir until a .multica/workspace.toml file is found.
// Returns the manifest file path and true on success, or ("", false) at the root.
func Find(startDir string, fs FS) (string, bool) {
	dir := filepath.Clean(startDir)
	for {
		candidate := filepath.Join(dir, dotMultica, manifestFile)
		if _, err := fs.ReadFile(candidate); err == nil {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// Parse decodes the raw bytes of a workspace.toml file.
func Parse(data []byte) (Manifest, error) {
	var m Manifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// ManifestDir returns the umbrella root directory for a manifest at the given path.
// Use this to resolve relative repo paths: filepath.Join(ManifestDir(path), repo.Path).
func ManifestDir(manifestPath string) string {
	return filepath.Dir(filepath.Dir(manifestPath))
}
