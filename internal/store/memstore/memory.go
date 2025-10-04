// Package memstore provides an in-memory implementation of the store interfaces.
// This implementation is designed for fast unit testing and does not persist data.
package memstore

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/yiblet/rem/internal/store"
)

// MemoryStore is an in-memory implementation of store.Store.
// It uses maps for storage and is thread-safe via mutexes.
// Data is not persisted and exists only for the lifetime of the process.
type MemoryStore struct {
	history *memoryHistoryStore
	config  *memoryConfigStore
}

// NewMemoryStore creates a new in-memory store for testing.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		history: newMemoryHistoryStore(),
		config:  newMemoryConfigStore(),
	}
}

// History returns the history store.
func (m *MemoryStore) History() store.HistoryStore {
	return m.history
}

// Config returns the config store.
func (m *MemoryStore) Config() store.ConfigStore {
	return m.config
}

// Close releases resources (no-op for memory store).
func (m *MemoryStore) Close() error {
	return nil
}

// memoryHistoryStore implements store.HistoryStore using in-memory maps.
type memoryHistoryStore struct {
	mu     sync.RWMutex
	items  map[uint]*historyEntry
	nextID uint
}

// historyEntry holds both the item metadata and content in memory.
type historyEntry struct {
	item    *store.HistoryItem
	content []byte
}

// newMemoryHistoryStore creates a new in-memory history store.
func newMemoryHistoryStore() *memoryHistoryStore {
	return &memoryHistoryStore{
		items:  make(map[uint]*historyEntry),
		nextID: 1,
	}
}

// Create stores a new history item by reading the entire content into memory.
func (m *memoryHistoryStore) Create(input *store.CreateHistoryInput) (*store.HistoryItem, error) {
	// Read entire content into memory
	content, err := io.ReadAll(input.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// Calculate SHA256 hash
	hash := sha256.Sum256(content)
	sha256Hash := hex.EncodeToString(hash[:])

	// Detect binary content
	isBinary := isBinary(content)

	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.nextID
	m.nextID++

	// Use input timestamp or current time
	timestamp := input.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	now := time.Now()
	item := &store.HistoryItem{
		ID:        id,
		Title:     input.Title,
		Timestamp: timestamp,
		IsBinary:  isBinary,
		Size:      int64(len(content)),
		SHA256:    sha256Hash,
		CreatedAt: now,
		UpdatedAt: now,
	}

	m.items[id] = &historyEntry{
		item:    item,
		content: content,
	}

	return item, nil
}

// List returns items sorted by timestamp descending (newest first - LIFO).
func (m *memoryHistoryStore) List(limit int) ([]*store.HistoryItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect all items
	items := make([]*store.HistoryItem, 0, len(m.items))
	for _, entry := range m.items {
		items = append(items, entry.item)
	}

	// Sort by timestamp descending (newest first - LIFO)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp.After(items[j].Timestamp)
	})

	// Apply limit
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	return items, nil
}

// Get retrieves a single item by ID (without content).
func (m *memoryHistoryStore) Get(id uint) (*store.HistoryItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.items[id]
	if !exists {
		return nil, fmt.Errorf("item not found: %d", id)
	}

	return entry.item, nil
}

// GetContent returns the content for an item as a ReadSeekCloser.
func (m *memoryHistoryStore) GetContent(id uint) (io.ReadSeekCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.items[id]
	if !exists {
		return nil, fmt.Errorf("item not found: %d", id)
	}

	return &bytesReadSeekCloser{reader: bytes.NewReader(entry.content)}, nil
}

// Delete removes an item by ID.
func (m *memoryHistoryStore) Delete(id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.items[id]; !exists {
		return fmt.Errorf("item not found: %d", id)
	}

	delete(m.items, id)
	return nil
}

// DeleteOldest removes the N oldest items by timestamp.
func (m *memoryHistoryStore) DeleteOldest(count int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get all items sorted by timestamp ascending (oldest first)
	items := make([]*store.HistoryItem, 0, len(m.items))
	for _, entry := range m.items {
		items = append(items, entry.item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp.Before(items[j].Timestamp)
	})

	// Delete oldest N items
	toDelete := count
	if toDelete > len(items) {
		toDelete = len(items)
	}

	for i := 0; i < toDelete; i++ {
		delete(m.items, items[i].ID)
	}

	return nil
}

