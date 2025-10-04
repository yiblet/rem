package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yiblet/rem/internal/clipboard/mockboard"
)

// newTestClipboard creates a mock clipboard for testing
func newTestClipboard() *mockboard.MockClipboard {
	return mockboard.New()
}

func TestNewAppModel(t *testing.T) {
	items := []*StackItem{}
	model := NewAppModel(items, newTestClipboard())

	if model.Width != 120 {
		t.Errorf("Expected width to be 120, got %d", model.Width)
	}
	if model.Height != 20 {
		t.Errorf("Expected height to be 20, got %d", model.Height)
	}
	if model.LeftWidth != 25 {
		t.Errorf("Expected left width to be 25, got %d", model.LeftWidth)
	}
	if model.ActivePane != LeftPane {
		t.Errorf("Expected active pane to be LeftPane, got %v", model.ActivePane)
	}
}

func TestNewAppModel_WithItems(t *testing.T) {
	items := []*StackItem{
		{Content: NewStringReadSeekCloser("Item 1 content"), Preview: "Item 1"},
		{Content: NewStringReadSeekCloser("Item 2 content"), Preview: "Item 2"},
	}

	app := NewAppModel(items, newTestClipboard())

	if app.ActivePane != LeftPane {
		t.Errorf("Expected active pane to be LeftPane, got %v", app.ActivePane)
	}
	if len(app.Items) != 2 {
		t.Errorf("Expected 2 items in app, got %d", len(app.Items))
	}
}

func TestAppModel_WindowResize(t *testing.T) {
	items := []*StackItem{{
		Content: NewStringReadSeekCloser("Test item content"),
		Preview: "Test item",
	}}
	app := NewAppModel(items, newTestClipboard())

	// Test window resize
	newModel, _ := app.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	updatedApp := newModel.(*AppModel)

	if updatedApp.Width != 140 {
		t.Errorf("Expected width to be 140, got %d", updatedApp.Width)
	}
	if updatedApp.Height != 30 {
		t.Errorf("Expected height to be 30, got %d", updatedApp.Height)
	}

	expectedRightWidth := 140 - 25 - 2 // width - leftWidth - spacing (2 for adjacent borders)
	if updatedApp.RightWidth != expectedRightWidth {
		t.Errorf("Expected right width to be %d, got %d", expectedRightWidth, updatedApp.RightWidth)
	}
}

func TestAppModel_TabSwitching(t *testing.T) {
	items := []*StackItem{{
		Content: NewStringReadSeekCloser("Test item content"),
		Preview: "Test item",
	}}
	app := NewAppModel(items, newTestClipboard())

	// Should start with left pane active
	if app.ActivePane != LeftPane {
		t.Errorf("Expected initial active pane to be LeftPane, got %v", app.ActivePane)
	}

	// Press tab to switch to right pane
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyTab})
	updatedApp := newModel.(*AppModel)

	if updatedApp.ActivePane != RightPane {
		t.Errorf("Expected active pane to be RightPane after tab, got %v", updatedApp.ActivePane)
	}

	// Press tab again to switch back to left pane
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyTab})
	updatedApp = newModel.(*AppModel)

	if updatedApp.ActivePane != LeftPane {
		t.Errorf("Expected active pane to be LeftPane after second tab, got %v", updatedApp.ActivePane)
	}
}

func TestAppModel_QuitKeys(t *testing.T) {
	items := []*StackItem{{
		Content: NewStringReadSeekCloser("Test item content"),
		Preview: "Test item",
	}}
	app := NewAppModel(items, newTestClipboard())

	quitKeys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("q")},
		{Type: tea.KeyCtrlC},
	}

	for i, keyMsg := range quitKeys {
		t.Run(fmt.Sprintf("key_%d", i), func(t *testing.T) {
			_, cmd := app.Update(keyMsg)
			if cmd == nil {
				t.Error("Expected quit command but got nil")
			}

			// We can't directly compare cmd to tea.Quit since it's a function,
			// but we can verify it's not nil which indicates a quit command was returned
		})
	}
}

func TestAppModel_LeftPaneNavigation(t *testing.T) {
	items := []*StackItem{
		{Content: NewStringReadSeekCloser("Item 1 content"), Preview: "Item 1"},
		{Content: NewStringReadSeekCloser("Item 2 content"), Preview: "Item 2"},
		{Content: NewStringReadSeekCloser("Item 3 content"), Preview: "Item 3"},
	}
	app := NewAppModel(items, newTestClipboard())

	// Should start at position 0
	if app.LeftPane.Cursor != 0 {
		t.Errorf("Expected cursor to be 0, got %d", app.LeftPane.Cursor)
	}

	// Press j (down)
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	updatedApp := newModel.(*AppModel)

	if updatedApp.LeftPane.Cursor != 1 {
		t.Errorf("Expected cursor to be 1 after j, got %d", updatedApp.LeftPane.Cursor)
	}

	// Press k (up)
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	updatedApp = newModel.(*AppModel)

	if updatedApp.LeftPane.Cursor != 0 {
		t.Errorf("Expected cursor to be 0 after k, got %d", updatedApp.LeftPane.Cursor)
	}
}

