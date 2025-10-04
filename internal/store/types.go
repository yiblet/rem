package store

import (
	"io"
	"time"
)

// HistoryItem represents a queue item without its content blob.
// This lightweight representation is used for listing and display
// purposes. Content must be retrieved separately via GetContent.
type HistoryItem struct {
	// ID is the unique identifier for this item.
	ID uint

	// Title is a user-provided or auto-generated title (max 80 characters).
	Title string

	// Timestamp is the creation time used for LIFO ordering.
	// Items with newer timestamps appear first in the queue.
	Timestamp time.Time

	// IsBinary indicates whether the content is binary data.
	// This affects display and preview behavior.
	IsBinary bool

	// Size is the total content size in bytes.
	Size int64

	// SHA256 is the hex-encoded SHA256 hash of the content.
	// Useful for deduplication and integrity verification.
	SHA256 string

	// CreatedAt is the timestamp when the item was first stored.
	// Managed automatically by the storage layer.
	CreatedAt time.Time

	// UpdatedAt is the timestamp of the last update.
	// Managed automatically by the storage layer.
	UpdatedAt time.Time
}

// CreateHistoryInput contains the data needed to create a new history item.
// The content is provided as an io.Reader to support streaming large files
// without loading them entirely into memory.
type CreateHistoryInput struct {
	// Title is the item's title (max 80 characters, required).
	// Should be sanitized and truncated before passing to Create.
	Title string

	// Content is the item's data as a stream (required).
	// Will be read completely and stored in chunks.
	// The reader will be consumed during the Create operation.
	Content io.Reader

	// Timestamp is the creation timestamp for LIFO ordering.
	// If zero, the storage layer should use the current time.
	Timestamp time.Time

	// IsBinary indicates if the content is binary.
	// If not set, the storage layer should detect it from the first chunk.
	IsBinary bool
}

// SearchQuery contains parameters for searching history items.
type SearchQuery struct {
	// Pattern is the regex pattern or text to search for.
	Pattern string

	// SearchTitle indicates whether to search in item titles.
	SearchTitle bool

	// SearchContent indicates whether to search in item content.
	SearchContent bool

	// Limit is the maximum number of results to return.
	// A value of 0 means no limit.
	Limit int

	// CaseSensitive indicates whether the search is case-sensitive.
	CaseSensitive bool
}

// SearchResult contains a single search result with match information.
type SearchResult struct {
	// Item is the matched history item (without content).
	Item *HistoryItem

	// Matches contains text snippets showing where the pattern matched.
	// This is optional and may be empty.
	Matches []string
}