// Count returns the total number of items.
func (m *memoryHistoryStore) Count() (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items), nil
}

// Clear removes all items.
func (m *memoryHistoryStore) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = make(map[uint]*historyEntry)
	return nil
}

// Search finds items matching the query pattern using regex.
func (m *memoryHistoryStore) Search(query *store.SearchQuery) ([]*store.HistoryItem, error) {
	if query.Pattern == "" {
		return []*store.HistoryItem{}, nil
	}

	// Compile regex pattern
	var re *regexp.Regexp
	var err error
	if query.CaseSensitive {
		re, err = regexp.Compile(query.Pattern)
	} else {
		re, err = regexp.Compile("(?i)" + query.Pattern)
	}
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Determine what to search
	searchTitle := query.SearchTitle
	searchContent := query.SearchContent
	// If both are false, search both (default behavior)
	if !searchTitle && !searchContent {
		searchTitle = true
		searchContent = true
	}

	var results []*store.HistoryItem

	// Search through all items
	for _, entry := range m.items {
		matched := false

		// Search in title if requested
		if searchTitle && re.MatchString(entry.item.Title) {
			matched = true
		}

		// Search in content if requested and not yet matched
		if !matched && searchContent {
			contentStr := string(entry.content)
			if re.MatchString(contentStr) {
				matched = true
			}
		}

		if matched {
			results = append(results, entry.item)

			// Check limit
			if query.Limit > 0 && len(results) >= query.Limit {
				break
			}
		}
	}

	// Sort results by timestamp descending (newest first - LIFO)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	return results, nil
}

// Close releases resources (no-op for memory store).
func (m *memoryHistoryStore) Close() error {
	return nil
}

// memoryConfigStore implements store.ConfigStore using an in-memory map.
type memoryConfigStore struct {
	mu     sync.RWMutex
	config map[string]string
}

// newMemoryConfigStore creates a new in-memory config store.
func newMemoryConfigStore() *memoryConfigStore {
	return &memoryConfigStore{
		config: make(map[string]string),
	}
}

// Get retrieves a configuration value by key.
func (m *memoryConfigStore) Get(key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, exists := m.config[key]
	if !exists {
		return "", fmt.Errorf("config key not found: %s", key)
	}

	return value, nil
}

// Set stores a configuration value.
func (m *memoryConfigStore) Set(key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config[key] = value
	return nil
}

// List returns a copy of all configuration key-value pairs.
func (m *memoryConfigStore) List() (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return copy to prevent external modification
	result := make(map[string]string, len(m.config))
	for k, v := range m.config {
		result[k] = v
	}

	return result, nil
}

// Delete removes a configuration key.
func (m *memoryConfigStore) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.config[key]; !exists {
		return fmt.Errorf("config key not found: %s", key)
	}

	delete(m.config, key)
	return nil
}

// Close releases resources (no-op for memory store).
func (m *memoryConfigStore) Close() error {
	return nil
}

// bytesReadSeekCloser wraps bytes.Reader to implement io.ReadSeekCloser.
type bytesReadSeekCloser struct {
	reader *bytes.Reader
}

// Read implements io.Reader.
func (b *bytesReadSeekCloser) Read(p []byte) (int, error) {
	return b.reader.Read(p)
}

// Seek implements io.Seeker.
func (b *bytesReadSeekCloser) Seek(offset int64, whence int) (int64, error) {
	return b.reader.Seek(offset, whence)
}

// Close implements io.Closer (no-op since bytes.Reader doesn't need closing).
func (b *bytesReadSeekCloser) Close() error {
	return nil
}

// isBinary detects if content contains binary data.
// Uses a simple heuristic: if content contains null bytes or has high ratio
// of non-printable characters, it's considered binary.
func isBinary(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// Check up to first 8KB for performance
	sampleSize := len(content)
	if sampleSize > 8192 {
		sampleSize = 8192
	}

	nonPrintable := 0
	for i := 0; i < sampleSize; i++ {
		b := content[i]

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
