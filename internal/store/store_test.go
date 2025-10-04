package store

import (
	"bytes"
	"io"
	"testing"
	"time"
)

// TestInterfaceCompilation verifies that the interfaces compile correctly.
// This test ensures all interface methods are properly defined.
func TestInterfaceCompilation(t *testing.T) {
	// This test will fail to compile if interfaces are malformed
	var _ HistoryStore = (*mockHistoryStore)(nil)
	var _ ConfigStore = (*mockConfigStore)(nil)
	var _ Store = (*mockStore)(nil)
}

// TestHistoryItemFields verifies HistoryItem has all required fields.
func TestHistoryItemFields(t *testing.T) {
	item := &HistoryItem{
		ID:        1,
		Title:     "test",
		Timestamp: time.Now(),
		IsBinary:  false,
		Size:      100,
		SHA256:    "abc123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if item.ID != 1 {
		t.Errorf("ID = %d, want 1", item.ID)
	}
	if item.Title != "test" {
		t.Errorf("Title = %s, want test", item.Title)
	}
	if item.Size != 100 {
		t.Errorf("Size = %d, want 100", item.Size)
	}
}

// TestCreateHistoryInputFields verifies CreateHistoryInput has all required fields.
func TestCreateHistoryInputFields(t *testing.T) {
	content := bytes.NewReader([]byte("test content"))
	ts := time.Now()

	input := &CreateHistoryInput{
		Title:     "test title",
		Content:   content,
		Timestamp: ts,
		IsBinary:  false,
	}

	if input.Title != "test title" {
		t.Errorf("Title = %s, want 'test title'", input.Title)
	}
	if input.Content != content {
		t.Error("Content reader not set correctly")
	}
	if !input.Timestamp.Equal(ts) {
		t.Errorf("Timestamp = %v, want %v", input.Timestamp, ts)
	}
	if input.IsBinary {
		t.Error("IsBinary should be false")
	}
}

// TestCreateHistoryInputContentIsReader verifies Content field is io.Reader.
func TestCreateHistoryInputContentIsReader(t *testing.T) {
	content := bytes.NewReader([]byte("test"))
	input := &CreateHistoryInput{
		Content: content,
	}

	// Verify we can read from it
	var r io.Reader = input.Content
	buf := make([]byte, 4)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	if n != 4 {
		t.Errorf("Read %d bytes, want 4", n)
	}
	if string(buf) != "test" {
		t.Errorf("Content = %s, want 'test'", string(buf))
	}
}

// TestSearchQueryFields verifies SearchQuery has all required fields.
func TestSearchQueryFields(t *testing.T) {
	query := &SearchQuery{
		Pattern:       "test.*pattern",
		SearchTitle:   true,
		SearchContent: false,
		Limit:         10,
		CaseSensitive: true,
	}

	if query.Pattern != "test.*pattern" {
		t.Errorf("Pattern = %s, want 'test.*pattern'", query.Pattern)
	}
	if !query.SearchTitle {
		t.Error("SearchTitle should be true")
	}
	if query.SearchContent {
		t.Error("SearchContent should be false")
	}
	if query.Limit != 10 {
		t.Errorf("Limit = %d, want 10", query.Limit)
	}
	if !query.CaseSensitive {
		t.Error("CaseSensitive should be true")
	}
}

// TestSearchResultFields verifies SearchResult has all required fields.
func TestSearchResultFields(t *testing.T) {
	item := &HistoryItem{ID: 1, Title: "test"}
	matches := []string{"match1", "match2"}

	result := &SearchResult{
		Item:    item,
		Matches: matches,
	}

	if result.Item != item {
		t.Error("Item not set correctly")
	}
	if len(result.Matches) != 2 {
		t.Errorf("Matches length = %d, want 2", len(result.Matches))
	}
	if result.Matches[0] != "match1" {
		t.Errorf("Matches[0] = %s, want 'match1'", result.Matches[0])
	}
}

// Mock implementations for interface compliance testing

type mockHistoryStore struct{}

func (m *mockHistoryStore) Create(item *CreateHistoryInput) (*HistoryItem, error) {
	return nil, nil
}

func (m *mockHistoryStore) List(limit int) ([]*HistoryItem, error) {
	return nil, nil
}

func (m *mockHistoryStore) Get(id uint) (*HistoryItem, error) {
	return nil, nil
}

func (m *mockHistoryStore) GetContent(id uint) (io.ReadSeekCloser, error) {
	return nil, nil
}

func (m *mockHistoryStore) Delete(id uint) error {
	return nil
}

func (m *mockHistoryStore) DeleteOldest(count int) error {
	return nil
}

func (m *mockHistoryStore) Count() (int, error) {
	return 0, nil
}

func (m *mockHistoryStore) Clear() error {
	return nil
}

func (m *mockHistoryStore) Search(query *SearchQuery) ([]*HistoryItem, error) {
	return nil, nil
}

func (m *mockHistoryStore) Close() error {
	return nil
}

type mockConfigStore struct{}

func (m *mockConfigStore) Get(key string) (string, error) {
	return "", nil
}

func (m *mockConfigStore) Set(key, value string) error {
	return nil
}

func (m *mockConfigStore) List() (map[string]string, error) {
	return nil, nil
}

func (m *mockConfigStore) Delete(key string) error {
	return nil
}

func (m *mockConfigStore) Close() error {
	return nil
}

type mockStore struct {
	history *mockHistoryStore
	config  *mockConfigStore
}

func (m *mockStore) History() HistoryStore {
	return m.history
}

func (m *mockStore) Config() ConfigStore {
	return m.config
}

func (m *mockStore) Close() error {
	return nil
}
