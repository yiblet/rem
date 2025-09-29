package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PaneType represents which pane is focused
type PaneType int

const (
	LeftPane PaneType = iota
	RightPane
)

// AppMsg represents messages that the app component handles
type AppMsg interface {
	isAppMsg()
}

// App-level message implementations
type WindowResizeMsg struct {
	Width  int
	Height int
}

func (WindowResizeMsg) isAppMsg() {}

type FocusChangeMsg struct {
	Pane PaneType
}

func (FocusChangeMsg) isAppMsg() {}

type QuitMsg struct{}

func (QuitMsg) isAppMsg() {}

type KeyPressMsg struct {
	Key string
}

func (KeyPressMsg) isAppMsg() {}


// AppModel orchestrates all sub-models
type AppModel struct {
	Width      int      // Window width
	Height     int      // Window height
	LeftWidth  int      // Left pane width
	RightWidth int      // Right pane width
	ActivePane PaneType // Currently focused pane

	// Sub-models
	LeftPane   LeftPaneModel
	RightPane  RightPaneModel
	Search     SearchModel
	Items      []*StackItem

	// Number input mode for multi-digit commands like "10j"
	NumberBuffer string   // Accumulates digits
	BufferPane   PaneType // Which pane the buffer applies to
}

// NewAppModel creates a new app model with all sub-models
func NewAppModel(items []*StackItem) AppModel {
	// Default dimensions that will be properly set on first resize
	defaultWidth := 120
	defaultHeight := 20
	defaultLeftWidth := 25
	defaultRightWidth := 90

	return AppModel{
		Width:      defaultWidth,
		Height:     defaultHeight,
		LeftWidth:  defaultLeftWidth,
		RightWidth: defaultRightWidth,
		ActivePane: LeftPane,
		LeftPane:   NewLeftPaneModel(defaultLeftWidth, defaultHeight),
		RightPane:  NewRightPaneModel(defaultRightWidth, defaultHeight),
		Search:     NewSearchModel(),
		Items:      items,
	}
}

// Update handles app-level messages and routes to appropriate sub-models
func (a *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle Bubble Tea messages first
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		return a.handleWindowResize(m)
	case tea.KeyMsg:
		return a.handleKeyPress(m)
	}

	return a, nil
}

// handleWindowResize processes window resize events
func (a *AppModel) handleWindowResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	a.Width = msg.Width
	a.Height = msg.Height

	// Ensure minimum total width of 30 characters
	minTotalWidth := 30
	if msg.Width < minTotalWidth {
		a.Width = minTotalWidth
	}

	// Calculate pane widths with proper constraints
	minLeftWidth := 15
	minRightWidth := 20
	borderSpacing := 3

	// If total width is too small, split proportionally
	if a.Width < minLeftWidth + minRightWidth + borderSpacing {
		// Very narrow - give each pane minimum space
		a.LeftWidth = minLeftWidth
		a.RightWidth = max(a.Width - a.LeftWidth - borderSpacing, minRightWidth)
	} else {
		// Normal case - use preferred left width, rest goes to right
		preferredLeftWidth := 25
		a.LeftWidth = min(preferredLeftWidth, a.Width/3) // Don't take more than 1/3
		a.RightWidth = a.Width - a.LeftWidth - borderSpacing

		// Ensure minimums are respected
		if a.LeftWidth < minLeftWidth {
			a.LeftWidth = minLeftWidth
			a.RightWidth = a.Width - a.LeftWidth - borderSpacing
		}
		if a.RightWidth < minRightWidth {
			a.RightWidth = minRightWidth
			a.LeftWidth = a.Width - a.RightWidth - borderSpacing
		}
	}

	// Update sub-models
	a.LeftPane.Update(ResizeLeftPaneMsg{Width: a.LeftWidth, Height: a.Height})
	a.RightPane.Update(ResizeRightPaneMsg{Width: a.RightWidth, Height: a.Height})

	// Update content in right pane when window resizes
	a.RightPane.Update(UpdateContentMsg{})

	return a, nil
}

