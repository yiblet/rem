package queue

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/yiblet/rem/internal/store"
)

const (
	DefaultMaxQueueSize = 255
)

// QueueManager manages the persistent LIFO queue using a store interface.
// It provides business logic for queue operations, title generation, and cleanup.
type QueueManager struct {
	store        store.Store
	historyLimit int
}

// NewQueueManager creates a new queue manager with the given store.
func NewQueueManager(s store.Store) (*QueueManager, error) {
	return NewQueueManagerWithConfig(s, DefaultMaxQueueSize)
}

// NewQueueManagerWithConfig creates a new queue manager with custom history limit.
func NewQueueManagerWithConfig(s store.Store, historyLimit int) (*QueueManager, error) {
	if historyLimit <= 0 {
		historyLimit = DefaultMaxQueueSize
	}

	qm := &QueueManager{
		store:        s,
		historyLimit: historyLimit,
	}

	return qm, nil
}

// Enqueue adds content with an optional title to the queue.
// If title is empty, generates title from first 4KB of content.
// Returns the created item with generated ID and metadata.
func (qm *QueueManager) Enqueue(content io.Reader, title string) (*store.HistoryItem, error) {
	// 1. Peek first chunk for title generation if needed
	var finalReader io.Reader
	var peekBuf []byte

	if title == "" {
		// Buffer first 4KB for title generation
		peekBuf = make([]byte, 4096)
		n, err := io.ReadFull(content, peekBuf)
		if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
			return nil, fmt.Errorf("failed to read content: %w", err)
		}
		peekBuf = peekBuf[:n]

		// Detect binary before generating title
		binary := isBinary(peekBuf)

		// Generate title from peeked content
		title = GenerateTitle(peekBuf, binary)

		// Create reader that replays peeked content + rest of stream
		finalReader = io.MultiReader(bytes.NewReader(peekBuf), content)
	} else {
		finalReader = content
	}

	// 2. Truncate title to 80 chars
	title = TruncateTitle(title, 80)

	// 3. Create store input (store handles chunking, hashing, binary detection)
	input := &store.CreateHistoryInput{
		Title:     title,
		Content:   finalReader,
		Timestamp: time.Now(),
	}

	// 4. Store in database (streaming into chunks)
	item, err := qm.store.History().Create(input)
	if err != nil {
		return nil, fmt.Errorf("failed to store item: %w", err)
	}

	// 5. Cleanup old items if over limit
	if err := qm.cleanupOldItems(); err != nil {
		return nil, fmt.Errorf("failed to cleanup: %w", err)
	}

	return item, nil
}

// List returns all items (excludes content), ordered newest first (LIFO).
func (qm *QueueManager) List() ([]*store.HistoryItem, error) {
	return qm.store.History().List(qm.historyLimit)
}

// Get returns an item by index (0 = newest).
func (qm *QueueManager) Get(index int) (*store.HistoryItem, error) {
	items, err := qm.List()
	if err != nil {
		return nil, err
	}

	if index < 0 || index >= len(items) {
		return nil, fmt.Errorf("index %d out of range (0-%d)", index, len(items)-1)
	}

	return items[index], nil
}

// GetContent returns a streaming reader for an item's content by ID.
func (qm *QueueManager) GetContent(id uint) (io.ReadSeekCloser, error) {
	return qm.store.History().GetContent(id)
}

// Delete removes an item by index.
func (qm *QueueManager) Delete(index int) error {
	item, err := qm.Get(index)
	if err != nil {
		return err
	}
	return qm.store.History().Delete(item.ID)
}

// Clear removes all items from the queue.
func (qm *QueueManager) Clear() error {
	return qm.store.History().Clear()
}

// Size returns the number of items in the queue.
func (qm *QueueManager) Size() (int, error) {
	return qm.store.History().Count()
}

// GetHistoryLimit returns the configured history limit.
func (qm *QueueManager) GetHistoryLimit() int {
	return qm.historyLimit
}

// Close releases store resources.
func (qm *QueueManager) Close() error {
	return qm.store.Close()
}

// cleanupOldItems removes items exceeding history limit.
func (qm *QueueManager) cleanupOldItems() error {
	count, err := qm.store.History().Count()
	if err != nil {
		return err
	}

	if count > qm.historyLimit {
		toDelete := count - qm.historyLimit
		return qm.store.History().DeleteOldest(toDelete)
	}

	return nil
}

// isBinary detects if data is binary by checking for non-printable characters
// and invalid UTF-8 sequences.
func isBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	// Check up to first 8KB for performance
	sampleSize := len(data)
	if sampleSize > 8192 {
		sampleSize = 8192
	}

	nonPrintable := 0
	for i := 0; i < sampleSize; i++ {
		b := data[i]

		// Null byte is a strong indicator of binary content
		if b == 0 {
			return true
		}

		// Count non-printable characters (excluding common whitespace)
		if b < 32 && b != '\n' && b != '\r' && b != '\t' {
			nonPrintable++
		}
	}

	// If more than 30% of characters are non-printable, consider it binary
	return float64(nonPrintable)/float64(sampleSize) > 0.3
}

// Legacy type aliases for backward compatibility
type StackManager = QueueManager
type StackItem = store.HistoryItem

// Legacy function aliases for backward compatibility
func NewStackManager(s store.Store) (*QueueManager, error) {
	return NewQueueManager(s)
}

// Legacy method aliases for backward compatibility

// Push is a legacy alias for Enqueue (without title parameter).
func (qm *QueueManager) Push(content io.Reader) (*store.HistoryItem, error) {
	return qm.Enqueue(content, "")
}
