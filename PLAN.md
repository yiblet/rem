# SQLite3 Storage Migration with Titles Feature Implementation Plan

## Current Project Status
**Feature**: SQLite3 Storage Migration with Titles
**Phase**: Planning and Design
**Goal**: Migrate from file-based storage to SQLite3 database with GORM, add titles feature (max 80 chars), and store all configuration in database. This is a backwards-incompatible change.

## Overview

This plan describes the migration from the current file-based storage system to a SQLite3 database using GORM as the ORM. The architecture uses a clean separation between storage interface and implementations, with the queue manager acting as a business logic layer.

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Queue Manager                           │
│                   (Business Logic)                          │
│  - LIFO ordering                                           │
│  - Cleanup policies                                        │
│  - Title generation                                        │
│  - Content validation                                      │
└─────────────────────────────────────────────────────────────┘
                          │
                          │ uses
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                   Store Interface                           │
│                (internal/store/)                            │
│  - HistoryStore interface                                  │
│  - ConfigStore interface                                   │
│  - Input/Output types                                      │
└─────────────────────────────────────────────────────────────┘
                          │
              ┌───────────┴───────────┐
              ▼                       ▼
┌──────────────────────────┐  ┌──────────────────────────┐
│   SQLite Implementation  │  │  In-Memory Implementation│
│  (store/dbstore/)      │  │  (store/memstore/)       │
│  - GORM models           │  │  - Maps/slices           │
│  - Database queries      │  │  - No persistence        │
│  - Streaming support     │  │  - Fast testing          │
└──────────────────────────┘  └──────────────────────────┘
```

### Key Changes
1. **Storage Architecture**: Interface-based with multiple implementations
2. **Storage Backend**: File-based → Store interface (SQLite + in-memory)
3. **Content Storage**: Individual files → Chunked storage (32KB chunks)
4. **Streaming Model**: True streaming with io.Reader/io.ReadSeekCloser interfaces
5. **Configuration**: YAML file → Database (SQLite)
6. **New Feature**: Titles for queue items (max 80 characters)
7. **Memory Efficiency**: ~32KB max memory usage regardless of content size
8. **No Size Limits**: Handles arbitrarily large files via chunked streaming
9. **Backwards Compatibility**: None - clean break from old storage model
10. **Testability**: Mockable store interface, fast in-memory implementation

## Store Interface Design

### Package Structure

```
internal/
├── store/
│   ├── store.go           # Store interfaces and types
│   ├── types.go           # Input/Output types
│   ├── dbstore/
│   │   ├── sqlite.go      # SQLite implementation
│   │   ├── models.go      # GORM models
│   │   └── sqlite_test.go # SQLite-specific tests
│   └── memstore/
│       ├── memory.go      # In-memory implementation
│       └── memory_test.go # In-memory tests
└── queue/
    ├── manager.go         # Queue manager (uses Store interface)
    ├── title.go           # Title generation utilities
    └── manager_test.go    # Tests using memstore
```

### Store Interfaces

```go
// Package: internal/store/store.go

package store

import (
    "io"
    "time"
)

// HistoryStore manages queue item persistence
type HistoryStore interface {
    // Create stores a new history item
    Create(item *CreateHistoryInput) (*HistoryItem, error)

    // List returns items ordered by timestamp (newest first)
    // Excludes content blob for performance
    List(limit int) ([]*HistoryItem, error)

    // Get retrieves a single item by ID (excludes content blob)
    Get(id uint) (*HistoryItem, error)

    // GetContent retrieves an item's content as a streaming reader
    GetContent(id uint) (io.ReadSeekCloser, error)

    // Delete removes an item by ID
    Delete(id uint) error

    // DeleteOldest removes the N oldest items
    DeleteOldest(count int) error

    // Count returns the total number of items
    Count() (int, error)

    // Clear removes all items
    Clear() error

    // Search finds items matching a pattern in title or content
    Search(query *SearchQuery) ([]*HistoryItem, error)

    // Close releases any resources (DB connections, etc.)
    Close() error
}

// ConfigStore manages configuration persistence
type ConfigStore interface {
    // Get retrieves a configuration value
    Get(key string) (string, error)

    // Set stores a configuration value
    Set(key, value string) error

    // List returns all configuration key-value pairs
    List() (map[string]string, error)

    // Delete removes a configuration key
    Delete(key string) error

    // Close releases any resources
    Close() error
}

// Store combines both history and config stores
type Store interface {
    History() HistoryStore
    Config() ConfigStore
    Close() error
}
```

### Store Types

```go
// Package: internal/store/types.go

package store

import (
    "io"
    "time"
)

// HistoryItem represents a queue item (without content blob)
type HistoryItem struct {
    ID        uint      // Unique identifier
    Title     string    // User-provided or auto-generated title (max 80 chars)
    Timestamp time.Time // Creation timestamp (for LIFO ordering)
    IsBinary  bool      // Binary content flag
    Size      int64     // Content size in bytes
    SHA256    string    // SHA256 hash (for binary files or deduplication)
    CreatedAt time.Time // GORM managed timestamp
    UpdatedAt time.Time // GORM managed timestamp
}

// CreateHistoryInput contains data for creating a new history item
type CreateHistoryInput struct {
    Title     string    // Title (max 80 chars, required)
    Content   io.Reader // Content stream (required) - will be chunked during storage
    Timestamp time.Time // Creation timestamp (defaults to now)
    IsBinary  bool      // Binary content flag (set after reading first chunk)
}

// SearchQuery contains search parameters
type SearchQuery struct {
    Pattern       string       // Regex pattern or text to search
    SearchTitle   bool         // Search in titles
    SearchContent bool         // Search in content
    Limit         int          // Max results (0 = no limit)
    CaseSensitive bool         // Case-sensitive search
}

