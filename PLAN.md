# TUI Elm Architecture Refactoring Plan

## Current Project Status
**Feature**: Convert TUI to Elm Architecture
**Phase**: Planning and Design
**Goal**: Refactor the existing TUI from a single-file monolithic structure to a clean Elm architecture with explicit Model/Msg/View patterns and sub-components.

## Current State Analysis

### Existing Structure
- **Single File**: All TUI code in `internal/tui/viewer.go` (~762 lines)
- **Monolithic Model**: Single `Model` struct containing all application state
- **Ad-hoc Messages**: Uses Bubble Tea's generic `tea.Msg` types with type switches
- **Scattered State**: Application state split between `Model` and `StackItem` structs
- **Helper-based Updates**: Update logic split into helper methods that mutate state

### Current State Issues
1. **Poor Separation of Concerns**: UI logic, business logic, and state management are mixed
2. **Difficult Testing**: Large functions make unit testing complex
3. **Hard to Extend**: Adding new features requires modifying multiple large functions
4. **No Component Reusability**: UI elements are tightly coupled to main model

## Target Elm Architecture Design

### Component-Specific Message Types
Each component handles only messages relevant to its domain:

#### App-Level Messages (AppMsg)
```go
type AppMsg interface {
    isAppMsg()
}

type WindowResizeMsg struct{ Width, Height int }
type FocusChangeMsg struct{ Pane PaneType }
type QuitMsg struct{}
type KeyPressMsg struct{ Key string } // Raw key that needs routing
```

#### Left Pane Messages (LeftPaneMsg)
```go
type LeftPaneMsg interface {
    isLeftPaneMsg()
}

type NavigateUpMsg struct{}
type NavigateDownMsg struct{}
type SelectItemMsg struct{ Index int }
type ResizeLeftPaneMsg struct{ Width, Height int }
```

#### Right Pane Messages (RightPaneMsg)
```go
type RightPaneMsg interface {
    isRightPaneMsg()
}

type ScrollUpMsg struct{}
type ScrollDownMsg struct{}
type ScrollToTopMsg struct{}
type ScrollToBottomMsg struct{}
type PageUpMsg struct{}
type PageDownMsg struct{}
type JumpMsg struct{ Direction string; Lines int }
type ResizeRightPaneMsg struct{ Width, Height int }
type UpdateContentMsg struct{ Item *StackItem }
```

#### Search Messages (SearchMsg)
```go
type SearchMsg interface {
    isSearchMsg()
}

type StartSearchMsg struct{}
type UpdateSearchInputMsg struct{ Input string }
type ExecuteSearchMsg struct{}
type CancelSearchMsg struct{}
type NextMatchMsg struct{}
type PrevMatchMsg struct{}
type ClearSearchMsg struct{}
```

### Model Structure
Break down the monolithic model into focused sub-models:

```go
// Top-level application model
type AppModel struct {
    WindowSize    WindowSize
    ActivePane    PaneType
    LeftPane      LeftPaneModel
    RightPane     RightPaneModel
    Search        SearchModel
    Items         []*StackItem
}

// Left pane (item list) model
type LeftPaneModel struct {
    Cursor      int
    Selected    int
    Width       int
    Height      int
}

// Right pane (content viewer) model
type RightPaneModel struct {
    Width       int
    Height      int
    ViewPos     int
    Content     *StackItem
}

// Search functionality model
type SearchModel struct {
    Active      bool
    Input       string
    Pattern     string
    Error       string
    Matches     []int
    CurrentMatch int
}
```

### Generic Component Interface
Each component implements a generic interface with its specific message type:

```go
// Generic component interface
type Component[M any] interface {
    View() string
    Update(M) error
}

// Component implementations
type LeftPaneComponent struct {
    Model LeftPaneModel
    Items []*StackItem
}

type RightPaneComponent struct {
    Model RightPaneModel
}

type SearchComponent struct {
    Model SearchModel
}

// App handles message routing and orchestration
type AppComponent struct {
    Model      AppModel
    LeftPane   LeftPaneComponent
    RightPane  RightPaneComponent
    Search     SearchComponent
}

// AppComponent routes messages to appropriate sub-components
func (a *AppComponent) Update(msg tea.Msg) error {
    // Parse tea.Msg into component-specific messages
    // Route to appropriate component
}
```

## Sub-Component Breakdown

### 1. Core Components
- **`AppComponent`**: Top-level orchestrator, handles window events and focus management
- **`LeftPaneComponent`**: Manages item list navigation and selection
- **`RightPaneComponent`**: Manages content viewing, scrolling, and display
- **`SearchComponent`**: Manages search input, execution, and match navigation

### 2. File Structure
Reorganize code into focused files:

```
internal/tui/
├── app.go              # Main AppModel, Update, and View
├── messages.go         # All message type definitions
├── leftpane.go         # LeftPaneModel and related functions
├── rightpane.go        # RightPaneModel and related functions
├── search.go           # SearchModel and search functionality
├── components.go       # Shared component interfaces and utilities
├── styles.go           # Lipgloss styles and theming
└── utils.go            # Utility functions (wrapText, min, max, etc.)
```

### 3. Component Responsibilities

#### AppComponent
- Window size management
- Focus management between panes
- Message routing to sub-components
- Overall layout orchestration

#### LeftPaneComponent
- Item navigation (up/down/select)
- Visual selection highlighting
- Item preview rendering
- Focus visual indicators

