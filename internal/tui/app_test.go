package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewAppModel(t *testing.T) {
	items := []*StackItem{}
	model := NewAppModel(items)

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

	app := NewAppModel(items)

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
	app := NewAppModel(items)

	// Test window resize
	newModel, _ := app.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	updatedApp := newModel.(*AppModel)

	if updatedApp.Width != 140 {
		t.Errorf("Expected width to be 140, got %d", updatedApp.Width)
	}
	if updatedApp.Height != 30 {
		t.Errorf("Expected height to be 30, got %d", updatedApp.Height)
	}

	expectedRightWidth := 140 - 25 - 3 // width - leftWidth - spacing
	if updatedApp.RightWidth != expectedRightWidth {
		t.Errorf("Expected right width to be %d, got %d", expectedRightWidth, updatedApp.RightWidth)
	}
}

func TestAppModel_TabSwitching(t *testing.T) {
	items := []*StackItem{{
		Content: NewStringReadSeekCloser("Test item content"),
		Preview: "Test item",
	}}
	app := NewAppModel(items)

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
	app := NewAppModel(items)

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
	app := NewAppModel(items)

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
	app := NewAppModel(items)

	// Switch to right pane
	app.ActivePane = RightPane
	// Initialize content and ensure lines are calculated
	content.UpdateWrappedLines(app.RightWidth - 6)
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
	app := NewAppModel(items)

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
	content.UpdateWrappedLines(app.RightWidth - 6)

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
	app := NewAppModel(items)

	// Set up search results manually
	app.ActivePane = RightPane
	content.UpdateWrappedLines(app.RightWidth - 6) // Ensure lines are calculated
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
	app := NewAppModel(items)

	// Start search mode
	app.Search.Update(StartSearchMsg{})
	app.Search.Update(UpdateSearchInputMsg{Input: "test"})

	// Press escape to cancel
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updatedApp := newModel.(*AppModel)

	if updatedApp.Search.IsActive() {
		t.Error("Expected search to be inactive after escape")
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
	app := NewAppModel(items)

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
	if !strings.Contains(view, "Left Pane:") {
		t.Error("Expected view to contain status line for left pane")
	}
}

func TestAppModel_StatusLineStates(t *testing.T) {
	items := []*StackItem{{
		Content: NewStringReadSeekCloser("Test item content"),
		Preview: "Test item",
	}}
	app := NewAppModel(items)
	app.Width = 120

	// Test left pane status
	statusLine := renderStatusLine(app)
	if !strings.Contains(statusLine, "Left Pane:") {
		t.Error("Expected left pane status line")
	}

	// Test right pane status
	app.ActivePane = RightPane
	statusLine = renderStatusLine(app)
	if !strings.Contains(statusLine, "Right Pane:") {
		t.Error("Expected right pane status line")
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
	app := NewAppModel(originalItems)
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
	app := NewAppModel(items)

	// Switch to right pane and initialize
	app.ActivePane = RightPane
	app.Init()

	// Test jump command through the update mechanism
	// Calculate max scroll for bounds checking
	content.UpdateWrappedLines(app.RightWidth - 6)
	availableHeight := app.RightPane.Height - 6
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
			app := NewAppModel(items)

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
			content.UpdateWrappedLines(app.RightWidth - 6)
			availableHeight := app.RightPane.Height - 6
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