func TestAppModel_RightPaneScrolling(t *testing.T) {
	content := &StackItem{
		Content: NewStringReadSeekCloser(strings.Repeat("Line\n", 50)),
		Preview: "Long content",
	}
	items := []*StackItem{content}
	app := NewAppModel(items, newTestClipboard())

	// Switch to right pane
	app.ActivePane = RightPane
	// Initialize content and ensure lines are calculated
	content.UpdateWrappedLines(app.RightWidth-6, app.RightPane.Height-6)
	app.Init()

	// Should start at position 0
	if app.RightPane.ViewPos != 0 {
		t.Errorf("Expected view position to be 0, got %d", app.RightPane.ViewPos)
	}

	// Press j (down)
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	updatedApp := newModel.(*AppModel)

	if updatedApp.RightPane.ViewPos != 1 {
		t.Errorf("Expected view position to be 1 after j, got %d", updatedApp.RightPane.ViewPos)
	}

	// Press k (up)
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	updatedApp = newModel.(*AppModel)

	if updatedApp.RightPane.ViewPos != 0 {
		t.Errorf("Expected view position to be 0 after k, got %d", updatedApp.RightPane.ViewPos)
	}
}

func TestAppModel_SearchMode(t *testing.T) {
	content := &StackItem{
		Content: NewStringReadSeekCloser("Line 1 test\nLine 2\nLine 3 test"),
		Preview: "Test content",
	}
	items := []*StackItem{content}
	app := NewAppModel(items, newTestClipboard())

	// Switch to right pane and initialize
	app.ActivePane = RightPane
	app.Init()

	// Start search mode with /
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	updatedApp := newModel.(*AppModel)

	if !updatedApp.Search.IsActive() {
		t.Error("Expected search to be active after pressing /")
	}

	// Type search input
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	updatedApp = newModel.(*AppModel)

	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	updatedApp = newModel.(*AppModel)

	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	updatedApp = newModel.(*AppModel)

	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	updatedApp = newModel.(*AppModel)

	if updatedApp.Search.GetInput() != "test" {
		t.Errorf("Expected search input to be 'test', got %q", updatedApp.Search.GetInput())
	}

	// Ensure content lines are calculated for search
	content.UpdateWrappedLines(app.RightWidth-6, app.RightPane.Height-6)

	// Execute search with enter
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updatedApp = newModel.(*AppModel)

	if updatedApp.Search.IsActive() {
		t.Error("Expected search to be inactive after pressing enter")
	}

	if updatedApp.Search.GetPattern() != "test" {
		t.Errorf("Expected search pattern to be 'test', got %q", updatedApp.Search.GetPattern())
	}

	if !updatedApp.Search.HasMatches() {
		t.Error("Expected search to have matches")
	}
}

func TestAppModel_SearchNavigation(t *testing.T) {
	content := &StackItem{
		Content: NewStringReadSeekCloser("Line 1 test\nLine 2\nLine 3 test"),
		Preview: "Test content",
	}
	items := []*StackItem{content}
	app := NewAppModel(items, newTestClipboard())

	// Set up search results manually
	app.ActivePane = RightPane
	content.UpdateWrappedLines(app.RightWidth-6, app.RightPane.Height-6) // Ensure lines are calculated
	app.Init()

	// Set up search pattern and perform search
	app.Search.Update(StartSearchMsg{})
	app.Search.Update(UpdateSearchInputMsg{Input: "test"})
	app.Search.Update(ExecuteSearchMsg{}) // Set pattern
	content.performSearch("test")
	app.Search.SetMatches(content.SearchMatches)

	// Test next match
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	updatedApp := newModel.(*AppModel)

	currentMatch, _ := updatedApp.Search.GetCurrentMatch()
	if currentMatch != 1 { // Should move to second match
		t.Errorf("Expected current match to be 1, got %d", currentMatch)
	}

	// Test previous match
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("N")})
	updatedApp = newModel.(*AppModel)

	currentMatch, _ = updatedApp.Search.GetCurrentMatch()
	if currentMatch != 0 { // Should move back to first match
		t.Errorf("Expected current match to be 0, got %d", currentMatch)
	}
}

func TestAppModel_SearchCancel(t *testing.T) {
	items := []*StackItem{{
		Content: NewStringReadSeekCloser("Test item content"),
		Preview: "Test item",
	}}
	app := NewAppModel(items, newTestClipboard())
	app.ActivePane = RightPane

	// Enter search mode properly
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	updatedApp := newModel.(*AppModel)

	if updatedApp.CurrentMode != SearchMode {
		t.Fatal("Failed to enter search mode")
	}

	// Add some input
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	updatedApp = newModel.(*AppModel)
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	updatedApp = newModel.(*AppModel)
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	updatedApp = newModel.(*AppModel)
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	updatedApp = newModel.(*AppModel)

	if updatedApp.Search.GetInput() != "test" {
		t.Errorf("Expected search input 'test', got '%s'", updatedApp.Search.GetInput())
	}

	// Press escape to cancel
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updatedApp = newModel.(*AppModel)

	if updatedApp.CurrentMode != NormalMode {
		t.Error("Expected to return to normal mode after escape")
	}
	if updatedApp.Search.GetInput() != "" {
		t.Error("Expected search input to be cleared after escape")
	}
}

func TestAppModel_View(t *testing.T) {
	items := []*StackItem{
		{Content: NewStringReadSeekCloser("Test item 1 content"), Preview: "Test item 1"},
		{Content: NewStringReadSeekCloser("Test item 2 content"), Preview: "Test item 2"},
	}
	app := NewAppModel(items, newTestClipboard())

	// Test initial view (width 0) - force width to 0 to test this condition
	app.Width = 0
	view := app.View()
	if !strings.Contains(view, "Initializing...") {
		t.Error("Expected view to show 'Initializing...' when width is 0")
	}

	// Set proper dimensions
	app.Width = 120
	app.Height = 20
	app.Init()

	view = app.View()
	if view == "" {
		t.Error("Expected non-empty view")
	}

	// Should contain elements from both panes
	if !strings.Contains(view, "Queue") {
		t.Error("Expected view to contain 'Queue' from left pane")
	}
	if !strings.Contains(view, "Content") {
		t.Error("Expected view to contain 'Content' from right pane")
	}
	if !strings.Contains(view, "Test item 1") {
		t.Error("Expected view to contain item preview")
	}

	// Should contain status line
	if !strings.Contains(view, "Press z for help") {
		t.Error("Expected view to contain simplified status line")
	}
}

