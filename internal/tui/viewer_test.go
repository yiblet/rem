package tui

import (
	"strings"
	"testing"
)

func TestLeftPaneRightBorderRendering(t *testing.T) {
	// Create test queue items
	items := []*QueueItem{
		{
			Content: NewStringReadSeekCloser("Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."),
			Preview: "Lorem ipsum dol...",
		},
		{
			Content: NewStringReadSeekCloser("SELECT * FROM users WHERE id = 1;"),
			Preview: "SELECT * FROM u...",
		},
	}

	// Create model with specific dimensions
	model := NewModel(items)
	model.width = 120
	model.height = 20
	model.leftWidth = 25
	model.rightWidth = 92 // 120 - 25 - 3

	// Update wrapped lines for the selected item
	if len(items) > 0 {
		items[0].UpdateWrappedLines(model.rightWidth-6, model.height-6)
	}

	// Render the view
	view := model.View()
	lines := strings.Split(view, "\n")

	// Check that we have content
	if len(lines) == 0 {
		t.Fatal("View should not be empty")
	}

	// Find the line that contains the queue content (should have borders)
	var queueLine string
	for _, line := range lines {
		if strings.Contains(line, "Queue") {
			queueLine = line
			break
		}
	}

	if queueLine == "" {
		t.Fatal("Could not find queue header line in view")
	}

	// Print the view for debugging
	t.Logf("Full view:\n%s", view)
	t.Logf("Queue line: '%s'", queueLine)
	t.Logf("Queue line length: %d", len(queueLine))

	// Check if the left pane has proper right border
	// The left pane should end with a border character (│) at position leftWidth-1
	// Looking at the rendered output, we should see the border characters

	// Find lines that should show the border between panes
	var borderCheckLine string
	for i, line := range lines {
		// Skip the status line and header lines, look for content lines
		if i > 2 && i < len(lines)-3 && len(line) > model.leftWidth {
			borderCheckLine = line
			break
		}
	}

	if borderCheckLine == "" {
		t.Fatal("Could not find a suitable line to check border")
	}

	t.Logf("Border check line: '%s'", borderCheckLine)
	t.Logf("Character at position %d: '%c' (code: %d)", model.leftWidth-1, borderCheckLine[model.leftWidth-1], int(borderCheckLine[model.leftWidth-1]))

	// The issue is that the right border of the left pane is missing
	// We should see a border character (│ or similar) but we likely see a space
	// This test will help identify the exact issue
}

func TestModelViewPanesAlignment(t *testing.T) {
	items := []*QueueItem{
		{
			Content: NewStringReadSeekCloser("Test content"),
			Preview: "Test content",
		},
	}

	model := NewModel(items)
	model.UpdateMockSize(120, 15) // Use more realistic dimensions

	view := model.View()
	lines := strings.Split(view, "\n")

	// Check the specific issue with borders - both should be present
	// Use the pure view functions
	leftPane, _ := LeftPaneView(model.app.LeftPane, model.app.Items, false)
	rightPane, _ := RightPaneView(model.app.RightPane, model.app.Items[0], model.app.Search, false, 0)

	leftLines := strings.Split(leftPane, "\n")
	rightLines := strings.Split(rightPane, "\n")

	t.Logf("Left pane (first few lines):")
	for i, line := range leftLines[:min(5, len(leftLines))] {
		t.Logf("  [%d]: '%s' (len: %d)", i, line, len(line))
	}

	t.Logf("Right pane (first few lines):")
	for i, line := range rightLines[:min(5, len(rightLines))] {
		t.Logf("  [%d]: '%s' (len: %d)", i, line, len(line))
	}

	// Test the main fix: check that borders are preserved when joining panes
	var borderTestLine string
	for i, line := range lines {
		if i > 2 && i < len(lines)-3 && len(line) > 25 && strings.Contains(line, "│") {
			borderTestLine = line
			break
		}
	}

	if borderTestLine == "" {
		t.Fatal("Could not find a line with borders to test")
	}

	t.Logf("Border test line: '%s'", borderTestLine)

	// Count border characters
	borderCount := strings.Count(borderTestLine, "│")
	if borderCount < 2 {
		t.Errorf("Expected at least 2 border characters (│), got %d", borderCount)
	}

	// Verify that we have both left pane borders (left edge and right edge)
	firstBorder := strings.Index(borderTestLine, "│")
	lastBorder := strings.LastIndex(borderTestLine, "│")

	if firstBorder == -1 || lastBorder == -1 || firstBorder == lastBorder {
		t.Errorf("Left pane should have both left and right borders. First: %d, Last: %d", firstBorder, lastBorder)
	} else {
		t.Logf("Both borders found: left at %d, right at %d", firstBorder, lastBorder)
	}
}