#### RightPaneComponent
- Content scrolling (line-by-line, page, jump)
- Content rendering with line wrapping
- Search result highlighting
- Scroll position indicators

#### SearchComponent
- Search input handling
- Pattern compilation and validation
- Match finding and navigation
- Search result visualization

## Incremental Implementation Plan

### Phase 1: Search Component (Standalone)
**Why First**: Self-contained, minimal dependencies, clear boundaries
**Deliverables**:
- `search.go` - SearchModel, SearchMsg types, and SearchComponent
- `search_test.go` - Unit tests for search functionality
- Test integration with existing StackItem search methods

**Implementation Steps**:
1. Create SearchMsg interface and message types
2. Create SearchModel struct
3. Implement SearchComponent with Update/View methods
4. Write comprehensive unit tests
5. Run `go test ./internal/tui` to ensure no regressions

### Phase 2: Left Pane Component
**Why Second**: Simple navigation logic, depends only on item list
**Deliverables**:
- `leftpane.go` - LeftPaneModel, LeftPaneMsg types, and LeftPaneComponent
- `leftpane_test.go` - Navigation and selection tests
- Integration with StackItem list

**Implementation Steps**:
1. Create LeftPaneMsg interface and message types
2. Create LeftPaneModel struct
3. Implement LeftPaneComponent with Update/View methods
4. Write unit tests for navigation boundaries and selection
5. Run `go test ./internal/tui` to ensure functionality

### Phase 3: Right Pane Component
**Why Third**: More complex scrolling logic, depends on SearchComponent
**Deliverables**:
- `rightpane.go` - RightPaneModel, RightPaneMsg types, and RightPaneComponent
- `rightpane_test.go` - Scrolling, content display, and search integration tests
- Integration with SearchComponent for highlighting

**Implementation Steps**:
1. Create RightPaneMsg interface and message types
2. Create RightPaneModel struct
3. Implement RightPaneComponent with Update/View methods
4. Integrate with SearchComponent for match highlighting
5. Write tests for scrolling boundaries and content rendering
6. Run `go test ./internal/tui` to verify all components work

### Phase 4: App Component Integration
**Why Last**: Orchestrates all sub-components, message routing complexity
**Deliverables**:
- `app.go` - AppModel, AppMsg types, AppComponent, and message routing
- `app_test.go` - Integration tests for component interactions
- Updated main `viewer.go` to use new AppComponent

**Implementation Steps**:
1. Create AppMsg interface and routing message types
2. Create AppComponent that wraps all sub-components
3. Implement message routing from `tea.Msg` to component-specific messages
4. Update main Update/View methods to delegate to AppComponent
5. Write integration tests for focus changes and message routing
6. Run full test suite to ensure complete functionality

### Phase 5: Legacy Compatibility & Cleanup
**Deliverables**:
- Maintain existing public API for backward compatibility
- Remove old monolithic code from `viewer.go`
- Update package documentation and examples

**Implementation Steps**:
1. Ensure `NewModel()` and existing APIs still work
2. Remove unused code from `viewer.go`
3. Update imports and clean up file structure
4. Run full test suite including existing integration tests
5. Performance verification - no regressions

## Testing Strategy

### Unit Tests
- **Message Handling**: Test each update function with specific message types
- **Model State**: Test model initialization, validation, and state consistency
- **View Rendering**: Test component view functions with different model states
- **Search Logic**: Test search pattern compilation, matching, and navigation

### Integration Tests
- **Pane Interactions**: Test focus changes and message routing between panes
- **Search Integration**: Test end-to-end search workflow across components
- **Window Resize**: Test layout recalculation and content rewrapping

### Property-Based Tests
- **Navigation Bounds**: Test navigation doesn't exceed item list boundaries
- **Scroll Bounds**: Test scrolling doesn't exceed content boundaries
- **Search Consistency**: Test search results are consistent across operations

## Success Criteria

### Functional Requirements
- [ ] All existing TUI functionality works identically
- [ ] No regression in user experience or performance
- [ ] All keyboard shortcuts and interactions preserved

### Code Quality Requirements
- [ ] Explicit message types for all user actions
- [ ] Pure update functions (no in-place mutation)
- [ ] Component-based view architecture
- [ ] <100 lines per file (focused responsibility)
- [ ] >90% test coverage on business logic

### Maintainability Requirements
- [ ] Easy to add new features (new message types, components)
- [ ] Clear separation between UI, business logic, and state
- [ ] Self-documenting code structure and interfaces
- [ ] Consistent error handling and validation patterns

## Next Steps

1. **Start Phase 1**: Begin with message type definitions in `messages.go`
2. **Create Branch**: Create feature branch `feat/elm-architecture-refactor`
3. **Incremental Implementation**: Implement each phase with working code at every step
4. **Continuous Testing**: Run existing tests after each phase to ensure no regressions
5. **Documentation**: Update ARCHITECTURE.md as components are implemented

## Risk Mitigation

### Technical Risks
- **Breaking Changes**: Maintain backward compatibility through incremental refactoring
- **Performance Impact**: Profile rendering performance before and after changes
- **Test Coverage**: Ensure existing tests pass throughout refactoring

### Timeline Risks
- **Scope Creep**: Focus strictly on refactoring, not new features
- **Complexity**: Break work into small, reviewable chunks
- **Integration**: Test component interactions early and often

---

**Last Updated**: 2025-09-28
**Status**: Planning Complete, Ready for Implementation