func TestAppModel_StatusLineStates(t *testing.T) {
	items := []*StackItem{{
		Content: NewStringReadSeekCloser("Test item content"),
		Preview: "Test item",
	}}
	app := NewAppModel(items, newTestClipboard())
	app.Width = 120

	// Test normal status line
	statusLine := renderStatusLine(app)
	if !strings.Contains(statusLine, "Press z for help") {
		t.Error("Expected simplified status line")
	}

	// Test help mode status
	app.CurrentMode = HelpMode
	statusLine = renderStatusLine(app)
	if !strings.Contains(statusLine, "Help Mode") {
		t.Error("Expected help mode status line")
	}

	// Test search mode status
	app.Search.Update(StartSearchMsg{})
	app.Search.Update(UpdateSearchInputMsg{Input: "test"})
	statusLine = renderStatusLine(app)
	if !strings.Contains(statusLine, "/test") {
		t.Error("Expected search input in status line")
	}

	// Test search error status
	app.Search.Update(UpdateSearchInputMsg{Input: "["})
	app.Search.Update(ExecuteSearchMsg{})
	statusLine = renderStatusLine(app)
	if !strings.Contains(statusLine, "Error:") {
		t.Error("Expected error message in status line")
	}
}

func TestAppModel_SetItems(t *testing.T) {
	originalItems := []*StackItem{
		{Content: NewStringReadSeekCloser("Item 1 content"), Preview: "Item 1"},
		{Content: NewStringReadSeekCloser("Item 2 content"), Preview: "Item 2"},
	}
	app := NewAppModel(originalItems, newTestClipboard())
	app.Init()

	// Set new items
	newItems := []*StackItem{
		{Content: NewStringReadSeekCloser("New Item 1 content"), Preview: "New Item 1"},
		{Content: NewStringReadSeekCloser("New Item 2 content"), Preview: "New Item 2"},
		{Content: NewStringReadSeekCloser("New Item 3 content"), Preview: "New Item 3"},
	}
	app.SetItems(newItems)

	if len(app.Items) != 3 {
		t.Errorf("Expected 3 items after SetItems, got %d", len(app.Items))
	}

	// Check that right pane view position was reset
	if app.RightPane.ViewPos != 0 {
		t.Error("Expected right pane view position to be reset to 0 after SetItems")
	}
}

func TestAppModel_JumpCommands(t *testing.T) {
	content := &StackItem{
		Content: NewStringReadSeekCloser(strings.Repeat("Line\n", 50)),
		Preview: "Long content",
	}
	items := []*StackItem{content}
	app := NewAppModel(items, newTestClipboard())

	// Switch to right pane and initialize
	app.ActivePane = RightPane
	app.Init()

	// Test jump command through the update mechanism
	// Calculate max scroll for bounds checking
	availableHeight := app.RightPane.Height - 6
	content.UpdateWrappedLines(app.RightWidth-6, availableHeight)
	maxScroll := 0
	if len(content.Lines) > availableHeight {
		maxScroll = len(content.Lines) - availableHeight
	}

	// Test "10j" jump command - send as separate key events
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	updatedApp := newModel.(*AppModel)

	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("0")})
	updatedApp = newModel.(*AppModel)

	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	updatedApp = newModel.(*AppModel)

	// Should move down by 10 lines (or to max scroll if less than 10)
	expectedPos := min(10, maxScroll)
	if updatedApp.RightPane.ViewPos != expectedPos {
		t.Errorf("Expected view position to be %d after '10j', got %d", expectedPos, updatedApp.RightPane.ViewPos)
	}
}

