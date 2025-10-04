package memstore

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yiblet/rem/internal/store"
)

// TestMemoryStore_Basic tests basic store creation and interface compliance.
func TestMemoryStore_Basic(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	// Check that History() returns a valid HistoryStore
	if s.History() == nil {
		t.Fatal("History() returned nil")
	}

	// Check that Config() returns a valid ConfigStore
	if s.Config() == nil {
		t.Fatal("Config() returned nil")
	}

	// Check that Close() doesn't error
	if err := s.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}
}

// TestHistoryStore_CreateAndGet tests creating and retrieving items.
func TestHistoryStore_CreateAndGet(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

	// Create an item
	content := []byte("Hello, World!")
	input := &store.CreateHistoryInput{
		Title:     "Test Item",
		Content:   bytes.NewReader(content),
		Timestamp: time.Now(),
	}

	item, err := h.Create(input)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Verify item fields
	if item.ID == 0 {
		t.Error("Created item has zero ID")
	}
	if item.Title != "Test Item" {
		t.Errorf("Title = %q, want %q", item.Title, "Test Item")
	}
	if item.Size != int64(len(content)) {
		t.Errorf("Size = %d, want %d", item.Size, len(content))
	}
	if item.IsBinary {
		t.Error("IsBinary = true, want false for text content")
	}
	if item.SHA256 == "" {
		t.Error("SHA256 is empty")
	}

	// Get the item back
	retrieved, err := h.Get(item.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	if retrieved.ID != item.ID {
		t.Errorf("Retrieved ID = %d, want %d", retrieved.ID, item.ID)
	}
	if retrieved.Title != item.Title {
		t.Errorf("Retrieved Title = %q, want %q", retrieved.Title, item.Title)
	}
}

// TestHistoryStore_GetContent tests retrieving content.
func TestHistoryStore_GetContent(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

	// Create an item
	content := []byte("Hello, World!")
	input := &store.CreateHistoryInput{
		Title:   "Test Item",
		Content: bytes.NewReader(content),
	}

	item, err := h.Create(input)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Get content
	reader, err := h.GetContent(item.ID)
	if err != nil {
		t.Fatalf("GetContent() error: %v", err)
	}
	defer reader.Close()

	// Read content
	retrieved, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}

	if !bytes.Equal(retrieved, content) {
		t.Errorf("Retrieved content = %q, want %q", retrieved, content)
	}
}

