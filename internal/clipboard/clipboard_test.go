package clipboard

import (
	"runtime"
	"testing"
)

func TestReadWrite(t *testing.T) {
	// Skip on Windows as it's not supported
	if runtime.GOOS == "windows" {
		t.Skip("Windows clipboard not supported")
	}

	testData := []byte("test clipboard content")

	// Write to clipboard
	if err := Write(testData); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read from clipboard
	data, err := Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	// Compare
	if string(data) != string(testData) {
		t.Errorf("Read data mismatch: got %q, want %q", string(data), string(testData))
	}
}

func TestWriteEmpty(t *testing.T) {
	// Skip on Windows as it's not supported
	if runtime.GOOS == "windows" {
		t.Skip("Windows clipboard not supported")
	}

	// Write empty content
	if err := Write([]byte("")); err != nil {
		t.Fatalf("Write empty failed: %v", err)
	}

	// Read should succeed
	data, err := Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if len(data) != 0 {
		t.Errorf("Expected empty clipboard, got %d bytes", len(data))
	}
}

func TestMultilineContent(t *testing.T) {
	// Skip on Windows as it's not supported
	if runtime.GOOS == "windows" {
		t.Skip("Windows clipboard not supported")
	}

	testData := []byte("line 1\nline 2\nline 3")

	// Write to clipboard
	if err := Write(testData); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read from clipboard
	data, err := Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	// Compare
	if string(data) != string(testData) {
		t.Errorf("Read data mismatch: got %q, want %q", string(data), string(testData))
	}
}

func TestWindowsError(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	// Both Read and Write should return errors on Windows
	if _, err := Read(); err == nil {
		t.Error("Expected error on Windows Read, got nil")
	}

	if err := Write([]byte("test")); err == nil {
		t.Error("Expected error on Windows Write, got nil")
	}
}