// SearchResult contains search results with match information
type SearchResult struct {
    Item     *HistoryItem // The matched item
    Matches  []string     // Matched text snippets (optional)
}
```

## Database Schema Design (SQLite Implementation)

### Chunked Storage Architecture

To enable true streaming without memory constraints, content is stored in 32KB chunks. This allows:
- **Streaming Writes**: Read from `io.Reader` → write chunks progressively
- **Streaming Reads**: Read chunks sequentially → expose as `io.Reader`
- **No Memory Limits**: Handle arbitrarily large files without loading into memory
- **Efficient Seeking**: Seek to specific chunks for random access

### Table 1: `history`
Stores queue item metadata (no content blob).

```go
// Package: internal/store/dbstore/models.go

type HistoryItemModel struct {
    ID        uint      `gorm:"primaryKey;autoIncrement"`
    Title     string    `gorm:"size:80;not null;index"`  // User-provided or auto-generated title
    Timestamp time.Time `gorm:"not null;index"`          // Creation timestamp
    IsBinary  bool      `gorm:"not null;default:false"`  // Binary content flag
    Size      int64     `gorm:"not null"`                // Total content size in bytes
    SHA256    string    `gorm:"size:64"`                 // SHA256 hash (computed during write)
    CreatedAt time.Time `gorm:"autoCreateTime"`          // GORM managed timestamp
    UpdatedAt time.Time `gorm:"autoUpdateTime"`          // GORM managed timestamp

    // One-to-many relationship with file chunks
    Chunks    []FileChunkModel `gorm:"foreignKey:HistoryID;constraint:OnDelete:CASCADE"`
}

func (HistoryItemModel) TableName() string {
    return "history"
}

// ToHistoryItem converts GORM model to store type
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
```

**Indexes:**
- Primary key on `id` (auto-increment)
- Index on `timestamp` for efficient LIFO ordering (DESC)
- Index on `title` for efficient title-based searches

**Design Decisions:**
- Content removed from history table (stored in chunks)
- One-to-many relationship with file_chunks
- Cascade delete ensures chunks are removed when item is deleted
- SHA256 computed incrementally during chunk writes

### Table 2: `file_chunks`
Stores content chunks for streaming support (32KB per chunk).

```go
// Package: internal/store/dbstore/models.go

const ChunkSize = 32 * 1024 // 32KB chunks

