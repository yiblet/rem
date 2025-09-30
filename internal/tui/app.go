package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.design/x/clipboard"
)

// PaneType represents which pane is focused
type PaneType int

const (
	LeftPane PaneType = iota
	RightPane
)

// UIMode represents the current modal state of the application
type UIMode int

const (
	NormalMode UIMode = iota
	SearchMode
	HelpMode
	NumberInputMode
	DeleteMode
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

type flashExpiredMsg struct{}

func (flashExpiredMsg) isAppMsg() {}

// AppModel orchestrates all sub-models
type AppModel struct {
	Width       int      // Window width
	Height      int      // Window height
	LeftWidth   int      // Left pane width
	RightWidth  int      // Right pane width
	ActivePane  PaneType // Currently focused pane
	CurrentMode UIMode   // Current modal state

	// Sub-models
	LeftPane  LeftPaneModel
	RightPane RightPaneModel
	Search    SearchModel
	Modal     ModalModel
	Items     []*StackItem

	// Number input mode for multi-digit commands like "10j"
	NumberBuffer string   // Accumulates digits
	BufferPane   PaneType // Which pane the buffer applies to

	// Flash message for temporary notifications
	FlashMessage string    // The message to display
	FlashExpiry  time.Time // When the message should disappear
}

// NewAppModel creates a new app model with all sub-models
func NewAppModel(items []*StackItem) AppModel {
	// Default dimensions that will be properly set on first resize
	defaultWidth := 120
	defaultHeight := 20
	defaultLeftWidth := 25
	defaultRightWidth := 90

	return AppModel{
		Width:       defaultWidth,
		Height:      defaultHeight,
		LeftWidth:   defaultLeftWidth,
		RightWidth:  defaultRightWidth,
		ActivePane:  LeftPane,
		CurrentMode: NormalMode,
		LeftPane:    NewLeftPaneModel(defaultLeftWidth, defaultHeight),
		RightPane:   NewRightPaneModel(defaultRightWidth, defaultHeight),
		Search:      NewSearchModel(),
		Modal:       NewModalModel(),
		Items:       items,
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
	case flashExpiredMsg:
		// Clear flash message when it expires
		a.FlashMessage = ""
		a.FlashExpiry = time.Time{}
		return a, nil
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
	borderSpacing := 2 // Account for adjacent borders (no space separator)

	// If total width is too small, split proportionally
	if a.Width < minLeftWidth+minRightWidth+borderSpacing {
		// Very narrow - give each pane minimum space
		a.LeftWidth = minLeftWidth
		a.RightWidth = max(a.Width-a.LeftWidth-borderSpacing, minRightWidth)
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

// handleKeyPress processes key press events using mode-first architecture
func (a *AppModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// MODE-FIRST ARCHITECTURE: Check current mode before processing any keys
	switch a.CurrentMode {
	case SearchMode:
		return a.handleSearchModeKeys(key)
	case HelpMode:
		return a.handleHelpModeKeys(key)
	case NumberInputMode:
		return a.handleNumberInputModeKeys(key)
	case DeleteMode:
		return a.handleDeleteModeKeys(key)
	case NormalMode:
		return a.handleNormalModeKeys(key)
	default:
		// Fallback to normal mode for unknown modes
		return a.handleNormalModeKeys(key)
	}
}

// handleSearchModeKeys processes keys when in search mode
func (a *AppModel) handleSearchModeKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c":
		// Force quit is always available
		return a, tea.Quit
	case "esc":
		// Cancel search and return to normal mode
		a.Search.Update(CancelSearchMsg{})
		a.CurrentMode = NormalMode
		return a, nil
	case "enter":
		// Execute search and return to normal mode
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
		a.CurrentMode = NormalMode
		return a, nil
	case "backspace", "ctrl+h":
		// Remove last character
		if len(a.Search.GetInput()) > 0 {
			newInput := a.Search.GetInput()[:len(a.Search.GetInput())-1]
			a.Search.Update(UpdateSearchInputMsg{Input: newInput})
		}
		return a, nil
	default:
		// Add character to search input (only printable characters)
		if len(key) == 1 && key[0] >= 32 && key[0] <= 126 {
			newInput := a.Search.GetInput() + key
			a.Search.Update(UpdateSearchInputMsg{Input: newInput})
		}
		return a, nil
	}
}

// handleHelpModeKeys processes keys when in help mode
func (a *AppModel) handleHelpModeKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c":
		// Force quit is always available
		return a, tea.Quit
	case "z", "esc", "q":
		// Exit help mode and return to normal mode
		a.CurrentMode = NormalMode
		return a, nil
	default:
		// All other keys are ignored in help mode
		return a, nil
	}
}

