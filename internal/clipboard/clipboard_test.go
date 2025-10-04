package clipboard

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/yiblet/rem/internal/clipboard/mockboard"
	"github.com/yiblet/rem/internal/clipboard/sysboard"
)

// saveAndRestoreClipboard saves the current clipboard content and returns a function to restore it
func saveAndRestoreClipboard(t *testing.T, sys *sysboard.SystemClipboard) func() {
	// Read current clipboard content
	reader, err := sys.Read()
	if err != nil {
		t.Logf("Warning: could not read original clipboard content: %v", err)
		return func() {} // Return no-op if we can't read
	}

	originalContent, err := io.ReadAll(reader)
	reader.Close()
	if err != nil {
		t.Logf("Warning: could not read original clipboard content: %v", err)
		return func() {} // Return no-op if we can't read
	}

	// Return restore function
	return func() {
		if err := sys.Write(bytes.NewReader(originalContent)); err != nil {
			t.Logf("Warning: could not restore clipboard content: %v", err)
		}
	}
}

func TestMockClipboard(t *testing.T) {
	mock := mockboard.New()
	testData := []byte("test clipboard content")

	// Write to mock clipboard
	if err := mock.Write(bytes.NewReader(testData)); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read from mock clipboard
	reader, err := mock.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	// Compare
	if string(data) != string(testData) {
		t.Errorf("Read data mismatch: got %q, want %q", string(data), string(testData))
	}

	// Verify GetData works
	if string(mock.GetData()) != string(testData) {
		t.Errorf("GetData mismatch: got %q, want %q", string(mock.GetData()), string(testData))
	}
}

func TestSystemClipboardReadWrite(t *testing.T) {
	sys := sysboard.New()

	// Skip if clipboard is not supported
	if !sys.IsSupported() {
		t.Skip("System clipboard not supported on this platform")
	}

	// Save and restore original clipboard content
	restore := saveAndRestoreClipboard(t, sys)
	defer restore()

	testData := []byte("test clipboard content")

	// Write to clipboard
	if err := sys.Write(bytes.NewReader(testData)); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read from clipboard
	reader, err := sys.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	// Compare
	if string(data) != string(testData) {
		t.Errorf("Read data mismatch: got %q, want %q", string(data), string(testData))
	}
}

func TestSystemClipboardEmpty(t *testing.T) {
	sys := sysboard.New()

	// Skip if clipboard is not supported
	if !sys.IsSupported() {
		t.Skip("System clipboard not supported on this platform")
	}

	// Save and restore original clipboard content
	restore := saveAndRestoreClipboard(t, sys)
	defer restore()

	// Write empty content
	if err := sys.Write(strings.NewReader("")); err != nil {
		t.Fatalf("Write empty failed: %v", err)
	}

	// Read should succeed
	reader, err := sys.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(data) != 0 {
		t.Errorf("Expected empty clipboard, got %d bytes", len(data))
	}
}

func TestSystemClipboardMultiline(t *testing.T) {
	sys := sysboard.New()

	// Skip if clipboard is not supported
	if !sys.IsSupported() {
		t.Skip("System clipboard not supported on this platform")
	}

	// Save and restore original clipboard content
	restore := saveAndRestoreClipboard(t, sys)
	defer restore()

	testData := []byte("line 1\nline 2\nline 3")

	// Write to clipboard
	if err := sys.Write(bytes.NewReader(testData)); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read from clipboard
	reader, err := sys.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	// Compare
	if string(data) != string(testData) {
		t.Errorf("Read data mismatch: got %q, want %q", string(data), string(testData))
	}
}

func TestSystemClipboardLarge(t *testing.T) {
	sys := sysboard.New()

	// Skip if clipboard is not supported
	if !sys.IsSupported() {
		t.Skip("System clipboard not supported on this platform")
	}

	// Save and restore original clipboard content
	restore := saveAndRestoreClipboard(t, sys)
	defer restore()

	// Create large content (1MB)
	testData := bytes.Repeat([]byte("abcdefghij"), 100000)

	// Write to clipboard
	if err := sys.Write(bytes.NewReader(testData)); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read from clipboard
	reader, err := sys.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	// Compare lengths
	if len(data) != len(testData) {
		t.Errorf("Read data length mismatch: got %d, want %d", len(data), len(testData))
	}

	// Spot check some bytes
	if !bytes.Equal(data[:100], testData[:100]) {
		t.Errorf("Read data content mismatch at start")
	}
}

func TestUnsupportedPlatform(t *testing.T) {
	sys := sysboard.New()

	// Skip if platform is supported
	if sys.IsSupported() {
		t.Skip("Platform is supported, skipping unsupported test")
	}

	// Both Read and Write should return errors on unsupported platforms
	if _, err := sys.Read(); err == nil {
		t.Error("Expected error on unsupported platform Read, got nil")
	}

	if err := sys.Write(strings.NewReader("test")); err == nil {
		t.Error("Expected error on unsupported platform Write, got nil")
	}
}

func TestIsSupported(t *testing.T) {
	// Test that IsSupported is implemented
	sys := sysboard.New()
	mock := mockboard.New()

	// Mock should always be supported
	if !mock.IsSupported() {
		t.Error("Mock clipboard should always be supported")
	}

	// System clipboard support depends on platform
	_ = sys.IsSupported() // Just verify it doesn't panic
}
