package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// LeftPaneMsg represents messages that the left pane component handles
type LeftPaneMsg interface {
	isLeftPaneMsg()
}

// Left pane message implementations
type NavigateUpMsg struct{}

func (NavigateUpMsg) isLeftPaneMsg() {}

type NavigateDownMsg struct {
	MaxIndex int // Maximum valid index for bounds checking
}

func (NavigateDownMsg) isLeftPaneMsg() {}

type SelectItemMsg struct {
	Index int
}

func (SelectItemMsg) isLeftPaneMsg() {}

type GoToTopMsg struct{}

func (GoToTopMsg) isLeftPaneMsg() {}

type GoToBottomMsg struct {
	MaxIndex int
}

func (GoToBottomMsg) isLeftPaneMsg() {}

type JumpToIndexMsg struct {
	Index    int
	MaxIndex int
}

func (JumpToIndexMsg) isLeftPaneMsg() {}

type ResizeLeftPaneMsg struct {
	Width  int
	Height int
}

func (ResizeLeftPaneMsg) isLeftPaneMsg() {}

// LeftPaneModel holds the state for the left pane (item list)
type LeftPaneModel struct {
	Cursor   int // Current cursor position
	Selected int // Currently selected item index
	Width    int // Pane width
	Height   int // Pane height
}

// NewLeftPaneModel creates a new left pane model with default values
func NewLeftPaneModel(width, height int) LeftPaneModel {
	return LeftPaneModel{
		Cursor:   0,
		Selected: 0,
		Width:    width,
		Height:   height,
	}
}

// LeftPaneModel implements the Model interface for the left pane
func (l *LeftPaneModel) Update(msg LeftPaneMsg) error {
	switch m := msg.(type) {
	case NavigateUpMsg:
		if l.Cursor > 0 {
			l.Cursor--
			l.Selected = l.Cursor
		}
	case NavigateDownMsg:
		if l.Cursor < m.MaxIndex {
			l.Cursor++
			l.Selected = l.Cursor
		}
	case GoToTopMsg:
		l.Cursor = 0
		l.Selected = 0
	case GoToBottomMsg:
		if m.MaxIndex >= 0 {
			l.Cursor = m.MaxIndex
			l.Selected = m.MaxIndex
		}
	case JumpToIndexMsg:
		if m.Index >= 0 && m.Index <= m.MaxIndex {
			l.Cursor = m.Index
			l.Selected = m.Index
		}
	case SelectItemMsg:
		// Validate index bounds - parent component will validate against item count
		if m.Index >= 0 {
			l.Cursor = m.Index
			l.Selected = m.Index
		}
	case ResizeLeftPaneMsg:
		l.Width = m.Width
		l.Height = m.Height
	}
	return nil
}

// LeftPaneView renders the left pane as a pure function
func LeftPaneView(model LeftPaneModel, items []*StackItem, focused bool) (string, error) {
	borderColor := "62"
	if focused {
		borderColor = "205" // Highlight focused pane
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(0, 1).
		Width(model.Width).
		Height(model.Height - 4).
		Inline(false)

	var content strings.Builder
	title := "Queue"
	if focused {
		title = "● " + title // Active indicator
	}
	content.WriteString(lipgloss.NewStyle().Bold(true).Render(title) + "\n\n")

	for i, item := range items {
		// Calculate available width for preview (account for "N. " prefix and padding)
		availableWidth := model.Width - 6 // Account for borders, padding, and "N. "
		preview := item.Preview

		// Remove any newlines from preview first
		preview = strings.ReplaceAll(preview, "\n", " ")

		// Truncate preview if too long
		if len(preview) > availableWidth {
			preview = preview[:availableWidth-3] + "..."
		}

		line := fmt.Sprintf("%d. %s", i, preview)

		// Ensure the line doesn't exceed the available width
		if len(line) > model.Width-4 { // Account for borders and padding
			line = line[:model.Width-7] + "..."
		}

		if i == model.Cursor {
			line = lipgloss.NewStyle().
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("230")).
				Width(model.Width - 4). // Force width constraint
				Render(line)
		}

		content.WriteString(line + "\n")
	}

	return style.Render(content.String()), nil
}