package dbstore

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/yiblet/rem/internal/store"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SQLiteStore is a SQLite-backed implementation of store.Store
type SQLiteStore struct {
	db     *gorm.DB
	dbPath string
}

// NewSQLiteStore creates a new SQLite-backed store at the specified path.
// It initializes the database schema and sets up default configuration.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign key constraints in SQLite
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Run auto-migration for all models
	if err := db.AutoMigrate(&HistoryItemModel{}, &FileChunkModel{}, &ConfigItemModel{}); err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}

	store := &SQLiteStore{
		db:     db,
		dbPath: dbPath,
	}

	// Initialize default config
	if err := store.initDefaultConfig(); err != nil {
		return nil, fmt.Errorf("failed to init config: %w", err)
	}

	return store, nil
}

// History returns the history store
func (s *SQLiteStore) History() store.HistoryStore {
	return &sqliteHistoryStore{db: s.db}
}

// Config returns the config store
func (s *SQLiteStore) Config() store.ConfigStore {
	return &sqliteConfigStore{db: s.db}
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// initDefaultConfig sets up default configuration values
func (s *SQLiteStore) initDefaultConfig() error {
	defaults := map[string]string{
		"history_limit": "255",
		"show_binary":   "false",
		"db_version":    "1",
	}

	configStore := s.Config()
	for key, value := range defaults {
		// Only set if not already present
		if _, err := configStore.Get(key); err != nil {
			if err := configStore.Set(key, value); err != nil {
				return err
			}
		}
	}

	return nil
}

// sqliteHistoryStore implements store.HistoryStore using SQLite with chunked storage
type sqliteHistoryStore struct {
	db *gorm.DB
}

// Create stores a new history item with chunked content streaming
func (s *sqliteHistoryStore) Create(input *store.CreateHistoryInput) (*store.HistoryItem, error) {
	// 1. Create history item record (without size/SHA256 yet)
	item := &HistoryItemModel{
		Title:     input.Title,
		Timestamp: input.Timestamp,
		IsBinary:  false, // Determined from first chunk
	}
	if err := s.db.Create(item).Error; err != nil {
		return nil, fmt.Errorf("failed to create history item: %w", err)
	}

	// 2. Stream content into chunks
	hasher := sha256.New()
	reader := io.TeeReader(input.Content, hasher) // Hash while reading

	buffer := make([]byte, ChunkSize)
	sequence := 0
	totalSize := int64(0)
	firstChunk := true

	for {
		n, err := io.ReadFull(reader, buffer)
		if n > 0 {
			// Detect binary from first chunk
			if firstChunk {
				item.IsBinary = isBinary(buffer[:n])
				firstChunk = false
			}

			// Store chunk
			chunk := &FileChunkModel{
				HistoryID: item.ID,
				Sequence:  sequence,
				Data:      append([]byte(nil), buffer[:n]...), // Copy slice
			}
			if err := s.db.Create(chunk).Error; err != nil {
				// Rollback: delete item (CASCADE will delete chunks)
				s.db.Delete(item)
				return nil, fmt.Errorf("failed to create chunk: %w", err)
			}

			sequence++
			totalSize += int64(n)
		}

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			// Rollback: delete item (CASCADE will delete chunks)
			s.db.Delete(item)
			return nil, fmt.Errorf("failed to read content: %w", err)
		}
	}

	// 3. Update item with final size and hash
	item.Size = totalSize
	item.SHA256 = hex.EncodeToString(hasher.Sum(nil))
	if err := s.db.Save(item).Error; err != nil {
		return nil, fmt.Errorf("failed to update item metadata: %w", err)
	}

	return item.ToHistoryItem(), nil
}

// List returns items ordered by timestamp (newest first), excluding content
func (s *sqliteHistoryStore) List(limit int) ([]*store.HistoryItem, error) {
	var models []*HistoryItemModel

	query := s.db.
		Select("id", "title", "timestamp", "is_binary", "size", "sha256", "created_at", "updated_at").
		Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}

	items := make([]*store.HistoryItem, len(models))
	for i, model := range models {
		items[i] = model.ToHistoryItem()
	}

	return items, nil
}

