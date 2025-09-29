package tui

import (
	"testing"
)

func TestNewSearchModel(t *testing.T) {
	model := NewSearchModel()

	if model.Active {
		t.Error("Expected Active to be false")
	}
	if model.Input != "" {
		t.Error("Expected Input to be empty")
	}
	if model.Pattern != "" {
		t.Error("Expected Pattern to be empty")
	}
	if model.Error != "" {
		t.Error("Expected Error to be empty")
	}
	if model.Matches != nil {
		t.Error("Expected Matches to be nil")
	}
	if model.CurrentMatch != -1 {
		t.Error("Expected CurrentMatch to be -1")
	}
}

func TestSearchModel_StartSearch(t *testing.T) {
	model := NewSearchModel()

	err := model.Update(StartSearchMsg{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !model.IsActive() {
		t.Error("Expected model to be active after StartSearchMsg")
	}
	if model.GetInput() != "" {
		t.Error("Expected input to be cleared when starting search")
	}
	if model.GetError() != "" {
		t.Error("Expected error to be cleared when starting search")
	}
}

func TestSearchModel_UpdateInput(t *testing.T) {
	model := NewSearchModel()
	model.Update(StartSearchMsg{}) // Start search first

	testInput := "test query"
	err := model.Update(UpdateSearchInputMsg{Input: testInput})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if model.GetInput() != testInput {
		t.Errorf("Expected input to be %q, got %q", testInput, model.GetInput())
	}
}

func TestSearchModel_ExecuteSearch(t *testing.T) {
	model := NewSearchModel()
	model.Update(StartSearchMsg{})
	model.Update(UpdateSearchInputMsg{Input: "test"})

	err := model.Update(ExecuteSearchMsg{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if model.IsActive() {
		t.Error("Expected search to be inactive after execution")
	}
	if model.GetPattern() != "test" {
		t.Errorf("Expected pattern to be 'test', got %q", model.GetPattern())
	}
	if model.GetError() != "" {
		t.Errorf("Expected no error, got %q", model.GetError())
	}
}

func TestSearchModel_ExecuteSearchError(t *testing.T) {
	model := NewSearchModel()
	model.Update(StartSearchMsg{})
	model.Update(UpdateSearchInputMsg{Input: "["}) // Invalid regex

	err := model.Update(ExecuteSearchMsg{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !model.IsActive() {
		t.Error("Expected search to remain active after error")
	}
	if model.GetError() == "" {
		t.Error("Expected error message to be set")
	}
	if model.GetPattern() != "" {
		t.Error("Expected pattern to remain empty after error")
	}
}

func TestSearchModel_CancelSearch(t *testing.T) {
	model := NewSearchModel()
	model.Update(StartSearchMsg{})
	model.Update(UpdateSearchInputMsg{Input: "test input"})

	err := model.Update(CancelSearchMsg{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if model.IsActive() {
		t.Error("Expected search to be inactive after cancel")
	}
	if model.GetInput() != "" {
		t.Error("Expected input to be cleared after cancel")
	}
}

func TestSearchModel_SetMatches(t *testing.T) {
	model := NewSearchModel()
	matches := []int{0, 2, 5, 8}

	model.SetMatches(matches)

	if !model.HasMatches() {
		t.Error("Expected model to have matches")
	}

	currentMatch, totalMatches := model.GetCurrentMatch()
	if currentMatch != 0 {
		t.Errorf("Expected current match to be 0, got %d", currentMatch)
	}
	if totalMatches != 4 {
		t.Errorf("Expected total matches to be 4, got %d", totalMatches)
	}
}

func TestSearchModel_NextMatch(t *testing.T) {
	model := NewSearchModel()
	matches := []int{0, 2, 5}
	model.SetMatches(matches)

	// Move to next match
	model.Update(NextMatchMsg{})
	currentMatch, _ := model.GetCurrentMatch()
	if currentMatch != 1 {
		t.Errorf("Expected current match to be 1, got %d", currentMatch)
	}

	// Move to next match again
	model.Update(NextMatchMsg{})
	currentMatch, _ = model.GetCurrentMatch()
	if currentMatch != 2 {
		t.Errorf("Expected current match to be 2, got %d", currentMatch)
	}

	// Wrap around to first match
	model.Update(NextMatchMsg{})
	currentMatch, _ = model.GetCurrentMatch()
	if currentMatch != 0 {
		t.Errorf("Expected current match to wrap to 0, got %d", currentMatch)
	}
}

func TestSearchModel_PrevMatch(t *testing.T) {
	model := NewSearchModel()
	matches := []int{0, 2, 5}
	model.SetMatches(matches)

	// Move to previous match (should wrap to last)
	model.Update(PrevMatchMsg{})
	currentMatch, _ := model.GetCurrentMatch()
	if currentMatch != 2 {
		t.Errorf("Expected current match to wrap to 2, got %d", currentMatch)
	}

	// Move to previous match again
	model.Update(PrevMatchMsg{})
	currentMatch, _ = model.GetCurrentMatch()
	if currentMatch != 1 {
		t.Errorf("Expected current match to be 1, got %d", currentMatch)
	}
}

func TestSearchModel_GetCurrentMatchLine(t *testing.T) {
	model := NewSearchModel()
	matches := []int{1, 3, 7}
	model.SetMatches(matches)

	// Should return first match line
	matchLine := model.GetCurrentMatchLine()
	if matchLine != 1 {
		t.Errorf("Expected current match line to be 1, got %d", matchLine)
	}

	// Move to next match and check line
	model.Update(NextMatchMsg{})
	matchLine = model.GetCurrentMatchLine()
	if matchLine != 3 {
		t.Errorf("Expected current match line to be 3, got %d", matchLine)
	}
}

func TestSearchModel_GetMatches(t *testing.T) {
	model := NewSearchModel()
	matches := []int{1, 3, 5, 7}
	model.SetMatches(matches)

	retrievedMatches := model.GetMatches()
	if len(retrievedMatches) != len(matches) {
		t.Errorf("Expected %d matches, got %d", len(matches), len(retrievedMatches))
	}

	for i, match := range matches {
		if retrievedMatches[i] != match {
			t.Errorf("Expected match %d to be %d, got %d", i, match, retrievedMatches[i])
		}
	}
}

func TestSearchModel_NoMatches(t *testing.T) {
	model := NewSearchModel()

	if model.HasMatches() {
		t.Error("Expected model to have no matches initially")
	}

	matchLine := model.GetCurrentMatchLine()
	if matchLine != -1 {
		t.Errorf("Expected current match line to be -1, got %d", matchLine)
	}

	currentMatch, totalMatches := model.GetCurrentMatch()
	if currentMatch != -1 {
		t.Errorf("Expected current match to be -1, got %d", currentMatch)
	}
	if totalMatches != 0 {
		t.Errorf("Expected total matches to be 0, got %d", totalMatches)
	}
}