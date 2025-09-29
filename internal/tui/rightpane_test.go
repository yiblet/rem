package tui

import (
	"strings"
	"testing"
)

func TestNewRightPaneModel(t *testing.T) {
	width, height := 80, 20
	model := NewRightPaneModel(width, height)

	if model.Width != width {
		t.Errorf("Expected width to be %d, got %d", width, model.Width)
	}
	if model.Height != height {
		t.Errorf("Expected height to be %d, got %d", height, model.Height)
	}
	if model.ViewPos != 0 {
		t.Errorf("Expected ViewPos to be 0, got %d", model.ViewPos)
	}
}

func TestRightPaneView(t *testing.T) {
	width, height := 80, 20
	model := NewRightPaneModel(width, height)
	search := NewSearchModel()

	// Test with no content
	view, err := RightPaneView(model, nil, search, false, 0)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if view == "" {
		t.Error("Expected non-empty view")
	}
	if !strings.Contains(view, "No content selected") {
		t.Error("Expected view to contain 'No content selected'")
	}
}

func TestRightPaneModel_ScrollUp(t *testing.T) {
	model := NewRightPaneModel(80, 20)

	// Set view position to 5
	model.ViewPos = 5

	// Scroll up
	err := model.Update(ScrollUpMsg{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if model.ViewPos != 4 {
		t.Errorf("Expected view position to be 4, got %d", model.ViewPos)
	}

	// Scroll up from position 0 (should not change)
	model.ViewPos = 0
	model.Update(ScrollUpMsg{})
	if model.ViewPos != 0 {
		t.Errorf("Expected view position to stay at 0, got %d", model.ViewPos)
	}
}

func TestRightPaneModel_ScrollDown(t *testing.T) {
	model := NewRightPaneModel(80, 20)
	maxScroll := 10

	// Scroll down
	err := model.Update(ScrollDownMsg{MaxScroll: maxScroll})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if model.ViewPos != 1 {
		t.Errorf("Expected view position to be 1, got %d", model.ViewPos)
	}

	// Scroll down to max
	model.ViewPos = maxScroll
	model.Update(ScrollDownMsg{MaxScroll: maxScroll})
	if model.ViewPos != maxScroll {
		t.Errorf("Expected view position to stay at %d, got %d", maxScroll, model.ViewPos)
	}
}

func TestRightPaneModel_ScrollToTop(t *testing.T) {
	model := NewRightPaneModel(80, 20)
	model.ViewPos = 10

	err := model.Update(ScrollToTopMsg{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if model.ViewPos != 0 {
		t.Errorf("Expected view position to be 0, got %d", model.ViewPos)
	}
}

func TestRightPaneModel_ScrollToBottom(t *testing.T) {
	model := NewRightPaneModel(80, 20)
	maxScroll := 15

	err := model.Update(ScrollToBottomMsg{MaxScroll: maxScroll})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if model.ViewPos != maxScroll {
		t.Errorf("Expected view position to be %d, got %d", maxScroll, model.ViewPos)
	}
}

func TestRightPaneModel_PageUp(t *testing.T) {
	model := NewRightPaneModel(80, 20)
	model.ViewPos = 10

	err := model.Update(PageUpMsg{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Page size is (height - 6) / 2 = (20 - 6) / 2 = 7
	expectedPos := max(10-7, 0)
	if model.ViewPos != expectedPos {
		t.Errorf("Expected view position to be %d, got %d", expectedPos, model.ViewPos)
	}
}

func TestRightPaneModel_PageDown(t *testing.T) {
	model := NewRightPaneModel(80, 20)
	maxScroll := 20

	err := model.Update(PageDownMsg{MaxScroll: maxScroll})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Page size is (height - 6) / 2 = (20 - 6) / 2 = 7
	expectedPos := min(0+7, maxScroll)
	if model.ViewPos != expectedPos {
		t.Errorf("Expected view position to be %d, got %d", expectedPos, model.ViewPos)
	}
}

func TestRightPaneModel_Jump(t *testing.T) {
	model := NewRightPaneModel(80, 20)
	maxScroll := 20

	// Test jump down
	err := model.Update(JumpMsg{Direction: "j", Lines: 5, MaxScroll: maxScroll})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if model.ViewPos != 5 {
		t.Errorf("Expected view position to be 5, got %d", model.ViewPos)
	}

	// Test jump up
	model.Update(JumpMsg{Direction: "k", Lines: 3, MaxScroll: maxScroll})
	if model.ViewPos != 2 {
		t.Errorf("Expected view position to be 2, got %d", model.ViewPos)
	}
}

func TestRightPaneModel_Resize(t *testing.T) {
	model := NewRightPaneModel(80, 20)

	err := model.Update(ResizeRightPaneMsg{Width: 100, Height: 25})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if model.Width != 100 {
		t.Errorf("Expected width to be 100, got %d", model.Width)
	}
	if model.Height != 25 {
		t.Errorf("Expected height to be 25, got %d", model.Height)
	}
}

func TestRightPaneModel_UpdateContent(t *testing.T) {
	model := NewRightPaneModel(80, 20)
	model.ViewPos = 10

	// UpdateContent should reset view position to 0
	err := model.Update(UpdateContentMsg{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if model.ViewPos != 0 {
		t.Errorf("Expected view position to be reset to 0, got %d", model.ViewPos)
	}
}

func TestRightPaneView_WithContent(t *testing.T) {
	model := NewRightPaneModel(80, 20)
	search := NewSearchModel()

	content := &StackItem{
		Content: NewStringReadSeekCloser("Line 1\nLine 2\nLine 3\nLine 4\nLine 5"),
		Preview: "Test content",
	}

	// Ensure lines are calculated
	content.UpdateWrappedLines(model.Width - 6)

	view, err := RightPaneView(model, content, search, false, 0)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if view == "" {
		t.Error("Expected non-empty view")
	}
	if !strings.Contains(view, "Content [0]") {
		t.Error("Expected view to contain content title")
	}
	if !strings.Contains(view, "Line 1") {
		t.Error("Expected view to contain first line")
	}
}

func TestRightPaneView_Focused(t *testing.T) {
	model := NewRightPaneModel(80, 20)
	search := NewSearchModel()

	content := &StackItem{
		Content: NewStringReadSeekCloser("Test content"),
		Preview: "Test content",
	}

	// Test focused view with content
	view, err := RightPaneView(model, content, search, true, 0)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(view, "● Content [0]") {
		t.Error("Expected focused view to contain '● Content [0]'")
	}
}