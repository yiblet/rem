package tui

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// QueueItem represents an item in the rem queue
type QueueItem struct {
	Content       io.ReadSeekCloser
	Preview       string
	Lines         []string // cached wrapped lines
	ViewPos       int      // current view position (line number)
	SearchPattern string   // current search pattern
	SearchMatches []int    // line numbers with matches
	SearchIndex   int      // current match index (-1 if no search active)
}

// GetFullContent reads the entire content from the ReadSeekCloser
func (q *QueueItem) GetFullContent() (string, error) {
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

// UpdateWrappedLines recalculates wrapped lines based on width
func (q *QueueItem) UpdateWrappedLines(width int) error {
	content, err := q.GetFullContent()
	if err != nil {
		return err
	}

	q.Lines = strings.Split(wrapText(content, width), "\n")

	// Re-run search if we have an active pattern
	if q.SearchPattern != "" {
		q.performSearch(q.SearchPattern)
	}

	return nil
}

// performSearch searches for a regex pattern and populates SearchMatches
func (q *QueueItem) performSearch(pattern string) error {
	if pattern == "" {
		q.SearchPattern = ""
		q.SearchMatches = nil
		q.SearchIndex = -1
		return nil
	}

	regex, err := regexp.Compile("(?i)" + pattern) // Case-insensitive by default
	if err != nil {
		return err
	}

	q.SearchPattern = pattern
	q.SearchMatches = nil
	q.SearchIndex = -1

	// Search through all lines
	for lineNum, line := range q.Lines {
		if regex.MatchString(line) {
			q.SearchMatches = append(q.SearchMatches, lineNum)
		}
	}

	// If we found matches, set to first match
	if len(q.SearchMatches) > 0 {
		q.SearchIndex = 0
	}

	return nil
}

// NextMatch moves to the next search match
func (q *QueueItem) NextMatch() bool {
	if len(q.SearchMatches) == 0 {
		return false
	}

	q.SearchIndex = (q.SearchIndex + 1) % len(q.SearchMatches)
	return true
}

// PrevMatch moves to the previous search match
func (q *QueueItem) PrevMatch() bool {
	if len(q.SearchMatches) == 0 {
		return false
	}

	q.SearchIndex = (q.SearchIndex - 1 + len(q.SearchMatches)) % len(q.SearchMatches)
	return true
}

// GetCurrentMatchLine returns the line number of the current match
func (q *QueueItem) GetCurrentMatchLine() int {
	if q.SearchIndex >= 0 && q.SearchIndex < len(q.SearchMatches) {
		return q.SearchMatches[q.SearchIndex]
	}
	return -1
}

// ClearSearch clears the current search
func (q *QueueItem) ClearSearch() {
	q.SearchPattern = ""
	q.SearchMatches = nil
	q.SearchIndex = -1
}

type Model struct {
	cursor        int
	selected      int
	width         int
	height        int
	leftWidth     int
	rightWidth    int
	focusedPane   int    // 0 = left, 1 = right
	searchMode    bool   // true when entering search pattern
	searchInput   string // current search input
	searchError   string // search error message
	items         []*QueueItem
}

func NewModel(items []*QueueItem) Model {
	return Model{
		cursor:      0,
		selected:    0,
		leftWidth:   25,
		rightWidth:  50, // Will be recalculated on resize
		focusedPane: 0,  // Start with left pane focused
		items:       items,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.rightWidth = msg.Width - m.leftWidth - 3 // Account for borders and spacing
		if m.rightWidth < 20 {
			m.rightWidth = 20
		}
		// Recalculate wrapped lines for current item when window size changes
		if m.selected < len(m.items) {
			m.items[m.selected].UpdateWrappedLines(m.rightWidth - 6)
		}

	case tea.KeyMsg:
		key := msg.String()

		// Handle search mode first before any other key processing
		if m.searchMode {
			switch key {
			case "enter":
				// Execute search
				if m.selected < len(m.items) {
					item := m.items[m.selected]
					if err := item.performSearch(m.searchInput); err != nil {
						m.searchError = err.Error()
					} else {
						m.searchError = ""
						// Jump to first match if found
						if matchLine := item.GetCurrentMatchLine(); matchLine >= 0 {
							// Center the match on screen
							availableHeight := m.height - 6
							item.ViewPos = max(0, matchLine-availableHeight/2)
							item.ViewPos = min(item.ViewPos, m.getMaxScroll(item))
						}
					}
				}
				m.searchMode = false
			case "esc":
				// Cancel search
				m.searchMode = false
				m.searchInput = ""
				m.searchError = ""
			case "backspace", "ctrl+h":
				// Remove last character
				if len(m.searchInput) > 0 {
					m.searchInput = m.searchInput[:len(m.searchInput)-1]
				}
			default:
				// Add character to search input (only printable characters)
				if len(key) == 1 && key[0] >= 32 && key[0] <= 126 {
					m.searchInput += key
				}
			}
		} else {
			// Handle normal mode keys
			switch key {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "esc":
				// Only quit if not in search mode
				return m, tea.Quit

			case "tab":
				// Toggle between left and right pane
				m.focusedPane = (m.focusedPane + 1) % 2

			default:
				// Handle pane-specific controls
				if m.focusedPane == 0 {
					// Left pane controls
					switch key {
					case "up", "k":
						if m.cursor > 0 {
							m.cursor--
							m.selected = m.cursor
							// Update wrapped lines for newly selected item
							if m.selected < len(m.items) {
								m.items[m.selected].UpdateWrappedLines(m.rightWidth - 6)
							}
						}
					case "down", "j":
						if m.cursor < len(m.items)-1 {
							m.cursor++
							m.selected = m.cursor
							// Update wrapped lines for newly selected item
							if m.selected < len(m.items) {
								m.items[m.selected].UpdateWrappedLines(m.rightWidth - 6)
							}
						}
					}
				} else {
					// Right pane controls (pager-like)
					if m.selected < len(m.items) {
						item := m.items[m.selected]
						maxScroll := m.getMaxScroll(item)

						// Handle search-related keys first
						switch key {
						case "/", "?":
							// Start search mode (forward or backward)
							m.searchMode = true
							m.searchInput = ""
							m.searchError = ""
						case "n":
							// Next search match
							if item.NextMatch() {
								if matchLine := item.GetCurrentMatchLine(); matchLine >= 0 {
									// Center the match on screen
									availableHeight := m.height - 6
									item.ViewPos = max(0, matchLine-availableHeight/2)
									item.ViewPos = min(item.ViewPos, maxScroll)
								}
							}
						case "N":
							// Previous search match
							if item.PrevMatch() {
								if matchLine := item.GetCurrentMatchLine(); matchLine >= 0 {
									// Center the match on screen
									availableHeight := m.height - 6
									item.ViewPos = max(0, matchLine-availableHeight/2)
									item.ViewPos = min(item.ViewPos, maxScroll)
								}
							}
						default:
							// Handle number prefixes (e.g., "10j", "5k")
							if matched, _ := regexp.MatchString(`^\d+[jk]$`, key); matched {
								numStr := key[:len(key)-1]
								direction := key[len(key)-1:]
								if num, err := strconv.Atoi(numStr); err == nil {
									if direction == "j" {
										item.ViewPos = min(item.ViewPos+num, maxScroll)
									} else if direction == "k" {
										item.ViewPos = max(item.ViewPos-num, 0)
									}
								}
							} else {
								switch key {
								case "up", "k":
									if item.ViewPos > 0 {
										item.ViewPos--
									}
								case "down", "j":
									if item.ViewPos < maxScroll {
										item.ViewPos++
									}
								case "g":
									item.ViewPos = 0 // Go to top
								case "G":
									item.ViewPos = maxScroll // Go to bottom
								case "ctrl+u":
									// Page up (half page)
									pageSize := (m.height - 6) / 2
									item.ViewPos = max(item.ViewPos-pageSize, 0)
								case "ctrl+d":
									// Page down (half page)
									pageSize := (m.height - 6) / 2
									item.ViewPos = min(item.ViewPos+pageSize, maxScroll)
								case "ctrl+b":
									// Full page up
									pageSize := m.height - 6
									item.ViewPos = max(item.ViewPos-pageSize, 0)
								case "ctrl+f":
									// Full page down
									pageSize := m.height - 6
									item.ViewPos = min(item.ViewPos+pageSize, maxScroll)
								}
							}
						}
					}
				}
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	leftPane := m.renderLeftPane()
	rightPane := m.renderRightPane()

	// Join left and right panes side by side
	leftLines := strings.Split(leftPane, "\n")
	rightLines := strings.Split(rightPane, "\n")

	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}

	var result strings.Builder
	for i := 0; i < maxLines; i++ {
		leftLine := ""
		rightLine := ""

		if i < len(leftLines) {
			leftLine = leftLines[i]
		}
		if i < len(rightLines) {
			rightLine = rightLines[i]
		}

		// Pad left line to ensure consistent spacing
		for len(leftLine) < m.leftWidth {
			leftLine += " "
		}

		result.WriteString(leftLine + " " + rightLine + "\n")
	}

	// Add status line at bottom
	var statusLine string
	if m.searchMode {
		// Show search input with cursor
		statusLine = fmt.Sprintf("/%s", m.searchInput)
		if len(m.searchError) > 0 {
			statusLine += fmt.Sprintf(" (Error: %s)", m.searchError)
		} else {
			statusLine += " (Enter to search, Esc to cancel)"
		}
	} else if m.selected < len(m.items) && len(m.items[m.selected].SearchMatches) > 0 {
		// Show search results
		item := m.items[m.selected]
		currentMatch := item.SearchIndex + 1
		totalMatches := len(item.SearchMatches)
		statusLine = fmt.Sprintf("Pattern: %s - Match %d of %d", item.SearchPattern, currentMatch, totalMatches)
	} else {
		// Show help text
		if m.focusedPane == 0 {
			statusLine = "Left Pane: ↑/k ↓/j (navigate & view) Tab (switch) q (quit)"
		} else {
			statusLine = "Right Pane: ↑/k ↓/j (scroll) / (search) n/N (next/prev) g/G (top/bottom) ##j/##k (jump) q (quit)"
		}
	}

	statusStyle := lipgloss.NewStyle().
		Width(m.width)

	result.WriteString("\n" + statusStyle.Render(statusLine))

	return result.String()
}

func (m Model) renderLeftPane() string {
	borderColor := "62"
	if m.focusedPane == 0 {
		borderColor = "205" // Highlight focused pane
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(0, 1).
		Width(m.leftWidth - 2).
		Height(m.height - 4).
		MaxWidth(m.leftWidth - 2). // Prevent any overflow
		Inline(false)

	var content strings.Builder
	title := "Queue"
	if m.focusedPane == 0 {
		title = "● " + title // Active indicator
	}
	content.WriteString(lipgloss.NewStyle().Bold(true).Render(title) + "\n\n")

	for i, item := range m.items {
		// Calculate available width for preview (account for "N. " prefix and padding)
		availableWidth := m.leftWidth - 6 // Account for borders, padding, and "N. "
		preview := item.Preview

		// Remove any newlines from preview first
		preview = strings.ReplaceAll(preview, "\n", " ")

		// Truncate preview if too long
		if len(preview) > availableWidth {
			preview = preview[:availableWidth-3] + "..."
		}

		line := fmt.Sprintf("%d. %s", i, preview)

		// Ensure the line doesn't exceed the available width
		if len(line) > m.leftWidth-4 { // Account for borders and padding
			line = line[:m.leftWidth-7] + "..."
		}

		if i == m.cursor {
			line = lipgloss.NewStyle().
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("230")).
				Width(m.leftWidth-4). // Force width constraint
				Render(line)
		}

		content.WriteString(line + "\n")
	}

	return style.Render(content.String())
}

func (m Model) renderRightPane() string {
	borderColor := "62"
	if m.focusedPane == 1 {
		borderColor = "205" // Highlight focused pane
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(0, 1).
		Width(m.rightWidth - 2).
		Height(m.height - 4)

	var content strings.Builder

	if m.selected < 0 || m.selected >= len(m.items) {
		content.WriteString(lipgloss.NewStyle().Bold(true).Render("Content") + "\n\n")
		content.WriteString("Invalid selection")
	} else {
		item := m.items[m.selected]

		// Build title with scroll indicator
		title := fmt.Sprintf("Content [%d]", m.selected)
		if m.focusedPane == 1 {
			title = "● " + title // Active indicator
		}

		// Ensure lines are calculated
		if len(item.Lines) == 0 {
			item.UpdateWrappedLines(m.rightWidth - 6)
		}

		if len(item.Lines) > 0 && m.getMaxScroll(item) > 0 {
			// Add scroll position indicator
			totalLines := len(item.Lines)
			topLine := item.ViewPos + 1
			bottomLine := min(item.ViewPos+(m.height-6), totalLines)
			title += fmt.Sprintf(" (%d-%d/%d)", topLine, bottomLine, totalLines)
		}
		content.WriteString(lipgloss.NewStyle().Bold(true).Render(title) + "\n\n")

		// Show the visible portion based on view position
		availableHeight := m.height - 6 // Account for borders and headers
		startLine := item.ViewPos
		endLine := min(startLine+availableHeight, len(item.Lines))

		// Create a set of match lines for quick lookup
		matchLines := make(map[int]bool)
		for _, matchLine := range item.SearchMatches {
			matchLines[matchLine] = true
		}

		for i := startLine; i < endLine; i++ {
			if i < len(item.Lines) {
				line := item.Lines[i]

				// Highlight search matches
				if matchLines[i] && item.SearchPattern != "" {
					// Compile regex for highlighting
					if regex, err := regexp.Compile("(?i)" + item.SearchPattern); err == nil {
						// Find all matches in the line
						matches := regex.FindAllStringIndex(line, -1)
						if len(matches) > 0 {
							var highlightedLine strings.Builder
							lastEnd := 0

							for _, match := range matches {
								// Add text before match
								highlightedLine.WriteString(line[lastEnd:match[0]])

								// Add highlighted match
								matchText := line[match[0]:match[1]]
								if i == item.GetCurrentMatchLine() {
									// Current match - use different highlighting
									highlightedLine.WriteString(lipgloss.NewStyle().
										Background(lipgloss.Color("220")).
										Foreground(lipgloss.Color("0")).
										Render(matchText))
								} else {
									// Other matches
									highlightedLine.WriteString(lipgloss.NewStyle().
										Background(lipgloss.Color("11")).
										Foreground(lipgloss.Color("0")).
										Render(matchText))
								}

								lastEnd = match[1]
							}

							// Add remaining text after last match
							highlightedLine.WriteString(line[lastEnd:])
							line = highlightedLine.String()
						}
					}
				}

				content.WriteString(line + "\n")
			}
		}
	}

	return style.Render(content.String())
}

func (m Model) getMaxScroll(item *QueueItem) int {
	availableHeight := m.height - 6 // Account for borders and headers
	if len(item.Lines) <= availableHeight {
		return 0
	}
	return len(item.Lines) - availableHeight
}

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