// TestAppModel_PaneBorderAlignment tests that left and right panes render the same number of lines
// to ensure borders align properly, preventing the scrolling visual bug where right pane borders
// appear on different lines than left pane borders.
func TestAppModel_PaneBorderAlignment(t *testing.T) {
	testCases := []struct {
		name        string
		contentSize int
		height      int
		scrollPos   int
	}{
		{
			name:        "short content no scroll",
			contentSize: 5,
			height:      15,
			scrollPos:   0,
		},
		{
			name:        "long content no scroll",
			contentSize: 30,
			height:      15,
			scrollPos:   0,
		},
		{
			name:        "long content with scroll",
			contentSize: 30,
			height:      15,
			scrollPos:   5,
		},
		{
			name:        "very short window",
			contentSize: 20,
			height:      10,
			scrollPos:   3,
		},
		{
			name:        "tall window",
			contentSize: 15,
			height:      25,
			scrollPos:   0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create content with specific number of lines
			contentLines := make([]string, tc.contentSize)
			for i := 0; i < tc.contentSize; i++ {
				contentLines[i] = fmt.Sprintf("Line %d with some content to test wrapping behavior", i+1)
			}
			contentStr := strings.Join(contentLines, "\n")

			content := &StackItem{
				Content: NewStringReadSeekCloser(contentStr),
				Preview: "Test content for border alignment",
			}
			items := []*StackItem{content}
			app := NewAppModel(items, newTestClipboard())

			// Set specific dimensions
			app.Width = 120
			app.Height = tc.height
			app.LeftWidth = 25
			app.RightWidth = 90

			// Update sub-models with dimensions
			app.LeftPane.Update(ResizeLeftPaneMsg{Width: app.LeftWidth, Height: app.Height})
			app.RightPane.Update(ResizeRightPaneMsg{Width: app.RightWidth, Height: app.Height})
			app.RightPane.Update(UpdateContentMsg{})

			// Set scroll position
			availableHeight := app.RightPane.Height - 6
			content.UpdateWrappedLines(app.RightWidth-6, availableHeight)
			maxScroll := 0
			if len(content.Lines) > availableHeight {
				maxScroll = len(content.Lines) - availableHeight
			}
			if tc.scrollPos <= maxScroll {
				app.RightPane.ViewPos = tc.scrollPos
			}

			// Render both panes separately
			leftPaneView, err := LeftPaneView(app.LeftPane, app.Items, app.ActivePane == LeftPane)
			if err != nil {
				t.Fatalf("Error rendering left pane: %v", err)
			}

			rightPaneView, err := RightPaneView(app.RightPane, content, app.Search, app.ActivePane == RightPane, 0)
			if err != nil {
				t.Fatalf("Error rendering right pane: %v", err)
			}

			// Split into lines and count
			leftLines := strings.Split(leftPaneView, "\n")
			rightLines := strings.Split(rightPaneView, "\n")

			// Critical test: both panes should render the same number of lines
			if len(leftLines) != len(rightLines) {
				t.Errorf("Border alignment issue: left pane has %d lines, right pane has %d lines. "+
					"This will cause border misalignment in the TUI. "+
					"Content size: %d, Height: %d, Scroll: %d",
					len(leftLines), len(rightLines), tc.contentSize, tc.height, tc.scrollPos)
			}

			// Verify that the complete AppView renders properly
			appView, err := AppView(app)
			if err != nil {
				t.Fatalf("Error rendering app view: %v", err)
			}

			appLines := strings.Split(appView, "\n")

			// Find lines with border characters to verify alignment
			var borderLines []string
			for _, line := range appLines {
				if strings.Contains(line, "│") {
					borderLines = append(borderLines, line)
				}
			}

			// Check that border characters appear in expected positions consistently
			if len(borderLines) > 0 {
				// All border lines should have consistent structure
				firstBorderLine := borderLines[0]
				leftBorderPos := strings.Index(firstBorderLine, "│")

				for i, borderLine := range borderLines {
					currentLeftBorderPos := strings.Index(borderLine, "│")
					if currentLeftBorderPos != leftBorderPos {
						t.Errorf("Inconsistent border alignment at line %d: expected left border at pos %d, got %d",
							i, leftBorderPos, currentLeftBorderPos)
					}
				}
			}

			// Verify no empty trailing lines in pane views (which could cause alignment issues)
			if len(leftLines) > 0 && strings.TrimSpace(leftLines[len(leftLines)-1]) == "" {
				t.Error("Left pane view has trailing empty line which could cause alignment issues")
			}
			if len(rightLines) > 0 && strings.TrimSpace(rightLines[len(rightLines)-1]) == "" {
				t.Error("Right pane view has trailing empty line which could cause alignment issues")
			}
		})
	}
}

func TestAppModel_HelpMode(t *testing.T) {
	items := []*StackItem{{
		Content: NewStringReadSeekCloser("Test content"),
		Preview: "Test item",
	}}
	app := NewAppModel(items, newTestClipboard())
	app.Width = 120
	app.Height = 20

	// Initially not in help mode
	if app.CurrentMode != NormalMode {
		t.Error("Expected CurrentMode to be NormalMode initially")
	}

	// Press 'z' to enter help mode
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	updatedApp := newModel.(*AppModel)

	if updatedApp.CurrentMode != HelpMode {
		t.Error("Expected CurrentMode to be HelpMode after pressing 'z'")
	}

	// Press 'z' again to exit help mode
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	updatedApp = newModel.(*AppModel)

	if updatedApp.CurrentMode != NormalMode {
		t.Error("Expected CurrentMode to be NormalMode after pressing 'z' again")
	}
}

func TestAppModel_HelpModeView(t *testing.T) {
	items := []*StackItem{{
		Content: NewStringReadSeekCloser("Test content"),
		Preview: "Test item",
	}}
	app := NewAppModel(items, newTestClipboard())
	app.Width = 120
	app.Height = 20
	app.CurrentMode = HelpMode

	// Get view in help mode
	view, err := AppView(app)
	if err != nil {
		t.Fatalf("Error rendering help view: %v", err)
	}

	// Should contain help content
	if !strings.Contains(view, "rem - Enhanced Clipboard Queue Manager") {
		t.Error("Expected help view to contain help title")
	}

	// Should contain key binding information
	if !strings.Contains(view, "NAVIGATION COMMANDS") {
		t.Error("Expected help view to contain navigation commands section")
	}

	if !strings.Contains(view, "PANE SWITCHING") {
		t.Error("Expected help view to contain pane switching section")
	}

	// Should contain 'z' key reference
	if !strings.Contains(view, "z           Toggle this help screen") {
		t.Error("Expected help view to contain 'z' key reference")
	}

	// Should contain proper status line for help mode
	if !strings.Contains(view, "Help Mode - Press z to return") {
		t.Error("Expected help mode status line")
	}
}

func TestAppModel_HelpModeStatusLine(t *testing.T) {
	app := NewAppModel([]*StackItem{}, newTestClipboard())
	app.Width = 120

	// Test normal mode status line
	statusLine := renderStatusLine(app)
	if !strings.Contains(statusLine, "Press z for help") {
		t.Error("Expected normal mode to show 'Press z for help'")
	}

	// Test help mode status line
	app.CurrentMode = HelpMode
	statusLine = renderStatusLine(app)
	if !strings.Contains(statusLine, "Help Mode - Press z to return") {
		t.Error("Expected help mode to show 'Press z to return'")
	}
}

