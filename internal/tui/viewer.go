package tui

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// StringReadSeekCloser wraps a string to implement io.ReadSeekCloser
type StringReadSeekCloser struct {
	content string
	pos     int64
}

func NewStringReadSeekCloser(content string) *StringReadSeekCloser {
	return &StringReadSeekCloser{content: content, pos: 0}
}

func (s *StringReadSeekCloser) Read(p []byte) (n int, err error) {
	if s.pos >= int64(len(s.content)) {
		return 0, io.EOF
	}
	n = copy(p, s.content[s.pos:])
	s.pos += int64(n)
	return n, nil
}

func (s *StringReadSeekCloser) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = s.pos + offset
	case io.SeekEnd:
		newPos = int64(len(s.content)) + offset
	default:
		return 0, fmt.Errorf("invalid whence")
	}
	if newPos < 0 {
		return 0, fmt.Errorf("negative position")
	}
	if newPos > int64(len(s.content)) {
		newPos = int64(len(s.content))
	}
	s.pos = newPos
	return newPos, nil
}

func (s *StringReadSeekCloser) Close() error {
	return nil
}

// StackItem represents an item in the rem queue
type StackItem struct {
	ID             string // Unique identifier for this item
	Content        io.ReadSeekCloser
	Preview        string
	Lines          []string     // cached wrapped lines (viewport window)
	LinesStart     int          // first source line number in cache
	LinesEnd       int          // last source line number in cache
	CachedWidth    int          // width used for cached lines (0 = not cached)
	ViewPos        int          // current view position (line number)
	SearchPattern  string       // current search pattern
	SearchMatches  []int        // line numbers with matches
	SearchIndex    int          // current match index (-1 if no search active)
	SearchLimitHit bool         // true if search stopped at 99 matches
	IsBinary       bool         // true if content is binary
	Size           int64        // size in bytes (useful for binary files)
	SHA256         string       // SHA256 hash (for binary files)
	DeleteFunc     func() error // function to delete this item from persistent storage

	pager *Pager // NEW: Streaming pager for content access
}

// GetFullContent reads the entire content from the ReadSeekCloser
func (q *StackItem) GetFullContent() (string, error) {
	// Save current position
	currentPos, err := q.Content.Seek(0, io.SeekCurrent)
	if err != nil {
		return "", err
	}

	// Read from beginning
	if _, err := q.Content.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	content, err := io.ReadAll(q.Content)
	if err != nil {
		return "", err
	}

	// Restore position
	if _, err := q.Content.Seek(currentPos, io.SeekStart); err != nil {
		return "", err
	}

	return string(content), nil
}

// UpdateWrappedLines recalculates wrapped lines based on width using streaming pager
// Loads only viewport window + buffer for memory efficiency
func (q *StackItem) UpdateWrappedLines(width, height int) error {
	// Check if we need to recalculate
	// Note: When height > LinesEnd, the viewport is larger than the content,
	// so we shouldn't trigger recalc based on the bottom edge check
	needsRecalc := q.CachedWidth != width ||
		q.ViewPos < q.LinesStart ||
		(q.LinesEnd > height && q.ViewPos >= q.LinesEnd-height)

	if !needsRecalc && len(q.Lines) > 0 {
		// Cache is valid
		return nil
	}

	// Handle binary content specially
	if q.IsBinary {
		// Calculate SHA256 lazily if not already done
		if q.SHA256 == "" {
			if err := q.calculateSHA256(); err != nil {
				// If we can't calculate SHA256, just show without it
				q.SHA256 = "Error calculating hash"
			}
		}
		// Create a formatted display for binary files
		q.Lines = q.formatBinaryInfo()
		q.CachedWidth = width
		return nil
	}

	// Initialize pager if not already done
	if q.pager == nil {
		q.pager = NewPager(q.Content)
	}

	// Calculate viewport window: ViewPos ± buffer
	const bufferLines = 50
	windowStart := max(0, q.ViewPos-bufferLines)
	windowEnd := q.ViewPos + height + bufferLines

	// Seek to start of window
	if _, err := q.pager.Seek(0, io.SeekStart); err != nil {
		return err
	}

	// Skip to windowStart line
	for i := 0; i < windowStart; i++ {
		_, err := q.pager.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	// Read and wrap lines in window
	q.Lines = nil
	lineNum := windowStart
	for lineNum < windowEnd {
		sourceLine, err := q.pager.ReadLine()
		if err != nil && err != io.EOF {
			return err
		}

		// Remove trailing newline for wrapping
		sourceLine = strings.TrimSuffix(sourceLine, "\n")

		// Process line if not empty
		if sourceLine != "" {
			// Wrap this source line
			wrappedLines := WrapText(sourceLine, width)
			q.Lines = append(q.Lines, wrappedLines...)
			lineNum++
		}

		// Break after processing if we hit EOF
		if err == io.EOF {
			break
		}
	}

	q.LinesStart = windowStart
	q.LinesEnd = lineNum
	q.CachedWidth = width

	// Re-run search if active
	if q.SearchPattern != "" {
		q.performSearch(q.SearchPattern)
	}

	return nil
}

// formatBinaryInfo creates a formatted display for binary files
func (q *StackItem) formatBinaryInfo() []string {
	lines := []string{
		"",
		"═══════════════════════════════════════",
		"           BINARY FILE",
		"═══════════════════════════════════════",
		"",
	}

	// Add file size
	lines = append(lines, fmt.Sprintf("Size: %s", formatBytes(q.Size)))
	lines = append(lines, "")

	// Add SHA256 hash
	if q.SHA256 != "" {
		lines = append(lines, "SHA256:")
		lines = append(lines, q.SHA256)
		lines = append(lines, "")
	}

	lines = append(lines, "This content cannot be displayed as text.")
	lines = append(lines, "")

	return lines
}

// formatBytes formats byte count into human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d bytes", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// calculateSHA256 computes the SHA256 hash of the content by streaming
func (q *StackItem) calculateSHA256() error {
	// Save current position
	currentPos, err := q.Content.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}

	// Seek to beginning
	if _, err := q.Content.Seek(0, io.SeekStart); err != nil {
		return err
	}

	// Calculate SHA256 by streaming (doesn't load entire file into memory)
	hasher := sha256.New()
	if _, err := io.Copy(hasher, q.Content); err != nil {
		// Restore position before returning error
		q.Content.Seek(currentPos, io.SeekStart)
		return err
	}

	q.SHA256 = hex.EncodeToString(hasher.Sum(nil))

	// Restore original position
	if _, err := q.Content.Seek(currentPos, io.SeekStart); err != nil {
		return err
	}

	return nil
}

