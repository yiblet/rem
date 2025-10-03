package queue

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	DefaultMaxQueueSize = 255
	FileExtension       = ".rem" // File extension for queue items
	// ContentDir removed - remfs now points directly to the history directory
)

// FileSystem interface for dependency injection and testability
type FileSystem interface {
	fs.FS
	fs.ReadDirFS
	WriteFile(name string, data []byte, perm fs.FileMode) error
	OpenForWrite(name string, perm fs.FileMode) (io.WriteCloser, error)
	Remove(name string) error
	MkdirAll(name string, perm fs.FileMode) error
}

// QueueManager manages the persistent LIFO queue
type QueueManager struct {
	fs           FileSystem
	historyLimit int
}

// QueueItem represents a single item in the queue
type QueueItem struct {
	ID        string // Unique identifier (filename without extension)
	Timestamp time.Time
	FilePath  string
	Preview   string
	IsBinary  bool   // true if content is binary
	Size      int64  // size in bytes (useful for binary files)
	SHA256    string // SHA256 hash (for binary files)
}

// NewQueueManager creates a new queue manager with the provided filesystem
func NewQueueManager(filesystem FileSystem) (*QueueManager, error) {
	return NewQueueManagerWithConfig(filesystem, DefaultMaxQueueSize)
}

// NewQueueManagerWithConfig creates a new queue manager with the provided filesystem and history limit
func NewQueueManagerWithConfig(filesystem FileSystem, historyLimit int) (*QueueManager, error) {
	if historyLimit <= 0 {
		historyLimit = DefaultMaxQueueSize
	}

	qm := &QueueManager{
		fs:           filesystem,
		historyLimit: historyLimit,
	}

	// The remfs already points to the history directory, no subdirectory needed

	return qm, nil
}

// FileSystem returns the underlying filesystem
func (qm *QueueManager) FileSystem() FileSystem {
	return qm.fs
}

// Enqueue adds a new item to the queue from an io.Reader
func (qm *QueueManager) Enqueue(content io.Reader) (*QueueItem, error) {
	now := time.Now()
	filename := now.Format("2006-01-02T15-04-05.000000Z07-00") + FileExtension
	filePath := filename

	// Open file for streaming write
	file, err := qm.fs.OpenForWrite(filePath, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create content file: %w", err)
	}
	defer file.Close()

	// Setup multi-writer: file + SHA256 hasher
	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)

	// Stream content to both file and hasher
	written, err := io.Copy(writer, content)
	if err != nil {
		qm.fs.Remove(filePath) // Clean up partial file
		return nil, fmt.Errorf("failed to write content: %w", err)
	}

	if written == 0 {
		qm.fs.Remove(filePath) // Clean up empty file
		return nil, fmt.Errorf("no content to store")
	}

	// Close file before reopening for reading
	file.Close()

	// Read first 4KB from file for preview/binary detection
	fileReader, err := qm.fs.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to reopen file for metadata: %w", err)
	}
	defer fileReader.Close()

	previewBuf := make([]byte, 4096)
	n, _ := fileReader.Read(previewBuf)
	previewBuf = previewBuf[:n]

	// Generate metadata from sample
	preview := qm.generatePreview(previewBuf)
	binary := isBinary(previewBuf)

	// Calculate SHA256 from hasher (for binary files)
	var sha256Hash string
	if binary {
		sha256Hash = hex.EncodeToString(hasher.Sum(nil))
	}

	item := &QueueItem{
		ID:        strings.TrimSuffix(filename, FileExtension),
		Timestamp: now,
		FilePath:  filePath,
		Preview:   preview,
		IsBinary:  binary,
		Size:      written,
		SHA256:    sha256Hash,
	}

	// Clean up old files if we exceed max size
	if err := qm.cleanupOldFiles(); err != nil {
		return nil, fmt.Errorf("failed to cleanup old files: %w", err)
	}

	return item, nil
}

