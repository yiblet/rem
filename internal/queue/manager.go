package queue

import (
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	DefaultMaxStackSize = 255
	// ContentDir removed - remfs now points directly to the history directory
)

// FileSystem interface for dependency injection and testability
type FileSystem interface {
	fs.FS
	fs.ReadDirFS
	WriteFile(name string, data []byte, perm fs.FileMode) error
	Remove(name string) error
	MkdirAll(name string, perm fs.FileMode) error
}

// StackManager manages the persistent LIFO stack
type StackManager struct {
	fs           FileSystem
	historyLimit int
}

// StackItem represents a single item in the stack
type StackItem struct {
	Timestamp time.Time
	FilePath  string
	Preview   string
}

// NewStackManager creates a new stack manager with the provided filesystem
func NewStackManager(filesystem FileSystem) (*StackManager, error) {
	return NewStackManagerWithConfig(filesystem, DefaultMaxStackSize)
}

// NewStackManagerWithConfig creates a new stack manager with the provided filesystem and history limit
func NewStackManagerWithConfig(filesystem FileSystem, historyLimit int) (*StackManager, error) {
	if historyLimit <= 0 {
		historyLimit = DefaultMaxStackSize
	}

	qm := &StackManager{
		fs:           filesystem,
		historyLimit: historyLimit,
	}

	// The remfs already points to the history directory, no subdirectory needed

	return qm, nil
}

// FileSystem returns the underlying filesystem
func (qm *StackManager) FileSystem() FileSystem {
	return qm.fs
}

// Push adds a new item to the stack from an io.Reader
func (qm *StackManager) Push(content io.Reader) (*StackItem, error) {
	now := time.Now()
	filename := now.Format("2006-01-02T15-04-05.000000Z07-00") + ".txt"
	filePath := filename

	// Read all content into memory first
	data, err := io.ReadAll(content)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no content to store")
	}

	// Write content to file
	if err := qm.fs.WriteFile(filePath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write content file: %w", err)
	}

	// Generate preview
	preview := qm.generatePreview(data)

	item := &StackItem{
		Timestamp: now,
		FilePath:  filePath,
		Preview:   preview,
	}

	// Clean up old files if we exceed max size
	if err := qm.cleanupOldFiles(); err != nil {
		return nil, fmt.Errorf("failed to cleanup old files: %w", err)
	}

	return item, nil
}

// generatePreview creates a preview string for the content
func (qm *StackManager) generatePreview(data []byte) string {
	// Use first 100 bytes for preview
	previewLen := len(data)
	if previewLen > 100 {
		previewLen = 100
	}

	// Convert to string and clean up
	previewStr := string(data[:previewLen])
	previewStr = strings.ReplaceAll(previewStr, "\n", " ")
	previewStr = strings.ReplaceAll(previewStr, "\r", " ")
	previewStr = strings.ReplaceAll(previewStr, "\t", " ")

	// Collapse multiple spaces
	words := strings.Fields(previewStr)
	previewStr = strings.Join(words, " ")

	if len(previewStr) > 50 {
		previewStr = previewStr[:50] + "..."
	}

	if previewStr == "" {
		previewStr = "[binary content]"
	}

	return previewStr
}

// cleanupOldFiles removes the oldest files if stack exceeds configured history limit
func (qm *StackManager) cleanupOldFiles() error {
	files, err := qm.fs.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read content directory: %w", err)
	}

	// Filter only content files (not directories or other files)
	var contentFiles []fs.DirEntry
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".txt") {
			contentFiles = append(contentFiles, file)
		}
	}

	// If we have historyLimit or fewer files, no cleanup needed
	if len(contentFiles) <= qm.historyLimit {
		return nil
	}

	// Sort files by name (which corresponds to timestamp due to ISO format)
	sort.Slice(contentFiles, func(i, j int) bool {
		return contentFiles[i].Name() < contentFiles[j].Name()
	})

	// Remove oldest files to get back to historyLimit
	filesToRemove := len(contentFiles) - qm.historyLimit
	for i := 0; i < filesToRemove; i++ {
		filePath := contentFiles[i].Name()
		if err := qm.fs.Remove(filePath); err != nil {
			return fmt.Errorf("failed to remove old file %s: %w", filePath, err)
		}
	}

	return nil
}

