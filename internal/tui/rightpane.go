package tui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RightPaneMsg represents messages that the right pane component handles
type RightPaneMsg interface {
	isRightPaneMsg()
}

// Right pane message implementations
type ScrollUpMsg struct{}

func (ScrollUpMsg) isRightPaneMsg() {}

type ScrollDownMsg struct {
	MaxScroll int
}

func (ScrollDownMsg) isRightPaneMsg() {}

type ScrollToTopMsg struct{}

func (ScrollToTopMsg) isRightPaneMsg() {}

type ScrollToBottomMsg struct {
	MaxScroll int
}

func (ScrollToBottomMsg) isRightPaneMsg() {}

type PageUpMsg struct{}

func (PageUpMsg) isRightPaneMsg() {}

type PageDownMsg struct {
	MaxScroll int
}

func (PageDownMsg) isRightPaneMsg() {}

type JumpMsg struct {
	Direction string // "j" for down, "k" for up
	Lines     int
	MaxScroll int
}

func (JumpMsg) isRightPaneMsg() {}

type ResizeRightPaneMsg struct {
	Width  int
	Height int
}

func (ResizeRightPaneMsg) isRightPaneMsg() {}

type UpdateContentMsg struct {
	// Content will be passed to view functions, not stored in model
}

func (UpdateContentMsg) isRightPaneMsg() {}

// RightPaneModel holds the state for the right pane (content viewer)
type RightPaneModel struct {
	Width   int // Pane width
	Height  int // Pane height
	ViewPos int // Current view position (line number)
}

// NewRightPaneModel creates a new right pane model with default values
func NewRightPaneModel(width, height int) RightPaneModel {
	return RightPaneModel{
		Width:   width,
		Height:  height,
		ViewPos: 0,
	}
}

// RightPaneModel implements the Model interface for the right pane
func (r *RightPaneModel) Update(msg RightPaneMsg) error {
	switch m := msg.(type) {
	case ScrollUpMsg:
		if r.ViewPos > 0 {
			r.ViewPos--
		}
	case ScrollDownMsg:
		if r.ViewPos < m.MaxScroll {
			r.ViewPos++
		}
	case ScrollToTopMsg:
		r.ViewPos = 0
	case ScrollToBottomMsg:
		r.ViewPos = m.MaxScroll
	case PageUpMsg:
		// Page up (half page)
		pageSize := (r.Height - 6) / 2
		r.ViewPos = max(r.ViewPos-pageSize, 0)
	case PageDownMsg:
		// Page down (half page)
		pageSize := (r.Height - 6) / 2
		r.ViewPos = min(r.ViewPos+pageSize, m.MaxScroll)
	case JumpMsg:
		switch m.Direction {
		case "j": // down
			r.ViewPos = min(r.ViewPos+m.Lines, m.MaxScroll)
		case "k": // up
			r.ViewPos = max(r.ViewPos-m.Lines, 0)
		}
	case ResizeRightPaneMsg:
		r.Width = m.Width
		r.Height = m.Height
	case UpdateContentMsg:
		r.ViewPos = 0 // Reset view position when content changes
	}
	return nil
}