// performSearch searches for a regex pattern using streaming, limiting to 99 matches
func (q *StackItem) performSearch(pattern string) error {
	if pattern == "" {
		q.SearchPattern = ""
		q.SearchMatches = nil
		q.SearchIndex = -1
		q.SearchLimitHit = false
		return nil
	}

	regex, err := regexp.Compile("(?i)" + pattern) // Case-insensitive by default
	if err != nil {
		return err
	}

	q.SearchPattern = pattern
	q.SearchMatches = nil
	q.SearchIndex = -1
	q.SearchLimitHit = false

	// Use default width if CachedWidth not set
	wrapWidth := q.CachedWidth
	if wrapWidth == 0 {
		wrapWidth = 80 // Default width for search without wrapping calculation
	}

	// Initialize pager for searching
	if q.pager == nil {
		q.pager = NewPager(q.Content)
	}

	// Seek to beginning
	if _, err := q.pager.Seek(0, io.SeekStart); err != nil {
		return err
	}

	// Stream through file, matching lines (limit to 99 matches)
	const maxMatches = 99
	wrappedLineNum := 0

	for {
		sourceLine, err := q.pager.ReadLine()
		if err != nil && err != io.EOF {
			return err
		}

		// Remove trailing newline for matching
		sourceLine = strings.TrimSuffix(sourceLine, "\n")

		// Process line if not empty
		if sourceLine != "" {
			// Check if this source line matches
			if regex.MatchString(sourceLine) {
				// Add all wrapped line numbers for this source line
				wrappedLines := WrapText(sourceLine, wrapWidth)
				for i := range wrappedLines {
					q.SearchMatches = append(q.SearchMatches, wrappedLineNum+i)

					// Check if we hit the limit
					if len(q.SearchMatches) >= maxMatches {
						q.SearchLimitHit = true
						goto done
					}
				}
			}

			// Count wrapped lines
			wrappedLines := WrapText(sourceLine, wrapWidth)
			wrappedLineNum += len(wrappedLines)
		}

		// Break after processing if we hit EOF
		if err == io.EOF {
			break
		}
	}

done:
	// Set to first match if any found
	if len(q.SearchMatches) > 0 {
		q.SearchIndex = 0
	}

	return nil
}

// NextMatch moves to the next search match
func (q *StackItem) NextMatch() bool {
	if len(q.SearchMatches) == 0 {
		return false
	}

	q.SearchIndex = (q.SearchIndex + 1) % len(q.SearchMatches)
	return true
}

// PrevMatch moves to the previous search match
func (q *StackItem) PrevMatch() bool {
	if len(q.SearchMatches) == 0 {
		return false
	}

	q.SearchIndex = (q.SearchIndex - 1 + len(q.SearchMatches)) % len(q.SearchMatches)
	return true
}

