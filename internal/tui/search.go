package tui

import (
	"regexp"
)

// SearchMsg represents messages that the search component handles
type SearchMsg interface {
	isSearchMsg()
}

// Search message implementations
type StartSearchMsg struct{}

func (StartSearchMsg) isSearchMsg() {}

type UpdateSearchInputMsg struct {
	Input string
}

func (UpdateSearchInputMsg) isSearchMsg() {}

type ExecuteSearchMsg struct{}

func (ExecuteSearchMsg) isSearchMsg() {}

type CancelSearchMsg struct{}

func (CancelSearchMsg) isSearchMsg() {}

type NextMatchMsg struct{}

func (NextMatchMsg) isSearchMsg() {}

type PrevMatchMsg struct{}

func (PrevMatchMsg) isSearchMsg() {}

type ClearSearchMsg struct{}

func (ClearSearchMsg) isSearchMsg() {}

// SearchModel holds the state for search functionality
type SearchModel struct {
	Active       bool     // true when in search mode
	Input        string   // current search input
	Pattern      string   // compiled search pattern
	Error        string   // search error message
	Matches      []int    // line numbers with matches
	CurrentMatch int      // current match index (-1 if no matches)
}

// NewSearchModel creates a new search model with default values
func NewSearchModel() SearchModel {
	return SearchModel{
		Active:       false,
		Input:        "",
		Pattern:      "",
		Error:        "",
		Matches:      nil,
		CurrentMatch: -1,
	}
}

// SearchModel implements the Model interface for search functionality
func (s *SearchModel) Update(msg SearchMsg) error {
	switch m := msg.(type) {
	case StartSearchMsg:
		s.Active = true
		s.Input = ""
		s.Error = ""
	case UpdateSearchInputMsg:
		s.Input = m.Input
	case ExecuteSearchMsg:
		// Validate and compile the search pattern
		if s.Input == "" {
			s.Pattern = ""
			s.Matches = nil
			s.CurrentMatch = -1
			s.Error = ""
			s.Active = false
		} else {
			// Try to compile regex pattern (case-insensitive)
			if _, err := regexp.Compile("(?i)" + s.Input); err != nil {
				s.Error = err.Error()
				// Keep search active when there's an error so user can correct it
				return nil
			}
			s.Pattern = s.Input
			s.Error = ""
			s.Active = false
		}
	case CancelSearchMsg:
		s.Active = false
		s.Input = ""
		s.Error = ""
	case NextMatchMsg:
		if len(s.Matches) > 0 {
			s.CurrentMatch = (s.CurrentMatch + 1) % len(s.Matches)
		}
	case PrevMatchMsg:
		if len(s.Matches) > 0 {
			s.CurrentMatch = (s.CurrentMatch - 1 + len(s.Matches)) % len(s.Matches)
		}
	case ClearSearchMsg:
		s.Pattern = ""
		s.Matches = nil
		s.CurrentMatch = -1
		s.Error = ""
	}
	return nil
}

// SearchView renders the search state (typically used in status line)
func SearchView(model SearchModel) (string, error) {
	// This is used by other views for status line and highlighting
	// Individual components can access the model state directly
	return "", nil
}

// IsActive returns whether search mode is currently active
func (s *SearchModel) IsActive() bool {
	return s.Active
}

// GetInput returns the current search input
func (s *SearchModel) GetInput() string {
	return s.Input
}

// GetPattern returns the current search pattern
func (s *SearchModel) GetPattern() string {
	return s.Pattern
}

// GetError returns the current search error
func (s *SearchModel) GetError() string {
	return s.Error
}

// HasMatches returns whether there are any search matches
func (s *SearchModel) HasMatches() bool {
	return len(s.Matches) > 0
}

// GetCurrentMatch returns the current match index and total matches
func (s *SearchModel) GetCurrentMatch() (int, int) {
	if len(s.Matches) == 0 {
		return -1, 0
	}
	return s.CurrentMatch, len(s.Matches)
}

// GetCurrentMatchLine returns the line number of the current match
func (s *SearchModel) GetCurrentMatchLine() int {
	if s.CurrentMatch >= 0 && s.CurrentMatch < len(s.Matches) {
		return s.Matches[s.CurrentMatch]
	}
	return -1
}

// GetMatches returns all match line numbers
func (s *SearchModel) GetMatches() []int {
	return s.Matches
}

// SetMatches updates the search matches and resets current match
func (s *SearchModel) SetMatches(matches []int) {
	s.Matches = matches
	if len(matches) > 0 {
		s.CurrentMatch = 0
	} else {
		s.CurrentMatch = -1
	}
}