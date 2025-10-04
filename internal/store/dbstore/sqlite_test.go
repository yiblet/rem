package dbstore

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yiblet/rem/internal/store"
)

// setupTestDB creates a temporary database for testing
func setupTestDB(t *testing.T) (*SQLiteStore, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	st, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}

	cleanup := func() {
		st.Close()
	}

	return st, cleanup
}

// TestNewSQLiteStore tests database initialization
func TestNewSQLiteStore(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	// Verify store was created
	if st == nil {
		t.Fatal("expected store to be created")
	}

	// Verify default config was initialized
	historyLimit, err := st.Config().Get("history_limit")
	if err != nil {
		t.Fatalf("failed to get history_limit: %v", err)
	}
	if historyLimit != "255" {
		t.Errorf("expected history_limit=255, got %s", historyLimit)
	}

	showBinary, err := st.Config().Get("show_binary")
	if err != nil {
		t.Fatalf("failed to get show_binary: %v", err)
	}
	if showBinary != "false" {
		t.Errorf("expected show_binary=false, got %s", showBinary)
	}

	dbVersion, err := st.Config().Get("db_version")
	if err != nil {
		t.Fatalf("failed to get db_version: %v", err)
	}
	if dbVersion != "1" {
		t.Errorf("expected db_version=1, got %s", dbVersion)
	}
}

// TestHistoryStore_Create tests creating history items with chunked storage
func TestHistoryStore_Create(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name     string
		title    string
		content  string
		wantSize int64
	}{
		{
			name:     "small text content",
			title:    "Small Text",
			content:  "Hello, World!",
			wantSize: 13,
		},
		{
			name:     "empty content",
			title:    "Empty",
			content:  "",
			wantSize: 0,
		},
		{
			name:     "content with newlines",
			title:    "Multi-line",
			content:  "Line 1\nLine 2\nLine 3",
			wantSize: 20, // Actual byte count of the literal string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &store.CreateHistoryInput{
				Title:     tt.title,
				Content:   strings.NewReader(tt.content),
				Timestamp: time.Now(),
			}

			item, err := st.History().Create(input)
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			if item.ID == 0 {
				t.Error("expected non-zero ID")
			}
			if item.Title != tt.title {
				t.Errorf("expected title=%s, got %s", tt.title, item.Title)
			}
			if item.Size != tt.wantSize {
				t.Errorf("expected size=%d, got %d", tt.wantSize, item.Size)
			}

			// Verify SHA256 hash
			expectedHash := sha256.Sum256([]byte(tt.content))
			expectedHashStr := hex.EncodeToString(expectedHash[:])
			if item.SHA256 != expectedHashStr {
				t.Errorf("expected SHA256=%s, got %s", expectedHashStr, item.SHA256)
			}
		})
	}
}

