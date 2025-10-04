package queue

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/yiblet/rem/internal/store/memstore"
)

func TestQueueManager_Basic(t *testing.T) {
	ms := memstore.NewMemoryStore()
	defer ms.Close()

	qm, err := NewQueueManager(ms)
	if err != nil {
		t.Fatalf("Failed to create queue manager: %v", err)
	}
	defer qm.Close()

	// Test empty queue
	size, err := qm.Size()
	if err != nil {
		t.Fatalf("Failed to get size: %v", err)
	}
	if size != 0 {
		t.Errorf("Expected empty queue, got size %d", size)
	}

	// Test enqueue with auto-generated title
	content := strings.NewReader("Hello, World!")
	item, err := qm.Enqueue(content, "")
	if err != nil {
		t.Fatalf("Failed to enqueue: %v", err)
	}

	if item.Title != "Hello, World!" {
		t.Errorf("Expected title 'Hello, World!', got '%s'", item.Title)
	}

	// Test size after enqueue
	size, err = qm.Size()
	if err != nil {
		t.Fatalf("Failed to get size: %v", err)
	}
	if size != 1 {
		t.Errorf("Expected size 1, got %d", size)
	}

	// Test get
	retrievedItem, err := qm.Get(0)
	if err != nil {
		t.Fatalf("Failed to get item: %v", err)
	}

	if retrievedItem.Title != item.Title {
		t.Errorf("Expected title '%s', got '%s'", item.Title, retrievedItem.Title)
	}

	// Test content reader
	reader, err := qm.GetContent(retrievedItem.ID)
	if err != nil {
		t.Fatalf("Failed to get content reader: %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read content: %v", err)
	}

	if string(data) != "Hello, World!" {
		t.Errorf("Expected content 'Hello, World!', got '%s'", string(data))
	}
}

func TestQueueManager_ExplicitTitle(t *testing.T) {
	ms := memstore.NewMemoryStore()
	defer ms.Close()

	qm, err := NewQueueManager(ms)
	if err != nil {
		t.Fatalf("Failed to create queue manager: %v", err)
	}
	defer qm.Close()

	// Test enqueue with explicit title
	content := strings.NewReader("Some content that would generate a different title")
	item, err := qm.Enqueue(content, "My Custom Title")
	if err != nil {
		t.Fatalf("Failed to enqueue: %v", err)
	}

	if item.Title != "My Custom Title" {
		t.Errorf("Expected title 'My Custom Title', got '%s'", item.Title)
	}

	// Verify content is still correct
	reader, err := qm.GetContent(item.ID)
	if err != nil {
		t.Fatalf("Failed to get content: %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read content: %v", err)
	}

	expected := "Some content that would generate a different title"
	if string(data) != expected {
		t.Errorf("Expected content '%s', got '%s'", expected, string(data))
	}
}

func TestQueueManager_TitleTruncation(t *testing.T) {
	ms := memstore.NewMemoryStore()
	defer ms.Close()

	qm, err := NewQueueManager(ms)
	if err != nil {
		t.Fatalf("Failed to create queue manager: %v", err)
	}
	defer qm.Close()

	// Create a title longer than 80 characters
	longTitle := strings.Repeat("a", 100)
	content := strings.NewReader("content")
	item, err := qm.Enqueue(content, longTitle)
	if err != nil {
		t.Fatalf("Failed to enqueue: %v", err)
	}

	// Title should be truncated to 80 chars (77 + "...")
	if len(item.Title) != 80 {
		t.Errorf("Expected title length 80, got %d", len(item.Title))
	}

	if !strings.HasSuffix(item.Title, "...") {
		t.Errorf("Expected truncated title to end with '...', got '%s'", item.Title)
	}
}

