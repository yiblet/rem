# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

rem is a LIFO clipboard stack manager that enhances `pbcopy`/`pbpaste` with persistent clipboard history. It provides both CLI commands and an interactive TUI for managing clipboard content.

## Development Commands

### Build and Test
```bash
# Build the project
go build

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./internal/queue ./internal/tui

# Run a specific test
go test -run TestQueueManager_Basic ./internal/queue

# Test specific package
go test ./internal/queue
```

### Running the Application
```bash
# Build and run CLI commands
./rem store < input.txt        # Store from stdin
./rem store filename.txt       # Store from file
./rem store -c                 # Store from clipboard
./rem get                      # Launch interactive TUI
./rem get 0                    # Get top stack item
./rem get -c 1                 # Copy second item to clipboard

# Demo commands for testing
go run ./cmd/demo/             # Populate stack with test data
go run ./cmd/test-integration/ # Test TUI rendering
```

## Architecture Overview

### Core Components

**4-Package Internal Architecture:**
- `internal/queue/` - LIFO stack management with file persistence
- `internal/cli/` - Command-line interface using go-arg
- `internal/tui/` - Interactive dual-pane TUI using Bubble Tea
- `internal/remfs/` - fs.FS abstraction for testable filesystem operations

### Key Design Principles

**LIFO Stack Model**: Items are stored newest-first (index 0 = most recent). The stack auto-cleans after 20 items.

**Stream-Based Content**: All content is modeled as `io.ReadSeekCloser` for memory efficiency and consistent interfaces across input sources.

**fs.FS Abstraction**: The `remfs` package provides a testable filesystem interface rooted at `~/.config/rem/`, enabling in-memory testing.

**Legacy Compatibility**: Type aliases maintain backward compatibility:
- `QueueManager` → `StackManager`
- `QueueItem` → `StackItem`
- `Enqueue()` → `Push()`

### Data Flow

**Storage Path**: Content → StackManager.Push() → ISO timestamp files in `~/.config/rem/content/`

**Retrieval Path**: StackManager.Get(index) → File reader → Stream content to CLI/TUI

**TUI Integration**: CLI and TUI share the same StackManager, with TUI converting StackItems to UI-specific types.

### File Persistence

Content files use ISO timestamp names with microsecond precision for ordering:
```
~/.config/rem/content/
├── 2025-09-28T10-15-30.123456-07-00.txt
├── 2025-09-28T10-16-45.789012-07-00.txt
```

The newest files (latest timestamps) correspond to stack index 0.

## Important Implementation Details

### Testing Strategy
- `internal/queue` has comprehensive tests for core stack operations
- `internal/tui` tests UI rendering and dual-pane layout
- Uses in-memory filesystem for isolated unit tests
- Tests validate LIFO behavior and size limits

### Constants and Configuration
- `MaxStackSize = 20` - Maximum items in stack before auto-cleanup
- Content directory: `content/` within rem config directory
- Preview length: ~50 characters with truncation

### TUI Key Bindings
- Left pane: j/k navigation, Tab to switch panes
- Right pane: j/k scroll, `/` search, n/N next/prev match, g/G top/bottom
- Global: q to quit

### Error Handling
CLI provides clean error messages without emojis. All user-facing output is text-only for terminal compatibility.

## Documentation Maintenance (CRITICAL)

**MUST ALWAYS KEEP UP-TO-DATE**: As you work on this project, you MUST maintain these two critical documentation files:

### PLAN.md - Current Feature and Project Status
- **Purpose**: Maintains information about the current feature being worked on, project phases, and implementation progress
- **Contents**: Current project status, phase completion tracking, success criteria, next steps, and immediate action items
- **Update when**: Starting new features, completing phases, changing project direction, or updating implementation status
- **Critical for**: Understanding where the project stands and what needs to be done next

### ARCHITECTURE.md - High-Level Project Design
- **Purpose**: Documents the high-level architecture and design decisions for the entire project
- **Contents**: System overview, component relationships, design principles, data flow, and architectural patterns
- **Update when**: Adding new packages, changing core designs, modifying data flow, or implementing new architectural patterns
- **Critical for**: Understanding how all the pieces fit together and making consistent design decisions

**These files are the project's source of truth for current state and overall design. Never make significant changes without updating them accordingly.**