// GetCurrentMatchLine returns the line number of the current match
func (q *StackItem) GetCurrentMatchLine() int {
	if q.SearchIndex >= 0 && q.SearchIndex < len(q.SearchMatches) {
		return q.SearchMatches[q.SearchIndex]
	}
	return -1
}

// ClearSearch clears the current search
func (q *StackItem) ClearSearch() {
	q.SearchPattern = ""
	q.SearchMatches = nil
	q.SearchIndex = -1
}

type Model struct {
	// Legacy fields maintained for backward compatibility
	cursor      int
	selected    int
	width       int
	height      int
	leftWidth   int
	rightWidth  int
	focusedPane int    // 0 = left, 1 = right
	searchMode  bool   // true when entering search pattern
	searchInput string // current search input
	searchError string // search error message
	items       []*StackItem

	// New Elm architecture model
	app *AppModel
}

func NewModel(items []*StackItem) Model {
	app := NewAppModel(items)

	return Model{
		cursor:      0,
		selected:    0,
		leftWidth:   25,
		rightWidth:  50, // Will be recalculated on resize
		focusedPane: 0,  // Start with left pane focused
		items:       items,
		app:         &app,
	}
}

// UpdateMockSize is a helper method for testing that simulates a window resize
func (m *Model) UpdateMockSize(width, height int) {
	// Update legacy fields for compatibility
	m.width = width
	m.height = height
	m.rightWidth = width - m.leftWidth - 3 // Account for borders and spacing
	if m.rightWidth < 20 {
		m.rightWidth = 20
	}

	// Delegate to app model
	newModel, _ := m.app.Update(tea.WindowSizeMsg{Width: width, Height: height})
	m.app = newModel.(*AppModel)

	// Sync legacy fields from app component
	m.syncFromApp()
}

func (m Model) Init() tea.Cmd {
	return m.app.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Delegate to app model
	newAppModel, cmd := m.app.Update(msg)
	m.app = newAppModel.(*AppModel)

	// Sync legacy fields from app model
	m.syncFromApp()

	return m, cmd
}

func (m Model) View() string {
	// Delegate to app component
	return m.app.View()
}

// syncFromApp updates legacy fields from the app model state for backward compatibility
func (m *Model) syncFromApp() {
	// Sync cursor and selection
	m.cursor = m.app.LeftPane.Cursor
	m.selected = m.app.LeftPane.Selected

	// Sync active pane (convert PaneType to int)
	if m.app.ActivePane == LeftPane {
		m.focusedPane = 0
	} else {
		m.focusedPane = 1
	}

	// Sync search state
	m.searchMode = m.app.Search.IsActive()
	m.searchInput = m.app.Search.GetInput()
	m.searchError = m.app.Search.GetError()

	// Sync dimensions
	m.width = m.app.Width
	m.height = m.app.Height
	m.leftWidth = m.app.LeftWidth
	m.rightWidth = m.app.RightWidth
}

// Legacy type alias for backward compatibility
type QueueItem = StackItem

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	var result strings.Builder

	for lineIndex, line := range lines {
		if lineIndex > 0 {
			result.WriteString("\n")
		}

		// If line is empty, just add it
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}

		// If line fits within width, add it as-is
		if len(line) <= width {
			result.WriteString(line)
			continue
		}

		// Line is too long, need to wrap
		words := strings.Fields(line)
		if len(words) == 0 {
			continue
		}

		var currentLine strings.Builder
		currentLength := 0

		for _, word := range words {
			wordLength := len(word)

			// If the word itself is longer than width, break it up
			if wordLength > width {
				// First, add current line if it has content
				if currentLength > 0 {
					result.WriteString(currentLine.String() + "\n")
					currentLine.Reset()
					currentLength = 0
				}

				// Break up the long word
				for len(word) > width {
					result.WriteString(word[:width] + "\n")
					word = word[width:]
				}
				if len(word) > 0 {
					currentLine.WriteString(word)
					currentLength = len(word)
				}
				continue
			}

			// If adding this word would exceed the width, start a new line
			if currentLength > 0 && currentLength+1+wordLength > width {
				result.WriteString(currentLine.String() + "\n")
				currentLine.Reset()
				currentLength = 0
			}

			// Add word to current line
			if currentLength > 0 {
				currentLine.WriteString(" ")
				currentLength++
			}
			currentLine.WriteString(word)
			currentLength += wordLength
		}

		// Add the remaining content in currentLine
		if currentLength > 0 {
			result.WriteString(currentLine.String())
		}
	}

	return result.String()
}