// Get retrieves a single item by ID, excluding content
func (s *sqliteHistoryStore) Get(id uint) (*store.HistoryItem, error) {
	var model HistoryItemModel

	if err := s.db.
		Select("id", "title", "timestamp", "is_binary", "size", "sha256", "created_at", "updated_at").
		First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("item not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	return model.ToHistoryItem(), nil
}

// GetContent returns a streaming reader for an item's content
func (s *sqliteHistoryStore) GetContent(id uint) (io.ReadSeekCloser, error) {
	// Get item size
	var item HistoryItemModel
	if err := s.db.Select("size").First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("item not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	return &ChunkedReader{
		db:        s.db,
		historyID: id,
		totalSize: item.Size,
		chunkSeq:  0,
		chunkPos:  0,
	}, nil
}

// Delete removes an item by ID (CASCADE deletes chunks)
func (s *sqliteHistoryStore) Delete(id uint) error {
	result := s.db.Delete(&HistoryItemModel{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete item: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("item not found: %d", id)
	}
	return nil
}

// DeleteOldest removes the N oldest items based on timestamp
func (s *sqliteHistoryStore) DeleteOldest(count int) error {
	// Get IDs of oldest items
	var ids []uint
	err := s.db.Model(&HistoryItemModel{}).
		Select("id").
		Order("timestamp ASC").
		Limit(count).
		Pluck("id", &ids).Error

	if err != nil {
		return fmt.Errorf("failed to find oldest items: %w", err)
	}

	if len(ids) == 0 {
		return nil
	}

	// Delete by IDs (CASCADE deletes chunks)
	if err := s.db.Delete(&HistoryItemModel{}, ids).Error; err != nil {
		return fmt.Errorf("failed to delete items: %w", err)
	}

	return nil
}

// Count returns the total number of items
func (s *sqliteHistoryStore) Count() (int, error) {
	var count int64
	if err := s.db.Model(&HistoryItemModel{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count items: %w", err)
	}
	return int(count), nil
}

// Clear removes all items
func (s *sqliteHistoryStore) Clear() error {
	if err := s.db.Session(&gorm.Session{AllowGlobalUpdate: true}).
		Delete(&HistoryItemModel{}).Error; err != nil {
		return fmt.Errorf("failed to clear history: %w", err)
	}
	return nil
}

// Search finds items matching a pattern in title or content using regex
func (s *sqliteHistoryStore) Search(query *store.SearchQuery) ([]*store.HistoryItem, error) {
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

	// Determine what to search
	searchTitle := query.SearchTitle
	searchContent := query.SearchContent
	// If both are false, search both (default behavior)
	if !searchTitle && !searchContent {
		searchTitle = true
		searchContent = true
	}

	// Get all items (ordered by timestamp DESC - newest first)
	var models []*HistoryItemModel
	dbQuery := s.db.
		Select("id", "title", "timestamp", "is_binary", "size", "sha256", "created_at", "updated_at").
		Order("timestamp DESC")

	if err := dbQuery.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to list items for search: %w", err)
	}

	var results []*store.HistoryItem

	// Search through items
	for _, model := range models {
		matched := false

		// Search in title if requested
		if searchTitle && re.MatchString(model.Title) {
			matched = true
		}

		// Search in content if requested and not yet matched
		if !matched && searchContent {
			// Load all chunks for this item and search
			var chunks []FileChunkModel
			err := s.db.Where("history_id = ?", model.ID).
				Order("sequence ASC").
				Find(&chunks).Error
			if err != nil {
				return nil, fmt.Errorf("failed to load chunks for item %d: %w", model.ID, err)
			}

			// Reconstruct content from chunks and search
			var contentBuilder strings.Builder
			for _, chunk := range chunks {
				contentBuilder.Write(chunk.Data)
			}
			contentStr := contentBuilder.String()

			if re.MatchString(contentStr) {
				matched = true
			}
		}

		if matched {
			results = append(results, model.ToHistoryItem())

			// Check limit
			if query.Limit > 0 && len(results) >= query.Limit {
				break
			}
		}
	}

	return results, nil
}

// Close releases any resources
func (s *sqliteHistoryStore) Close() error {
	return nil // No-op, parent store handles DB closing
}

// ChunkedReader implements io.ReadSeekCloser for chunked content
type ChunkedReader struct {
	db        *gorm.DB
	historyID uint
	totalSize int64

	position int64  // Current read position
	chunkBuf []byte // Current chunk buffer
	chunkSeq int    // Current chunk sequence
	chunkPos int    // Position within current chunk
}

// Read reads data from the chunked content
func (c *ChunkedReader) Read(p []byte) (int, error) {
	if c.position >= c.totalSize {
		return 0, io.EOF
	}

	totalRead := 0

	for totalRead < len(p) && c.position < c.totalSize {
		// Load next chunk if needed
		if c.chunkBuf == nil || c.chunkPos >= len(c.chunkBuf) {
			if err := c.loadChunk(c.chunkSeq); err != nil {
				if totalRead > 0 {
					return totalRead, nil
				}
				return 0, err
			}
			c.chunkSeq++
			c.chunkPos = 0
		}

		// Copy from current chunk
		n := copy(p[totalRead:], c.chunkBuf[c.chunkPos:])
		c.chunkPos += n
		c.position += int64(n)
		totalRead += n
	}

	return totalRead, nil
}

// Seek seeks to a specific position in the content
func (c *ChunkedReader) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = c.position + offset
	case io.SeekEnd:
		newPos = c.totalSize + offset
	default:
		return 0, fmt.Errorf("invalid whence value")
	}

	if newPos < 0 {
		return 0, fmt.Errorf("negative seek position")
	}
	if newPos > c.totalSize {
		newPos = c.totalSize
	}

	// Calculate which chunk and offset within chunk
	chunkSeq := int(newPos / ChunkSize)
	chunkOffset := int(newPos % ChunkSize)

	// Load the target chunk if different from current
	if chunkSeq != c.chunkSeq || c.chunkBuf == nil {
		if err := c.loadChunk(chunkSeq); err != nil {
			return 0, err
		}
	}

	c.position = newPos
	c.chunkSeq = chunkSeq
	c.chunkPos = chunkOffset

	return newPos, nil
}