func TestAppModel_PaneSwitchingWithH(t *testing.T) {
	items := []*StackItem{{
		Content: NewStringReadSeekCloser("Test content"),
		Preview: "Test item",
	}}
	app := NewAppModel(items, newTestClipboard())
	app.ActivePane = RightPane

	// Press 'h' to switch to left pane
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	updatedApp := newModel.(*AppModel)

	if updatedApp.ActivePane != LeftPane {
		t.Error("Expected 'h' to switch to left pane")
	}

	// Press 'h' again when already on left pane (should be no-op)
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	updatedApp = newModel.(*AppModel)

	if updatedApp.ActivePane != LeftPane {
		t.Error("Expected 'h' to be no-op when already on left pane")
	}
}

func TestAppModel_ModalStateTransitions(t *testing.T) {
	items := []*StackItem{{
		Content: NewStringReadSeekCloser("Test content"),
		Preview: "Test item",
	}}
	app := NewAppModel(items, newTestClipboard())

	// Initially in normal mode
	if app.CurrentMode != NormalMode {
		t.Errorf("Expected initial mode to be NormalMode, got %v", app.CurrentMode)
	}

	// Enter help mode with 'z'
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	updatedApp := newModel.(*AppModel)

	if updatedApp.CurrentMode != HelpMode {
		t.Errorf("Expected help mode after 'z', got %v", updatedApp.CurrentMode)
	}

	// Exit help mode with 'z' again
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	updatedApp = newModel.(*AppModel)

	if updatedApp.CurrentMode != NormalMode {
		t.Errorf("Expected normal mode after second 'z', got %v", updatedApp.CurrentMode)
	}

	// Enter search mode with '/' (from right pane)
	updatedApp.ActivePane = RightPane
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	updatedApp = newModel.(*AppModel)

	if updatedApp.CurrentMode != SearchMode {
		t.Errorf("Expected search mode after '/', got %v", updatedApp.CurrentMode)
	}

	// Exit search mode with Esc
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updatedApp = newModel.(*AppModel)

	if updatedApp.CurrentMode != NormalMode {
		t.Errorf("Expected normal mode after Esc from search, got %v", updatedApp.CurrentMode)
	}

	// Enter number input mode with digit
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("5")})
	updatedApp = newModel.(*AppModel)

	if updatedApp.CurrentMode != NumberInputMode {
		t.Errorf("Expected number input mode after digit, got %v", updatedApp.CurrentMode)
	}

	if updatedApp.NumberBuffer != "5" {
		t.Errorf("Expected number buffer to be '5', got '%s'", updatedApp.NumberBuffer)
	}

	// Exit number input mode with Esc
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updatedApp = newModel.(*AppModel)

	if updatedApp.CurrentMode != NormalMode {
		t.Errorf("Expected normal mode after Esc from number input, got %v", updatedApp.CurrentMode)
	}

	if updatedApp.NumberBuffer != "" {
		t.Errorf("Expected empty number buffer after Esc, got '%s'", updatedApp.NumberBuffer)
	}
}

func TestAppModel_SearchModeBlocksPaneNavigation(t *testing.T) {
	items := []*StackItem{{
		Content: NewStringReadSeekCloser("Test content"),
		Preview: "Test item",
	}}
	app := NewAppModel(items, newTestClipboard())
	app.ActivePane = RightPane

	// Enter search mode
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	updatedApp := newModel.(*AppModel)

	if updatedApp.CurrentMode != SearchMode {
		t.Fatal("Failed to enter search mode")
	}

	// Try to switch panes with 'h' - should add 'h' to search input instead
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	updatedApp = newModel.(*AppModel)

	// Should still be in search mode and on right pane
	if updatedApp.CurrentMode != SearchMode {
		t.Error("Expected to remain in search mode after 'h'")
	}
	if updatedApp.ActivePane != RightPane {
		t.Error("Expected to remain on right pane while in search mode")
	}

	// Search input should contain 'h'
	if updatedApp.Search.GetInput() != "h" {
		t.Errorf("Expected search input to be 'h', got '%s'", updatedApp.Search.GetInput())
	}

	// Try other navigation keys - should all be added to search input
	keys := []string{"l", "j", "k"}
	expectedInput := "h"

	for _, key := range keys {
		newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		updatedApp = newModel.(*AppModel)

		expectedInput += key
		if updatedApp.Search.GetInput() != expectedInput {
			t.Errorf("After key '%s', expected search input '%s', got '%s'", key, expectedInput, updatedApp.Search.GetInput())
		}
		if updatedApp.CurrentMode != SearchMode {
			t.Errorf("After key '%s', expected to remain in search mode", key)
		}
	}

	// Test tab key specifically - should also be added to search input
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyTab})
	updatedApp = newModel.(*AppModel)

	// Tab character should be added to search input (tab is ASCII 9, but in search input it's treated as regular input)
	// The tea.KeyTab should be converted to a tab character in search
	if updatedApp.CurrentMode != SearchMode {
		t.Error("Expected to remain in search mode after tab key")
	}
}

