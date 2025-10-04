// Package mockboard provides a mock clipboard implementation for testing.
package mockboard

import (
	"bytes"
	"io"
)

// MockClipboard implements Clipboard for testing
type MockClipboard struct {
	data []byte
}

// New creates a new MockClipboard instance
func New() *MockClipboard {
	return &MockClipboard{}
}

// Read implements Clipboard.Read for MockClipboard
func (m *MockClipboard) Read() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(m.data)), nil
}

// Write implements Clipboard.Write for MockClipboard
func (m *MockClipboard) Write(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.data = data
	return nil
}

// SetData sets the mock clipboard data directly (for testing)
func (m *MockClipboard) SetData(data []byte) {
	m.data = data
}

// GetData returns the current clipboard data (for testing)
func (m *MockClipboard) GetData() []byte {
	return m.data
}

// IsSupported always returns true for the mock clipboard
func (m *MockClipboard) IsSupported() bool {
	return true
}
