package tui

import (
	"strings"
	"testing"
)

// TestStackItem_WrappingRecalculationOnWidthChange verifies that lines are
// recalculated when the width changes
func TestStackItem_WrappingRecalculationOnWidthChange(t *testing.T) {
	// Create a StackItem with long content
	longLine := "This is a very long line that will need different wrapping at different widths to ensure proper display"
	content := &StackItem{
		Content: NewStringReadSeekCloser(longLine),
		Preview: "Long content",
	}

	// First wrap at width 30
	err := content.UpdateWrappedLines(30, 100)
	if err != nil {
		t.Fatalf("Failed to wrap at width 30: %v", err)
	}

	firstLineCount := len(content.Lines)
	firstCachedWidth := content.CachedWidth

	if firstCachedWidth != 30 {
		t.Errorf("Expected CachedWidth to be 30, got %d", firstCachedWidth)
	}

	// Verify all lines fit within width 30
	for i, line := range content.Lines {
		if len(line) > 30 {
			t.Errorf("Line %d exceeds width 30: %q (len=%d)", i, line, len(line))
		}
	}

	// Now wrap at width 50 - should recalculate
	err = content.UpdateWrappedLines(50, 100)
	if err != nil {
		t.Fatalf("Failed to wrap at width 50: %v", err)
	}

	secondLineCount := len(content.Lines)
	secondCachedWidth := content.CachedWidth

	if secondCachedWidth != 50 {
		t.Errorf("Expected CachedWidth to be 50, got %d", secondCachedWidth)
	}

	// Verify all lines fit within width 50
	for i, line := range content.Lines {
		if len(line) > 50 {
			t.Errorf("Line %d exceeds width 50: %q (len=%d)", i, line, len(line))
		}
	}

	// With wider width, we should have fewer lines (less wrapping needed)
	if secondLineCount >= firstLineCount {
		t.Errorf("Expected fewer lines at width 50 (%d) than at width 30 (%d)", secondLineCount, firstLineCount)
	}
}

// TestStackItem_WrappingNotRecalculatedWhenWidthSame verifies that lines are
// NOT recalculated when the width hasn't changed (performance optimization)
func TestStackItem_WrappingNotRecalculatedWhenWidthSame(t *testing.T) {
	content := &StackItem{
		Content: NewStringReadSeekCloser("Some content that will be wrapped"),
		Preview: "Test",
	}

	// First wrap
	err := content.UpdateWrappedLines(40, 100)
	if err != nil {
		t.Fatalf("Failed to wrap: %v", err)
	}

	// Manually modify the Lines to detect if they get recalculated
	originalFirstLine := content.Lines[0]
	content.Lines[0] = "MODIFIED_LINE"

	// Call UpdateWrappedLines again with same width
	err = content.UpdateWrappedLines(40, 100)
	if err != nil {
		t.Fatalf("Failed to wrap: %v", err)
	}

	// If cache worked correctly, the modification should still be there
	// (because UpdateWrappedLines should have returned early)
	if content.Lines[0] != "MODIFIED_LINE" {
		t.Errorf("Lines were recalculated when they shouldn't have been")
	}

	// Now change width - should recalculate and overwrite our modification
	err = content.UpdateWrappedLines(50, 100)
	if err != nil {
		t.Fatalf("Failed to wrap: %v", err)
	}

	// The modification should be gone now
	if content.Lines[0] == "MODIFIED_LINE" {
		t.Errorf("Lines were not recalculated when width changed")
	}

	// The original content should be back (or at least not our modification)
	if !strings.Contains(strings.Join(content.Lines, " "), "Some content") {
		t.Errorf("Content not properly recalculated after width change")
	}

	// Restore for cleanup
	_ = originalFirstLine
}

// TestStackItem_WrappingRecalculatedOnFirstCall verifies that lines are
// calculated the first time even if CachedWidth is 0
func TestStackItem_WrappingRecalculatedOnFirstCall(t *testing.T) {
	content := &StackItem{
		Content:     NewStringReadSeekCloser("Test content"),
		Preview:     "Test",
		CachedWidth: 0, // Explicitly 0 (uninitialized)
		Lines:       nil,
	}

	err := content.UpdateWrappedLines(40, 100)
	if err != nil {
		t.Fatalf("Failed to wrap: %v", err)
	}

	if len(content.Lines) == 0 {
		t.Errorf("Lines should have been calculated on first call")
	}

	if content.CachedWidth != 40 {
		t.Errorf("Expected CachedWidth to be 40, got %d", content.CachedWidth)
	}
}