type FileChunkModel struct {
    ID        uint   `gorm:"primaryKey;autoIncrement"`
    HistoryID uint   `gorm:"not null;index:idx_history_seq"` // Foreign key to history
    Sequence  int    `gorm:"not null;index:idx_history_seq"` // Chunk order (0, 1, 2, ...)
    Data      []byte `gorm:"type:blob;not null"`             // Chunk data (max 32KB)

    CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (FileChunkModel) TableName() string {
    return "file_chunks"
}
```

**Indexes:**
- Primary key on `id` (auto-increment)
- Composite index on `(history_id, sequence)` for efficient ordered retrieval
- Foreign key constraint with CASCADE delete

**Design Decisions:**
- Fixed 32KB chunk size for predictable memory usage
- Sequence number ensures correct ordering when streaming out
- Composite index enables efficient range queries: `WHERE history_id = ? ORDER BY sequence`
- Last chunk may be < 32KB (remainder)

### Table 3: `config`
Stores configuration as key-value pairs.

```go
// Package: internal/store/dbstore/models.go

type ConfigItemModel struct {
    Key       string    `gorm:"primaryKey;size:100"`
    Value     string    `gorm:"type:text;not null"`
    CreatedAt time.Time `gorm:"autoCreateTime"`
    UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (ConfigItemModel) TableName() string {
    return "config"
}
```

**Default Configuration Keys:**
- `history_limit`: Maximum queue size (default: "255")
- `show_binary`: Show binary file previews (default: "false")
- `db_version`: Schema version for future migrations (default: "1")

## SQLite Implementation

### SQLite Store Structure

```go
// Package: internal/store/dbstore/sqlite.go

type SQLiteStore struct {
    db     *gorm.DB
    dbPath string
}

// NewSQLiteStore creates a new SQLite-backed store
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
    db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Silent),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    // Run auto-migration
    if err := db.AutoMigrate(&HistoryItemModel{}, &ConfigItemModel{}); err != nil {
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

func (s *SQLiteStore) History() store.HistoryStore {
    return &sqliteHistoryStore{db: s.db}
}

func (s *SQLiteStore) Config() store.ConfigStore {
    return &sqliteConfigStore{db: s.db}
}

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
```

### SQLite History Store

```go
// Package: internal/store/dbstore/sqlite.go

type sqliteHistoryStore struct {
    db *gorm.DB
}

func (s *sqliteHistoryStore) Create(input *store.CreateHistoryInput) (*store.HistoryItem, error) {
    model := &HistoryItemModel{
        Title:     input.Title,
        Content:   input.Content,
        Timestamp: input.Timestamp,
        IsBinary:  input.IsBinary,
        Size:      int64(len(input.Content)),
        SHA256:    input.SHA256,
    }

    if err := s.db.Create(model).Error; err != nil {
        return nil, fmt.Errorf("failed to create history item: %w", err)
    }

    return model.ToHistoryItem(), nil
}

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

func (s *sqliteHistoryStore) Get(id uint) (*store.HistoryItem, error) {
    var model HistoryItemModel

    if err := s.db.
        Select("id", "title", "timestamp", "is_binary", "size", "sha256", "created_at", "updated_at").
        First(&model, id).Error; err != nil {
        return nil, fmt.Errorf("failed to get item: %w", err)
    }

    return model.ToHistoryItem(), nil
}

func (s *sqliteHistoryStore) GetContent(id uint) (io.ReadSeekCloser, error) {
    var content []byte

    if err := s.db.Model(&HistoryItemModel{}).
        Select("content").
        Where("id = ?", id).
        Pluck("content", &content).Error; err != nil {
        return nil, fmt.Errorf("failed to get content: %w", err)
    }

    return &bytesReadSeekCloser{reader: bytes.NewReader(content)}, nil
}

func (s *sqliteHistoryStore) Delete(id uint) error {
    result := s.db.Delete(&HistoryItemModel{}, id)
    if result.Error != nil {
        return fmt.Errorf("failed to delete item: %w", result.Error)
    }
    if result.RowsAffected == 0 {
        return fmt.Errorf("item not found")
    }
    return nil
}

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

    // Delete by IDs
    if err := s.db.Delete(&HistoryItemModel{}, ids).Error; err != nil {
        return fmt.Errorf("failed to delete items: %w", err)
    }

    return nil
}

func (s *sqliteHistoryStore) Count() (int, error) {
    var count int64
    if err := s.db.Model(&HistoryItemModel{}).Count(&count).Error; err != nil {
        return 0, fmt.Errorf("failed to count items: %w", err)
    }
    return int(count), nil
}

func (s *sqliteHistoryStore) Clear() error {
    if err := s.db.Session(&gorm.Session{AllowGlobalUpdate: true}).
        Delete(&HistoryItemModel{}).Error; err != nil {
        return fmt.Errorf("failed to clear history: %w", err)
    }
    return nil
}

func (s *sqliteHistoryStore) Search(query *store.SearchQuery) ([]*store.HistoryItem, error) {
    // Implementation uses regex pattern matching on title and/or content
    // See detailed implementation in Phase 3
    return nil, fmt.Errorf("not implemented")
}

func (s *sqliteHistoryStore) Close() error {
    return nil // No-op, parent store handles DB closing
}

// bytesReadSeekCloser wraps bytes.Reader to implement io.ReadSeekCloser
type bytesReadSeekCloser struct {
    reader *bytes.Reader
}

func (b *bytesReadSeekCloser) Read(p []byte) (int, error) {
    return b.reader.Read(p)
}

func (b *bytesReadSeekCloser) Seek(offset int64, whence int) (int64, error) {
    return b.reader.Seek(offset, whence)
}

func (b *bytesReadSeekCloser) Close() error {
    return nil // bytes.Reader doesn't need closing
}
```

### SQLite Config Store

```go
// Package: internal/store/dbstore/sqlite.go

type sqliteConfigStore struct {
    db *gorm.DB
}

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

func (s *sqliteConfigStore) Set(key, value string) error {
    model := &ConfigItemModel{
        Key:   key,
        Value: value,
    }

    // Upsert: update if exists, insert if not
    result := s.db.Where("key = ?", key).
        Assign(map[string]interface{}{"value": value}).
        FirstOrCreate(model)

    if result.Error != nil {
        return fmt.Errorf("failed to set config: %w", result.Error)
    }

    return nil
}

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

func (s *sqliteConfigStore) Close() error {
    return nil // No-op, parent store handles DB closing
}
```

## In-Memory Implementation

### Memory Store Structure

```go
// Package: internal/store/memstore/memory.go

type MemoryStore struct {
    mu      sync.RWMutex
    history *memoryHistoryStore
    config  *memoryConfigStore
}

// NewMemoryStore creates a new in-memory store (for testing)
func NewMemoryStore() *MemoryStore {
    return &MemoryStore{
        history: newMemoryHistoryStore(),
        config:  newMemoryConfigStore(),
    }
}

func (m *MemoryStore) History() store.HistoryStore {
    return m.history
}

func (m *MemoryStore) Config() store.ConfigStore {
    return m.config
}

func (m *MemoryStore) Close() error {
    return nil // No resources to clean up
}
```

### Memory History Store

```go
// Package: internal/store/memstore/memory.go

type memoryHistoryStore struct {
    mu        sync.RWMutex
    items     map[uint]*historyEntry
    nextID    uint
}

type historyEntry struct {
    item    *store.HistoryItem
    content []byte
}

func newMemoryHistoryStore() *memoryHistoryStore {
    return &memoryHistoryStore{
        items:  make(map[uint]*historyEntry),
        nextID: 1,
    }
}

func (m *memoryHistoryStore) Create(input *store.CreateHistoryInput) (*store.HistoryItem, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    id := m.nextID
    m.nextID++

    item := &store.HistoryItem{
        ID:        id,
        Title:     input.Title,
        Timestamp: input.Timestamp,
        IsBinary:  input.IsBinary,
        Size:      int64(len(input.Content)),
        SHA256:    input.SHA256,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    m.items[id] = &historyEntry{
        item:    item,
        content: input.Content,
    }

    return item, nil
}

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

func (m *memoryHistoryStore) Get(id uint) (*store.HistoryItem, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    entry, exists := m.items[id]
    if !exists {
        return nil, fmt.Errorf("item not found: %d", id)
    }

    return entry.item, nil
}

func (m *memoryHistoryStore) GetContent(id uint) (io.ReadSeekCloser, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    entry, exists := m.items[id]
    if !exists {
        return nil, fmt.Errorf("item not found: %d", id)
    }

    return &bytesReadSeekCloser{reader: bytes.NewReader(entry.content)}, nil
}

func (m *memoryHistoryStore) Delete(id uint) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if _, exists := m.items[id]; !exists {
        return fmt.Errorf("item not found: %d", id)
    }

    delete(m.items, id)
    return nil
}

func (m *memoryHistoryStore) DeleteOldest(count int) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    // Get all items sorted by timestamp
    items := make([]*store.HistoryItem, 0, len(m.items))
    for _, entry := range m.items {
        items = append(items, entry.item)
    }

    sort.Slice(items, func(i, j int) bool {
        return items[i].Timestamp.Before(items[j].Timestamp) // Oldest first
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

func (m *memoryHistoryStore) Count() (int, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return len(m.items), nil
}

func (m *memoryHistoryStore) Clear() error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.items = make(map[uint]*historyEntry)
    return nil
}

func (m *memoryHistoryStore) Search(query *store.SearchQuery) ([]*store.HistoryItem, error) {
    // Implementation uses regex pattern matching on title and/or content
    // See detailed implementation in Phase 3
    return nil, fmt.Errorf("not implemented")
}

func (m *memoryHistoryStore) Close() error {
    return nil
}

// bytesReadSeekCloser implementation (same as SQLite)
type bytesReadSeekCloser struct {
    reader *bytes.Reader
}

func (b *bytesReadSeekCloser) Read(p []byte) (int, error) {
    return b.reader.Read(p)
}

func (b *bytesReadSeekCloser) Seek(offset int64, whence int) (int64, error) {
    return b.reader.Seek(offset, whence)
}

func (b *bytesReadSeekCloser) Close() error {
    return nil
}
```

### Memory Config Store

```go
// Package: internal/store/memstore/memory.go

type memoryConfigStore struct {
    mu     sync.RWMutex
    config map[string]string
}

func newMemoryConfigStore() *memoryConfigStore {
    return &memoryConfigStore{
        config: make(map[string]string),
    }
}

func (m *memoryConfigStore) Get(key string) (string, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    value, exists := m.config[key]
    if !exists {
        return "", fmt.Errorf("config key not found: %s", key)
    }

    return value, nil
}

func (m *memoryConfigStore) Set(key, value string) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.config[key] = value
    return nil
}

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

func (m *memoryConfigStore) Delete(key string) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if _, exists := m.config[key]; !exists {
        return fmt.Errorf("config key not found: %s", key)
    }

    delete(m.config, key)
    return nil
}

