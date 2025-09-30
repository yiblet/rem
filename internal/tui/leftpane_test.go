package tui

import (
	"strings"
	"testing"
)

func TestNewLeftPaneModel(t *testing.T) {
	width, height := 30, 20
	model := NewLeftPaneModel(width, height)

	if model.Cursor != 0 {
		t.Errorf("Expected cursor to be 0, got %d", model.Cursor)
	}
	if model.Selected != 0 {
		t.Errorf("Expected selected to be 0, got %d", model.Selected)
	}
	if model.Width != width {
		t.Errorf("Expected width to be %d, got %d", width, model.Width)
	}
	if model.Height != height {
		t.Errorf("Expected height to be %d, got %d", height, model.Height)
	}
}

func TestLeftPaneView(t *testing.T) {
	items := []*StackItem{
		{Preview: "Item 1"},
		{Preview: "Item 2"},
	}
	width, height := 30, 20
	model := NewLeftPaneModel(width, height)

	view, err := LeftPaneView(model, items, false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if view == "" {
		t.Error("Expected non-empty view")
	}
	if !strings.Contains(view, "Queue") {
		t.Error("Expected view to contain 'Queue'")
	}
	if !strings.Contains(view, "Item 1") {
		t.Error("Expected view to contain 'Item 1'")
	}
	if !strings.Contains(view, "Item 2") {
		t.Error("Expected view to contain 'Item 2'")
	}
}

func TestLeftPaneModel_NavigateUp(t *testing.T) {
	model := NewLeftPaneModel(30, 20)

	// Start at position 2
	model.Update(SelectItemMsg{Index: 2})
	if model.Cursor != 2 {
		t.Errorf("Expected cursor to be 2, got %d", model.Cursor)
	}

	// Navigate up once
	err := model.Update(NavigateUpMsg{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if model.Cursor != 1 {
		t.Errorf("Expected cursor to be 1, got %d", model.Cursor)
	}
	if model.Selected != 1 {
		t.Errorf("Expected selected to be 1, got %d", model.Selected)
	}

	// Navigate up again
	model.Update(NavigateUpMsg{})
	if model.Cursor != 0 {
		t.Errorf("Expected cursor to be 0, got %d", model.Cursor)
	}

	// Try to navigate up from position 0 (should not change)
	model.Update(NavigateUpMsg{})
	if model.Cursor != 0 {
		t.Errorf("Expected cursor to stay at 0, got %d", model.Cursor)
	}
}

func TestLeftPaneModel_NavigateDown(t *testing.T) {
	model := NewLeftPaneModel(30, 20)
	maxIndex := 2 // Simulate 3 items (0, 1, 2)

	// Start at position 0
	if model.Cursor != 0 {
		t.Errorf("Expected cursor to be 0, got %d", model.Cursor)
	}

	// Navigate down once
	err := model.Update(NavigateDownMsg{MaxIndex: maxIndex})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if model.Cursor != 1 {
		t.Errorf("Expected cursor to be 1, got %d", model.Cursor)
	}
	if model.Selected != 1 {
		t.Errorf("Expected selected to be 1, got %d", model.Selected)
	}

	// Navigate down again
	model.Update(NavigateDownMsg{MaxIndex: maxIndex})
	if model.Cursor != 2 {
		t.Errorf("Expected cursor to be 2, got %d", model.Cursor)
	}

	// Try to navigate down from last position (should not change)
	model.Update(NavigateDownMsg{MaxIndex: maxIndex})
	if model.Cursor != 2 {
		t.Errorf("Expected cursor to stay at 2, got %d", model.Cursor)
	}
}

func TestLeftPaneModel_SelectItem(t *testing.T) {
	model := NewLeftPaneModel(30, 20)

	// Select item at index 1
	err := model.Update(SelectItemMsg{Index: 1})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if model.Cursor != 1 {
		t.Errorf("Expected cursor to be 1, got %d", model.Cursor)
	}
	if model.Selected != 1 {
		t.Errorf("Expected selected to be 1, got %d", model.Selected)
	}

	// Try to select invalid index (negative) - should be ignored
	model.Update(SelectItemMsg{Index: -1})
	if model.Cursor != 1 {
		t.Errorf("Expected cursor to remain 1 after invalid selection, got %d", model.Cursor)
	}

	// Select valid index
	model.Update(SelectItemMsg{Index: 2})
	if model.Cursor != 2 {
		t.Errorf("Expected cursor to be 2, got %d", model.Cursor)
	}
}

func TestLeftPaneModel_ResizeLeftPane(t *testing.T) {
	model := NewLeftPaneModel(30, 20)

	newWidth, newHeight := 40, 25
	err := model.Update(ResizeLeftPaneMsg{Width: newWidth, Height: newHeight})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if model.Width != newWidth {
		t.Errorf("Expected width to be %d, got %d", newWidth, model.Width)
	}
	if model.Height != newHeight {
		t.Errorf("Expected height to be %d, got %d", newHeight, model.Height)
	}
}

func TestLeftPaneView_Focused(t *testing.T) {
	items := []*StackItem{
		{Preview: "Test item 1"},
		{Preview: "Test item 2"},
	}
	model := NewLeftPaneModel(30, 20)

	// Test unfocused view
	view, err := LeftPaneView(model, items, false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if view == "" {
		t.Error("Expected non-empty view")
	}
	if !strings.Contains(view, "Queue") {
		t.Error("Expected view to contain 'Queue' title")
	}
	if !strings.Contains(view, "Test item 1") {
		t.Error("Expected view to contain item preview")
	}

	// Test focused view
	focusedView, err := LeftPaneView(model, items, true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(focusedView, "● Queue") {
		t.Error("Expected focused view to contain '● Queue' title")
	}

	// Views should be different (focused has different styling)
	if view == focusedView {
		t.Error("Expected focused and unfocused views to be different")
	}
}

func TestLeftPaneView_EmptyItems(t *testing.T) {
	model := NewLeftPaneModel(30, 20)

	view, err := LeftPaneView(model, []*StackItem{}, false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if view == "" {
		t.Error("Expected non-empty view even with no items")
	}
	if !strings.Contains(view, "Queue") {
		t.Error("Expected view to contain 'Queue' title even with no items")
	}
}