func TestQueueManager_MaxSize(t *testing.T) {
	ms := memstore.NewMemoryStore()
	defer ms.Close()

	qm, err := NewQueueManager(ms)
	if err != nil {
		t.Fatalf("Failed to create queue manager: %v", err)
	}
	defer qm.Close()

	// Add more than DefaultMaxQueueSize items
	for i := 0; i < DefaultMaxQueueSize+5; i++ {
		content := strings.NewReader(fmt.Sprintf("Content %d", i))
		_, err := qm.Enqueue(content, "")
		if err != nil {
			t.Fatalf("Failed to enqueue item %d: %v", i, err)
		}
	}

	// Check that size is limited to DefaultMaxQueueSize
	size, err := qm.Size()
	if err != nil {
		t.Fatalf("Failed to get size: %v", err)
	}

	if size != DefaultMaxQueueSize {
		t.Errorf("Expected size %d, got %d", DefaultMaxQueueSize, size)
	}

	// Check that newest items are kept (LIFO - should have items 5-259)
	items, err := qm.List()
	if err != nil {
		t.Fatalf("Failed to list items: %v", err)
	}

	// First item should be newest (Content 259)
	if !strings.Contains(items[0].Title, "Content 259") {
		t.Errorf("Expected newest item to contain 'Content 259', got '%s'", items[0].Title)
	}

	// Last item should be Content 5 (oldest remaining after removing 0-4)
	lastIndex := len(items) - 1
	if !strings.Contains(items[lastIndex].Title, "Content 5") {
		t.Errorf("Expected oldest item to contain 'Content 5', got '%s'", items[lastIndex].Title)
	}
}

func TestQueueManager_TitleGeneration(t *testing.T) {
	ms := memstore.NewMemoryStore()
	defer ms.Close()

	qm, err := NewQueueManager(ms)
	if err != nil {
		t.Fatalf("Failed to create queue manager: %v", err)
	}
	defer qm.Close()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple text",
			input:    "Short text",
			expected: "Short text",
		},
		{
			name:     "Text with newlines - uses first line",
			input:    "First line\nSecond line\nThird line",
			expected: "First line",
		},
		{
			name:     "Text with whitespace",
			input:    "Text with\nnewlines\tand\ttabs",
			expected: "Text with",
		},
		{
			name:     "Long text gets truncated",
			input:    strings.Repeat("a", 100),
			expected: strings.Repeat("a", 77) + "...",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content := strings.NewReader(tc.input)
			item, err := qm.Enqueue(content, "")
			if err != nil {
				t.Fatalf("Failed to enqueue: %v", err)
			}

			if item.Title != tc.expected {
				t.Errorf("Expected title '%s', got '%s'", tc.expected, item.Title)
			}
		})
	}
}

func TestQueueManager_Clear(t *testing.T) {
	ms := memstore.NewMemoryStore()
	defer ms.Close()

	qm, err := NewStackManager(ms)
	if err != nil {
		t.Fatalf("Failed to create queue manager: %v", err)
	}
	defer qm.Close()

	// Add some items to the queue
	for i := 0; i < 5; i++ {
		content := strings.NewReader(fmt.Sprintf("Item %d", i))
		_, err := qm.Push(content)
		if err != nil {
			t.Fatalf("Failed to push item %d: %v", i, err)
		}
	}

	// Verify items were added
	size, err := qm.Size()
	if err != nil {
		t.Fatalf("Failed to get size: %v", err)
	}
	if size != 5 {
		t.Errorf("Expected size 5, got %d", size)
	}

	// Clear the queue
	err = qm.Clear()
	if err != nil {
		t.Fatalf("Failed to clear queue: %v", err)
	}

	// Verify queue is empty
	size, err = qm.Size()
	if err != nil {
		t.Fatalf("Failed to get size after clear: %v", err)
	}
	if size != 0 {
		t.Errorf("Expected size 0 after clear, got %d", size)
	}

	// Test clearing empty queue (should not error)
	err = qm.Clear()
	if err != nil {
		t.Fatalf("Failed to clear empty queue: %v", err)
	}
}

func TestQueueManager_Delete(t *testing.T) {
	ms := memstore.NewMemoryStore()
	defer ms.Close()

	qm, err := NewQueueManager(ms)
	if err != nil {
		t.Fatalf("Failed to create queue manager: %v", err)
	}
	defer qm.Close()

	// Add some items
	for i := 0; i < 3; i++ {
		content := strings.NewReader(fmt.Sprintf("Item %d", i))
		_, err := qm.Enqueue(content, "")
		if err != nil {
			t.Fatalf("Failed to enqueue item %d: %v", i, err)
		}
	}

	// Delete item at index 1 (middle item - "Item 1" in LIFO order is "Item 1")
	err = qm.Delete(1)
	if err != nil {
		t.Fatalf("Failed to delete item: %v", err)
	}

	// Verify size decreased
	size, err := qm.Size()
	if err != nil {
		t.Fatalf("Failed to get size: %v", err)
	}
	if size != 2 {
		t.Errorf("Expected size 2 after delete, got %d", size)
	}

	// Verify correct items remain (Item 2 and Item 0 in LIFO order)
	items, err := qm.List()
	if err != nil {
		t.Fatalf("Failed to list items: %v", err)
	}

	if !strings.Contains(items[0].Title, "Item 2") {
		t.Errorf("Expected first item to contain 'Item 2', got '%s'", items[0].Title)
	}

	if !strings.Contains(items[1].Title, "Item 0") {
		t.Errorf("Expected second item to contain 'Item 0', got '%s'", items[1].Title)
	}
}

