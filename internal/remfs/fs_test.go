package remfs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewWithHistoryPath_Default(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Set temporary home for testing
	os.Setenv("HOME", tempDir)

	// Test default case (empty history path)
	remfs, err := NewWithHistoryPath("")
	if err != nil {
		t.Fatalf("NewWithHistoryPath(\"\") failed: %v", err)
	}

	expectedPath := filepath.Join(tempDir, ConfigDir, DefaultHistDir)
	if remfs.Root() != expectedPath {
		t.Errorf("Expected root path %s, got %s", expectedPath, remfs.Root())
	}

	// Check that directory was created
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected directory %s to be created", expectedPath)
	}
}

func TestNewWithHistoryPath_Absolute(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	customPath := filepath.Join(tempDir, "custom-history")

	// Test absolute path
	remfs, err := NewWithHistoryPath(customPath)
	if err != nil {
		t.Fatalf("NewWithHistoryPath(%s) failed: %v", customPath, err)
	}

	if remfs.Root() != customPath {
		t.Errorf("Expected root path %s, got %s", customPath, remfs.Root())
	}

	// Check that directory was created
	if _, err := os.Stat(customPath); os.IsNotExist(err) {
		t.Errorf("Expected directory %s to be created", customPath)
	}
}

func TestNewWithHistoryPath_Relative(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Set temporary home for testing
	os.Setenv("HOME", tempDir)

	// Test relative path
	relativePath := "custom"
	remfs, err := NewWithHistoryPath(relativePath)
	if err != nil {
		t.Fatalf("NewWithHistoryPath(%s) failed: %v", relativePath, err)
	}

	expectedPath := filepath.Join(tempDir, ConfigDir, relativePath)
	if remfs.Root() != expectedPath {
		t.Errorf("Expected root path %s, got %s", expectedPath, remfs.Root())
	}

	// Check that directory was created
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected directory %s to be created", expectedPath)
	}
}

func TestMigrateFromLegacyLocation_NoLegacyDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	newHistoryPath := filepath.Join(tempDir, "history")
	legacyPath := filepath.Join(tempDir, "content")

	// Create only the new directory
	if err := os.MkdirAll(newHistoryPath, 0755); err != nil {
		t.Fatalf("Failed to create new history directory: %v", err)
	}

	remfs := &RemFS{root: newHistoryPath}

	// Should not fail when legacy directory doesn't exist
	err := remfs.migrateFromLegacyLocation(legacyPath)
	if err != nil {
		t.Errorf("migrateFromLegacyLocation should not fail when legacy directory doesn't exist: %v", err)
	}
}

func TestMigrateFromLegacyLocation_SuccessfulMigration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	newHistoryPath := filepath.Join(tempDir, "history")
	legacyPath := filepath.Join(tempDir, "content")

	// Create legacy directory with test files
	if err := os.MkdirAll(legacyPath, 0755); err != nil {
		t.Fatalf("Failed to create legacy directory: %v", err)
	}

	testFiles := map[string]string{
		"2025-09-28T10-15-30.123456-07-00.txt": "content 1",
		"2025-09-28T10-16-45.789012-07-00.txt": "content 2",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(legacyPath, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create new history directory
	if err := os.MkdirAll(newHistoryPath, 0755); err != nil {
		t.Fatalf("Failed to create new history directory: %v", err)
	}

	remfs := &RemFS{root: newHistoryPath}

	// Perform migration
	err := remfs.migrateFromLegacyLocation(legacyPath)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Check that files were migrated
	for filename, expectedContent := range testFiles {
		migratedPath := filepath.Join(newHistoryPath, filename)
		actualContent, err := os.ReadFile(migratedPath)
		if err != nil {
			t.Errorf("Failed to read migrated file %s: %v", filename, err)
			continue
		}

		if string(actualContent) != expectedContent {
			t.Errorf("Migrated file %s has incorrect content. Expected %q, got %q",
				filename, expectedContent, string(actualContent))
		}
	}

	// Check that migration marker was created
	configDir := filepath.Dir(newHistoryPath)
	migrationMarker := filepath.Join(configDir, ".migration_complete")
	if _, err := os.Stat(migrationMarker); err != nil {
		t.Errorf("Migration marker should be created at %s", migrationMarker)
	}
}

func TestMigrateFromLegacyLocation_ConflictingFiles(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	newHistoryPath := filepath.Join(tempDir, "history")
	legacyPath := filepath.Join(tempDir, "content")

	// Create legacy directory with test file
	if err := os.MkdirAll(legacyPath, 0755); err != nil {
		t.Fatalf("Failed to create legacy directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyPath, "test.txt"), []byte("legacy"), 0644); err != nil {
		t.Fatalf("Failed to create legacy test file: %v", err)
	}

	// Create new directory with existing file
	if err := os.MkdirAll(newHistoryPath, 0755); err != nil {
		t.Fatalf("Failed to create new history directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(newHistoryPath, "existing.txt"), []byte("existing"), 0644); err != nil {
		t.Fatalf("Failed to create existing test file: %v", err)
	}

	remfs := &RemFS{root: newHistoryPath}

	// Migration should fail when both directories have files
	err := remfs.migrateFromLegacyLocation(legacyPath)
	if err == nil {
		t.Error("Expected migration to fail when both directories contain files")
	}
}

func TestMigrateFromLegacyLocation_AlreadyMigrated(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	newHistoryPath := filepath.Join(tempDir, "history")
	legacyPath := filepath.Join(tempDir, "content")

	// Create directories
	if err := os.MkdirAll(legacyPath, 0755); err != nil {
		t.Fatalf("Failed to create legacy directory: %v", err)
	}
	if err := os.MkdirAll(newHistoryPath, 0755); err != nil {
		t.Fatalf("Failed to create new history directory: %v", err)
	}

	// Create migration marker
	configDir := filepath.Dir(newHistoryPath)
	migrationMarker := filepath.Join(configDir, ".migration_complete")
	if err := os.WriteFile(migrationMarker, []byte("done"), 0644); err != nil {
		t.Fatalf("Failed to create migration marker: %v", err)
	}

	remfs := &RemFS{root: newHistoryPath}

	// Should not attempt migration when marker exists
	err := remfs.migrateFromLegacyLocation(legacyPath)
	if err != nil {
		t.Errorf("Migration should succeed when already completed: %v", err)
	}
}

func TestRemFS_BasicOperations(t *testing.T) {
	tempDir := t.TempDir()
	remfs := NewWithRoot(tempDir)

	// Test WriteFile
	testContent := []byte("hello world")
	err := remfs.WriteFile("test.txt", testContent, 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Test Open and read content
	file, err := remfs.Open("test.txt")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer file.Close()

	// Read content
	buffer := make([]byte, len(testContent))
	n, err := file.Read(buffer)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if n != len(testContent) || string(buffer[:n]) != string(testContent) {
		t.Errorf("Expected content %q, got %q", string(testContent), string(buffer[:n]))
	}

	// Test ReadDir
	entries, err := remfs.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	found := false
	for _, entry := range entries {
		if entry.Name() == "test.txt" {
			found = true
			break
		}
	}

	if !found {
		t.Error("test.txt should be found in directory listing")
	}

	// Test Remove
	err = remfs.Remove("test.txt")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify file was removed
	_, err = remfs.Open("test.txt")
	if !os.IsNotExist(err) {
		t.Error("File should not exist after removal")
	}
}
