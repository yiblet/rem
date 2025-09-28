# rem - Enhanced Clipboard Stack Manager Architecture

## Overview

`rem` is a powerful clipboard management tool that extends `pbcopy` and `pbpaste` with a persistent FIFO queue and interactive TUI viewer. It provides seamless integration with existing clipboard workflows while adding advanced features like search, position tracking, and multi-format content handling.

## Core Concept

Unlike traditional clipboard managers that replace the clipboard, `rem` works as a **queue-based clipboard enhancer**:

- **Queue Model (FIFO)**: Items enter at the back, can be retrieved from any position
- **Stream-Based Content**: All content is modeled as `io.ReadSeekCloser` for efficient handling
- **Position Memory**: Each item remembers scroll position and search state independently
- **Non-Destructive Operations**: `get` operations are peek operations by default

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                           rem CLI                                │
├─────────────────────────────────────────────────────────────────┤
│  store        │  get          │  Interactive Viewer (rem/rem get) │
│  (Input Ops)  │  (Output Ops) │  (TUI Browser)                    │
└─────────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────────┐
│                      Core Queue Manager                         │
│  - Queue Operations (FIFO)                                     │
│  - Persistence Layer                                           │
│  - Content Type Detection                                      │
│  - Metadata Management                                         │
└─────────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────────┐
│                   Storage & Content Layer                       │
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │  File Storage   │  │   Clipboard     │  │   Network       │ │
│  │  ~/.config/rem/ │  │   Integration   │  │   Streams       │ │
│  │                 │  │                 │  │   (Future)      │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────────┐
│                    Interactive TUI Layer                        │
│                                                                 │
│  ┌──────────────┐              ┌─────────────────────────────┐  │
│  │ Left Pane    │              │ Right Pane                  │  │
│  │ Queue List   │◄────────────►│ Content Viewer & Pager     │  │
│  │ - Navigation │              │ - Text Wrapping            │  │
│  │ - Previews   │              │ - Search & Highlighting    │  │
│  │ - Selection  │              │ - Position Memory          │  │
│  └──────────────┘              └─────────────────────────────┘  │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ Status Line                                                 │ │
│  │ - Command Input (Search)                                   │ │
│  │ - Status & Help                                            │ │
│  │ - Error Messages                                           │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Component Architecture

### 1. CLI Interface Layer

#### Command Structure
```bash
# Input Operations (Queue Addition)
rem store                    # Read from stdin
rem store -c                 # Read from clipboard
rem store file.txt          # Read from file

# Output Operations (Queue Retrieval)
rem get                     # Interactive TUI viewer
rem get N                   # Output entry N to stdout
rem get -c N                # Copy entry N to clipboard
rem get N file.txt          # Write entry N to file

# Interactive Browser
rem                         # Launch TUI (same as rem get)
```

#### Verb Design Philosophy
- **`store`**: Explicit about input source and direction
- **`get`**: Explicit about output destination and format
- **Context flags**: `-c` for clipboard operations
- **Positional arguments**: Files for input/output
- **No arguments**: Default behaviors (stdin for store, TUI for get)

### 2. Core Queue Manager

#### Current Queue Implementation
```go
type QueueManager struct {
    fs FileSystem  // fs.FS abstraction rooted at config directory
}

type QueueItem struct {
    Filename    string    // ISO timestamp-based filename
    Timestamp   time.Time // microsecond precision
    Preview     string    // first ~100 chars for display
}

type FileSystem interface {
    fs.FS
    WriteFile(name string, data []byte, perm fs.FileMode) error
    Remove(name string) error
    MkdirAll(path string, perm fs.FileMode) error
}
```

#### Implementation Details
- **File-based persistence**: Each item stored as individual file with ISO timestamp name
- **fs.FS abstraction**: RemFS package provides filesystem rooted at `~/.config/rem/`
- **Content directory**: All queue items stored in `content/` subdirectory
- **Microsecond precision**: Filenames use RFC3339 with microsecond precision to prevent collisions
- **Auto-cleanup**: Automatic removal of oldest items when exceeding 20 item limit

#### FIFO Queue Operations
- **Enqueue**: Add items to back of queue
- **Peek**: Non-destructive access to any position
- **Auto-eviction**: Remove oldest items when queue exceeds size limit
- **Persistence**: Automatic save/restore of queue state

### 3. Content Abstraction Layer

#### Stream-Based Content Model
```go
type ContentReader interface {
    io.ReadSeekCloser
    GetMetadata() ItemMetadata
}

// Implementations
type FileContentReader struct { ... }      // For file input
type ClipboardContentReader struct { ... } // For clipboard input
type StringContentReader struct { ... }    // For string/stdin input
```

#### Benefits of Stream Model
- **Efficient Memory Usage**: Large files aren't fully loaded into memory
- **Consistent Interface**: Same operations work on files, clipboard, network streams
- **Position Tracking**: Seek operations enable position memory
- **Extensible**: Easy to add new content sources

### 4. Persistence Layer

#### Current Storage Structure
```
~/.config/rem/
└── content/                           # Queue content storage
    ├── 2025-09-27T20-01-11.787997-07-00.txt  # ISO timestamp files
    ├── 2025-09-27T20-01-11.788268-07-00.txt
    └── ...
```

