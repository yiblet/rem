package queue

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// MemoryFileSystem implements FileSystem for in-memory testing
type MemoryFileSystem struct {
	fstest.MapFS
}

func NewMemoryFileSystem() *MemoryFileSystem {
	return &MemoryFileSystem{
		MapFS: make(fstest.MapFS),
	}
}

func (mfs *MemoryFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	mfs.MapFS[name] = &fstest.MapFile{
		Data: data,
		Mode: perm,
	}
	return nil
}

func (mfs *MemoryFileSystem) Remove(name string) error {
	delete(mfs.MapFS, name)
	return nil
}

func (mfs *MemoryFileSystem) MkdirAll(path string, perm os.FileMode) error {
	// In memory filesystem doesn't need directory creation
	return nil
}

// memoryFileWriter implements io.WriteCloser for MemoryFileSystem
type memoryFileWriter struct {
	fs   *MemoryFileSystem
	name string
	perm os.FileMode
	buf  []byte
}

func (w *memoryFileWriter) Write(p []byte) (n int, err error) {
	w.buf = append(w.buf, p...)
	return len(p), nil
}

func (w *memoryFileWriter) Close() error {
	// Write buffer to filesystem when closed
	w.fs.MapFS[w.name] = &fstest.MapFile{
		Data: w.buf,
		Mode: w.perm,
	}
	return nil
}

func (mfs *MemoryFileSystem) OpenForWrite(name string, perm os.FileMode) (io.WriteCloser, error) {
	return &memoryFileWriter{
		fs:   mfs,
		name: name,
		perm: perm,
		buf:  nil,
	}, nil
}

func (mfs *MemoryFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	var entries []fs.DirEntry

	// Handle "." as root directory
	if name == "." {
		name = ""
	}

	prefix := name
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	for path := range mfs.MapFS {
		match := false
		relativePath := ""

		if prefix == "" {
			// Root directory: include files with no path separators
			if !strings.Contains(path, "/") {
				match = true
				relativePath = path
			}
		} else if strings.HasPrefix(path, prefix) {
			// Subdirectory: remove prefix to get relative name
			relativePath = strings.TrimPrefix(path, prefix)
			// Only include direct children (no further slashes)
			if !strings.Contains(relativePath, "/") && relativePath != "" {
				match = true
			}
		}

		if match {
			entries = append(entries, &memoryDirEntry{
				name: relativePath,
				file: mfs.MapFS[path],
			})
		}
	}

	// Sort entries by name for consistent results
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	return entries, nil
}

// memoryDirEntry implements fs.DirEntry for in-memory files
type memoryDirEntry struct {
	name string
	file *fstest.MapFile
}

func (e *memoryDirEntry) Name() string {
	return e.name
}

func (e *memoryDirEntry) IsDir() bool {
	return e.file.Mode.IsDir()
}

func (e *memoryDirEntry) Type() fs.FileMode {
	return e.file.Mode.Type()
}

func (e *memoryDirEntry) Info() (fs.FileInfo, error) {
	return &memoryFileInfo{
		name: e.name,
		mode: e.file.Mode,
		size: int64(len(e.file.Data)),
	}, nil
}

// memoryFileInfo implements fs.FileInfo for in-memory files
type memoryFileInfo struct {
	name string
	mode fs.FileMode
	size int64
}

func (fi *memoryFileInfo) Name() string       { return fi.name }
func (fi *memoryFileInfo) Size() int64        { return fi.size }
func (fi *memoryFileInfo) Mode() fs.FileMode  { return fi.mode }
func (fi *memoryFileInfo) ModTime() time.Time { return time.Time{} }
func (fi *memoryFileInfo) IsDir() bool        { return fi.mode.IsDir() }
func (fi *memoryFileInfo) Sys() interface{}   { return nil }

func TestQueueManager_Basic(t *testing.T) {
	memFS := NewMemoryFileSystem()
	qm, err := NewQueueManager(memFS)
	if err != nil {
		t.Fatalf("Failed to create queue manager: %v", err)
	}

	// Test empty queue
	size, err := qm.Size()
	if err != nil {
		t.Fatalf("Failed to get size: %v", err)
	}
	if size != 0 {
		t.Errorf("Expected empty queue, got size %d", size)
	}

	// Test enqueue
	content := strings.NewReader("Hello, World!")
	item, err := qm.Enqueue(content)
	if err != nil {
		t.Fatalf("Failed to enqueue: %v", err)
	}

	if item.Preview != "Hello, World!" {
		t.Errorf("Expected preview 'Hello, World!', got '%s'", item.Preview)
	}

	// Test size after enqueue
	size, err = qm.Size()
	if err != nil {
		t.Fatalf("Failed to get size: %v", err)
	}
	if size != 1 {
		t.Errorf("Expected size 1, got %d", size)
	}

	// Test get
	retrievedItem, err := qm.Get(0)
	if err != nil {
		t.Fatalf("Failed to get item: %v", err)
	}

	if retrievedItem.Preview != item.Preview {
		t.Errorf("Expected preview '%s', got '%s'", item.Preview, retrievedItem.Preview)
	}

	// Test content reader
	reader, err := retrievedItem.GetContentReader(memFS)
	if err != nil {
		t.Fatalf("Failed to get content reader: %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read content: %v", err)
	}

	if string(data) != "Hello, World!" {
		t.Errorf("Expected content 'Hello, World!', got '%s'", string(data))
	}
}

