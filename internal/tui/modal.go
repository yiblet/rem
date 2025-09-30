package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ModalMsg represents messages that the modal component handles
type ModalMsg interface {
	isModalMsg()
}

// Modal message implementations
type ShowModalMsg struct {
	Title   string
	Content string
	Options string
}

func (ShowModalMsg) isModalMsg() {}

type HideModalMsg struct{}

func (HideModalMsg) isModalMsg() {}

// ModalModel holds the state for modal dialogs
type ModalModel struct {
	Active  bool
	Title   string
	Content string
	Options string
	Width   int
	Height  int
}

// NewModalModel creates a new modal model
func NewModalModel() ModalModel {
	return ModalModel{
		Active: false,
		Width:  60,
		Height: 10,
	}
}

// Update handles modal messages
func (m *ModalModel) Update(msg ModalMsg) error {
	switch msg := msg.(type) {
	case ShowModalMsg:
		m.Active = true
		m.Title = msg.Title
		m.Content = msg.Content
		m.Options = msg.Options
	case HideModalMsg:
		m.Active = false
		m.Title = ""
		m.Content = ""
		m.Options = ""
	}
	return nil
}

// ModalView renders the modal as a pure function
func ModalView(model ModalModel, backgroundView string, windowWidth, windowHeight int) string {
	if !model.Active {
		return backgroundView
	}

	// Build modal content
	modalContent := model.Title
	if model.Content != "" {
		modalContent += "\n\n" + model.Content
	}
	if model.Options != "" {
		modalContent += "\n\n" + model.Options
	}

	// Calculate modal dimensions
	modalWidth := model.Width
	modalHeight := model.Height

	// Ensure modal fits within window
	if modalWidth > windowWidth-4 {
		modalWidth = windowWidth - 4
	}
	if modalHeight > windowHeight-4 {
		modalHeight = windowHeight - 4
	}

	// Create modal style with border
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("9")). // Bright red border
		Padding(1, 2).
		Width(modalWidth).
		Height(modalHeight).
		Align(lipgloss.Center, lipgloss.Center)

	modal := modalStyle.Render(modalContent)

	// Split background and modal into lines
	backgroundLines := strings.Split(backgroundView, "\n")
	modalLines := strings.Split(modal, "\n")

	// Calculate position to center the modal
	modalStartY := (windowHeight - len(modalLines)) / 2
	modalStartX := (windowWidth - lipgloss.Width(modalLines[0])) / 2

	if modalStartY < 0 {
		modalStartY = 0
	}
	if modalStartX < 0 {
		modalStartX = 0
	}

	// Overlay modal on background
	var result strings.Builder

	for i := 0; i < len(backgroundLines); i++ {
		if i > 0 {
			result.WriteString("\n")
		}

		// Check if this line should show the modal
		modalLineIdx := i - modalStartY
		if modalLineIdx >= 0 && modalLineIdx < len(modalLines) {
			// This line shows the modal overlaid on the background
			bgLine := backgroundLines[i]
			bgWidth := lipgloss.Width(bgLine)
			modalLine := modalLines[modalLineIdx]
			modalWidth := lipgloss.Width(modalLine)

			// Show background before modal (if modalStartX > 0)
			if modalStartX > 0 && bgWidth > 0 {
				// Truncate background to show only the part before the modal
				beforeModal := truncateToVisualWidth(bgLine, modalStartX)
				result.WriteString(beforeModal)
			}

			// Show the modal
			result.WriteString(modalLine)

			// Show background after modal or pad to maintain line width
			endX := modalStartX + modalWidth
			if endX < bgWidth {
				// There's background content after the modal
				afterModal := truncateFromVisualWidth(bgLine, endX)
				result.WriteString(afterModal)
			} else if endX > bgWidth {
				// Modal extends beyond background, but we don't need to pad
				// The modal rendering handles its own width
			}
			// If endX == bgWidth, the modal exactly fills the line - nothing more needed
		} else {
			// No modal on this line, show background as-is
			result.WriteString(backgroundLines[i])
		}
	}

	return result.String()
}

// ShowDeleteConfirmation creates a delete confirmation modal
func ShowDeleteConfirmation(itemPreview string, itemIndex int) ShowModalMsg {
	return ShowModalMsg{
		Title: "Delete Item?",
		Content: fmt.Sprintf("Item: %s\nIndex: %d\n\nAre you sure you want to delete this item?",
			itemPreview, itemIndex),
		Options: "[Y] Yes, delete    [N] No, cancel",
	}
}

// truncateToVisualWidth truncates a styled string to the specified visual width
func truncateToVisualWidth(s string, targetWidth int) string {
	if targetWidth <= 0 {
		return ""
	}

	currentWidth := 0
	runes := []rune(s)
	inEscape := false
	var result strings.Builder

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Track ANSI escape sequences (they don't count toward visual width)
		if r == '\x1b' {
			inEscape = true
		}

		if inEscape {
			result.WriteRune(r)
			if r == 'm' {
				inEscape = false
			}
			continue
		}

		// Count visual width (normal characters count as 1)
		if currentWidth >= targetWidth {
			break
		}

		result.WriteRune(r)
		currentWidth++
	}

	return result.String()
}

// truncateFromVisualWidth returns the portion of a styled string starting from the specified visual position
func truncateFromVisualWidth(s string, startWidth int) string {
	if startWidth <= 0 {
		return s
	}

	currentWidth := 0
	runes := []rune(s)
	inEscape := false
	startIdx := -1
	var pendingEscapes strings.Builder

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Track ANSI escape sequences
		if r == '\x1b' {
			inEscape = true
			if startIdx < 0 {
				pendingEscapes.WriteRune(r)
			}
		} else if inEscape {
			if startIdx < 0 {
				pendingEscapes.WriteRune(r)
			}
			if r == 'm' {
				inEscape = false
			}
		} else {
			// Normal visible character
			if currentWidth >= startWidth && startIdx < 0 {
				startIdx = i
			}
			currentWidth++
		}
	}

	if startIdx < 0 {
		// Start width is beyond the string
		return ""
	}

	// Include any pending escape codes that were before the start
	return pendingEscapes.String() + string(runes[startIdx:])
}