#### Implemented Persistence Strategy
- **File-based storage**: Each queue item as individual timestamped file
- **ISO timestamp naming**: RFC3339 format with microsecond precision
- **RemFS abstraction**: Filesystem interface rooted at `~/.config/rem/`
- **Lazy loading**: Content accessed via `io.ReadSeekCloser` on demand
- **Auto-cleanup**: Oldest files removed when queue exceeds 20 items
- **Testable design**: In-memory filesystem for unit tests

### 5. Interactive TUI Layer

#### Current Architecture Pattern
```go
type Model struct {
    // Queue items with stream-based content
    items       []*QueueItem
    cursor      int
    selected    int

    // UI State
    focusedPane int  // 0=left, 1=right
    dimensions  Dimensions

    // Modal State
    mode        Mode
    searchInput string
    searchError string
}

type QueueItem struct {
    Content       io.ReadSeekCloser
    Preview       string
    Lines         []string // cached wrapped lines
    ViewPos       int      // current view position
    SearchPattern string   // current search pattern
    SearchMatches []int    // line numbers with matches
    SearchIndex   int      // current match index
}
```

#### Dual-Pane Design
- **Left Pane (25 chars)**: Queue browser with previews
- **Right Pane (Flexible)**: Full content viewer with pager functionality
- **Status Line**: Command input, search feedback, help text

#### Pager Features (Less-Compatible)
- **Navigation**: j/k, g/G, Ctrl+u/d, Ctrl+b/f, #j/#k
- **Search**: `/pattern`, `n`/`N` for next/prev match
- **Highlighting**: Regex matches with current match emphasis
- **Position Memory**: Each item remembers scroll position independently

## Data Flow

### Store Operation Flow
```
Input Source → Content Detection → Queue Addition → Persistence → Preview Generation
     │               │                    │              │              │
   stdin         ContentType         QueueManager    Database      TUI Update
   file          Validation          .Enqueue()      .Save()       .Refresh()
   clipboard     Size Limits
```

### Get Operation Flow
```
User Request → Queue Lookup → Content Retrieval → Output Formatting → Destination
     │              │              │                    │               │
   rem get 5    QueueManager    ContentReader      Format Selection   stdout
   TUI Nav      .Get(index)     .Seek()/.Read()    Text/Binary       clipboard
                                                                      file
```

### Interactive Browsing Flow
```
TUI Input → Mode Handling → State Update → Content Rendering → Display
    │            │             │              │                 │
  Key Press   HandleKey()   Model Update   Pager Logic      View()
  /search     SearchMode    ViewState      Text Wrap        Render
  j/k nav     NavMode       Selection      Highlighting     Screen
```

## Design Principles

### 1. **Explicit Over Implicit**
- Clear command verbs (`store`, `get`) vs ambiguous operations
- Explicit source/destination flags (`-c`, file arguments)
- Predictable behavior with minimal magic

### 2. **Stream-First Architecture**
- All content modeled as `io.ReadSeekCloser`
- Lazy loading for memory efficiency
- Position tracking for enhanced UX

### 3. **Non-Destructive Operations**
- `get` operations are peeks, not pops
- Queue persists across sessions
- Undo-friendly design

### 4. **Unix Philosophy Compatibility**
- Works with existing clipboard workflows
- Pipe-friendly CLI interface
- Small, focused, composable operations

### 5. **Performance-Conscious Design**
- Lazy content loading
- Efficient text wrapping and search
- Minimal memory footprint for large content

## Security Considerations

### Content Isolation
- Content stored in user-specific config directory
- No network operations by default
- Safe handling of binary content

### Input Validation
- Size limits on stored content
- Content type validation
- Regex pattern validation for search

### Privacy
- Local-only storage by default
- Optional cleanup/expiration policies
- No telemetry or external communication

## Current Implementation Status

### ✅ Completed Components
1. **Core Queue Manager**: File-based persistence with auto-cleanup (`internal/queue/`)
2. **RemFS Abstraction**: fs.FS interface rooted at config directory (`internal/remfs/`)
3. **Interactive TUI**: Dual-pane viewer with search and navigation (`internal/tui/`)
4. **Stream Architecture**: io.ReadSeekCloser-based content model
5. **Testing Framework**: In-memory filesystem for unit tests
6. **Integration Testing**: TUI working with real queue data

### 🚧 Next Phase: CLI Commands
1. **`rem store` operations**: stdin, file, clipboard input
2. **`rem get` operations**: stdout, clipboard, file output
3. **Command parsing**: Argument handling and validation
4. **Clipboard integration**: Platform-specific clipboard APIs

### 🔮 Future Enhancements
1. **Configuration system**: Queue size limits and preferences
2. **Advanced content handling**: Binary content, syntax highlighting
3. **Network integration**: HTTP/HTTPS content sources
4. **Encryption**: Optional encryption for sensitive content
5. **Plugin system**: Custom content processors
6. **Full-text search**: Indexing for large content collections

### Plugin Architecture (Future)
```go
type ContentProcessor interface {
    CanHandle(contentType string) bool
    Process(content io.ReadSeeker) (io.ReadSeekCloser, error)
    GeneratePreview(content io.ReadSeeker) string
}
```

This architecture provides a solid foundation built on fs.FS abstraction for testability, with a working TUI and queue manager ready for CLI command implementation.