func TestQueueManager_BinaryDetection(t *testing.T) {
	ms := memstore.NewMemoryStore()
	defer ms.Close()

	qm, err := NewStackManager(ms)
	if err != nil {
		t.Fatalf("Failed to create queue manager: %v", err)
	}
	defer qm.Close()

	// Test with binary content
	binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}
	item, err := qm.Push(strings.NewReader(string(binaryData)))
	if err != nil {
		t.Fatalf("Failed to push binary content: %v", err)
	}

	if !item.IsBinary {
		t.Error("Expected binary content to be detected as binary")
	}

	if item.Title != "[binary content]" {
		t.Errorf("Expected title '[binary content]', got '%s'", item.Title)
	}

	if item.Size != int64(len(binaryData)) {
		t.Errorf("Expected size %d, got %d", len(binaryData), item.Size)
	}

	if item.SHA256 == "" {
		t.Error("Expected SHA256 hash to be calculated for binary content")
	}

	// Test with text content
	textData := "This is plain text content"
	textItem, err := qm.Push(strings.NewReader(textData))
	if err != nil {
		t.Fatalf("Failed to push text content: %v", err)
	}

	if textItem.IsBinary {
		t.Error("Expected text content to not be detected as binary")
	}

	if textItem.Size != int64(len(textData)) {
		t.Errorf("Expected size %d, got %d", len(textData), textItem.Size)
	}
}

func TestQueueManager_LIFO_Ordering(t *testing.T) {
	ms := memstore.NewMemoryStore()
	defer ms.Close()

	qm, err := NewQueueManager(ms)
	if err != nil {
		t.Fatalf("Failed to create queue manager: %v", err)
	}
	defer qm.Close()

	// Add items with explicit timestamps to ensure ordering
	for i := 0; i < 5; i++ {
		content := strings.NewReader(fmt.Sprintf("Item %d", i))
		_, err := qm.Enqueue(content, "")
		if err != nil {
			t.Fatalf("Failed to enqueue item %d: %v", i, err)
		}
		// Small delay to ensure different timestamps
		time.Sleep(1 * time.Millisecond)
	}

	// List should return items in LIFO order (newest first)
	items, err := qm.List()
	if err != nil {
		t.Fatalf("Failed to list items: %v", err)
	}

	if len(items) != 5 {
		t.Fatalf("Expected 5 items, got %d", len(items))
	}

	// Verify LIFO order: Item 4, Item 3, Item 2, Item 1, Item 0
	for i, item := range items {
		expectedTitle := fmt.Sprintf("Item %d", 4-i)
		if !strings.Contains(item.Title, expectedTitle) {
			t.Errorf("Expected item at index %d to contain '%s', got '%s'", i, expectedTitle, item.Title)
		}
	}
}

func TestQueueManager_StreamingContent(t *testing.T) {
	ms := memstore.NewMemoryStore()
	defer ms.Close()

	qm, err := NewQueueManager(ms)
	if err != nil {
		t.Fatalf("Failed to create queue manager: %v", err)
	}
	defer qm.Close()

	// Create a large content (1MB)
	largeContent := strings.Repeat("A", 1024*1024)
	item, err := qm.Enqueue(strings.NewReader(largeContent), "Large File")
	if err != nil {
		t.Fatalf("Failed to enqueue large content: %v", err)
	}

	if item.Size != int64(len(largeContent)) {
		t.Errorf("Expected size %d, got %d", len(largeContent), item.Size)
	}

	// Read content back in streaming fashion
	reader, err := qm.GetContent(item.ID)
	if err != nil {
		t.Fatalf("Failed to get content: %v", err)
	}
	defer reader.Close()

	// Read in chunks to verify streaming works
	buf := make([]byte, 32*1024) // 32KB chunks
	totalRead := 0
	for {
		n, err := reader.Read(buf)
		totalRead += n
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read chunk: %v", err)
		}
	}

	if totalRead != len(largeContent) {
		t.Errorf("Expected to read %d bytes, got %d", len(largeContent), totalRead)
	}
}

