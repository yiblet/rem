package tui

import (
	"io"
	"strings"
	"testing"
)

func TestPager_ReadLine(t *testing.T) {
	content := "Line 1\nLine 2\nLine 3\n"
	reader := strings.NewReader(content)
	pager := NewPager(reader)

	// Read first line
	line, err := pager.ReadLine()
	if err != nil {
		t.Fatalf("Failed to read first line: %v", err)
	}
	if line != "Line 1\n" {
		t.Errorf("Expected 'Line 1\\n', got %q", line)
	}

	// Read second line
	line, err = pager.ReadLine()
	if err != nil {
		t.Fatalf("Failed to read second line: %v", err)
	}
	if line != "Line 2\n" {
		t.Errorf("Expected 'Line 2\\n', got %q", line)
	}

	// Read third line
	line, err = pager.ReadLine()
	if err != nil {
		t.Fatalf("Failed to read third line: %v", err)
	}
	if line != "Line 3\n" {
		t.Errorf("Expected 'Line 3\\n', got %q", line)
	}

	// Read EOF
	_, err = pager.ReadLine()
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}
}

func TestPager_Seek(t *testing.T) {
	content := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	reader := strings.NewReader(content)
	pager := NewPager(reader)

	// Read first 5 bytes
	buf := make([]byte, 5)
	n, err := pager.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	if n != 5 || string(buf) != "ABCDE" {
		t.Errorf("Expected 'ABCDE', got %q", string(buf))
	}

	// Seek to beginning
	pos, err := pager.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("Failed to seek: %v", err)
	}
	if pos != 0 {
		t.Errorf("Expected position 0, got %d", pos)
	}

	// Read again from beginning
	n, err = pager.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read after seek: %v", err)
	}
	if n != 5 || string(buf) != "ABCDE" {
		t.Errorf("Expected 'ABCDE' after seek, got %q", string(buf))
	}

	// Seek to middle
	pos, err = pager.Seek(10, io.SeekStart)
	if err != nil {
		t.Fatalf("Failed to seek to middle: %v", err)
	}
	if pos != 10 {
		t.Errorf("Expected position 10, got %d", pos)
	}

	// Read from middle
	n, err = pager.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from middle: %v", err)
	}
	if n != 5 || string(buf) != "KLMNO" {
		t.Errorf("Expected 'KLMNO' from middle, got %q", string(buf))
	}
}

func TestPager_ReadRune(t *testing.T) {
	content := "Hello, 世界!"
	reader := strings.NewReader(content)
	pager := NewPager(reader)

	// Read 'H'
	r, size, err := pager.ReadRune()
	if err != nil {
		t.Fatalf("Failed to read rune: %v", err)
	}
	if r != 'H' || size != 1 {
		t.Errorf("Expected 'H' (size 1), got %q (size %d)", r, size)
	}

	// Skip to multibyte character
	pager.Seek(7, io.SeekStart)

	// Read '世'
	r, size, err = pager.ReadRune()
	if err != nil {
		t.Fatalf("Failed to read multibyte rune: %v", err)
	}
	if r != '世' || size != 3 {
		t.Errorf("Expected '世' (size 3), got %q (size %d)", r, size)
	}
}

func TestPager_EmptyContent(t *testing.T) {
	content := ""
	reader := strings.NewReader(content)
	pager := NewPager(reader)

	// Try to read from empty pager
	_, err := pager.ReadLine()
	if err != io.EOF {
		t.Errorf("Expected EOF for empty content, got %v", err)
	}
}

func TestPager_SingleLine(t *testing.T) {
	content := "Single line without newline"
	reader := strings.NewReader(content)
	pager := NewPager(reader)

	// Read line (should get EOF but also the content)
	line, err := pager.ReadLine()
	if err != io.EOF {
		t.Errorf("Expected EOF for line without newline, got %v", err)
	}
	if line != content {
		t.Errorf("Expected %q, got %q", content, line)
	}
}

func TestPager_SeekAndReadLine(t *testing.T) {
	content := "Line 1\nLine 2\nLine 3\n"
	reader := strings.NewReader(content)
	pager := NewPager(reader)

	// Read first line
	line, _ := pager.ReadLine()
	if line != "Line 1\n" {
		t.Errorf("Expected 'Line 1\\n', got %q", line)
	}

	// Seek back to start
	pager.Seek(0, io.SeekStart)

	// Read first line again
	line, _ = pager.ReadLine()
	if line != "Line 1\n" {
		t.Errorf("Expected 'Line 1\\n' after seek, got %q", line)
	}

	// Seek to second line (position 7)
	pager.Seek(7, io.SeekStart)

	// Read second line
	line, _ = pager.ReadLine()
	if line != "Line 2\n" {
		t.Errorf("Expected 'Line 2\\n', got %q", line)
	}
}

func TestPager_Close(t *testing.T) {
	content := "Test content"
	reader := strings.NewReader(content)
	pager := NewPager(reader)

	// Close should not error for strings.Reader (no Close method)
	err := pager.Close()
	if err != nil {
		t.Errorf("Expected no error on close, got %v", err)
	}
}

func TestPager_ReadAllThenSeekAndReadAgain(t *testing.T) {
	content := "Line 1\nLine 2\nLine 3\n"
	reader := strings.NewReader(content)
	pager := NewPager(reader)

	// Read all lines until EOF
	lines := []string{}
	for {
		line, err := pager.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read line: %v", err)
		}
		lines = append(lines, line)
	}

	if len(lines) != 3 {
		t.Errorf("Expected 3 lines on first read, got %d", len(lines))
	}

	// Seek back to beginning
	_, err := pager.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("Failed to seek to beginning: %v", err)
	}

	// Read all lines again
	lines = []string{}
	for {
		line, err := pager.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read line on second pass: %v", err)
		}
		lines = append(lines, line)
	}

	if len(lines) != 3 {
		t.Errorf("Expected 3 lines on second read after seek, got %d", len(lines))
	}

	// Verify content
	if lines[0] != "Line 1\n" || lines[1] != "Line 2\n" || lines[2] != "Line 3\n" {
		t.Errorf("Content mismatch on second read: %v", lines)
	}
}