func TestQueueManager_MaxSize(t *testing.T) {
	memFS := NewMemoryFileSystem()
	qm, err := NewQueueManager(memFS)
	if err != nil {
		t.Fatalf("Failed to create queue manager: %v", err)
	}

	// Add more than DefaultMaxQueueSize items
	for i := 0; i < DefaultMaxQueueSize+5; i++ {
		content := strings.NewReader(fmt.Sprintf("Content %d", i))
		_, err := qm.Enqueue(content)
		if err != nil {
			t.Fatalf("Failed to enqueue item %d: %v", i, err)
		}
	}

	// Check that size is limited to DefaultMaxQueueSize
	size, err := qm.Size()
	if err != nil {
		t.Fatalf("Failed to get size: %v", err)
	}

	if size != DefaultMaxQueueSize {
		t.Errorf("Expected size %d, got %d", DefaultMaxQueueSize, size)
	}

	// Check that newest items are kept (should have items 5-259)
	items, err := qm.List()
	if err != nil {
		t.Fatalf("Failed to list items: %v", err)
	}

	// First item should be newest (Content 259, since we added 0-259)
	if !strings.Contains(items[0].Preview, "Content 259") {
		t.Errorf("Expected newest item to contain 'Content 259', got '%s'", items[0].Preview)
	}

	// Last item should be Content 5 (oldest remaining after removing 0-4)
	lastIndex := len(items) - 1
	if !strings.Contains(items[lastIndex].Preview, "Content 5") {
		t.Errorf("Expected oldest item to contain 'Content 5', got '%s'", items[lastIndex].Preview)
	}
}

func TestQueueManager_Preview(t *testing.T) {
	memFS := NewMemoryFileSystem()
	qm, err := NewQueueManager(memFS)
	if err != nil {
		t.Fatalf("Failed to create queue manager: %v", err)
	}

	testCases := []struct {
		input    string
		expected string
	}{
		{"Short text", "Short text"},
		{"Text with\nnewlines\tand\ttabs", "Text with newlines and tabs"},
		{strings.Repeat("a", 100), strings.Repeat("a", 50) + "..."},
		{"", "[binary content]"},
	}

	for i, tc := range testCases {
		content := strings.NewReader(tc.input)
		item, err := qm.Enqueue(content)
		if err != nil {
			if tc.input == "" {
				continue // Empty content should error
			}
			t.Fatalf("Failed to enqueue test case %d: %v", i, err)
		}

		if item.Preview != tc.expected {
			t.Errorf("Test case %d: expected preview '%s', got '%s'", i, tc.expected, item.Preview)
		}
	}
}

func TestQueueManager_Clear(t *testing.T) {
	memFS := NewMemoryFileSystem()
	qm, err := NewStackManager(memFS)
	if err != nil {
		t.Fatalf("Failed to create queue manager: %v", err)
	}

	// Add some items to the queue
	for i := 0; i < 5; i++ {
		content := strings.NewReader(fmt.Sprintf("Item %d", i))
		_, err := qm.Push(content)
		if err != nil {
			t.Fatalf("Failed to push item %d: %v", i, err)
		}
	}

	// Verify items were added
	size, err := qm.Size()
	if err != nil {
		t.Fatalf("Failed to get size: %v", err)
	}
	if size != 5 {
		t.Errorf("Expected size 5, got %d", size)
	}

	// Clear the queue
	err = qm.Clear()
	if err != nil {
		t.Fatalf("Failed to clear queue: %v", err)
	}

	// Verify queue is empty
	size, err = qm.Size()
	if err != nil {
		t.Fatalf("Failed to get size after clear: %v", err)
	}
	if size != 0 {
		t.Errorf("Expected size 0 after clear, got %d", size)
	}

	// Test clearing empty queue (should not error)
	err = qm.Clear()
	if err != nil {
		t.Fatalf("Failed to clear empty queue: %v", err)
	}
}