func TestAppModel_HelpModeBlocksAllNavigation(t *testing.T) {
	items := []*StackItem{{
		Content: NewStringReadSeekCloser("Test content"),
		Preview: "Test item",
	}}
	app := NewAppModel(items, newTestClipboard())
	app.ActivePane = RightPane

	// Enter help mode
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	updatedApp := newModel.(*AppModel)

	if updatedApp.CurrentMode != HelpMode {
		t.Fatal("Failed to enter help mode")
	}

	initialPane := updatedApp.ActivePane

	// Try various navigation keys - should all be ignored
	navigationKeys := []string{"h", "l", "j", "k", "tab", "/", "n", "N"}

	for _, key := range navigationKeys {
		newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		updatedApp = newModel.(*AppModel)

		if updatedApp.CurrentMode != HelpMode {
			t.Errorf("After key '%s', expected to remain in help mode, got %v", key, updatedApp.CurrentMode)
		}
		if updatedApp.ActivePane != initialPane {
			t.Errorf("After key '%s', pane should not change in help mode", key)
		}
	}

	// Only 'z', 'q', and 'esc' should exit help mode
	exitKeys := []string{"z", "q", "esc"}

	for _, key := range exitKeys {
		// Re-enter help mode
		updatedApp.CurrentMode = HelpMode

		var keyMsg tea.KeyMsg
		if key == "esc" {
			keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
		} else {
			keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		}

		newModel, _ = updatedApp.Update(keyMsg)
		updatedApp = newModel.(*AppModel)

		if updatedApp.CurrentMode == HelpMode {
			t.Errorf("Key '%s' should exit help mode", key)
		}
	}
}

func TestAppModel_NumberInputModeHandling(t *testing.T) {
	items := []*StackItem{{
		Content: NewStringReadSeekCloser("Test content"),
		Preview: "Test item",
	}}
	app := NewAppModel(items, newTestClipboard())

	// Enter number input mode with '1'
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	updatedApp := newModel.(*AppModel)

	if updatedApp.CurrentMode != NumberInputMode {
		t.Fatal("Failed to enter number input mode")
	}
	if updatedApp.NumberBuffer != "1" {
		t.Errorf("Expected number buffer '1', got '%s'", updatedApp.NumberBuffer)
	}

	// Add more digits
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("0")})
	updatedApp = newModel.(*AppModel)

	if updatedApp.NumberBuffer != "10" {
		t.Errorf("Expected number buffer '10', got '%s'", updatedApp.NumberBuffer)
	}
	if updatedApp.CurrentMode != NumberInputMode {
		t.Error("Should remain in number input mode")
	}

	// Execute movement command 'j' - should execute 10j and return to normal mode
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	updatedApp = newModel.(*AppModel)

	if updatedApp.CurrentMode != NormalMode {
		t.Error("Should return to normal mode after executing movement command")
	}
	if updatedApp.NumberBuffer != "" {
		t.Errorf("Number buffer should be cleared after command execution, got '%s'", updatedApp.NumberBuffer)
	}

	// Test backspace in number input mode
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	updatedApp = newModel.(*AppModel)
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("5")})
	updatedApp = newModel.(*AppModel)

	if updatedApp.NumberBuffer != "25" {
		t.Errorf("Expected number buffer '25', got '%s'", updatedApp.NumberBuffer)
	}

	// Backspace should remove last digit
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	updatedApp = newModel.(*AppModel)

	if updatedApp.NumberBuffer != "2" {
		t.Errorf("Expected number buffer '2' after backspace, got '%s'", updatedApp.NumberBuffer)
	}
	if updatedApp.CurrentMode != NumberInputMode {
		t.Error("Should remain in number input mode after backspace")
	}

	// Backspace on single digit should return to normal mode
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	updatedApp = newModel.(*AppModel)

	if updatedApp.CurrentMode != NormalMode {
		t.Error("Should return to normal mode after backspacing single digit")
	}
	if updatedApp.NumberBuffer != "" {
		t.Errorf("Number buffer should be empty after returning to normal mode, got '%s'", updatedApp.NumberBuffer)
	}
}

func TestAppModel_DeleteMode(t *testing.T) {
	items := []*StackItem{
		{Content: NewStringReadSeekCloser("Item 0"), Preview: "Item 0"},
		{Content: NewStringReadSeekCloser("Item 1"), Preview: "Item 1"},
		{Content: NewStringReadSeekCloser("Item 2"), Preview: "Item 2"},
	}

	model := NewAppModel(items, newTestClipboard())
	model.ActivePane = LeftPane

	// Press 'd' to enter delete mode
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updatedApp := newModel.(*AppModel)

	if updatedApp.CurrentMode != DeleteMode {
		t.Errorf("Expected DeleteMode, got %v", updatedApp.CurrentMode)
	}

	// Press 'n' to cancel
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	updatedApp = newModel.(*AppModel)

	if updatedApp.CurrentMode != NormalMode {
		t.Error("Should return to normal mode after canceling deletion")
	}
	if len(updatedApp.Items) != 3 {
		t.Errorf("Expected 3 items after canceling deletion, got %d", len(updatedApp.Items))
	}
}

func TestAppModel_DeleteConfirm(t *testing.T) {
	items := []*StackItem{
		{Content: NewStringReadSeekCloser("Item 0"), Preview: "Item 0"},
		{Content: NewStringReadSeekCloser("Item 1"), Preview: "Item 1"},
		{Content: NewStringReadSeekCloser("Item 2"), Preview: "Item 2"},
	}

	model := NewAppModel(items, newTestClipboard())
	model.ActivePane = LeftPane
	model.LeftPane.Selected = 1 // Select middle item

	// Press 'd' to enter delete mode
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updatedApp := newModel.(*AppModel)

	// Press 'y' to confirm deletion
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	updatedApp = newModel.(*AppModel)

	if updatedApp.CurrentMode != NormalMode {
		t.Error("Should return to normal mode after confirming deletion")
	}
	if len(updatedApp.Items) != 2 {
		t.Errorf("Expected 2 items after deletion, got %d", len(updatedApp.Items))
	}

	// Check that the correct item was deleted (item at index 1)
	if updatedApp.Items[0].Preview != "Item 0" {
		t.Errorf("Expected first item to be 'Item 0', got '%s'", updatedApp.Items[0].Preview)
	}
	if updatedApp.Items[1].Preview != "Item 2" {
		t.Errorf("Expected second item to be 'Item 2', got '%s'", updatedApp.Items[1].Preview)
	}
}