// loadChunk loads a specific chunk from the database
func (c *ChunkedReader) loadChunk(sequence int) error {
	var chunk FileChunkModel
	err := c.db.Where("history_id = ? AND sequence = ?", c.historyID, sequence).
		First(&chunk).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return io.EOF
		}
		return fmt.Errorf("failed to load chunk: %w", err)
	}
	c.chunkBuf = chunk.Data
	return nil
}

// Close releases the chunk buffer
func (c *ChunkedReader) Close() error {
	c.chunkBuf = nil
	return nil
}

// sqliteConfigStore implements store.ConfigStore using SQLite
type sqliteConfigStore struct {
	db *gorm.DB
}

// Get retrieves a configuration value by key
func (s *sqliteConfigStore) Get(key string) (string, error) {
	var model ConfigItemModel
	if err := s.db.First(&model, "key = ?", key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("config key not found: %s", key)
		}
		return "", fmt.Errorf("failed to get config: %w", err)
	}
	return model.Value, nil
}

// Set stores a configuration value (upsert)
func (s *sqliteConfigStore) Set(key, value string) error {
	model := &ConfigItemModel{
		Key:   key,
		Value: value,
	}

	// Upsert: update if exists, insert if not
	result := s.db.Where("key = ?", key).
		Assign(map[string]interface{}{"value": value, "updated_at": s.db.NowFunc()}).
		FirstOrCreate(model)

	if result.Error != nil {
		return fmt.Errorf("failed to set config: %w", result.Error)
	}

	return nil
}

// List returns all configuration key-value pairs
func (s *sqliteConfigStore) List() (map[string]string, error) {
	var models []ConfigItemModel
	if err := s.db.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to list config: %w", err)
	}

	result := make(map[string]string, len(models))
	for _, model := range models {
		result[model.Key] = model.Value
	}

	return result, nil
}

// Delete removes a configuration key
func (s *sqliteConfigStore) Delete(key string) error {
	result := s.db.Delete(&ConfigItemModel{}, "key = ?", key)
	if result.Error != nil {
		return fmt.Errorf("failed to delete config: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("config key not found: %s", key)
	}
	return nil
}

// Close releases any resources
func (s *sqliteConfigStore) Close() error {
	return nil // No-op, parent store handles DB closing
}

// isBinary detects if content is binary based on the first chunk
func isBinary(data []byte) bool {
	// Check for null bytes or high concentration of non-printable characters
	nullCount := 0
	nonPrintable := 0
	sampleSize := min(len(data), 512)

	for i := range sampleSize {
		b := data[i]
		if b == 0 {
			nullCount++
		}
		if b < 32 && b != '\n' && b != '\r' && b != '\t' {
			nonPrintable++
		}
	}

	// If more than 10% null bytes or 30% non-printable, consider binary
	if nullCount > sampleSize/10 || nonPrintable > sampleSize*3/10 {
		return true
	}

	return false
}
