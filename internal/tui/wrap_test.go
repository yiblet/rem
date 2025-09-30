package tui

import (
	"strings"
	"testing"
)

func TestWrapText_FitsWithinWidth(t *testing.T) {
	text := "Hello world"
	width := 20
	result := WrapText(text, width)

	if len(result) != 1 {
		t.Errorf("Expected 1 line, got %d", len(result))
	}
	if result[0] != text {
		t.Errorf("Expected %q, got %q", text, result[0])
	}
}

func TestWrapText_SimpleWrap(t *testing.T) {
	text := "Hello world this is a test"
	width := 15
	result := WrapText(text, width)

	// Each line should be <= width
	for i, line := range result {
		if len(line) > width {
			t.Errorf("Line %d exceeds width: %q (len=%d, max=%d)", i, line, len(line), width)
		}
	}

	// Content should be preserved (modulo whitespace normalization)
	joined := strings.Join(result, " ")
	if !strings.Contains(joined, "Hello") || !strings.Contains(joined, "world") {
		t.Errorf("Content not preserved: %v", result)
	}
}

func TestWrapText_LongWordBreak(t *testing.T) {
	text := "ThisIsAVeryLongWordThatExceedsTheMaxWidth"
	width := 10
	result := WrapText(text, width)

	// Should break the long word
	for i, line := range result {
		if len(line) > width {
			t.Errorf("Line %d exceeds width: %q (len=%d, max=%d)", i, line, len(line), width)
		}
	}

	// Should have multiple lines
	if len(result) < 2 {
		t.Errorf("Expected word to be broken into multiple lines, got %d", len(result))
	}
}

func TestWrapText_PreservesNewlines(t *testing.T) {
	text := "Line 1\nLine 2\nLine 3"
	width := 20
	result := WrapText(text, width)

	if len(result) != 3 {
		t.Errorf("Expected 3 lines (newlines preserved), got %d", len(result))
	}
	if result[0] != "Line 1" || result[1] != "Line 2" || result[2] != "Line 3" {
		t.Errorf("Lines not preserved correctly: %v", result)
	}
}

func TestWrapText_EmptyLines(t *testing.T) {
	text := "Line 1\n\nLine 3"
	width := 20
	result := WrapText(text, width)

	if len(result) != 3 {
		t.Errorf("Expected 3 lines (with empty), got %d", len(result))
	}
	if result[1] != "" {
		t.Errorf("Expected empty line at index 1, got %q", result[1])
	}
}

func TestWrapText_ZeroWidth(t *testing.T) {
	text := "Hello"
	result := WrapText(text, 0)

	if len(result) != 0 {
		t.Errorf("Expected empty result for zero width, got %v", result)
	}
}

func TestWrapText_NegativeWidth(t *testing.T) {
	text := "Hello"
	result := WrapText(text, -5)

	if len(result) != 0 {
		t.Errorf("Expected empty result for negative width, got %v", result)
	}
}

func TestWrapText_ExactWidth(t *testing.T) {
	text := "1234567890"
	width := 10
	result := WrapText(text, width)

	if len(result) != 1 {
		t.Errorf("Expected 1 line, got %d", len(result))
	}
	if result[0] != text {
		t.Errorf("Expected %q, got %q", text, result[0])
	}
}

func TestWrapText_MultipleSpaces(t *testing.T) {
	text := "Hello    world"
	width := 20
	result := WrapText(text, width)

	// Should handle multiple spaces (they'll be normalized to single spaces in output)
	for i, line := range result {
		if len(line) > width {
			t.Errorf("Line %d exceeds width: %q", i, line)
		}
	}
}

func TestWrapText_MixedContent(t *testing.T) {
	text := "# CLAUDE.md\n\nThis file provides guidance to Claude Code (claude.ai/code) when working with code in this repository."
	width := 40
	result := WrapText(text, width)

	// All lines should fit
	for i, line := range result {
		if len(line) > width {
			t.Errorf("Line %d exceeds width: %q (len=%d)", i, line, len(line))
		}
	}

	// Should preserve paragraph structure (empty line between sections)
	foundEmpty := false
	for _, line := range result {
		if line == "" {
			foundEmpty = true
			break
		}
	}
	if !foundEmpty {
		t.Errorf("Expected empty line to be preserved")
	}
}

func TestWrapText_ManyLines(t *testing.T) {
	text := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	width := 20
	result := WrapText(text, width)

	// Should preserve all lines (no height truncation)
	if len(result) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(result))
	}

	if result[0] != "Line 1" || result[1] != "Line 2" || result[2] != "Line 3" {
		t.Errorf("Expected all lines preserved, got %v", result)
	}
}

func TestWrapText_LongLinesWrap(t *testing.T) {
	// Long lines that need wrapping
	text := "This is a very long line that needs wrapping\nAnother long line that needs wrapping\nYet another line"
	width := 20
	result := WrapText(text, width)

	// Should wrap all lines
	for i, line := range result {
		if len(line) > width {
			t.Errorf("Line %d exceeds width: %q (len=%d)", i, line, len(line))
		}
	}

	// Should have more than 3 lines due to wrapping
	if len(result) < 3 {
		t.Errorf("Expected at least 3 lines after wrapping, got %d", len(result))
	}
}

func TestWrapLine_SingleWord(t *testing.T) {
	line := "Hello"
	width := 10
	result := wrapLine(line, width)

	if len(result) != 1 {
		t.Errorf("Expected 1 line, got %d", len(result))
	}
	if result[0] != "Hello" {
		t.Errorf("Expected 'Hello', got %q", result[0])
	}
}

func TestWrapLine_MultipleWords(t *testing.T) {
	line := "The quick brown fox"
	width := 10
	result := wrapLine(line, width)

	// Should wrap into multiple lines
	if len(result) < 2 {
		t.Errorf("Expected multiple lines, got %d", len(result))
	}

	for i, l := range result {
		if len(l) > width {
			t.Errorf("Line %d exceeds width: %q", i, l)
		}
	}
}

func TestSplitWords_BasicSplit(t *testing.T) {
	text := "Hello world test"
	words := splitWords(text)

	expected := []string{"Hello", "world", "test"}
	if len(words) != len(expected) {
		t.Errorf("Expected %d words, got %d", len(expected), len(words))
	}

	for i, word := range words {
		if word != expected[i] {
			t.Errorf("Word %d: expected %q, got %q", i, expected[i], word)
		}
	}
}

func TestSplitWords_MultipleSpaces(t *testing.T) {
	text := "Hello    world"
	words := splitWords(text)

	expected := []string{"Hello", "world"}
	if len(words) != len(expected) {
		t.Errorf("Expected %d words, got %d: %v", len(expected), len(words), words)
	}
}

func TestSplitWords_Empty(t *testing.T) {
	text := ""
	words := splitWords(text)

	if len(words) != 0 {
		t.Errorf("Expected 0 words, got %d", len(words))
	}
}