func (m *memoryConfigStore) Close() error {
    return nil
}
```

## Queue Manager Refactoring

The queue manager becomes a business logic layer that uses the store interface:

```go
// Package: internal/queue/manager.go

type QueueManager struct {
    store        store.Store
    historyLimit int
}

// NewQueueManager creates a queue manager with the given store
func NewQueueManager(s store.Store) (*QueueManager, error) {
    return NewQueueManagerWithConfig(s, DefaultMaxQueueSize)
}

// NewQueueManagerWithConfig creates a queue manager with custom history limit
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

// Enqueue adds content with an optional title
func (qm *QueueManager) Enqueue(content io.Reader, title string) (*store.HistoryItem, error) {
    // 1. Peek first chunk for title generation if needed
    var finalReader io.Reader
    if title == "" {
        // Buffer first 4KB for title generation
        peekBuf := make([]byte, 4096)
        n, err := io.ReadFull(content, peekBuf)
        if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
            return nil, fmt.Errorf("failed to read content: %w", err)
        }
        peekBuf = peekBuf[:n]

        // Generate title from peeked content
        title = GenerateTitle(peekBuf, isBinary(peekBuf))

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

// List returns all items (excludes content)
func (qm *QueueManager) List() ([]*store.HistoryItem, error) {
    return qm.store.History().List(qm.historyLimit)
}

// Get returns an item by index (0 = newest)
func (qm *QueueManager) Get(index int) (*store.HistoryItem, error) {
    items, err := qm.List()
    if err != nil {
        return nil, err
    }

    if index < 0 || index >= len(items) {
        return nil, fmt.Errorf("index %d out of range", index)
    }

    return items[index], nil
}

// GetContent returns a streaming reader for an item's content
func (qm *QueueManager) GetContent(id uint) (io.ReadSeekCloser, error) {
    return qm.store.History().GetContent(id)
}

// Delete removes an item by index
func (qm *QueueManager) Delete(index int) error {
    item, err := qm.Get(index)
    if err != nil {
        return err
    }
    return qm.store.History().Delete(item.ID)
}

// Clear removes all items
func (qm *QueueManager) Clear() error {
    return qm.store.History().Clear()
}

// cleanupOldItems removes items exceeding history limit
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

// Close releases store resources
func (qm *QueueManager) Close() error {
    return qm.store.Close()
}
```

## Streaming Architecture with Store Interface

### Chunked Storage Model

Content is stored in 32KB chunks to enable true streaming without memory constraints. This architecture supports files of any size.

### Write Path (Enqueue): Stream-to-Chunks

The store accepts `io.Reader` and streams content directly into chunk storage:

```go
func (s *sqliteHistoryStore) Create(input *store.CreateHistoryInput) (*store.HistoryItem, error) {
    // 1. Create history item record (without size/SHA256 yet)
    item := &HistoryItemModel{
        Title:     input.Title,
        Timestamp: input.Timestamp,
        IsBinary:  false, // Determined from first chunk
    }
    if err := s.db.Create(item).Error; err != nil {
        return nil, err
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
                // Rollback: delete item and chunks
                s.db.Delete(item)
                return nil, err
            }

            sequence++
            totalSize += int64(n)
        }

        if err == io.EOF || err == io.ErrUnexpectedEOF {
            break
        }
        if err != nil {
            // Rollback: delete item and chunks
            s.db.Delete(item)
            return nil, err
        }
    }

    // 3. Update item with final size and hash
    item.Size = totalSize
    item.SHA256 = hex.EncodeToString(hasher.Sum(nil))
    if err := s.db.Save(item).Error; err != nil {
        return nil, err
    }

    return item.ToHistoryItem(), nil
}
```

**Benefits:**
- No memory buffering required (32KB max at any time)
- Handles arbitrarily large files
- SHA256 computed incrementally
- Binary detection from first chunk
- Transaction-like behavior with rollback on failure

### Read Path (Get): Chunks-to-Stream

The store provides `GetContent()` which returns an `io.ReadSeekCloser` that streams chunks on demand:

```go
// ChunkedReader implements io.ReadSeekCloser for chunked content
type ChunkedReader struct {
    db        *gorm.DB
    historyID uint
    totalSize int64

    position  int64 // Current read position
    chunkBuf  []byte // Current chunk buffer
    chunkSeq  int    // Current chunk sequence
    chunkPos  int    // Position within current chunk
}

