package remfs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	ConfigDir      = ".config/rem"
	DefaultHistDir = "history" // New default (was "content")
	LegacyHistDir  = "content" // Legacy directory name
)

// RemFS is a filesystem rooted at the rem configuration directory
type RemFS struct {
	root string
}

// New creates a new RemFS rooted at ~/.config/rem/
func New() (*RemFS, error) {
	return NewWithHistoryPath("")
}

// NewWithHistoryPath creates a new RemFS with custom history location
// If historyPath is empty, uses default ~/.config/rem/history/
// If historyPath is absolute, uses it directly as the full history directory
// If historyPath is relative, treats as subdirectory of ~/.config/rem/
func NewWithHistoryPath(historyPath string) (*RemFS, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	var historyDir string

	if historyPath == "" {
		// Default case: ~/.config/rem/history/
		configPath := filepath.Join(homeDir, ConfigDir)
		historyDir = filepath.Join(configPath, DefaultHistDir)
	} else if filepath.IsAbs(historyPath) {
		// Absolute path: use directly as the full history directory
		historyDir = historyPath
	} else {
		// Relative path: treat as subdirectory of ~/.config/rem/
		configPath := filepath.Join(homeDir, ConfigDir)
		historyDir = filepath.Join(configPath, historyPath)
	}

	// Ensure the directory exists
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return nil, err
	}

	remfs := &RemFS{root: historyDir}

	// Perform migration if needed (only for default case)
	if historyPath == "" {
		legacyPath := filepath.Join(homeDir, ConfigDir, LegacyHistDir)
		if err := remfs.migrateFromLegacyLocation(legacyPath); err != nil {
			return nil, err
		}
	}

	return remfs, nil
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

// migrateFromLegacyLocation migrates history files from the legacy content/ directory
// to the new history/ directory location
func (rfs *RemFS) migrateFromLegacyLocation(legacyPath string) error {
	// Check if legacy directory exists
	if _, err := os.Stat(legacyPath); os.IsNotExist(err) {
		// No legacy directory, nothing to migrate
		return nil
	}

	// Check if migration marker already exists
	configDir := filepath.Dir(rfs.root)
	migrationMarker := filepath.Join(configDir, ".migration_complete")
	if _, err := os.Stat(migrationMarker); err == nil {
		// Migration already completed
		return nil
	}

	// Read all files from legacy directory
	entries, err := os.ReadDir(legacyPath)
	if err != nil {
		return fmt.Errorf("failed to read legacy directory: %w", err)
	}

	// Check if new directory has any files (avoid double migration)
	newEntries, err := os.ReadDir(rfs.root)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to check new directory: %w", err)
	}

	if len(newEntries) > 0 && len(entries) > 0 {
		// Both directories have files, don't auto-migrate to avoid data loss
		return fmt.Errorf("both legacy (%s) and new (%s) directories contain files; manual migration required", legacyPath, rfs.root)
	}

	// Migrate files from legacy to new location
	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories
		}

		srcPath := filepath.Join(legacyPath, entry.Name())
		dstPath := filepath.Join(rfs.root, entry.Name())

		// Copy file content
		srcData, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("failed to read legacy file %s: %w", entry.Name(), err)
		}

		if err := os.WriteFile(dstPath, srcData, 0644); err != nil {
			return fmt.Errorf("failed to write migrated file %s: %w", entry.Name(), err)
		}
	}

	// Create migration marker
	markerData := []byte("Migration from content/ to history/ completed")
	if err := os.WriteFile(migrationMarker, markerData, 0644); err != nil {
		// Log warning but don't fail migration
		// This marker is just to avoid redundant migrations
	}

	return nil
}