// RightPaneView renders the right pane as a pure function
func RightPaneView(model RightPaneModel, content *StackItem, searchModel SearchModel, focused bool, selectedIndex int) (string, error) {
	borderColor := "62"
	if focused {
		borderColor = "205" // Highlight focused pane
		if searchModel.IsActive() {
			// Show the border in yellow when in search mode
			borderColor = "220"
		}
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(0, 1).
		Width(model.Width - 2).
		Height(model.Height - 4)

	var contentBuilder strings.Builder

	if content == nil {
		contentBuilder.WriteString(lipgloss.NewStyle().Bold(true).Render("Content") + "\n\n")
		contentBuilder.WriteString("No content selected")
	} else {
		// Build title with item title (Preview contains the title)
		title := fmt.Sprintf("Content [%d]", selectedIndex)
		if focused {
			title = "● " + title // Active indicator
		}

		// Add item title if available (truncate to fit available width)
		if content.Preview != "" {
			maxTitleWidth := model.Width - 20 // Account for borders, padding, and Content [N] text
			itemTitle := content.Preview
			if len(itemTitle) > maxTitleWidth {
				itemTitle = itemTitle[:maxTitleWidth-3] + "..."
			}
			title += ": " + itemTitle
		}

		// Calculate available height once
		availableHeight := model.Height - 6 // Account for borders and headers

		// Ensure lines are wrapped for current width
		// UpdateWrappedLines is smart - it only recalculates if width changed
		content.UpdateWrappedLines(model.Width-6, availableHeight)

		maxScroll := getMaxScroll(model, content)
		if len(content.Lines) > 0 && maxScroll > 0 {
			// Add scroll position indicator
			totalLines := len(content.Lines)
			topLine := model.ViewPos + 1
			bottomLine := min(model.ViewPos+availableHeight, totalLines)
			title += fmt.Sprintf(" (%d-%d/%d)", topLine, bottomLine, totalLines)
		}
		contentBuilder.WriteString(lipgloss.NewStyle().Bold(true).Render(title) + "\n\n")

		// Show the visible portion based on view position
		startLine := model.ViewPos
		endLine := min(startLine+availableHeight, len(content.Lines))

		// Create a set of match lines for quick lookup
		matchLines := make(map[int]bool)
		for _, matchLine := range searchModel.GetMatches() {
			matchLines[matchLine] = true
		}

		for i := startLine; i < endLine; i++ {
			if i < len(content.Lines) {
				line := content.Lines[i]

				// Highlight search matches
				if matchLines[i] && searchModel.GetPattern() != "" {
					line = highlightSearchMatches(line, searchModel.GetPattern(), i == searchModel.GetCurrentMatchLine())
				}

				contentBuilder.WriteString(line + "\n")
			}
		}
	}

	contentStr := strings.TrimSuffix(contentBuilder.String(), "\n")
	return style.Render(contentStr), nil
}

// highlightSearchMatches highlights search matches in a line (pure function)
func highlightSearchMatches(line, pattern string, isCurrentMatch bool) string {
	// Compile regex for highlighting
	regex, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return line // Return original line if regex fails
	}

	// Find all matches in the line
	matches := regex.FindAllStringIndex(line, -1)
	if len(matches) == 0 {
		return line
	}

	var highlightedLine strings.Builder
	lastEnd := 0

	for _, match := range matches {
		// Add text before match
		highlightedLine.WriteString(line[lastEnd:match[0]])

		// Add highlighted match
		matchText := line[match[0]:match[1]]
		if isCurrentMatch {
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
	return highlightedLine.String()
}

// getMaxScroll returns the maximum scroll position (pure function)
func getMaxScroll(model RightPaneModel, content *StackItem) int {
	if content == nil {
		return 0
	}
	availableHeight := model.Height - 6 // Account for borders and headers
	if len(content.Lines) <= availableHeight {
		return 0
	}
	return len(content.Lines) - availableHeight
}

// scrollToMatch calculates the view position to center a match line (pure function)
func scrollToMatch(model RightPaneModel, content *StackItem, matchLine int) int {
	availableHeight := model.Height - 6
	newViewPos := max(0, matchLine-availableHeight/2)
	maxScroll := getMaxScroll(model, content)
	return min(newViewPos, maxScroll)
}

// parseJumpCommand parses jump commands like "10j", "5k" (pure function)
func parseJumpCommand(command string) (string, int, error) {
	matched, err := regexp.MatchString(`^\d+[jk]$`, command)
	if err != nil {
		return "", 0, err
	}
	if !matched {
		return "", 0, fmt.Errorf("invalid jump command: %s", command)
	}

	numStr := command[:len(command)-1]
	direction := command[len(command)-1:]

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return "", 0, err
	}

	return direction, num, nil
}