// handleKeyPress processes key press events and routes them to appropriate models
func (a *AppModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle global keys first
	switch key {
	case "ctrl+c", "q":
		return a, tea.Quit
	case "esc":
		// If search is active, cancel it; otherwise quit
		if a.Search.IsActive() {
			a.Search.Update(CancelSearchMsg{})
			return a, nil
		}
		return a, tea.Quit
	case "tab":
		// Toggle between left and right pane
		if a.ActivePane == LeftPane {
			a.ActivePane = RightPane
		} else {
			a.ActivePane = LeftPane
		}
		return a, nil
	}

	// Handle search mode keys
	if a.Search.IsActive() {
		return a.handleSearchModeKeys(key)
	}

	// Handle number mode input
	if newBuffer, handled := handleNumberMode(a.NumberBuffer, a.BufferPane, a.ActivePane, key); handled {
		a.NumberBuffer = newBuffer
		if a.NumberBuffer != "" {
			a.BufferPane = a.ActivePane
		}
		return a, nil
	}

	// Get multiplier and clear buffer if we have one
	multiplier := 1
	if a.NumberBuffer != "" && a.BufferPane == a.ActivePane {
		if num, err := strconv.Atoi(a.NumberBuffer); err == nil {
			multiplier = num
		}
		a.NumberBuffer = ""
	}

	// Check if this is a movement command that should use the multiplier
	if isMovementCommand(key) {
		return a.executeCommand(multiplier, key, a.ActivePane)
	}

	// Clear buffer for non-movement keys and route to pane-specific handlers
	a.NumberBuffer = ""
	switch a.ActivePane {
	case LeftPane:
		return a.handleLeftPaneKeys(key)
	case RightPane:
		return a.handleRightPaneKeys(key)
	}

	return a, nil
}

// handleSearchModeKeys processes keys when in search mode
func (a *AppModel) handleSearchModeKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "enter":
		// Execute search
		a.Search.Update(ExecuteSearchMsg{})

		// If search was successful and we have matches, perform the search on current item
		if a.Search.GetError() == "" && a.LeftPane.Selected < len(a.Items) {
			selectedItem := a.Items[a.LeftPane.Selected]
			if selectedItem != nil && a.Search.GetPattern() != "" {
				// Perform search on the item
				selectedItem.performSearch(a.Search.GetPattern())

				// Update search model with matches
				a.Search.SetMatches(selectedItem.SearchMatches)

				// Jump to first match if found
				if matchLine := a.Search.GetCurrentMatchLine(); matchLine >= 0 {
					newViewPos := scrollToMatch(a.RightPane, selectedItem, matchLine)
					a.RightPane.ViewPos = newViewPos
				}
			}
		}
	case "backspace", "ctrl+h":
		// Remove last character
		if len(a.Search.GetInput()) > 0 {
			newInput := a.Search.GetInput()[:len(a.Search.GetInput())-1]
			a.Search.Update(UpdateSearchInputMsg{Input: newInput})
		}
	default:
		// Add character to search input (only printable characters)
		if len(key) == 1 && key[0] >= 32 && key[0] <= 126 {
			newInput := a.Search.GetInput() + key
			a.Search.Update(UpdateSearchInputMsg{Input: newInput})
		}
	}

	return a, nil
}

// handleLeftPaneKeys processes keys when left pane is focused
func (a *AppModel) handleLeftPaneKeys(key string) (tea.Model, tea.Cmd) {
	// Only handle non-movement keys here, movement keys are handled by executeCommand
	switch key {
	default:
		// Handle other left pane specific keys here if needed in the future
	}

	return a, nil
}