// TestHistoryStore_List tests listing items with LIFO ordering.
func TestHistoryStore_List(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

	// Create items with different timestamps
	baseTime := time.Now()
	items := []struct {
		title     string
		timestamp time.Time
	}{
		{"First", baseTime},
		{"Second", baseTime.Add(1 * time.Second)},
		{"Third", baseTime.Add(2 * time.Second)},
	}

	for _, item := range items {
		input := &store.CreateHistoryInput{
			Title:     item.title,
			Content:   bytes.NewReader([]byte(item.title)),
			Timestamp: item.timestamp,
		}
		if _, err := h.Create(input); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	// List all items
	list, err := h.List(0)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(list) != 3 {
		t.Fatalf("List() returned %d items, want 3", len(list))
	}

	// Verify LIFO ordering (newest first)
	expectedOrder := []string{"Third", "Second", "First"}
	for i, expected := range expectedOrder {
		if list[i].Title != expected {
			t.Errorf("List()[%d].Title = %q, want %q", i, list[i].Title, expected)
		}
	}
}

// TestHistoryStore_ListWithLimit tests listing with a limit.
func TestHistoryStore_ListWithLimit(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

	// Create 5 items
	for i := 0; i < 5; i++ {
		input := &store.CreateHistoryInput{
			Title:     string(rune('A' + i)),
			Content:   bytes.NewReader([]byte{byte('A' + i)}),
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		}
		if _, err := h.Create(input); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	// List with limit of 3
	list, err := h.List(3)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("List(3) returned %d items, want 3", len(list))
	}
}

// TestHistoryStore_Delete tests deleting items.
func TestHistoryStore_Delete(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

	// Create an item
	input := &store.CreateHistoryInput{
		Title:   "Test Item",
		Content: bytes.NewReader([]byte("content")),
	}

	item, err := h.Create(input)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Delete the item
	if err := h.Delete(item.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// Verify it's gone
	if _, err := h.Get(item.ID); err == nil {
		t.Error("Get() succeeded after Delete(), want error")
	}

	// Try to delete again (should error)
	if err := h.Delete(item.ID); err == nil {
		t.Error("Delete() succeeded on non-existent item, want error")
	}
}

// TestHistoryStore_DeleteOldest tests deleting oldest items.
func TestHistoryStore_DeleteOldest(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

	// Create items with different timestamps
	baseTime := time.Now()
	for i := 0; i < 5; i++ {
		input := &store.CreateHistoryInput{
			Title:     string(rune('A' + i)),
			Content:   bytes.NewReader([]byte{byte('A' + i)}),
			Timestamp: baseTime.Add(time.Duration(i) * time.Second),
		}
		if _, err := h.Create(input); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	// Delete 2 oldest items (A and B)
	if err := h.DeleteOldest(2); err != nil {
		t.Fatalf("DeleteOldest() error: %v", err)
	}

	// List remaining items
	list, err := h.List(0)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(list) != 3 {
		t.Fatalf("List() returned %d items, want 3", len(list))
	}

	// Verify remaining items are C, D, E (newest first)
	expectedTitles := []string{"E", "D", "C"}
	for i, expected := range expectedTitles {
		if list[i].Title != expected {
			t.Errorf("List()[%d].Title = %q, want %q", i, list[i].Title, expected)
		}
	}
}

// TestHistoryStore_Count tests counting items.
func TestHistoryStore_Count(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

	// Initially empty
	count, err := h.Count()
	if err != nil {
		t.Fatalf("Count() error: %v", err)
	}
	if count != 0 {
		t.Errorf("Count() = %d, want 0", count)
	}

	// Add 3 items
	for i := 0; i < 3; i++ {
		input := &store.CreateHistoryInput{
			Title:   "Item",
			Content: bytes.NewReader([]byte("content")),
		}
		if _, err := h.Create(input); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	count, err = h.Count()
	if err != nil {
		t.Fatalf("Count() error: %v", err)
	}
	if count != 3 {
		t.Errorf("Count() = %d, want 3", count)
	}
}

// TestHistoryStore_Clear tests clearing all items.
func TestHistoryStore_Clear(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

	// Add items
	for i := 0; i < 3; i++ {
		input := &store.CreateHistoryInput{
			Title:   "Item",
			Content: bytes.NewReader([]byte("content")),
		}
		if _, err := h.Create(input); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	// Clear
	if err := h.Clear(); err != nil {
		t.Fatalf("Clear() error: %v", err)
	}

	// Verify empty
	count, err := h.Count()
	if err != nil {
		t.Fatalf("Count() error: %v", err)
	}
	if count != 0 {
		t.Errorf("Count() after Clear() = %d, want 0", count)
	}
}

// TestHistoryStore_EmptyStore tests operations on an empty store.
func TestHistoryStore_EmptyStore(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

	// Get non-existent item
	if _, err := h.Get(999); err == nil {
		t.Error("Get() on empty store succeeded, want error")
	}

	// Get content for non-existent item
	if _, err := h.GetContent(999); err == nil {
		t.Error("GetContent() on empty store succeeded, want error")
	}

	// Delete non-existent item
	if err := h.Delete(999); err == nil {
		t.Error("Delete() on empty store succeeded, want error")
	}

	// List should return empty slice
	list, err := h.List(0)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("List() on empty store returned %d items, want 0", len(list))
	}
}

// TestHistoryStore_BinaryContent tests binary content detection.
func TestHistoryStore_BinaryContent(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

	// Create item with binary content (contains null bytes)
	binaryContent := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
	input := &store.CreateHistoryInput{
		Title:   "Binary Item",
		Content: bytes.NewReader(binaryContent),
	}

	item, err := h.Create(input)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if !item.IsBinary {
		t.Error("IsBinary = false, want true for binary content")
	}
}

// TestConfigStore_GetAndSet tests config get and set operations.
func TestConfigStore_GetAndSet(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	c := s.Config()

	// Set a value
	if err := c.Set("key1", "value1"); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	// Get the value
	value, err := c.Get("key1")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if value != "value1" {
		t.Errorf("Get() = %q, want %q", value, "value1")
	}

	// Update the value
	if err := c.Set("key1", "value2"); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	value, err = c.Get("key1")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if value != "value2" {
		t.Errorf("Get() after update = %q, want %q", value, "value2")
	}
}

// TestConfigStore_List tests listing all config values.
func TestConfigStore_List(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	c := s.Config()

	// Set multiple values
	configs := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for k, v := range configs {
		if err := c.Set(k, v); err != nil {
			t.Fatalf("Set() error: %v", err)
		}
	}

	// List all
	list, err := c.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(list) != len(configs) {
		t.Errorf("List() returned %d items, want %d", len(list), len(configs))
	}

	for k, v := range configs {
		if list[k] != v {
			t.Errorf("List()[%q] = %q, want %q", k, list[k], v)
		}
	}

	// Verify List returns a copy (modifying it doesn't affect the store)
	list["key1"] = "modified"
	value, _ := c.Get("key1")
	if value != "value1" {
		t.Error("Modifying List() result affected the store")
	}
}

// TestConfigStore_Delete tests deleting config keys.
func TestConfigStore_Delete(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	c := s.Config()

	// Set a value
	if err := c.Set("key1", "value1"); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	// Delete it
	if err := c.Delete("key1"); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// Verify it's gone
	if _, err := c.Get("key1"); err == nil {
		t.Error("Get() succeeded after Delete(), want error")
	}

	// Try to delete again (should error)
	if err := c.Delete("key1"); err == nil {
		t.Error("Delete() succeeded on non-existent key, want error")
	}
}

// TestConfigStore_EmptyStore tests operations on an empty config store.
func TestConfigStore_EmptyStore(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	c := s.Config()

	// Get non-existent key
	if _, err := c.Get("nonexistent"); err == nil {
		t.Error("Get() on empty store succeeded, want error")
	}

	// Delete non-existent key
	if err := c.Delete("nonexistent"); err == nil {
		t.Error("Delete() on empty store succeeded, want error")
	}

	// List should return empty map
	list, err := c.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("List() on empty store returned %d items, want 0", len(list))
	}
}

// TestBytesReadSeekCloser_ReadAndSeek tests the bytesReadSeekCloser wrapper.
func TestBytesReadSeekCloser_ReadAndSeek(t *testing.T) {
	content := []byte("Hello, World!")
	reader := &bytesReadSeekCloser{reader: bytes.NewReader(content)}
	defer reader.Close()

	// Read first 5 bytes
	buf := make([]byte, 5)
	n, err := reader.Read(buf)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if n != 5 {
		t.Errorf("Read() returned %d bytes, want 5", n)
	}
	if string(buf) != "Hello" {
		t.Errorf("Read() = %q, want %q", buf, "Hello")
	}

	// Seek to beginning
	pos, err := reader.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("Seek() error: %v", err)
	}
	if pos != 0 {
		t.Errorf("Seek() position = %d, want 0", pos)
	}

	// Read again from beginning
	n, err = reader.Read(buf)
	if err != nil {
		t.Fatalf("Read() after Seek() error: %v", err)
	}
	if string(buf) != "Hello" {
		t.Errorf("Read() after Seek() = %q, want %q", buf, "Hello")
	}

	// Seek to end
	pos, err = reader.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatalf("Seek(end) error: %v", err)
	}
	if pos != int64(len(content)) {
		t.Errorf("Seek(end) position = %d, want %d", pos, len(content))
	}

	// Seek with current position
	pos, err = reader.Seek(-5, io.SeekCurrent)
	if err != nil {
		t.Fatalf("Seek(current) error: %v", err)
	}
	if pos != int64(len(content))-5 {
		t.Errorf("Seek(current) position = %d, want %d", pos, len(content)-5)
	}

	// Close should not error
	if err := reader.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}
}