// TestHistoryStore_CreateLargeContent tests chunked storage with content larger than one chunk
func TestHistoryStore_CreateLargeContent(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	// Create content larger than ChunkSize (32KB)
	largeContent := strings.Repeat("A", ChunkSize*3+1000) // 3 full chunks + partial
	input := &store.CreateHistoryInput{
		Title:     "Large Content",
		Content:   strings.NewReader(largeContent),
		Timestamp: time.Now(),
	}

	item, err := st.History().Create(input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if item.Size != int64(len(largeContent)) {
		t.Errorf("expected size=%d, got %d", len(largeContent), item.Size)
	}

	// Verify chunks were created
	var chunkCount int64
	st.db.Model(&FileChunkModel{}).Where("history_id = ?", item.ID).Count(&chunkCount)
	expectedChunks := 4 // 3 full + 1 partial
	if chunkCount != int64(expectedChunks) {
		t.Errorf("expected %d chunks, got %d", expectedChunks, chunkCount)
	}
}

// TestHistoryStore_List tests listing history items
func TestHistoryStore_List(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	// Create test items with different timestamps
	timestamps := []time.Time{
		time.Now().Add(-3 * time.Hour),
		time.Now().Add(-2 * time.Hour),
		time.Now().Add(-1 * time.Hour),
	}

	for i, ts := range timestamps {
		input := &store.CreateHistoryInput{
			Title:     string(rune('A' + i)),
			Content:   strings.NewReader("content"),
			Timestamp: ts,
		}
		_, err := st.History().Create(input)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// List all items
	items, err := st.History().List(0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// Verify LIFO order (newest first)
	if items[0].Title != "C" {
		t.Errorf("expected first item title=C, got %s", items[0].Title)
	}
	if items[1].Title != "B" {
		t.Errorf("expected second item title=B, got %s", items[1].Title)
	}
	if items[2].Title != "A" {
		t.Errorf("expected third item title=A, got %s", items[2].Title)
	}

	// Test with limit
	limitedItems, err := st.History().List(2)
	if err != nil {
		t.Fatalf("List(2) error = %v", err)
	}
	if len(limitedItems) != 2 {
		t.Errorf("expected 2 items with limit, got %d", len(limitedItems))
	}
}

// TestHistoryStore_Get tests retrieving a single item
func TestHistoryStore_Get(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	// Create an item
	input := &store.CreateHistoryInput{
		Title:     "Test Item",
		Content:   strings.NewReader("test content"),
		Timestamp: time.Now(),
	}
	created, err := st.History().Create(input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get the item
	item, err := st.History().Get(created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if item.ID != created.ID {
		t.Errorf("expected ID=%d, got %d", created.ID, item.ID)
	}
	if item.Title != "Test Item" {
		t.Errorf("expected title='Test Item', got %s", item.Title)
	}

	// Test getting non-existent item
	_, err = st.History().Get(9999)
	if err == nil {
		t.Error("expected error for non-existent item")
	}
}

// TestHistoryStore_GetContent tests retrieving content via ChunkedReader
func TestHistoryStore_GetContent(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	testContent := "Hello, this is test content for streaming!"
	input := &store.CreateHistoryInput{
		Title:     "Test Content",
		Content:   strings.NewReader(testContent),
		Timestamp: time.Now(),
	}
	item, err := st.History().Create(input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get content
	reader, err := st.History().GetContent(item.ID)
	if err != nil {
		t.Fatalf("GetContent() error = %v", err)
	}
	defer reader.Close()

	// Read all content
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(content) != testContent {
		t.Errorf("expected content=%s, got %s", testContent, string(content))
	}
}

// TestHistoryStore_GetContentLarge tests streaming large content
func TestHistoryStore_GetContentLarge(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	// Create large content (multiple chunks)
	largeContent := strings.Repeat("X", ChunkSize*2+500)
	input := &store.CreateHistoryInput{
		Title:     "Large Content",
		Content:   strings.NewReader(largeContent),
		Timestamp: time.Now(),
	}
	item, err := st.History().Create(input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get content and read incrementally
	reader, err := st.History().GetContent(item.ID)
	if err != nil {
		t.Fatalf("GetContent() error = %v", err)
	}
	defer reader.Close()

	// Read in small chunks to verify streaming
	var result bytes.Buffer
	buf := make([]byte, 1024) // Small buffer to force multiple chunk loads
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			result.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
	}

	if result.String() != largeContent {
		t.Errorf("content mismatch: expected length=%d, got length=%d",
			len(largeContent), result.Len())
	}
}

// TestChunkedReader_Seek tests seeking in chunked content
func TestChunkedReader_Seek(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	// Create content spanning multiple chunks
	content := strings.Repeat("0123456789", ChunkSize/5) // ~64KB
	input := &store.CreateHistoryInput{
		Title:     "Seekable Content",
		Content:   strings.NewReader(content),
		Timestamp: time.Now(),
	}
	item, err := st.History().Create(input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	reader, err := st.History().GetContent(item.ID)
	if err != nil {
		t.Fatalf("GetContent() error = %v", err)
	}
	defer reader.Close()

	// Test SeekStart
	pos, err := reader.Seek(100, io.SeekStart)
	if err != nil {
		t.Fatalf("Seek(100, SeekStart) error = %v", err)
	}
	if pos != 100 {
		t.Errorf("expected position=100, got %d", pos)
	}

	// Read and verify content after seek
	buf := make([]byte, 10)
	n, err := reader.Read(buf)
	if err != nil {
		t.Fatalf("Read() after seek error = %v", err)
	}
	if n != 10 {
		t.Errorf("expected to read 10 bytes, got %d", n)
	}
	expected := content[100:110]
	if string(buf) != expected {
		t.Errorf("expected content=%s, got %s", expected, string(buf))
	}

	// Test SeekEnd
	pos, err = reader.Seek(-10, io.SeekEnd)
	if err != nil {
		t.Fatalf("Seek(-10, SeekEnd) error = %v", err)
	}
	expectedPos := int64(len(content)) - 10
	if pos != expectedPos {
		t.Errorf("expected position=%d, got %d", expectedPos, pos)
	}

	// Test SeekCurrent
	pos, err = reader.Seek(5, io.SeekCurrent)
	if err != nil {
		t.Fatalf("Seek(5, SeekCurrent) error = %v", err)
	}
	expectedPos = int64(len(content)) - 5
	if pos != expectedPos {
		t.Errorf("expected position=%d, got %d", expectedPos, pos)
	}
}

// TestHistoryStore_Delete tests deleting items
func TestHistoryStore_Delete(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	// Create an item
	input := &store.CreateHistoryInput{
		Title:     "To Delete",
		Content:   strings.NewReader("content"),
		Timestamp: time.Now(),
	}
	item, err := st.History().Create(input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify chunks exist
	var chunkCount int64
	st.db.Model(&FileChunkModel{}).Where("history_id = ?", item.ID).Count(&chunkCount)
	if chunkCount == 0 {
		t.Fatal("expected chunks to exist")
	}

	// Delete the item
	err = st.History().Delete(item.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify item is gone
	_, err = st.History().Get(item.ID)
	if err == nil {
		t.Error("expected error when getting deleted item")
	}

	// Verify chunks are gone (CASCADE delete)
	st.db.Model(&FileChunkModel{}).Where("history_id = ?", item.ID).Count(&chunkCount)
	if chunkCount != 0 {
		t.Errorf("expected chunks to be deleted, found %d chunks", chunkCount)
	}

	// Test deleting non-existent item
	err = st.History().Delete(9999)
	if err == nil {
		t.Error("expected error when deleting non-existent item")
	}
}

// TestHistoryStore_DeleteOldest tests deleting oldest items
func TestHistoryStore_DeleteOldest(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	// Create items with different timestamps
	timestamps := []time.Time{
		time.Now().Add(-3 * time.Hour), // Oldest
		time.Now().Add(-2 * time.Hour),
		time.Now().Add(-1 * time.Hour), // Newest
	}

	for i, ts := range timestamps {
		input := &store.CreateHistoryInput{
			Title:     string(rune('A' + i)),
			Content:   strings.NewReader("content"),
			Timestamp: ts,
		}
		_, err := st.History().Create(input)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Delete oldest 2
	err := st.History().DeleteOldest(2)
	if err != nil {
		t.Fatalf("DeleteOldest() error = %v", err)
	}

	// Verify only 1 item remains
	items, err := st.History().List(0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item remaining, got %d", len(items))
	}

	// Verify it's the newest item
	if items[0].Title != "C" {
		t.Errorf("expected remaining item title=C, got %s", items[0].Title)
	}
}

// TestHistoryStore_Count tests counting items
func TestHistoryStore_Count(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	// Initially empty
	count, err := st.History().Count()
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 0 {
		t.Errorf("expected count=0, got %d", count)
	}

	// Add items
	for i := 0; i < 5; i++ {
		input := &store.CreateHistoryInput{
			Title:     "Item",
			Content:   strings.NewReader("content"),
			Timestamp: time.Now(),
		}
		_, err := st.History().Create(input)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	count, err = st.History().Count()
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 5 {
		t.Errorf("expected count=5, got %d", count)
	}
}

// TestHistoryStore_Clear tests clearing all items
func TestHistoryStore_Clear(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	// Add items
	for i := 0; i < 3; i++ {
		input := &store.CreateHistoryInput{
			Title:     "Item",
			Content:   strings.NewReader("content"),
			Timestamp: time.Now(),
		}
		_, err := st.History().Create(input)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Clear
	err := st.History().Clear()
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// Verify empty
	count, err := st.History().Count()
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 0 {
		t.Errorf("expected count=0 after clear, got %d", count)
	}
}

// TestConfigStore_GetSet tests config operations
func TestConfigStore_GetSet(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	// Set a value
	err := st.Config().Set("test_key", "test_value")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get the value
	value, err := st.Config().Get("test_key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if value != "test_value" {
		t.Errorf("expected value='test_value', got %s", value)
	}

	// Update the value
	err = st.Config().Set("test_key", "new_value")
	if err != nil {
		t.Fatalf("Set() update error = %v", err)
	}

	value, err = st.Config().Get("test_key")
	if err != nil {
		t.Fatalf("Get() after update error = %v", err)
	}
	if value != "new_value" {
		t.Errorf("expected value='new_value', got %s", value)
	}

	// Get non-existent key
	_, err = st.Config().Get("non_existent")
	if err == nil {
		t.Error("expected error for non-existent key")
	}
}

// TestConfigStore_List tests listing all config values
func TestConfigStore_List(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	// List should include default config
	configs, err := st.Config().List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	// Verify default configs exist
	if configs["history_limit"] != "255" {
		t.Errorf("expected history_limit=255, got %s", configs["history_limit"])
	}
	if configs["show_binary"] != "false" {
		t.Errorf("expected show_binary=false, got %s", configs["show_binary"])
	}
	if configs["db_version"] != "1" {
		t.Errorf("expected db_version=1, got %s", configs["db_version"])
	}
}

// TestConfigStore_Delete tests deleting config values
func TestConfigStore_Delete(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	// Set a value
	err := st.Config().Set("to_delete", "value")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Delete it
	err = st.Config().Delete("to_delete")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	_, err = st.Config().Get("to_delete")
	if err == nil {
		t.Error("expected error for deleted key")
	}

	// Delete non-existent key
	err = st.Config().Delete("non_existent")
	if err == nil {
		t.Error("expected error when deleting non-existent key")
	}
}

// TestIsBinary tests binary content detection
func TestIsBinary(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "text content",
			data: []byte("Hello, World! This is text content."),
			want: false,
		},
		{
			name: "content with newlines",
			data: []byte("Line 1\nLine 2\nLine 3"),
			want: false,
		},
		{
			name: "binary with null bytes",
			data: []byte{0x00, 0x01, 0x02, 0x03, 0x04},
			want: true,
		},
		{
			name: "binary with many non-printable",
			data: bytes.Repeat([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}, 50), // Control characters
			want: true,
		},
		{
			name: "empty content",
			data: []byte{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBinary(tt.data)
			if got != tt.want {
				t.Errorf("isBinary() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMemoryConstraint verifies that only one chunk is loaded at a time
func TestMemoryConstraint(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	// Create large content (multiple chunks)
	largeContent := strings.Repeat("X", ChunkSize*5) // 5 chunks
	input := &store.CreateHistoryInput{
		Title:     "Memory Test",
		Content:   strings.NewReader(largeContent),
		Timestamp: time.Now(),
	}
	item, err := st.History().Create(input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get content reader
	reader, err := st.History().GetContent(item.ID)
	if err != nil {
		t.Fatalf("GetContent() error = %v", err)
	}
	defer reader.Close()

	chunkedReader, ok := reader.(*ChunkedReader)
	if !ok {
		t.Fatal("expected ChunkedReader type")
	}

	// Read small amounts to verify chunk buffer size
	buf := make([]byte, 100)
	_, err = chunkedReader.Read(buf)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	// Verify only one chunk is loaded
	if len(chunkedReader.chunkBuf) > ChunkSize {
		t.Errorf("chunk buffer too large: %d bytes (max should be %d)",
			len(chunkedReader.chunkBuf), ChunkSize)
	}

	// Read through multiple chunks
	for i := 0; i < 3; i++ {
		largeBuf := make([]byte, ChunkSize+1000)
		_, err = chunkedReader.Read(largeBuf)
		if err != nil && err != io.EOF {
			t.Fatalf("Read() error = %v", err)
		}

		// Verify still only one chunk loaded
		if len(chunkedReader.chunkBuf) > ChunkSize {
			t.Errorf("chunk buffer too large after read %d: %d bytes",
				i, len(chunkedReader.chunkBuf))
		}
	}
}

// TestConcurrentAccess tests thread safety (basic test)
func TestConcurrentAccess(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	// Create an item
	input := &store.CreateHistoryInput{
		Title:     "Concurrent Test",
		Content:   strings.NewReader("test content"),
		Timestamp: time.Now(),
	}
	item, err := st.History().Create(input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Read concurrently (SQLite handles this via database locking)
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			defer func() { done <- true }()

			reader, err := st.History().GetContent(item.ID)
			if err != nil {
				t.Errorf("GetContent() error = %v", err)
				return
			}
			defer reader.Close()

			content, err := io.ReadAll(reader)
			if err != nil {
				t.Errorf("ReadAll() error = %v", err)
				return
			}

			if string(content) != "test content" {
				t.Errorf("unexpected content: %s", string(content))
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
}

// TestDatabaseFile verifies the database file is created correctly
func TestDatabaseFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "rem.db")

	st, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer st.Close()

	// Verify database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("database file not created at %s", dbPath)
	}
}

// TestHistoryStore_SearchTitleOnly tests searching in titles only.
func TestHistoryStore_SearchTitleOnly(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	h := st.History()

	// Create test items
	items := []struct {
		title   string
		content string
	}{
		{"Config File", "This contains passwords"},
		{"Password Manager", "Stores all your credentials"},
		{"Log Analysis", "Error logs from server"},
		{"Test Results", "All tests passed successfully"},
	}

	for _, item := range items {
		input := &store.CreateHistoryInput{
			Title:   item.title,
			Content: strings.NewReader(item.content),
		}
		if _, err := h.Create(input); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	// Search for "Config" in titles only
	query := &store.SearchQuery{
		Pattern:     "Config",
		SearchTitle: true,
	}

	results, err := h.Search(query)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Search() returned %d results, want 1", len(results))
	}

	if results[0].Title != "Config File" {
		t.Errorf("Search() result title = %q, want %q", results[0].Title, "Config File")
	}
}

// TestHistoryStore_SearchContentOnly tests searching in content only.
func TestHistoryStore_SearchContentOnly(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	h := st.History()

	// Create test items
	items := []struct {
		title   string
		content string
	}{
		{"Config File", "This contains passwords"},
		{"Password Manager", "Stores all your credentials"},
		{"Log Analysis", "Error logs from server"},
		{"Test Results", "All tests passed successfully"},
	}

	for _, item := range items {
		input := &store.CreateHistoryInput{
			Title:   item.title,
			Content: strings.NewReader(item.content),
		}
		if _, err := h.Create(input); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	// Search for "password" in content only (case-insensitive by default)
	query := &store.SearchQuery{
		Pattern:       "password",
		SearchContent: true,
	}

	results, err := h.Search(query)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Search() returned %d results, want 1", len(results))
	}

	if results[0].Title != "Config File" {
		t.Errorf("Search() result title = %q, want %q", results[0].Title, "Config File")
	}
}

// TestHistoryStore_SearchBoth tests searching in both title and content.
func TestHistoryStore_SearchBoth(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	h := st.History()

	// Create test items
	items := []struct {
		title   string
		content string
	}{
		{"Config File", "Contains settings"},
		{"Password Manager", "Stores all your credentials"},
		{"Log Analysis", "Error logs with password leaks"},
	}

	for _, item := range items {
		input := &store.CreateHistoryInput{
			Title:   item.title,
			Content: strings.NewReader(item.content),
		}
		if _, err := h.Create(input); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	// Search for "password" in both title and content (default behavior)
	query := &store.SearchQuery{
		Pattern: "password",
	}

	results, err := h.Search(query)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Search() returned %d results, want 2", len(results))
	}

	// Verify both matches
	titles := []string{results[0].Title, results[1].Title}
	wantTitles := map[string]bool{
		"Password Manager": true,
		"Log Analysis":     true,
	}

	for _, title := range titles {
		if !wantTitles[title] {
			t.Errorf("Unexpected result title: %q", title)
		}
	}
}

// TestHistoryStore_SearchCaseSensitive tests case-sensitive search.
func TestHistoryStore_SearchCaseSensitive(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	h := st.History()

	// Create test items
	items := []struct {
		title   string
		content string
	}{
		{"ERROR Log", "ERROR: Something went wrong"},
		{"Info Log", "error: minor issue"},
	}

	for _, item := range items {
		input := &store.CreateHistoryInput{
			Title:   item.title,
			Content: strings.NewReader(item.content),
		}
		if _, err := h.Create(input); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	// Case-sensitive search for "ERROR"
	query := &store.SearchQuery{
		Pattern:       "ERROR",
		CaseSensitive: true,
	}

	results, err := h.Search(query)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Search() returned %d results, want 1", len(results))
	}

	if results[0].Title != "ERROR Log" {
		t.Errorf("Search() result title = %q, want %q", results[0].Title, "ERROR Log")
	}

	// Case-insensitive search for "error" should find both
	query.CaseSensitive = false
	query.Pattern = "error"

	results, err = h.Search(query)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Case-insensitive search returned %d results, want 2", len(results))
	}
}

// TestHistoryStore_SearchLimit tests search limit parameter.
func TestHistoryStore_SearchLimit(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	h := st.History()

	// Create test items
	for i := 0; i < 5; i++ {
		input := &store.CreateHistoryInput{
			Title:   "Test Item",
			Content: strings.NewReader("test content"),
		}
		if _, err := h.Create(input); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	// Search with limit of 3
	query := &store.SearchQuery{
		Pattern: "Test",
		Limit:   3,
	}

	results, err := h.Search(query)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Search() with limit 3 returned %d results, want 3", len(results))
	}
}

// TestHistoryStore_SearchRegex tests regex pattern matching.
func TestHistoryStore_SearchRegex(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	h := st.History()

	// Create test items
	items := []struct {
		title   string
		content string
	}{
		{"test-001", "First test item"},
		{"test-002", "Second test item"},
		{"demo-001", "Demo item"},
	}

	for _, item := range items {
		input := &store.CreateHistoryInput{
			Title:   item.title,
			Content: strings.NewReader(item.content),
		}
		if _, err := h.Create(input); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	// Search for pattern "test-.*" (should match test-001 and test-002)
	query := &store.SearchQuery{
		Pattern:     "test-.*",
		SearchTitle: true,
	}

	results, err := h.Search(query)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Search() returned %d results, want 2", len(results))
	}

	// Search for pattern "^First" (should match content starting with "First")
	query = &store.SearchQuery{
		Pattern:       "^First",
		SearchContent: true,
	}

	results, err = h.Search(query)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Search() returned %d results, want 1", len(results))
	}

	if results[0].Title != "test-001" {
		t.Errorf("Search() result title = %q, want %q", results[0].Title, "test-001")
	}
}

// TestHistoryStore_SearchInvalidRegex tests handling of invalid regex patterns.
func TestHistoryStore_SearchInvalidRegex(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	h := st.History()

	// Try to search with invalid regex
	query := &store.SearchQuery{
		Pattern: "[invalid(regex",
	}

	_, err := h.Search(query)
	if err == nil {
		t.Error("Search() with invalid regex succeeded, want error")
	}
}

// TestHistoryStore_SearchEmptyPattern tests searching with empty pattern.
func TestHistoryStore_SearchEmptyPattern(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	h := st.History()

	// Create an item
	input := &store.CreateHistoryInput{
		Title:   "Test Item",
		Content: strings.NewReader("content"),
	}
	if _, err := h.Create(input); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Search with empty pattern should return empty results
	query := &store.SearchQuery{
		Pattern: "",
	}

	results, err := h.Search(query)
	if err != nil {
		t.Fatalf("Search() with empty pattern error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Search() with empty pattern returned %d results, want 0", len(results))
	}
}

// TestHistoryStore_SearchNoMatches tests searching when no items match.
func TestHistoryStore_SearchNoMatches(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	h := st.History()

	// Create items
	items := []struct {
		title   string
		content string
	}{
		{"Config File", "Settings"},
		{"Log Analysis", "Errors"},
	}

	for _, item := range items {
		input := &store.CreateHistoryInput{
			Title:   item.title,
			Content: strings.NewReader(item.content),
		}
		if _, err := h.Create(input); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	// Search for pattern that doesn't match anything
	query := &store.SearchQuery{
		Pattern: "nonexistent",
	}

	results, err := h.Search(query)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Search() with no matches returned %d results, want 0", len(results))
	}
}

// TestHistoryStore_SearchWithChunkedContent tests search on content spanning multiple chunks.
func TestHistoryStore_SearchWithChunkedContent(t *testing.T) {
	st, cleanup := setupTestDB(t)
	defer cleanup()

	h := st.History()

	// Create item with large content (multiple chunks)
	// Content will span multiple 32KB chunks
	largeContent := strings.Repeat("A", ChunkSize) + "SEARCHPATTERN" + strings.Repeat("B", ChunkSize)
	input := &store.CreateHistoryInput{
		Title:   "Large Item",
		Content: strings.NewReader(largeContent),
	}
	if _, err := h.Create(input); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Search for pattern in the middle of chunks
	query := &store.SearchQuery{
		Pattern:       "SEARCHPATTERN",
		SearchContent: true,
	}

	results, err := h.Search(query)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Search() returned %d results, want 1", len(results))
	}

	if results[0].Title != "Large Item" {
		t.Errorf("Search() result title = %q, want %q", results[0].Title, "Large Item")
	}
}