// handleNumberInputModeKeys processes keys when in number input mode
func (a *AppModel) handleNumberInputModeKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c":
		// Force quit is always available
		return a, tea.Quit
	case "esc":
		// Cancel number input and return to normal mode
		a.NumberBuffer = ""
		a.CurrentMode = NormalMode
		return a, nil
	case "backspace":
		// Remove last digit
		if len(a.NumberBuffer) > 1 {
			a.NumberBuffer = a.NumberBuffer[:len(a.NumberBuffer)-1]
		} else {
			a.NumberBuffer = ""
			a.CurrentMode = NormalMode
		}
		return a, nil
	default:
		// Handle digit input or execute command
		if key >= "0" && key <= "9" {
			a.NumberBuffer += key
			return a, nil
		} else if isMovementCommand(key) {
			// Execute command with multiplier and return to normal mode
			multiplier := 1
			if num, err := strconv.Atoi(a.NumberBuffer); err == nil {
				multiplier = num
			}
			a.NumberBuffer = ""
			a.CurrentMode = NormalMode
			return a.executeCommand(multiplier, key, a.ActivePane)
		} else {
			// Invalid key, cancel number input
			a.NumberBuffer = ""
			a.CurrentMode = NormalMode
			return a, nil
		}
	}
}

// handleDeleteModeKeys processes keys when in delete confirmation mode
func (a *AppModel) handleDeleteModeKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c":
		// Force quit is always available
		return a, tea.Quit
	case "y", "Y":
		// Confirm deletion
		if a.LeftPane.Selected < len(a.Items) && len(a.Items) > 0 {
			deletedIndex := a.LeftPane.Selected
			selectedItem := a.Items[deletedIndex]

			// Delete from persistent storage if DeleteFunc is provided
			if selectedItem.DeleteFunc != nil {
				if err := selectedItem.DeleteFunc(); err != nil {
					// Show error message and stay in delete mode
					a.Modal.Update(ShowModalMsg{
						Title:   "Delete Error",
						Content: fmt.Sprintf("Failed to delete item: %v", err),
						Options: "Press any key to continue",
					})
					return a, nil
				}
			}

			// Remove item from the Items slice
			a.Items = append(a.Items[:deletedIndex], a.Items[deletedIndex+1:]...)

			// Adjust cursor position if needed
			if len(a.Items) == 0 {
				// Queue is now empty
				a.LeftPane.Cursor = 0
				a.LeftPane.Selected = 0
			} else if a.LeftPane.Selected >= len(a.Items) {
				// Cursor was at the last item, move it back
				a.LeftPane.Cursor = len(a.Items) - 1
				a.LeftPane.Selected = a.LeftPane.Cursor
			}
			// else: cursor stays at the same index, now pointing to the next item

			// Update the right pane content
			a.RightPane.Update(UpdateContentMsg{})

			// Show success flash message
			flashCmd := a.setFlashMessage("Item deleted successfully", 2*time.Second)

			// Hide modal and return to normal mode
			a.Modal.Update(HideModalMsg{})
			a.CurrentMode = NormalMode
			return a, flashCmd
		}

		// Hide modal and return to normal mode
		a.Modal.Update(HideModalMsg{})
		a.CurrentMode = NormalMode
		return a, nil
	case "n", "N", "esc":
		// Cancel deletion - hide modal and return to normal mode
		a.Modal.Update(HideModalMsg{})
		a.CurrentMode = NormalMode
		return a, nil
	default:
		// Ignore other keys in delete mode
		return a, nil
	}
}