func TestAppModel_DeleteLastItem(t *testing.T) {
	items := []*StackItem{
		{Content: NewStringReadSeekCloser("Item 0"), Preview: "Item 0"},
		{Content: NewStringReadSeekCloser("Item 1"), Preview: "Item 1"},
		{Content: NewStringReadSeekCloser("Item 2"), Preview: "Item 2"},
	}

	model := NewAppModel(items, newTestClipboard())
	model.ActivePane = LeftPane
	model.LeftPane.Cursor = 2
	model.LeftPane.Selected = 2 // Select last item

	// Press 'd' then 'y' to delete last item
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updatedApp := newModel.(*AppModel)
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	updatedApp = newModel.(*AppModel)

	if len(updatedApp.Items) != 2 {
		t.Errorf("Expected 2 items after deletion, got %d", len(updatedApp.Items))
	}

	// Cursor should be moved back to last available item (index 1)
	if updatedApp.LeftPane.Cursor != 1 {
		t.Errorf("Expected cursor at index 1, got %d", updatedApp.LeftPane.Cursor)
	}
	if updatedApp.LeftPane.Selected != 1 {
		t.Errorf("Expected selected at index 1, got %d", updatedApp.LeftPane.Selected)
	}
}

func TestAppModel_DeleteAllItems(t *testing.T) {
	items := []*StackItem{
		{Content: NewStringReadSeekCloser("Only item"), Preview: "Only item"},
	}

	model := NewAppModel(items, newTestClipboard())
	model.ActivePane = LeftPane

	// Delete the only item
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updatedApp := newModel.(*AppModel)
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	updatedApp = newModel.(*AppModel)

	if len(updatedApp.Items) != 0 {
		t.Errorf("Expected 0 items after deleting all items, got %d", len(updatedApp.Items))
	}

	// Cursor should be at 0
	if updatedApp.LeftPane.Cursor != 0 {
		t.Errorf("Expected cursor at 0 when queue is empty, got %d", updatedApp.LeftPane.Cursor)
	}
}

func TestAppModel_DeleteModeOnlyInLeftPane(t *testing.T) {
	items := []*StackItem{
		{Content: NewStringReadSeekCloser("Item 0"), Preview: "Item 0"},
		{Content: NewStringReadSeekCloser("Item 1"), Preview: "Item 1"},
	}

	model := NewAppModel(items, newTestClipboard())
	model.ActivePane = RightPane // Focus on right pane

	// Press 'd' - should NOT enter delete mode since we're in right pane
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updatedApp := newModel.(*AppModel)

	if updatedApp.CurrentMode != NormalMode {
		t.Errorf("Should remain in NormalMode when pressing 'd' in right pane, got %v", updatedApp.CurrentMode)
	}
}

func TestAppModel_DeleteModeCancelWithEscape(t *testing.T) {
	items := []*StackItem{
		{Content: NewStringReadSeekCloser("Item 0"), Preview: "Item 0"},
		{Content: NewStringReadSeekCloser("Item 1"), Preview: "Item 1"},
	}

	model := NewAppModel(items, newTestClipboard())
	model.ActivePane = LeftPane

	// Enter delete mode
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updatedApp := newModel.(*AppModel)

	// Cancel with escape
	newModel, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyEscape})
	updatedApp = newModel.(*AppModel)

	if updatedApp.CurrentMode != NormalMode {
		t.Error("Should return to normal mode after pressing escape in delete mode")
	}
	if len(updatedApp.Items) != 2 {
		t.Errorf("Expected 2 items after canceling with escape, got %d", len(updatedApp.Items))
	}
}

func TestAppModel_CopyToClipboard(t *testing.T) {
	testContent := "Test clipboard content"
	items := []*StackItem{
		{Content: NewStringReadSeekCloser(testContent), Preview: "Test item", Size: int64(len(testContent))},
	}

	model := NewAppModel(items, newTestClipboard())
	model.ActivePane = LeftPane

	// Press 'c' to copy to clipboard
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	updatedApp := newModel.(*AppModel)

	// Should return a command (the flash message timer)
	if cmd == nil {
		t.Error("Expected a command to be returned for flash message")
	}

	// Flash message should be set
	if updatedApp.FlashMessage == "" {
		t.Error("Expected flash message to be set after copying")
	}

	// Flash message should contain byte count
	expectedByteCount := len(testContent)
	expectedMessage := fmt.Sprintf("Copied %d bytes to clipboard", expectedByteCount)
	if updatedApp.FlashMessage != expectedMessage {
		t.Errorf("Expected flash message '%s', got '%s'", expectedMessage, updatedApp.FlashMessage)
	}

	// Flash expiry should be set
	if updatedApp.FlashExpiry.IsZero() {
		t.Error("Expected flash expiry to be set")
	}
}