// handleRightPaneKeys processes keys when right pane is focused
func (a *AppModel) handleRightPaneKeys(key string) (tea.Model, tea.Cmd) {
	var maxScroll int
	if a.LeftPane.Selected < len(a.Items) {
		selectedItem := a.Items[a.LeftPane.Selected]
		maxScroll = getMaxScroll(a.RightPane, selectedItem)
	}

	switch key {
	case "/", "?":
		a.Search.Update(StartSearchMsg{})
	case "n":
		if a.Search.HasMatches() {
			a.Search.Update(NextMatchMsg{})
			if matchLine := a.Search.GetCurrentMatchLine(); matchLine >= 0 && a.LeftPane.Selected < len(a.Items) {
				selectedItem := a.Items[a.LeftPane.Selected]
				newViewPos := scrollToMatch(a.RightPane, selectedItem, matchLine)
				a.RightPane.ViewPos = newViewPos
			}
		}
	case "N":
		if a.Search.HasMatches() {
			a.Search.Update(PrevMatchMsg{})
			if matchLine := a.Search.GetCurrentMatchLine(); matchLine >= 0 && a.LeftPane.Selected < len(a.Items) {
				selectedItem := a.Items[a.LeftPane.Selected]
				newViewPos := scrollToMatch(a.RightPane, selectedItem, matchLine)
				a.RightPane.ViewPos = newViewPos
			}
		}
	case "ctrl+u":
		a.RightPane.Update(PageUpMsg{})
	case "ctrl+d":
		a.RightPane.Update(PageDownMsg{MaxScroll: maxScroll})
	case "ctrl+b":
		pageSize := a.Height - 6
		a.RightPane.Update(JumpMsg{Direction: "k", Lines: pageSize, MaxScroll: maxScroll})
	case "ctrl+f":
		pageSize := a.Height - 6
		a.RightPane.Update(JumpMsg{Direction: "j", Lines: pageSize, MaxScroll: maxScroll})
	}

	return a, nil
}

// Init initializes the app model (required by tea.Model interface)
func (a *AppModel) Init() tea.Cmd {
	// Initialize right pane content if there are items
	if len(a.Items) > 0 {
		a.RightPane.Update(UpdateContentMsg{})
	}
	return nil
}

// AppView renders the complete application using pure functions
func AppView(model AppModel) (string, error) {
	if model.Width == 0 {
		return "Initializing...", nil
	}

	leftPaneFocused := model.ActivePane == LeftPane
	rightPaneFocused := model.ActivePane == RightPane

	// Get current content
	var selectedItem *StackItem
	if model.LeftPane.Selected < len(model.Items) {
		selectedItem = model.Items[model.LeftPane.Selected]
	}

	leftPaneView, err := LeftPaneView(model.LeftPane, model.Items, leftPaneFocused)
	if err != nil {
		return "", err
	}

	rightPaneView, err := RightPaneView(model.RightPane, selectedItem, model.Search, rightPaneFocused, model.LeftPane.Selected)
	if err != nil {
		return "", err
	}

	// Join left and right panes side by side
	leftLines := strings.Split(leftPaneView, "\n")
	rightLines := strings.Split(rightPaneView, "\n")

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

		// Join with a single space as separator between panes
		result.WriteString(leftLine + " " + rightLine + "\n")
	}

	result.WriteString("\n" + renderStatusLine(model))

	return result.String(), nil
}

// View method for tea.Model compatibility
func (a *AppModel) View() string {
	view, _ := AppView(*a)
	return view
}

// renderStatusLine renders the bottom status line (pure function)
func renderStatusLine(model AppModel) string {
	var statusLine string

	if model.NumberBuffer != "" {
		// Show number buffer input
		statusLine = fmt.Sprintf("%s", model.NumberBuffer)
	} else if model.Search.IsActive() {
		// Show search input with cursor
		statusLine = fmt.Sprintf("/%s", model.Search.GetInput())
		if model.Search.GetError() != "" {
			statusLine += fmt.Sprintf(" (Error: %s)", model.Search.GetError())
		} else {
			statusLine += " (Enter to search, Esc to cancel)"
		}
	} else if model.Search.HasMatches() {
		// Show search results
		currentMatch, totalMatches := model.Search.GetCurrentMatch()
		statusLine = fmt.Sprintf("Pattern: %s - Match %d of %d", model.Search.GetPattern(), currentMatch+1, totalMatches)
	} else {
		// Show help text
		if model.ActivePane == LeftPane {
			statusLine = "Left Pane: ↑/k ↓/j (navigate) g/G (top/bottom) ##/##j/##k (jump) Tab (switch) q (quit)"
		} else {
			statusLine = "Right Pane: ↑/k ↓/j (scroll) / (search) n/N (next/prev) g/G (top/bottom) ##j/##k (jump) q (quit)"
		}
	}

	statusStyle := lipgloss.NewStyle().
		Width(model.Width)

	return statusStyle.Render(statusLine)
}