// handleNormalModeKeys processes keys when in normal mode
func (a *AppModel) handleNormalModeKeys(key string) (tea.Model, tea.Cmd) {
	// Handle global keys that work in normal mode
	switch key {
	case "ctrl+c", "q":
		return a, tea.Quit
	case "esc":
		return a, tea.Quit
	case "z":
		// Enter help mode
		a.CurrentMode = HelpMode
		return a, nil
	case "c":
		// Copy content to clipboard
		return a, a.copyToClipboard()
	case "tab":
		// Toggle between left and right pane
		if a.ActivePane == LeftPane {
			a.ActivePane = RightPane
		} else {
			a.ActivePane = LeftPane
		}
		return a, nil
	case "h", "left":
		// Switch to left pane (no-op if already on left)
		if a.ActivePane == RightPane {
			a.ActivePane = LeftPane
		}
		return a, nil
	case "l", "right":
		// Switch to right pane (no-op if already on right)
		if a.ActivePane == LeftPane {
			a.ActivePane = RightPane
		}
		return a, nil
	}

	// Handle number input (digits 1-9, 0 only after other digits)
	if key >= "1" && key <= "9" || (key == "0" && a.NumberBuffer != "") {
		a.NumberBuffer += key
		a.BufferPane = a.ActivePane
		a.CurrentMode = NumberInputMode
		return a, nil
	}

	// Check if this is a movement command that should use any existing multiplier
	if isMovementCommand(key) {
		multiplier := 1
		if a.NumberBuffer != "" && a.BufferPane == a.ActivePane {
			if num, err := strconv.Atoi(a.NumberBuffer); err == nil {
				multiplier = num
			}
			a.NumberBuffer = ""
		}
		return a.executeCommand(multiplier, key, a.ActivePane)
	}

	// Handle pane-specific keys
	switch a.ActivePane {
	case LeftPane:
		return a.handleLeftPaneKeys(key)
	case RightPane:
		return a.handleRightPaneKeys(key)
	}

	return a, nil
}

// handleLeftPaneKeys processes keys when left pane is focused in normal mode
func (a *AppModel) handleLeftPaneKeys(key string) (tea.Model, tea.Cmd) {
	// Only handle non-movement keys here, movement keys are handled by executeCommand
	switch key {
	case "d":
		// Enter delete confirmation mode if there are items
		if len(a.Items) > 0 && a.LeftPane.Selected < len(a.Items) {
			a.CurrentMode = DeleteMode
			// Show the delete confirmation modal
			selectedItem := a.Items[a.LeftPane.Selected]
			a.Modal.Update(ShowDeleteConfirmation(selectedItem.Preview, a.LeftPane.Selected))
		}
		return a, nil
	default:
		// Handle other left pane specific keys here if needed in the future
	}

	return a, nil
}

// handleRightPaneKeys processes keys when right pane is focused in normal mode
func (a *AppModel) handleRightPaneKeys(key string) (tea.Model, tea.Cmd) {
	var maxScroll int
	if a.LeftPane.Selected < len(a.Items) {
		selectedItem := a.Items[a.LeftPane.Selected]
		maxScroll = getMaxScroll(a.RightPane, selectedItem)
	}

	switch key {
	case "/", "?":
		// Enter search mode
		a.Search.Update(StartSearchMsg{})
		a.CurrentMode = SearchMode
		return a, nil
	case "n":
		if a.Search.HasMatches() {
			a.Search.Update(NextMatchMsg{})
			if matchLine := a.Search.GetCurrentMatchLine(); matchLine >= 0 && a.LeftPane.Selected < len(a.Items) {
				selectedItem := a.Items[a.LeftPane.Selected]
				newViewPos := scrollToMatch(a.RightPane, selectedItem, matchLine)
				a.RightPane.ViewPos = newViewPos
			}
		}
		return a, nil
	case "N":
		if a.Search.HasMatches() {
			a.Search.Update(PrevMatchMsg{})
			if matchLine := a.Search.GetCurrentMatchLine(); matchLine >= 0 && a.LeftPane.Selected < len(a.Items) {
				selectedItem := a.Items[a.LeftPane.Selected]
				newViewPos := scrollToMatch(a.RightPane, selectedItem, matchLine)
				a.RightPane.ViewPos = newViewPos
			}
		}
		return a, nil
	case "ctrl+u":
		a.RightPane.Update(PageUpMsg{})
		return a, nil
	case "ctrl+d":
		a.RightPane.Update(PageDownMsg{MaxScroll: maxScroll})
		return a, nil
	case "ctrl+b":
		pageSize := a.Height - 6
		a.RightPane.Update(JumpMsg{Direction: "k", Lines: pageSize, MaxScroll: maxScroll})
		return a, nil
	case "ctrl+f":
		pageSize := a.Height - 6
		a.RightPane.Update(JumpMsg{Direction: "j", Lines: pageSize, MaxScroll: maxScroll})
		return a, nil
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

	// Show help view if in help mode
	if model.CurrentMode == HelpMode {
		helpView := renderHelpView(model)
		return helpView + "\n\n" + renderStatusLine(model), nil
	}

	// Render normal view first (this will be the background for modal)
	normalView, err := renderNormalView(model)
	if err != nil {
		return "", err
	}

	// Overlay modal if active (for delete confirmation, etc.)
	if model.Modal.Active {
		return ModalView(model.Modal, normalView, model.Width, model.Height), nil
	}

	return normalView, nil
}

// renderNormalView renders the normal dual-pane view
func renderNormalView(model AppModel) (string, error) {
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

		// Join panes directly (borders provide visual separation)
		result.WriteString(leftLine + rightLine + "\n")
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

	// Prioritize flash message if active and not expired
	if model.FlashMessage != "" && time.Now().Before(model.FlashExpiry) {
		statusLine = model.FlashMessage
		// Use green color for flash messages
		statusStyle := lipgloss.NewStyle().
			Width(model.Width).
			Foreground(lipgloss.Color("10"))
		return statusStyle.Render(statusLine)
	}

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
		// Show mode-specific status text
		switch model.CurrentMode {
		case HelpMode:
			statusLine = "Help Mode - Press z to return to normal view, q to quit"
		case SearchMode:
			statusLine = fmt.Sprintf("Search: /%s (Enter to execute, Esc to cancel)", model.Search.GetInput())
		case NumberInputMode:
			statusLine = fmt.Sprintf("Number Input: %s (Enter command or Esc to cancel)", model.NumberBuffer)
		default: // NormalMode
			statusLine = "Press z for help, q to quit"
		}
	}

	statusStyle := lipgloss.NewStyle().
		Width(model.Width)

	return statusStyle.Render(statusLine)
}

