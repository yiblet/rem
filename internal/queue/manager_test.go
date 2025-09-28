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

func (mfs *MemoryFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	var entries []fs.DirEntry
	prefix := name
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	for path := range mfs.MapFS {
		if strings.HasPrefix(path, prefix) {
			// Remove prefix to get relative name
			relativePath := strings.TrimPrefix(path, prefix)
			// Only include direct children (no further slashes)
			if !strings.Contains(relativePath, "/") && relativePath != "" {
				entries = append(entries, &memoryDirEntry{
					name: relativePath,
					file: mfs.MapFS[path],
				})
			}
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
func (fi *memoryFileInfo) IsDir() bool         { return fi.mode.IsDir() }
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

	// Add more than MaxStackSize items
	for i := 0; i < MaxStackSize+5; i++ {
		content := strings.NewReader(fmt.Sprintf("Content %d", i))
		_, err := qm.Enqueue(content)
		if err != nil {
			t.Fatalf("Failed to enqueue item %d: %v", i, err)
		}
	}

	// Check that size is limited to MaxStackSize
	size, err := qm.Size()
	if err != nil {
		t.Fatalf("Failed to get size: %v", err)
	}

	if size != MaxStackSize {
		t.Errorf("Expected size %d, got %d", MaxStackSize, size)
	}

	// Check that newest items are kept (should have items 5-24)
	items, err := qm.List()
	if err != nil {
		t.Fatalf("Failed to list items: %v", err)
	}

	// First item should be newest (Content 24)
	if !strings.Contains(items[0].Preview, "Content 24") {
		t.Errorf("Expected newest item to contain 'Content 24', got '%s'", items[0].Preview)
	}

	// Last item should be Content 5 (oldest remaining)
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