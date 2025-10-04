package dbstore

import (
	"time"

	"github.com/yiblet/rem/internal/store"
)

// ChunkSize defines the size of each file chunk (32KB)
const ChunkSize = 32 * 1024

// HistoryItemModel represents a history item in the database.
// Content is stored separately in chunks, not in this table.
type HistoryItemModel struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	Title     string    `gorm:"size:80;not null;index"` // User-provided or auto-generated title
	Timestamp time.Time `gorm:"not null;index"`         // Creation timestamp for LIFO ordering
	IsBinary  bool      `gorm:"not null;default:false"` // Binary content flag
	Size      int64     `gorm:"not null"`               // Total content size in bytes
	SHA256    string    `gorm:"size:64"`                // SHA256 hash (computed during write)
	CreatedAt time.Time `gorm:"autoCreateTime"`         // GORM managed timestamp
	UpdatedAt time.Time `gorm:"autoUpdateTime"`         // GORM managed timestamp

	// One-to-many relationship with file chunks
	Chunks []FileChunkModel `gorm:"foreignKey:HistoryID;constraint:OnDelete:CASCADE"`
}

// TableName returns the table name for HistoryItemModel
func (HistoryItemModel) TableName() string {
	return "history_items"
}

// ToHistoryItem converts the GORM model to a store.HistoryItem
func (m *HistoryItemModel) ToHistoryItem() *store.HistoryItem {
	return &store.HistoryItem{
		ID:        m.ID,
		Title:     m.Title,
		Timestamp: m.Timestamp,
		IsBinary:  m.IsBinary,
		Size:      m.Size,
		SHA256:    m.SHA256,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// FileChunkModel represents a single chunk of file content.
// Content is split into 32KB chunks for streaming support.
type FileChunkModel struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	HistoryID uint      `gorm:"not null;index:idx_history_seq"` // Foreign key to history
	Sequence  int       `gorm:"not null;index:idx_history_seq"` // Chunk order (0, 1, 2, ...)
	Data      []byte    `gorm:"type:blob;not null"`             // Chunk data (max 32KB)
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// TableName returns the table name for FileChunkModel
func (FileChunkModel) TableName() string {
	return "file_chunks"
}

// ConfigItemModel represents a configuration key-value pair
type ConfigItemModel struct {
	Key       string    `gorm:"primaryKey;size:100"`
	Value     string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName returns the table name for ConfigItemModel
func (ConfigItemModel) TableName() string {
	return "config"
}