// isBinary detects if data is binary by checking for non-printable characters
// and invalid UTF-8 sequences
func isBinary(data []byte) bool {
	// Check first 512 bytes (or less if file is smaller)
	checkLen := len(data)
	if checkLen > 512 {
		checkLen = 512
	}

	if checkLen == 0 {
		return false
	}

	// Check if data is valid UTF-8
	if !utf8.Valid(data[:checkLen]) {
		return true
	}

	// Count non-printable characters (excluding common whitespace)
	nonPrintable := 0
	for _, b := range data[:checkLen] {
		// Allow common whitespace: space, tab, newline, carriage return
		if b < 32 && b != 9 && b != 10 && b != 13 {
			nonPrintable++
		}
		// Check for high byte values that might indicate binary
		if b == 127 || (b > 127 && b < 160) {
			nonPrintable++
		}
	}

	// If more than 30% of checked bytes are non-printable, consider it binary
	threshold := float64(checkLen) * 0.3
	return float64(nonPrintable) > threshold
}

// generatePreview creates a preview string for the content
func (qm *StackManager) generatePreview(data []byte) string {
	// Check if binary
	if isBinary(data) {
		return "[binary content]"
	}

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
		previewStr = "[empty]"
	}

	return previewStr
}

// cleanupOldFiles removes the oldest files if queue exceeds configured history limit
func (qm *QueueManager) cleanupOldFiles() error {
	files, err := qm.fs.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read content directory: %w", err)
	}

	// Filter only content files (not directories or other files)
	var contentFiles []fs.DirEntry
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), FileExtension) {
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

// List returns all items in the queue, sorted by timestamp (newest first - LIFO order)
func (qm *QueueManager) List() ([]*QueueItem, error) {
	files, err := qm.fs.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("failed to read content directory: %w", err)
	}

	var items []*QueueItem
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), FileExtension) {
			continue
		}

		// Parse timestamp from filename
		filename := strings.TrimSuffix(file.Name(), FileExtension)
		timestamp, err := time.Parse("2006-01-02T15-04-05.000000Z07-00", filename)
		if err != nil {
			continue // Skip files with invalid timestamp format
		}

		filePath := file.Name()
		preview, binary, size, err := qm.generateMetadataFromFile(filePath)
		if err != nil {
			preview = filename // Fallback to filename
			binary = false
			size = 0
		}

		items = append(items, &QueueItem{
			ID:        filename,
			Timestamp: timestamp,
			FilePath:  filePath,
			Preview:   preview,
			IsBinary:  binary,
			Size:      size,
			SHA256:    "", // SHA256 calculated lazily when viewing
		})
	}

	// Sort by timestamp, newest first (LIFO - Last In, First Out)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp.After(items[j].Timestamp)
	})

	return items, nil
}