// TestHistoryStore_ConcurrentOperations tests thread safety.
func TestHistoryStore_ConcurrentOperations(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

	var wg sync.WaitGroup
	numGoroutines := 10
	itemsPerGoroutine := 10

	// Concurrent creates
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				input := &store.CreateHistoryInput{
					Title:   "Item",
					Content: bytes.NewReader([]byte("content")),
				}
				if _, err := h.Create(input); err != nil {
					t.Errorf("Concurrent Create() error: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify count
	count, err := h.Count()
	if err != nil {
		t.Fatalf("Count() error: %v", err)
	}
	expected := numGoroutines * itemsPerGoroutine
	if count != expected {
		t.Errorf("Count() after concurrent creates = %d, want %d", count, expected)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := h.List(10); err != nil {
				t.Errorf("Concurrent List() error: %v", err)
			}
		}()
	}

	wg.Wait()
}

// TestConfigStore_ConcurrentOperations tests thread safety of config store.
func TestConfigStore_ConcurrentOperations(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	c := s.Config()

	var wg sync.WaitGroup
	numGoroutines := 10

	// Concurrent sets
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := string(rune('A' + id))
			value := string(rune('a' + id))
			if err := c.Set(key, value); err != nil {
				t.Errorf("Concurrent Set() error: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if _, err := c.List(); err != nil {
				t.Errorf("Concurrent List() error: %v", err)
			}
		}(i)
	}

	wg.Wait()
}

