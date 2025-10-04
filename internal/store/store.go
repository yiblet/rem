// Package store defines the storage interfaces for rem's persistence layer.
// It provides abstractions for both history (clipboard queue items) and
// configuration storage.
package store

import (
	"io"
)

// HistoryStore manages queue item persistence.
// It provides methods for creating, listing, retrieving, and deleting
// clipboard history items. Content is stored separately from metadata
// for efficient listing and streaming.
type HistoryStore interface {
	// Create stores a new history item from the provided input.
	// Content is read from the input's io.Reader and persisted.
	// Returns the created item with generated ID and metadata.
	Create(item *CreateHistoryInput) (*HistoryItem, error)

	// List returns items ordered by timestamp (newest first).
	// Content is excluded for performance - use GetContent to retrieve it.
	// If limit is 0, all items are returned. If limit > 0, at most limit items are returned.
	List(limit int) ([]*HistoryItem, error)

	// Get retrieves a single item by ID.
	// Content is excluded - use GetContent to retrieve it.
	Get(id uint) (*HistoryItem, error)

	// GetContent retrieves an item's content as a streaming reader.
	// The returned reader supports seeking for random access.
	// Caller is responsible for closing the reader.
	GetContent(id uint) (io.ReadSeekCloser, error)

	// Delete removes an item by ID.
	// Returns an error if the item does not exist.
	Delete(id uint) error

	// DeleteOldest removes the N oldest items based on timestamp.
	// If count exceeds the number of items, all items are deleted.
	DeleteOldest(count int) error

	// Count returns the total number of items in the store.
	Count() (int, error)

	// Clear removes all items from the store.
	Clear() error

	// Search finds items matching the query pattern.
	// Returns matching items with optional match snippets.
	Search(query *SearchQuery) ([]*HistoryItem, error)

	// Close releases any resources (DB connections, file handles, etc.).
	Close() error
}

// ConfigStore manages configuration persistence.
// Configuration is stored as key-value pairs.
type ConfigStore interface {
	// Get retrieves a configuration value by key.
	// Returns an error if the key does not exist.
	Get(key string) (string, error)

	// Set stores a configuration value.
	// If the key already exists, its value is updated.
	Set(key, value string) error

	// List returns all configuration key-value pairs.
	List() (map[string]string, error)

	// Delete removes a configuration key.
	// Returns an error if the key does not exist.
	Delete(key string) error

	// Close releases any resources.
	Close() error
}

// Store combines both history and config stores.
// Implementations provide access to both stores and manage
// their lifecycle as a single unit.
type Store interface {
	// History returns the history store for managing queue items.
	History() HistoryStore

	// Config returns the config store for managing settings.
	Config() ConfigStore

	// Close releases all resources for both stores.
	Close() error
}