// List returns all items in the stack, sorted by timestamp (newest first - LIFO order)
func (qm *StackManager) List() ([]*StackItem, error) {
	files, err := qm.fs.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("failed to read content directory: %w", err)
	}

	var items []*StackItem
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".txt") {
			continue
		}

		// Parse timestamp from filename
		filename := strings.TrimSuffix(file.Name(), ".txt")
		timestamp, err := time.Parse("2006-01-02T15-04-05.000000Z07-00", filename)
		if err != nil {
			continue // Skip files with invalid timestamp format
		}

		filePath := file.Name()
		preview, err := qm.generatePreviewFromFile(filePath)
		if err != nil {
			preview = filename // Fallback to filename
		}

		items = append(items, &StackItem{
			Timestamp: timestamp,
			FilePath:  filePath,
			Preview:   preview,
		})
	}

	// Sort by timestamp, newest first (LIFO - Last In, First Out)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp.After(items[j].Timestamp)
	})

	return items, nil
}

// generatePreviewFromFile creates a preview from a file path
func (qm *StackManager) generatePreviewFromFile(filePath string) (string, error) {
	file, err := qm.fs.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Read first 100 bytes for preview
	preview := make([]byte, 100)
	n, err := file.Read(preview)
	if err != nil && err != io.EOF {
		return "", err
	}

	return qm.generatePreview(preview[:n]), nil
}

// Get returns the item at the specified index (0 = top of stack, newest)
func (qm *StackManager) Get(index int) (*StackItem, error) {
	items, err := qm.List()
	if err != nil {
		return nil, err
	}

	if index < 0 || index >= len(items) {
		return nil, fmt.Errorf("index %d out of range (0-%d)", index, len(items)-1)
	}

	return items[index], nil
}

// Size returns the number of items in the stack
func (qm *StackManager) Size() (int, error) {
	items, err := qm.List()
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

// GetContentReader returns an io.ReadSeekCloser for the item's content
func (item *StackItem) GetContentReader(fs FileSystem) (io.ReadSeekCloser, error) {
	file, err := fs.Open(item.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open content file: %w", err)
	}

	// Ensure the file implements ReadSeekCloser
	if rsc, ok := file.(io.ReadSeekCloser); ok {
		return rsc, nil
	}

	// If not, we need to wrap it (for in-memory filesystems)
	return &readSeekCloserWrapper{file}, nil
}

// readSeekCloserWrapper wraps a file to provide Seek functionality
type readSeekCloserWrapper struct {
	file fs.File
}

func (w *readSeekCloserWrapper) Read(p []byte) (int, error) {
	return w.file.Read(p)
}

func (w *readSeekCloserWrapper) Close() error {
	return w.file.Close()
}

func (w *readSeekCloserWrapper) Seek(offset int64, whence int) (int64, error) {
	// Try to cast to io.Seeker
	if seeker, ok := w.file.(io.Seeker); ok {
		return seeker.Seek(offset, whence)
	}
	return 0, fmt.Errorf("seek not supported")
}

// Delete removes the item from the stack
func (qm *StackManager) Delete(index int) error {
	item, err := qm.Get(index)
	if err != nil {
		return err
	}

	if err := qm.fs.Remove(item.FilePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// Legacy type aliases for backward compatibility
type QueueManager = StackManager
type QueueItem = StackItem

// Legacy function aliases for backward compatibility
func NewQueueManager(filesystem FileSystem) (*StackManager, error) {
	return NewStackManager(filesystem)
}

// GetHistoryLimit returns the configured history limit
func (qm *StackManager) GetHistoryLimit() int {
	return qm.historyLimit
}

// Legacy method aliases for backward compatibility
func (qm *StackManager) Enqueue(content io.Reader) (*StackItem, error) {
	return qm.Push(content)
}

// Clear removes all items from the stack
func (qm *StackManager) Clear() error {
	items, err := qm.List()
	if err != nil {
		return fmt.Errorf("failed to list items: %w", err)
	}

	// Delete each item's file
	for _, item := range items {
		if err := qm.fs.Remove(item.FilePath); err != nil {
			return fmt.Errorf("failed to remove file %s: %w", item.FilePath, err)
		}
	}

	return nil
}

// SearchResult represents a search result with the item and its index
type SearchResult struct {
	Index int
	Item  *StackItem
}

// Search searches for items matching the given regex pattern
func (qm *StackManager) Search(pattern string) ([]*SearchResult, error) {
	// Compile the regex pattern
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Get all items from the stack
	items, err := qm.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}

	var results []*SearchResult

	// Search through each item
	for i, item := range items {
		// Open the file to read its content
		file, err := qm.fs.Open(item.FilePath)
		if err != nil {
			continue // Skip files that can't be opened
		}

		// Read the entire content
		content, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			continue // Skip files that can't be read
		}

		// Check if the content matches the pattern
		if re.Match(content) {
			results = append(results, &SearchResult{
				Index: i,
				Item:  item,
			})
		}
	}

	return results, nil
}
