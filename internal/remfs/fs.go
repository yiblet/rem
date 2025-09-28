package remfs

import (
	"io/fs"
	"os"
	"path/filepath"
)

const (
	ConfigDir = ".config/rem"
)

// RemFS is a filesystem rooted at the rem configuration directory
type RemFS struct {
	root string
}

// New creates a new RemFS rooted at ~/.config/rem/
func New() (*RemFS, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ConfigDir)

	// Ensure the directory exists
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return nil, err
	}

	return &RemFS{root: configPath}, nil
}

// NewWithRoot creates a RemFS with a custom root (for testing)
func NewWithRoot(root string) *RemFS {
	return &RemFS{root: root}
}

// Open implements fs.FS
func (rfs *RemFS) Open(name string) (fs.File, error) {
	// Validate the path
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	fullPath := filepath.Join(rfs.root, name)
	return os.Open(fullPath)
}

// ReadDir implements fs.ReadDirFS
func (rfs *RemFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}

	fullPath := filepath.Join(rfs.root, name)
	return os.ReadDir(fullPath)
}

// WriteFile writes data to a file relative to the rem config directory
func (rfs *RemFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	if !fs.ValidPath(name) {
		return &fs.PathError{Op: "writefile", Path: name, Err: fs.ErrInvalid}
	}

	fullPath := filepath.Join(rfs.root, name)
	dir := filepath.Dir(fullPath)

	// Ensure parent directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, data, perm)
}

// Remove removes a file relative to the rem config directory
func (rfs *RemFS) Remove(name string) error {
	if !fs.ValidPath(name) {
		return &fs.PathError{Op: "remove", Path: name, Err: fs.ErrInvalid}
	}

	fullPath := filepath.Join(rfs.root, name)
	return os.Remove(fullPath)
}

// MkdirAll creates directories relative to the rem config directory
func (rfs *RemFS) MkdirAll(name string, perm os.FileMode) error {
	if !fs.ValidPath(name) {
		return &fs.PathError{Op: "mkdirall", Path: name, Err: fs.ErrInvalid}
	}

	fullPath := filepath.Join(rfs.root, name)
	return os.MkdirAll(fullPath, perm)
}

// Root returns the root directory path
func (rfs *RemFS) Root() string {
	return rfs.root
}