// TestIsBinary tests the binary content detection helper.
func TestIsBinary(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    bool
	}{
		{
			name:    "empty",
			content: []byte{},
			want:    false,
		},
		{
			name:    "text",
			content: []byte("Hello, World!"),
			want:    false,
		},
		{
			name:    "text with newlines",
			content: []byte("Line 1\nLine 2\nLine 3"),
			want:    false,
		},
		{
			name:    "binary with null byte",
			content: []byte{0x00, 0x01, 0x02},
			want:    true,
		},
		{
			name:    "binary with many non-printable",
			content: bytes.Repeat([]byte{0x01, 0x02, 0x03, 0x04}, 100),
			want:    true,
		},
		{
			name:    "text with some control chars",
			content: []byte("Hello\x07World"),
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBinary(tt.content)
			if got != tt.want {
				t.Errorf("isBinary() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHistoryStore_LargeContent tests handling of large content.
func TestHistoryStore_LargeContent(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

	// Create 1MB of content
	largeContent := bytes.Repeat([]byte("A"), 1024*1024)
	input := &store.CreateHistoryInput{
		Title:   "Large Item",
		Content: bytes.NewReader(largeContent),
	}

	item, err := h.Create(input)
	if err != nil {
		t.Fatalf("Create() with large content error: %v", err)
	}

	if item.Size != int64(len(largeContent)) {
		t.Errorf("Size = %d, want %d", item.Size, len(largeContent))
	}

	// Verify content can be retrieved
	reader, err := h.GetContent(item.ID)
	if err != nil {
		t.Fatalf("GetContent() error: %v", err)
	}
	defer reader.Close()

	retrieved, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}

	if !bytes.Equal(retrieved, largeContent) {
		t.Error("Retrieved large content doesn't match original")
	}
}

// TestHistoryStore_StreamingRead tests that content can be read in chunks.
func TestHistoryStore_StreamingRead(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

	// Create item with known content
	content := []byte(strings.Repeat("Hello, World! ", 100))
	input := &store.CreateHistoryInput{
		Title:   "Stream Item",
		Content: bytes.NewReader(content),
	}

	item, err := h.Create(input)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Get content and read in chunks
	reader, err := h.GetContent(item.ID)
	if err != nil {
		t.Fatalf("GetContent() error: %v", err)
	}
	defer reader.Close()

	// Read in 100-byte chunks
	var result []byte
	buf := make([]byte, 100)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read() error: %v", err)
		}
	}

	if !bytes.Equal(result, content) {
		t.Error("Streaming read doesn't match original content")
	}
}

// TestHistoryStore_IDAutoIncrement tests that IDs are auto-incremented correctly.
func TestHistoryStore_IDAutoIncrement(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

	// Create 3 items
	var ids []uint
	for i := 0; i < 3; i++ {
		input := &store.CreateHistoryInput{
			Title:   "Item",
			Content: bytes.NewReader([]byte("content")),
		}
		item, err := h.Create(input)
		if err != nil {
			t.Fatalf("Create() error: %v", err)
		}
		ids = append(ids, item.ID)
	}

	// Verify IDs are sequential
	if ids[0] != 1 {
		t.Errorf("First ID = %d, want 1", ids[0])
	}
	if ids[1] != 2 {
		t.Errorf("Second ID = %d, want 2", ids[1])
	}
	if ids[2] != 3 {
		t.Errorf("Third ID = %d, want 3", ids[2])
	}
}

// TestHistoryStore_SearchTitleOnly tests searching in titles only.
func TestHistoryStore_SearchTitleOnly(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

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
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

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
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

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
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

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
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

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
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

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
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

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
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

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
	s := NewMemoryStore()
	defer s.Close()

	h := s.History()

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