// renderHelpView renders the help content as a single pane (pure function)
func renderHelpView(model AppModel) string {
	helpContent := `rem - Enhanced Clipboard Queue Manager

NAVIGATION COMMANDS:
  j, ↓        Move down (left pane: next item, right pane: scroll down)
  k, ↑        Move up (left pane: previous item, right pane: scroll up)
  g           Go to top (with number: go to line N)
  G           Go to bottom
  #j, #k      Jump N lines (e.g., 10j moves down 10 lines/items)

PANE SWITCHING:
  Tab         Toggle between left and right panes
  h, ←        Switch to left pane
  l, →        Switch to right pane
  z           Toggle this help screen

CONTENT VIEWING:
  /pattern    Search for text pattern in current item
  n           Next search match
  N           Previous search match
  Ctrl+u      Page up (right pane)
  Ctrl+d      Page down (right pane)
  Ctrl+b      Page up (full screen)
  Ctrl+f      Page down (full screen)

CLIPBOARD:
  c           Copy current item content to clipboard

HISTORY MANAGEMENT:
  d           Delete selected item (left pane only)

QUEUE BEHAVIOR:
  Index 0     Most recent item (top of queue)
  Index 1+    Older items in reverse chronological order
  Max 20      Queue automatically removes oldest items

GLOBAL COMMANDS:
  q           Quit
  Esc         Cancel search or quit
  Ctrl+c      Force quit

Press z again to return to normal view.`

	// Create a bordered help view
	helpStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		Width(model.Width - 4).
		Height(model.Height - 4)

	return helpStyle.Render(helpContent)
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

// setFlashMessage sets a flash message that will disappear after the specified duration
func (a *AppModel) setFlashMessage(message string, duration time.Duration) tea.Cmd {
	a.FlashMessage = message
	a.FlashExpiry = time.Now().Add(duration)
	return tea.Tick(duration, func(t time.Time) tea.Msg {
		return flashExpiredMsg{}
	})
}

// copyToClipboard copies the current item's content to the clipboard
func (a *AppModel) copyToClipboard() tea.Cmd {
	// Get the currently selected item
	if a.LeftPane.Selected >= len(a.Items) || len(a.Items) == 0 {
		return a.setFlashMessage("No item selected", 2*time.Second)
	}

	selectedItem := a.Items[a.LeftPane.Selected]
	if selectedItem == nil {
		return a.setFlashMessage("No item selected", 2*time.Second)
	}

	// Get the full content
	content, err := selectedItem.GetFullContent()
	if err != nil {
		return a.setFlashMessage(fmt.Sprintf("Error reading content: %v", err), 2*time.Second)
	}

	// Initialize clipboard
	err = clipboard.Init()
	if err != nil {
		return a.setFlashMessage(fmt.Sprintf("Error initializing clipboard: %v", err), 2*time.Second)
	}

	// Write to clipboard
	clipboard.Write(clipboard.FmtText, []byte(content))

	// Show success message with byte count
	byteCount := len(content)
	return a.setFlashMessage(fmt.Sprintf("Copied %d bytes to clipboard", byteCount), 2*time.Second)
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