// generatePreviewFromFile creates a preview from a file path
func (qm *QueueManager) generatePreviewFromFile(filePath string) (string, error) {
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

// generateMetadataFromFile reads file and returns lightweight metadata (preview, binary status, size)
// Only reads first 4KB for efficiency - SHA256 is calculated lazily when viewing
func (qm *QueueManager) generateMetadataFromFile(filePath string) (preview string, binary bool, size int64, err error) {
	// Get file info for size without reading content
	fileInfo, err := fs.Stat(qm.fs, filePath)
	if err != nil {
		return "", false, 0, err
	}
	size = fileInfo.Size()

	// Read only first 4KB for preview and binary detection
	file, err := qm.fs.Open(filePath)
	if err != nil {
		return "", false, size, err
	}
	defer file.Close()

	// Read up to 4KB for preview/detection
	previewData := make([]byte, 4096)
	n, err := file.Read(previewData)
	if err != nil && err != io.EOF {
		return "", false, size, err
	}
	previewData = previewData[:n]

	// Generate preview and detect binary
	preview = qm.generatePreview(previewData)
	binary = isBinary(previewData)

	return preview, binary, size, nil
}

// Get returns the item at the specified index (0 = top of queue, newest)
func (qm *QueueManager) Get(index int) (*QueueItem, error) {
	items, err := qm.List()
	if err != nil {
		return nil, err
	}

	if index < 0 || index >= len(items) {
		return nil, fmt.Errorf("index %d out of range (0-%d)", index, len(items)-1)
	}

	return items[index], nil
}

// Size returns the number of items in the queue
func (qm *QueueManager) Size() (int, error) {
	items, err := qm.List()
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

// GetContentReader returns an io.ReadSeekCloser for the item's content
func (item *QueueItem) GetContentReader(fs FileSystem) (io.ReadSeekCloser, error) {
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

// Delete removes the item from the queue by index (deprecated, use DeleteByID)
func (qm *QueueManager) Delete(index int) error {
	item, err := qm.Get(index)
	if err != nil {
		return err
	}

	if err := qm.fs.Remove(item.FilePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// DeleteByID removes the item from the queue by its unique ID
func (qm *QueueManager) DeleteByID(id string) error {
	// Construct the filename from the ID
	filename := id + FileExtension

	// Check if the file exists by attempting to stat it
	if _, err := qm.fs.Open(filename); err != nil {
		return fmt.Errorf("item not found: %w", err)
	}

	// Delete the file
	if err := qm.fs.Remove(filename); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// Legacy type aliases for backward compatibility
type StackManager = QueueManager
type StackItem = QueueItem

// Legacy function aliases for backward compatibility
func NewStackManager(filesystem FileSystem) (*QueueManager, error) {
	return NewQueueManager(filesystem)
}

// GetHistoryLimit returns the configured history limit
func (qm *QueueManager) GetHistoryLimit() int {
	return qm.historyLimit
}

// Legacy method aliases for backward compatibility
func (qm *QueueManager) Push(content io.Reader) (*QueueItem, error) {
	return qm.Enqueue(content)
}

// Clear removes all items from the queue
func (qm *QueueManager) Clear() error {
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
	Item  *QueueItem
}

// Search searches for items matching the given regex pattern
func (qm *QueueManager) SearchIter(pattern string) *IterError[SearchResult] {
	newIter := func(outerErr *error) iter.Seq[SearchResult] {
		return func(yield func(SearchResult) bool) {
			// Compile the regex pattern
			re, err := regexp.Compile(pattern)
			if err != nil {
				*outerErr = fmt.Errorf("invalid regex pattern: %w", err)
				return
			}

			// Get all items from the queue
			items, err := qm.List()
			if err != nil {
				*outerErr = fmt.Errorf("failed to list items: %w", err)
				return
			}

			// Search through each item
			for i, item := range items {
				res, ok := func() (SearchResult, bool) {
					// Open the file to read its content
					file, err := qm.fs.Open(item.FilePath)
					if err != nil {
						return SearchResult{}, false
					}
					defer file.Close()
					buf := bufio.NewReader(file)
					// Check if the content matches the pattern
					if re.MatchReader(buf) {
						return SearchResult{
							Index: i,
							Item:  item,
						}, true
					}

					return SearchResult{}, false
				}()
				if ok {
					if !yield(res) {
						return
					}
				}
			}
		}
	}

	return newIterError(newIter)
}

// Search searches for items matching the given regex pattern
func (qm *QueueManager) Search(pattern string) ([]SearchResult, error) {
	iter := qm.SearchIter(pattern)

	res := []SearchResult{}
	for result := range iter.Iter() {
		res = append(res, result)
	}

	if iter.Err() != nil {
		return nil, iter.Err()
	}

	return res, nil
}

type IterError[T any] struct {
	iter iter.Seq[T]
	err  error
}

func (e *IterError[T]) Err() error {
	return e.err
}

func (e *IterError[T]) Iter() iter.Seq[T] {
	return e.iter
}

func newIterError[T any](createIter func(*error) iter.Seq[T]) *IterError[T] {
	res := &IterError[T]{}
	res.iter = createIter(&res.err)
	return res
}