func (c *ChunkedReader) Read(p []byte) (int, error) {
    if c.position >= c.totalSize {
        return 0, io.EOF
    }

    // Load next chunk if needed
    if c.chunkBuf == nil || c.chunkPos >= len(c.chunkBuf) {
        if err := c.loadChunk(c.chunkSeq); err != nil {
            return 0, err
        }
        c.chunkSeq++
        c.chunkPos = 0
    }

    // Copy from current chunk
    n := copy(p, c.chunkBuf[c.chunkPos:])
    c.chunkPos += n
    c.position += int64(n)

    return n, nil
}

func (c *ChunkedReader) Seek(offset int64, whence int) (int64, error) {
    var newPos int64
    switch whence {
    case io.SeekStart:
        newPos = offset
    case io.SeekCurrent:
        newPos = c.position + offset
    case io.SeekEnd:
        newPos = c.totalSize + offset
    }

    if newPos < 0 || newPos > c.totalSize {
        return 0, fmt.Errorf("invalid seek position")
    }

    // Calculate which chunk and offset within chunk
    chunkSeq := int(newPos / ChunkSize)
    chunkOffset := int(newPos % ChunkSize)

    // Load the target chunk
    if err := c.loadChunk(chunkSeq); err != nil {
        return 0, err
    }

    c.position = newPos
    c.chunkSeq = chunkSeq
    c.chunkPos = chunkOffset

    return newPos, nil
}

func (c *ChunkedReader) loadChunk(sequence int) error {
    var chunk FileChunkModel
    err := c.db.Where("history_id = ? AND sequence = ?", c.historyID, sequence).
        First(&chunk).Error
    if err != nil {
        return err
    }
    c.chunkBuf = chunk.Data
    return nil
}

func (c *ChunkedReader) Close() error {
    c.chunkBuf = nil
    return nil
}