func TestAppModel_CopyFromRightPane(t *testing.T) {
	testContent := "Right pane test content"
	items := []*StackItem{
		{Content: NewStringReadSeekCloser(testContent), Preview: "Test item", Size: int64(len(testContent))},
	}

	model := NewAppModel(items, newTestClipboard())
	model.ActivePane = RightPane // Focus on right pane

	// Press 'c' to copy to clipboard
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	updatedApp := newModel.(*AppModel)

	// Should still copy from the currently selected item
	if cmd == nil {
		t.Error("Expected a command to be returned for flash message")
	}

	if updatedApp.FlashMessage == "" {
		t.Error("Expected flash message to be set after copying from right pane")
	}

	expectedByteCount := len(testContent)
	expectedMessage := fmt.Sprintf("Copied %d bytes to clipboard", expectedByteCount)
	if updatedApp.FlashMessage != expectedMessage {
		t.Errorf("Expected flash message '%s', got '%s'", expectedMessage, updatedApp.FlashMessage)
	}
}

func TestAppModel_CopyWithNoItems(t *testing.T) {
	// Empty items list
	model := NewAppModel([]*StackItem{}, newTestClipboard())

	// Press 'c' to copy
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	updatedApp := newModel.(*AppModel)

	// Should set an error flash message
	if cmd == nil {
		t.Error("Expected a command to be returned for flash message")
	}

	if updatedApp.FlashMessage == "" {
		t.Error("Expected flash message to be set when copying with no items")
	}

	if !strings.Contains(updatedApp.FlashMessage, "No item selected") {
		t.Errorf("Expected 'No item selected' message, got '%s'", updatedApp.FlashMessage)
	}
}

func TestAppModel_FlashMessageExpires(t *testing.T) {
	testContent := "Test content"
	items := []*StackItem{
		{Content: NewStringReadSeekCloser(testContent), Preview: "Test item"},
	}

	model := NewAppModel(items, newTestClipboard())

	// Copy to clipboard to set flash message
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	updatedApp := newModel.(*AppModel)

	// Flash message should be set
	if updatedApp.FlashMessage == "" {
		t.Fatal("Flash message should be set after copy")
	}

	// Simulate flash expiry message
	newModel, _ = updatedApp.Update(flashExpiredMsg{})
	updatedApp = newModel.(*AppModel)

	// Flash message should be cleared
	if updatedApp.FlashMessage != "" {
		t.Errorf("Expected flash message to be cleared after expiry, got '%s'", updatedApp.FlashMessage)
	}

	// Flash expiry should be reset
	if !updatedApp.FlashExpiry.IsZero() {
		t.Error("Expected flash expiry to be reset after expiry")
	}
}

func TestAppModel_FlashMessageInStatusLine(t *testing.T) {
	model := NewAppModel([]*StackItem{}, newTestClipboard())
	model.Width = 120

	// Set a flash message manually
	model.FlashMessage = "Test flash message"
	model.FlashExpiry = time.Now().Add(2 * time.Second)

	// Render status line
	statusLine := renderStatusLine(model)

	// Should show flash message
	if !strings.Contains(statusLine, "Test flash message") {
		t.Errorf("Expected status line to contain flash message, got '%s'", statusLine)
	}

	// Flash message should be shown with green color styling
	// (We can't directly test the color, but we can verify the message is there)
}

func TestAppModel_FlashMessagePriority(t *testing.T) {
	items := []*StackItem{{
		Content: NewStringReadSeekCloser("Test content"),
		Preview: "Test item",
	}}
	model := NewAppModel(items, newTestClipboard())
	model.Width = 120

	// Set up search to have a pattern
	model.Search.Update(StartSearchMsg{})
	model.Search.Update(UpdateSearchInputMsg{Input: "test"})
	model.Search.Update(ExecuteSearchMsg{})

	// Set flash message
	model.FlashMessage = "Flash message"
	model.FlashExpiry = time.Now().Add(2 * time.Second)

	// Render status line
	statusLine := renderStatusLine(model)

	// Flash message should take priority over search status
	if !strings.Contains(statusLine, "Flash message") {
		t.Error("Expected flash message to have priority in status line")
	}
	if strings.Contains(statusLine, "Pattern:") {
		t.Error("Search status should not be shown when flash message is active")
	}
}

func TestAppModel_DeleteModalRendering(t *testing.T) {
	items := []*StackItem{
		{Content: NewStringReadSeekCloser("Item 0"), Preview: "Item 0"},
		{Content: NewStringReadSeekCloser("Item 1"), Preview: "Item 1"},
	}

	model := NewAppModel(items, newTestClipboard())
	model.Width = 120
	model.Height = 20
	model.ActivePane = LeftPane
	model.CurrentMode = DeleteMode

	// Activate the delete modal (simulating pressing 'd')
	model.Modal.Update(ShowDeleteConfirmation("Item 0", 0))

	// Render the app view with delete modal
	view, err := AppView(model)
	if err != nil {
		t.Fatalf("Error rendering delete modal view: %v", err)
	}

	// Should contain the delete modal content
	if !strings.Contains(view, "Delete Item?") {
		t.Error("Expected delete modal to contain 'Delete Item?'")
	}

	// Should contain the item preview
	if !strings.Contains(view, "Item 0") {
		t.Error("Expected delete modal to show item preview")
	}

	// Should contain the confirmation options
	if !strings.Contains(view, "[Y] Yes, delete") {
		t.Error("Expected delete modal to contain confirmation options")
	}

	// Check that the view doesn't have excessive width (no cropping issues)
	// And check that line count matches expected (important for modal rendering)
	lines := strings.Split(view, "\n")

	// Render normal view to compare line counts
	model.Modal.Update(HideModalMsg{})
	model.CurrentMode = NormalMode
	normalView, _ := AppView(model)
	normalLines := strings.Split(normalView, "\n")

	// Delete modal should have same line count as normal view
	if len(lines) != len(normalLines) {
		t.Errorf("Delete modal has %d lines, expected %d (same as normal view)", len(lines), len(normalLines))
	}
}