func TestQueueManager_Search(t *testing.T) {
	memFS := NewMemoryFileSystem()
	qm, err := NewStackManager(memFS)
	if err != nil {
		t.Fatalf("Failed to create queue manager: %v", err)
	}

	// Add test items
	testItems := []string{
		"Hello World",
		"Error: something went wrong",
		"Debug: checking values",
		"Error: another problem occurred",
		"Success: operation completed",
	}

	for _, content := range testItems {
		_, err := qm.Push(strings.NewReader(content))
		if err != nil {
			t.Fatalf("Failed to push item: %v", err)
		}
	}

	tests := []struct {
		name          string
		pattern       string
		expectedCount int
		expectedIndex []int
	}{
		{
			name:          "Find all errors",
			pattern:       "Error:",
			expectedCount: 2,
			expectedIndex: []int{1, 3}, // LIFO order: item 4 is at index 1, item 2 is at index 3
		},
		{
			name:          "Find debug",
			pattern:       "Debug:",
			expectedCount: 1,
			expectedIndex: []int{2},
		},
		{
			name:          "Regex pattern",
			pattern:       "Error:.*wrong",
			expectedCount: 1,
			expectedIndex: []int{3}, // "Error: something went wrong" is at index 3
		},
		{
			name:          "Case sensitive",
			pattern:       "hello",
			expectedCount: 0,
			expectedIndex: []int{},
		},
		{
			name:          "No matches",
			pattern:       "NotFound",
			expectedCount: 0,
			expectedIndex: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := qm.Search(tt.pattern)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			for i, result := range results {
				if i >= len(tt.expectedIndex) {
					break
				}
				if result.Index != tt.expectedIndex[i] {
					t.Errorf("Result %d: expected index %d, got %d", i, tt.expectedIndex[i], result.Index)
				}
			}
		})
	}
}

func TestQueueManager_Search_InvalidRegex(t *testing.T) {
	memFS := NewMemoryFileSystem()
	qm, err := NewStackManager(memFS)
	if err != nil {
		t.Fatalf("Failed to create queue manager: %v", err)
	}

	// Test with invalid regex pattern
	_, err = qm.Search("[invalid")
	if err == nil {
		t.Error("Expected error for invalid regex pattern, got nil")
	}
}

func TestQueueManager_Search_EmptyQueue(t *testing.T) {
	memFS := NewMemoryFileSystem()
	qm, err := NewStackManager(memFS)
	if err != nil {
		t.Fatalf("Failed to create queue manager: %v", err)
	}

	// Search in empty queue
	results, err := qm.Search("test")
	if err != nil {
		t.Fatalf("Search in empty queue should not error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results in empty queue, got %d", len(results))
	}
}

func TestIsBinary(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "Plain text",
			data:     []byte("Hello, world! This is plain text."),
			expected: false,
		},
		{
			name:     "Text with newlines",
			data:     []byte("Line 1\nLine 2\nLine 3\n"),
			expected: false,
		},
		{
			name:     "Empty data",
			data:     []byte{},
			expected: false,
		},
		{
			name:     "Binary with null bytes",
			data:     []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05},
			expected: true,
		},
		{
			name:     "Binary with control characters",
			data:     []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			expected: true,
		},
		{
			name:     "Text with tabs and spaces",
			data:     []byte("Hello\tWorld\n\tIndented line\n"),
			expected: false,
		},
		{
			name:     "Invalid UTF-8",
			data:     []byte{0xFF, 0xFE, 0xFD},
			expected: true,
		},
		{
			name:     "Mixed content - mostly text",
			data:     append([]byte("Text content "), 0x01, 0x02),
			expected: false,
		},
		{
			name:     "Mixed content - mostly binary",
			data:     append(make([]byte, 100), []byte("text")...),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBinary(tt.data)
			if result != tt.expected {
				t.Errorf("isBinary(%q) = %v, expected %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestQueueManager_BinaryDetection(t *testing.T) {
	memFS := NewMemoryFileSystem()
	qm, err := NewStackManager(memFS)
	if err != nil {
		t.Fatalf("Failed to create queue manager: %v", err)
	}

	// Test with binary content
	binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}
	item, err := qm.Push(strings.NewReader(string(binaryData)))
	if err != nil {
		t.Fatalf("Failed to push binary content: %v", err)
	}

	if !item.IsBinary {
		t.Error("Expected binary content to be detected as binary")
	}

	if item.Preview != "[binary content]" {
		t.Errorf("Expected preview '[binary content]', got '%s'", item.Preview)
	}

	if item.Size != int64(len(binaryData)) {
		t.Errorf("Expected size %d, got %d", len(binaryData), item.Size)
	}

	if item.SHA256 == "" {
		t.Error("Expected SHA256 hash to be calculated for binary content")
	}

	// Test with text content
	textData := "This is plain text content"
	textItem, err := qm.Push(strings.NewReader(textData))
	if err != nil {
		t.Fatalf("Failed to push text content: %v", err)
	}

	if textItem.IsBinary {
		t.Error("Expected text content to not be detected as binary")
	}

	if textItem.SHA256 != "" {
		t.Error("Expected SHA256 to be empty for text content")
	}

	if textItem.Size != int64(len(textData)) {
		t.Errorf("Expected size %d, got %d", len(textData), textItem.Size)
	}
}