func (s *sqliteHistoryStore) GetContent(id uint) (io.ReadSeekCloser, error) {
    // Get item size
    var item HistoryItemModel
    if err := s.db.Select("size").First(&item, id).Error; err != nil {
        return nil, err
    }

    return &ChunkedReader{
        db:        s.db,
        historyID: id,
        totalSize: item.Size,
        chunkSeq:  0,
        chunkPos:  0,
    }, nil
}
```

**Benefits:**
- Only one chunk (32KB) in memory at a time
- Supports seeking for random access
- Lazy loading of chunks
- No size limits

### In-Memory Implementation

The in-memory store still uses simple byte slices (for testing):

```go
func (m *memoryHistoryStore) Create(input *store.CreateHistoryInput) (*store.HistoryItem, error) {
    // Read entire content into memory (acceptable for tests)
    content, err := io.ReadAll(input.Content)
    if err != nil {
        return nil, err
    }

    // Store as single blob
    id := m.nextID
    m.nextID++

    item := &store.HistoryItem{
        ID:        id,
        Title:     input.Title,
        Timestamp: input.Timestamp,
        IsBinary:  isBinary(content),
        Size:      int64(len(content)),
        SHA256:    computeSHA256(content),
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    m.items[id] = &historyEntry{
        item:    item,
        content: content,
    }

    return item, nil
}

func (m *memoryHistoryStore) GetContent(id uint) (io.ReadSeekCloser, error) {
    entry, exists := m.items[id]
    if !exists {
        return nil, fmt.Errorf("item not found")
    }

    return &bytesReadSeekCloser{reader: bytes.NewReader(entry.content)}, nil
}
```

### Optimized Listing: Metadata Only

The store interface separates item metadata from content:
- `List()` returns `HistoryItem` without any content/chunks
- `GetContent()` explicitly fetches chunks only when needed
- TUI left pane uses `List()` for instant rendering

### Memory Usage Analysis

**Write (Enqueue):**
- Memory: ~32KB buffer + hash state (~256 bytes) = ~32KB total
- Works for any file size

**Read (GetContent):**
- Memory: ~32KB chunk buffer = ~32KB total
- Works for any file size

**List (TUI):**
- Memory: Item metadata only (~200 bytes/item × 255 items) = ~50KB
- Fast and lightweight

## Title Feature Design

### Title Generation Utilities

```go
// Package: internal/queue/title.go

// GenerateTitle creates a title from a content sample (first few KB)
// Note: This only uses the first ~4KB for title generation, not the entire content
func GenerateTitle(sample []byte, isBinary bool) string {
    if isBinary {
        return "[binary content]"
    }

    if len(sample) == 0 {
        return "[empty]"
    }

    // Convert to string and get first non-empty line
    text := string(sample)
    lines := strings.Split(text, "\n")

    for _, line := range lines {
        cleaned := strings.TrimSpace(line)
        if cleaned != "" {
            // Found a non-empty line, use it as title
            return TruncateTitle(cleaned, 80)
        }
    }

    // Fallback: use cleaned content (collapse whitespace)
    cleaned := strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
    if cleaned == "" {
        return "[empty]"
    }

    return TruncateTitle(cleaned, 80)
}

// TruncateTitle ensures title is at most maxLen characters
func TruncateTitle(title string, maxLen int) string {
    title = strings.TrimSpace(title)

    if len(title) <= maxLen {
        return title
    }

    return title[:maxLen-3] + "..."
}

// SanitizeTitle removes control characters
func SanitizeTitle(title string) string {
    // Remove control characters except tab
    title = strings.Map(func(r rune) rune {
        if r < 32 && r != '\t' {
            return -1
        }
        return r
    }, title)

    // Collapse whitespace
    fields := strings.Fields(title)
    return strings.Join(fields, " ")
}
```

### Title in CLI

```bash
# Store with explicit title
echo "content" | rem store --title "My Important Note"

# Store from file with explicit title
rem store --title "Config File" config.yaml

# Store with auto-generated title (from first line)
echo "Hello World" | rem store
# Creates item with title "Hello World"
```

### Title Display in TUI

**Left Pane:**
- Show title instead of preview (first ~20 chars)
- Truncate with ellipsis if needed

**Right Pane Status:**
- Show full title (80 chars max)

## Implementation Phases

### Phase 1: Store Interface and Types [COMPLETED]
**Priority**: High - Foundation for all other phases
**Complexity**: Low-Medium
**Dependencies**: None
**Completed**: 2025-10-03
**Commit**: 3851ce5

#### 1.1 Store Interface Definition
- Create `internal/store/store.go` with store interfaces
- Create `internal/store/types.go` with input/output types
- Define `HistoryStore` and `ConfigStore` interfaces
- Define `CreateHistoryInput`, `HistoryItem`, `SearchQuery` types

#### 1.2 Package Structure
- Create directory structure: `store/`, `store/dbstore/`, `store/memstore/`
- Set up package documentation and comments
- Add interface compliance tests

**Phase 1 Testing:**
- **Unit Tests**: Interface type definitions compile correctly
- **Unit Tests**: Type conversion and validation logic

**Phase 1 Deliverables:**
- `internal/store/store.go` - Store interfaces
- `internal/store/types.go` - Input/Output types
- `internal/store/store_test.go` - Interface compliance tests

---

### Phase 2: In-Memory Store Implementation [COMPLETED]
**Priority**: High - Enables testing without SQLite
**Complexity**: Medium
**Dependencies**: Phase 1 complete
**Completed**: 2025-10-03
**Commit**: 7d1afb8

#### 2.1 Memory Store Structure
- Create `internal/store/memstore/memory.go`
- Implement `MemoryStore` with maps for storage
- Implement thread-safe operations with mutexes
- Implement LIFO ordering in `List()`

#### 2.2 Memory History Store
- Implement all `HistoryStore` interface methods
- Use map for items, track next ID
- Store content in memory alongside metadata
- Implement search functionality

#### 2.3 Memory Config Store
- Implement all `ConfigStore` interface methods
- Use simple map for key-value storage
- Thread-safe operations

**Phase 2 Testing:**
- **Unit Tests**: All HistoryStore methods with various inputs
- **Unit Tests**: All ConfigStore methods
- **Unit Tests**: LIFO ordering correctness
- **Unit Tests**: Thread safety (concurrent operations)
- **Unit Tests**: Edge cases (empty store, deletions, etc.)

**Phase 2 Deliverables:**
- `internal/store/memstore/memory.go` - In-memory implementation
- `internal/store/memstore/memory_test.go` - Comprehensive tests

---

### Phase 3: SQLite Store Implementation [COMPLETED]
**Priority**: High - Production storage backend
**Complexity**: Medium-High
**Dependencies**: Phase 1 complete
**Completed**: 2025-10-03
**Commit**: 209d641

#### 3.1 GORM Models
- Create `internal/store/dbstore/models.go`
- Define `HistoryItemModel` and `ConfigItemModel`
- Add GORM tags and indexes
- Implement model-to-type conversions

#### 3.2 SQLite Store Structure
- Create `internal/store/dbstore/sqlite.go`
- Implement `SQLiteStore` with GORM database
- Implement database initialization and migration
- Add default config initialization

#### 3.3 SQLite History Store
- Implement all `HistoryStore` interface methods
- Optimize queries (exclude content in List/Get)
- Implement content streaming with `GetContent()`
- Add proper error handling and transactions

#### 3.4 SQLite Config Store
- Implement all `ConfigStore` interface methods
- Use upsert pattern for Set()
- Handle not-found errors gracefully

**Phase 3 Testing:**
- **Unit Tests**: GORM model validation
- **Unit Tests**: Database initialization and schema
- **Integration Tests**: All HistoryStore methods with real SQLite
- **Integration Tests**: All ConfigStore methods
- **Integration Tests**: Content streaming functionality
- **Performance Tests**: Large content handling (up to 100MB)
- **Integration Tests**: Database file creation and cleanup

**Phase 3 Deliverables:**
- `internal/store/dbstore/models.go` - GORM models
- `internal/store/dbstore/sqlite.go` - SQLite implementation
- `internal/store/dbstore/sqlite_test.go` - Comprehensive tests

---

### Phase 4: Queue Manager Refactoring [COMPLETED]
**Priority**: High - Core business logic migration
**Complexity**: Medium-High
**Dependencies**: Phases 2 & 3 complete
**Completed**: 2025-10-03
**Commit**: d99f743

#### 4.1 Queue Manager Store Integration
- Refactor `QueueManager` to use `store.Store` interface
- Remove filesystem/remfs dependencies
- Update constructor to accept store interface
- Implement buffered enqueue with metadata generation

#### 4.2 Title Generation
- Create `internal/queue/title.go` with title utilities
- Implement `GenerateTitle()`, `TruncateTitle()`, `SanitizeTitle()`
- Integrate title generation into `Enqueue()`
- Add title parameter to `Enqueue()` signature

#### 4.3 Queue Operations Update
- Update `List()` to use store interface
- Update `Get()` to use store interface
- Add `GetContent()` method for streaming content
- Update `Delete()` and `Clear()` to use store interface
- Update cleanup logic to use store's `DeleteOldest()`

#### 4.4 Legacy Code Removal
- Remove `internal/remfs` package
- Remove file-based persistence logic
- Keep legacy type aliases where appropriate
- Update all queue tests to use memstore

**Phase 4 Testing:**
- **Unit Tests**: QueueManager with memstore (fast tests)
- **Integration Tests**: QueueManager with SQLite store
- **Unit Tests**: Title generation with various content types
- **Unit Tests**: Enqueue with different title scenarios
- **Integration Tests**: Full workflows (enqueue -> list -> get -> delete)
- **Performance Tests**: Large content and large queue sizes

**Phase 4 Deliverables:**
- Updated `internal/queue/manager.go` - Store-backed queue manager
- New `internal/queue/title.go` - Title utilities
- Updated `internal/queue/manager_test.go` - Tests with memstore
- Remove `internal/remfs/` package

---

### Phase 5: CLI Integration [COMPLETED]
**Priority**: High - User-facing changes
**Complexity**: Medium
**Dependencies**: Phases 3 & 4 complete
**Completed**: 2025-10-03
**Commit**: 4cc8708

#### 5.1 CLI Initialization Update
- Update CLI to create store based on configuration
- Add `--db-path` flag for custom database location
- Add `REM_DB_PATH` environment variable support
- Remove remfs initialization code

#### 5.2 Store Command Enhancement
- Add `--title` flag to store command
- Update store command to pass title to QueueManager
- Add content size warnings (for large content)
- Update success messages to show title

#### 5.3 Get Command Update
- Update get command to use new `GetContent()` method
- Ensure streaming still works correctly
- Update TUI launcher to use store-backed queue manager
- Update item display to show titles

#### 5.4 Config Command Update
- Update config commands to use store's ConfigStore
- Remove YAML-specific logic
- Maintain backwards-compatible behavior

**Phase 5 Testing:**
- **Integration Tests**: Full CLI workflows with SQLite
- **Integration Tests**: Database path configuration (flag and env var)
- **Integration Tests**: Title flag in store command
- **Integration Tests**: Config commands with store backend
- **Integration Tests**: Error handling for database failures

**Phase 5 Deliverables:**
- Updated `internal/cli/cli.go` - Store integration
- Updated `internal/cli/args.go` - New flags (--db-path, --title)
- Updated `internal/cli/cli_test.go` - Tests with memstore
- Updated `main.go` - New initialization logic

---

### Phase 6: TUI Updates [COMPLETED]
**Priority**: Medium - UX improvements
**Complexity**: Medium
**Dependencies**: Phase 5 complete
**Completed**: 2025-10-03
**Commit**: a8c40b4

#### 6.1 TUI Store Integration
- Update TUI to work with store-backed items
- Update item structure to use `store.HistoryItem`
- Update delete functionality to use store interface
- Ensure content loading uses `GetContent()`

#### 6.2 Title Display
- Update left pane to show titles instead of previews
- Update right pane status to show full title
- Adjust truncation and display logic
- Update help screen to mention titles

#### 6.3 Performance Optimization
- Implement lazy content loading
- Add content caching for viewed items
- Optimize rendering for large queues

**Phase 6 Testing:**
- **Integration Tests**: TUI with store backend
- **Integration Tests**: Title display in left and right panes
- **Integration Tests**: Delete operations from TUI
- **Performance Tests**: TUI with 1000+ items
- **Integration Tests**: Navigation and content viewing

**Phase 6 Deliverables:**
- Updated `internal/tui/app.go` - Store integration
- Updated `internal/tui/leftpane.go` - Title display
- Updated `internal/tui/rightpane.go` - Title in status
- Updated help screen

---

### Phase 7: Search Implementation [COMPLETED]
**Priority**: Medium - Advanced feature
**Complexity**: Medium-High
**Dependencies**: Phase 6 complete
**Completed**: 2025-10-03
**Commit**: a41493a

#### 7.1 Search in Store Interface
- Implement `Search()` in memstore
- Implement `Search()` in SQLite store
- Support regex patterns
- Support title-only and content-only searches

#### 7.2 CLI Search Enhancement
- Add `--title` and `--content` flags to search command
- Update search to use store's `Search()` method
- Display titles in search results
- Support search result formatting

#### 7.3 TUI Search Enhancement (Optional)
- Update TUI search to search titles
- Highlight matching titles
- Show title in search results

**Phase 7 Testing:**
- **Unit Tests**: Search pattern matching (regex)
- **Integration Tests**: Search with various patterns
- **Integration Tests**: Title-only and content-only searches
- **Performance Tests**: Search in large queues

**Phase 7 Deliverables:**
- Updated `internal/store/memstore/memory.go` - Search implementation
- Updated `internal/store/dbstore/sqlite.go` - Search implementation
- Updated `internal/cli/cli.go` - Enhanced search command
- Optional: Updated TUI search

---

### Phase 8: Documentation and Polish [TODO]
**Priority**: Medium - User communication
**Complexity**: Low
**Dependencies**: All phases complete

#### 8.1 Architecture Documentation
- Update ARCHITECTURE.md with new store design
- Document store interface and implementations
- Add diagrams for store architecture
- Document streaming strategy

#### 8.2 Development Documentation
- Update CLAUDE.md with new development workflow
- Document testing with memstore
- Add examples of using store interface
- Update build and test commands

#### 8.3 Migration Guide
- Write migration guide for users upgrading
- Document backwards incompatibility clearly
- Provide troubleshooting guide
- Add FAQ section

#### 8.4 Help System
- Update CLI help messages
- Update TUI help screen
- Add examples for new flags
- Document database location and management

**Phase 8 Deliverables:**
- Updated `ARCHITECTURE.md`
- Updated `CLAUDE.md`
- New `MIGRATION.md`
- Updated help text in CLI and TUI

---

## Testing Strategy

### Unit Tests
- Store interface types and conversions
- In-memory store implementation (fast)
- SQLite store implementation (with test database)
- Queue manager business logic (using memstore)
- Title generation utilities

### Integration Tests
- Full workflows (store -> list -> get -> delete)
- CLI commands with SQLite store
- TUI interaction with store backend
- Database initialization and migration
- Error scenarios and recovery

### Performance Tests
- Large content handling (up to 100MB)
- Large queue operations (1000+ items)
- Database query performance
- Memory usage profiling
- TUI responsiveness

### Test Organization
```
internal/
├── store/
│   ├── store_test.go          # Interface compliance tests
│   ├── memstore/
│   │   └── memory_test.go     # In-memory store tests
│   └── dbstore/
│       └── sqlite_test.go     # SQLite store tests
└── queue/
    ├── manager_test.go         # Queue manager tests (uses memstore)
    └── title_test.go           # Title utilities tests
```

## Success Criteria

### Functional Requirements
- [ ] Store interface fully defined and documented
- [ ] In-memory store implementation complete
- [ ] SQLite store implementation with chunked storage complete
- [ ] Chunked streaming works for write (io.Reader input)
- [ ] Chunked streaming works for read (io.ReadSeekCloser output)
- [ ] Queue manager uses store interface correctly
- [ ] Titles feature working with 80-char limit
- [ ] Auto-title generation from first 4KB sample
- [ ] LIFO queue behavior maintained
- [ ] Search works for titles and content
- [ ] TUI displays titles correctly
- [ ] Configuration stored in database
- [ ] Cascade delete removes chunks when items are deleted

### Performance Requirements
- [ ] List operation < 100ms for 1000 items (metadata only)
- [ ] Enqueue operation uses max ~32KB memory regardless of file size
- [ ] GetContent returns reader within 100ms (no full content load)
- [ ] Streaming read loads only one 32KB chunk at a time
- [ ] TUI navigation remains responsive with large files
- [ ] In-memory tests run fast (< 1s for full suite)
- [ ] Can handle multi-GB files without memory issues

### Quality Requirements
- [ ] Comprehensive test coverage (>80%)
- [ ] All tests pass with both memstore and SQLite
- [ ] Clear error messages for users
- [ ] Documentation updated and accurate
- [ ] Code follows Go best practices

### Architecture Requirements
- [ ] Store interface is mockable
- [ ] Implementations are swappable
- [ ] Queue manager has no storage dependencies
- [ ] Clean separation of concerns
- [ ] Easy to add new store implementations

## Risk Mitigation

### Data Safety
- Database transactions for critical operations
- Rollback on failure during enqueue
- Database file backup recommendations
- Export/import functionality for recovery

### Testing Strategy
- Fast tests with in-memory store
- Comprehensive tests with SQLite store
- Integration tests for full workflows
- Performance benchmarks

### Performance Risks
- Content size limits prevent memory exhaustion
- Database indexes for fast queries
- Lazy loading for content blobs
- Streaming interface for large content

### Migration Risks
- Clear documentation about backwards incompatibility
- Export/import tools for data migration
- Prominent warning in release notes
- Version bump (v1.x → v2.0)

## Dependencies

### New Go Packages
```go
import (
    "gorm.io/gorm"
    "gorm.io/driver/sqlite"
)
```

### Version Requirements
- GORM v1.25+ (latest stable)
- SQLite driver for GORM
- Go 1.21+ (for compatibility)

## Database Configuration

### Database Path Precedence
1. CLI `--db-path` flag (highest)
2. `REM_DB_PATH` environment variable
3. Default `~/.config/rem/rem.db` (lowest)

### Directory Structure

**Old (File-based):**
```
~/.config/rem/
├── history/
│   ├── 2025-09-28T10-15-30.123456-07-00.rem
│   └── ...
└── config.yaml
```

**New (Database):**
```
~/.config/rem/
└── rem.db                  # Single SQLite database file
```

## CLI Changes

### New Flags

```bash
# Database location
rem --db-path /custom/path/rem.db <command>
export REM_DB_PATH=/custom/path/rem.db

# Store with title
rem store --title "My Important Note"

# Search
rem search --title "keyword"    # Search titles only
rem search --content "keyword"  # Search content only
rem search "keyword"            # Search both (default)
```

## Next Steps

1. **Phase 1**: Define store interfaces and types
2. **Phase 2**: Implement in-memory store for testing
3. **Phase 3**: Implement SQLite store for production
4. **Phase 4**: Refactor queue manager to use store interface
5. **Phase 5**: Integrate store into CLI
6. **Phase 6**: Update TUI with title display
7. **Phase 7**: Implement search functionality
8. **Phase 8**: Documentation and migration guide

## Open Questions

1. **Database Compaction**: Auto-VACUUM or manual `rem db compact` command?
2. **Title Editing**: Allow post-creation title editing via CLI or TUI?
3. **Deduplication**: Detect duplicate content by SHA256 and reuse chunks?
4. **Encryption**: Support encrypted database files for sensitive content?
5. **Concurrent Access**: Should we support multiple rem instances accessing same DB with locking?
6. **Chunk Optimization**: Should we add chunk caching for recently accessed items?
7. **Search Performance**: Should we add full-text search indexes for titles/content?

## Backwards Incompatibility Notes

This is a **breaking change** from v1.x to v2.0.

**Migration Path:**
1. Optional export tool: `rem v1.x export queue.json`
2. Upgrade to v2.0
3. Optional import tool: `rem v2.0 import queue.json`

**User Communication:**
```
rem v2.0 introduces SQLite3 database storage and titles feature.
This is a backwards-incompatible change.

Your existing file-based queue will not be accessible in v2.0.
For migration instructions, see MIGRATION.md
```

---

**Last Updated**: 2025-10-03
**Status**: Phases 1-7 Complete - Ready for Phase 8 (Documentation)

## Implementation Summary

**Completed Phases:**
- ✅ Phase 1: Store Interface and Types (Commit: 3851ce5)
- ✅ Phase 2: In-Memory Store Implementation (Commit: 7d1afb8)
- ✅ Phase 3: SQLite Store Implementation (Commit: 209d641)
- ✅ Phase 4: Queue Manager Refactoring (Commit: d99f743)
- ✅ Phase 5: CLI Integration (Commit: 4cc8708)
- ✅ Phase 6: TUI Updates (Commit: a8c40b4)
- ✅ Phase 7: Search Implementation (Commit: a41493a)

**Remaining:**
- ⏳ Phase 8: Documentation and Polish

**All functional development complete!** The application is fully working with:
- SQLite3 database storage with 32KB chunked streaming
- Title feature (max 80 chars, auto-generated or explicit)
- Full regex search for titles and content
- Database-backed configuration
- Complete test coverage (all tests passing)