func TestIsBinary(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "Plain text",
			data:     []byte("Hello, world! This is plain text."),
			expected: false,
		},
		{
			name:     "Text with newlines",
			data:     []byte("Line 1\nLine 2\nLine 3\n"),
			expected: false,
		},
		{
			name:     "Empty data",
			data:     []byte{},
			expected: false,
		},
		{
			name:     "Binary with null bytes",
			data:     []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05},
			expected: true,
		},
		{
			name:     "Binary with control characters",
			data:     []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			expected: true,
		},
		{
			name:     "Text with tabs and spaces",
			data:     []byte("Hello\tWorld\n\tIndented line\n"),
			expected: false,
		},
		{
			name:     "Mixed content - mostly text",
			data:     append([]byte("Text content "), 0x01, 0x02),
			expected: false,
		},
		{
			name:     "Mixed content - mostly binary",
			data:     append(make([]byte, 100), []byte("text")...),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBinary(tt.data)
			if result != tt.expected {
				t.Errorf("isBinary() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestTitleGeneration(t *testing.T) {
	tests := []struct {
		name     string
		sample   []byte
		isBinary bool
		expected string
	}{
		{
			name:     "Binary content",
			sample:   []byte{0x00, 0x01, 0x02},
			isBinary: true,
			expected: "[binary content]",
		},
		{
			name:     "Empty content",
			sample:   []byte{},
			isBinary: false,
			expected: "[empty]",
		},
		{
			name:     "Single line",
			sample:   []byte("Hello, World!"),
			isBinary: false,
			expected: "Hello, World!",
		},
		{
			name:     "Multiple lines - uses first",
			sample:   []byte("First line\nSecond line\nThird line"),
			isBinary: false,
			expected: "First line",
		},
		{
			name:     "Leading whitespace",
			sample:   []byte("   \n\n  First line\nSecond line"),
			isBinary: false,
			expected: "First line",
		},
		{
			name:     "Control characters sanitized",
			sample:   []byte("Text with\x01control\x02chars"),
			isBinary: false,
			expected: "Text with control chars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateTitle(tt.sample, tt.isBinary)
			if result != tt.expected {
				t.Errorf("GenerateTitle() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestTruncateTitle(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		maxLen   int
		expected string
	}{
		{
			name:     "Short title, no truncation",
			title:    "Short",
			maxLen:   80,
			expected: "Short",
		},
		{
			name:     "Exact length, no truncation",
			title:    strings.Repeat("a", 80),
			maxLen:   80,
			expected: strings.Repeat("a", 80),
		},
		{
			name:     "Long title, truncate with ellipsis",
			title:    strings.Repeat("a", 100),
			maxLen:   80,
			expected: strings.Repeat("a", 77) + "...",
		},
		{
			name:     "Very short maxLen",
			title:    "Hello",
			maxLen:   3,
			expected: "...",
		},
		{
			name:     "Trim whitespace",
			title:    "  Title with spaces  ",
			maxLen:   80,
			expected: "Title with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateTitle(tt.title, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateTitle() = %q, expected %q", result, tt.expected)
			}
			if len(result) > tt.maxLen {
				t.Errorf("TruncateTitle() returned length %d, expected <= %d", len(result), tt.maxLen)
			}
		})
	}
}

func TestSanitizeTitle(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected string
	}{
		{
			name:     "Normal text",
			title:    "Normal text",
			expected: "Normal text",
		},
		{
			name:     "Control characters removed",
			title:    "Text\x01with\x02control\x03chars",
			expected: "Text with control chars",
		},
		{
			name:     "Multiple spaces collapsed",
			title:    "Too     many    spaces",
			expected: "Too many spaces",
		},
		{
			name:     "Tabs converted to spaces",
			title:    "Text\twith\ttabs",
			expected: "Text with tabs",
		},
		{
			name:     "Newlines collapsed",
			title:    "Text\nwith\nnewlines",
			expected: "Text with newlines",
		},
		{
			name:     "Mixed whitespace",
			title:    "  Text \n with \t mixed \r\n whitespace  ",
			expected: "Text with mixed whitespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeTitle(tt.title)
			if result != tt.expected {
				t.Errorf("SanitizeTitle() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