// handleNumberMode processes digit input and backspace for vim-style number prefixes
func handleNumberMode(currentBuffer string, targetPane PaneType, activePane PaneType, key string) (newBuffer string, handled bool) {
	// Handle digit input
	if key >= "0" && key <= "9" {
		return currentBuffer + key, true
	}

	// Handle backspace in number mode
	if key == "backspace" && currentBuffer != "" {
		if len(currentBuffer) > 1 {
			return currentBuffer[:len(currentBuffer)-1], true
		}
		return "", true
	}

	return currentBuffer, false
}

// isMovementCommand checks if a key is a movement command that can use multipliers
func isMovementCommand(key string) bool {
	switch key {
	case "up", "k", "down", "j", "g", "G":
		return true
	}
	return false
}

// executeCommand executes a command with a number multiplier on the specified pane
func (a *AppModel) executeCommand(multiplier int, key string, pane PaneType) (tea.Model, tea.Cmd) {
	maxIndex := len(a.Items) - 1

	if pane == LeftPane {
		switch key {
		case "up", "k":
			newCursor := max(a.LeftPane.Cursor-multiplier, 0)
			a.LeftPane.Update(JumpToIndexMsg{Index: newCursor, MaxIndex: maxIndex})
			a.RightPane.Update(UpdateContentMsg{})
		case "down", "j":
			newCursor := min(a.LeftPane.Cursor+multiplier, maxIndex)
			a.LeftPane.Update(JumpToIndexMsg{Index: newCursor, MaxIndex: maxIndex})
			a.RightPane.Update(UpdateContentMsg{})
		case "g":
			if multiplier > 1 {
				jumpIndex := min(max(multiplier-1, 0), maxIndex)
				a.LeftPane.Update(JumpToIndexMsg{Index: jumpIndex, MaxIndex: maxIndex})
			} else {
				a.LeftPane.Update(GoToTopMsg{})
			}
			a.RightPane.Update(UpdateContentMsg{})
		case "G":
			a.LeftPane.Update(GoToBottomMsg{MaxIndex: maxIndex})
			a.RightPane.Update(UpdateContentMsg{})
		}
	} else { // RightPane
		var maxScroll int
		if a.LeftPane.Selected < len(a.Items) {
			selectedItem := a.Items[a.LeftPane.Selected]
			maxScroll = getMaxScroll(a.RightPane, selectedItem)
		}

		switch key {
		case "up", "k":
			a.RightPane.Update(JumpMsg{Direction: "k", Lines: multiplier, MaxScroll: maxScroll})
		case "down", "j":
			a.RightPane.Update(JumpMsg{Direction: "j", Lines: multiplier, MaxScroll: maxScroll})
		case "g":
			if multiplier > 1 {
				newViewPos := max(multiplier-1, 0)
				if newViewPos > maxScroll {
					newViewPos = maxScroll
				}
				a.RightPane.ViewPos = newViewPos
			} else {
				a.RightPane.Update(ScrollToTopMsg{})
			}
		case "G":
			a.RightPane.Update(ScrollToBottomMsg{MaxScroll: maxScroll})
		}
	}

	return a, nil
}

// SetItems updates the items list in the app model
func (a *AppModel) SetItems(items []*StackItem) {
	a.Items = items

	// Adjust cursor if it's beyond the new item count
	if a.LeftPane.Cursor >= len(items) {
		if len(items) > 0 {
			a.LeftPane.Cursor = len(items) - 1
			a.LeftPane.Selected = a.LeftPane.Cursor
		} else {
			a.LeftPane.Cursor = 0
			a.LeftPane.Selected = 0
		}
	}

	// Reset right pane content
	a.RightPane.Update(UpdateContentMsg{})
}