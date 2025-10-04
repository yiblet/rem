# rem - Enhanced Clipboard Queue Manager

A powerful clipboard management tool that extends `pbcopy` and `pbpaste` with a persistent LIFO queue of clipboard history.

## Overview

`rem` maintains a configurable history of clipboard entries, allowing you to access and manage up to 255 items (configurable). It provides an intuitive interactive TUI with dual-pane viewing, powerful search capabilities, and seamless integration with your existing clipboard workflow.

## Features

- **Persistent LIFO Queue**: Maintains clipboard history in local storage with newest items at index 0
- **Interactive TUI Viewer**: Dual-pane interface with vim-style navigation and search
- **Configurable History**: Set custom history limits and storage locations
- **Multiple Input Methods**: Accept input from stdin, files, or clipboard
- **Multiple Output Methods**: Output to stdout, files, or clipboard
- **Advanced Search**: Regex-based search across entire history
- **History Management**: Clear all history or delete individual items
- **Binary File Support**: Detects and handles binary content appropriately
- **Titled Entries**: Add custom titles to stored items for easier identification
- **SQLite Storage**: Fast and reliable database-backed storage
- **Environment Variables**: Customize database location via `REM_DB_PATH`

## Installation

### Install via Pre-built Binary

```bash
# Linux (amd64)
curl -L https://github.com/yiblet/rem/releases/latest/download/rem-linux-amd64.tar.gz | tar xz
mv rem-linux-amd64 ~/.local/bin/rem  # If ~/.local/bin isn't in your $PATH, you should put rem somewhere else like /usr/local/bin

# Linux (arm64)
curl -L https://github.com/yiblet/rem/releases/latest/download/rem-linux-arm64.tar.gz | tar xz
mv rem-linux-arm64 ~/.local/bin/rem  # If ~/.local/bin isn't in your $PATH, you should put rem somewhere else like /usr/local/bin

# macOS (Intel)
curl -L https://github.com/yiblet/rem/releases/latest/download/rem-darwin-amd64.tar.gz | tar xz
mv rem-darwin-amd64 ~/.local/bin/rem  # If ~/.local/bin isn't in your $PATH, you should put rem somewhere else like /usr/local/bin

# macOS (Apple Silicon)
curl -L https://github.com/yiblet/rem/releases/latest/download/rem-darwin-arm64.tar.gz | tar xz
mv rem-darwin-arm64 ~/.local/bin/rem  # If ~/.local/bin isn't in your $PATH, you should put rem somewhere else like /usr/local/bin

# Windows
# Download from: https://github.com/yiblet/rem/releases/latest/download/rem-windows-amd64.exe.zip
```

### Install via Go

```bash
go install github.com/yiblet/rem@latest
```

## Quick Start

```bash
# Store some content
echo "Hello, world!" | rem store
rem store --title "My Note" file.txt

# Launch interactive viewer
rem get

# Or access items directly
rem get 0        # Output most recent item
rem get -c 1     # Copy second item to clipboard
```

## Usage

### Store Operations (Enqueue to Queue)

```bash
# Store from stdin (auto-generated title)
echo "content" | rem store
cat file.txt | rem store

# Store from files (supports multiple files)
rem store file.txt
rem store file1.txt file2.txt file3.txt

# Store with custom title
rem store --title "My Note" file.txt
rem store -t "Important" file1.txt file2.txt

# Store from clipboard
rem store -c
```

### Get Operations (Access Queue)

```bash
# Interactive TUI viewer (most common usage)
rem get

# Output to stdout
rem get 0     # Most recent item (top of queue)
rem get 1     # Second most recent item
rem get 5     # Sixth most recent item

# Copy to clipboard
rem get -c 0  # Copy most recent to clipboard
rem get -c 2  # Copy third item to clipboard

# Save to file
rem get 0 output.txt  # Save most recent to file
rem get 2 data.txt    # Save third item to file
```

### Configuration Management

```bash
# List all configuration settings
rem config list

# Get specific configuration value
rem config get history_limit

# Set configuration values
rem config set history_limit 100      # Set max items to 100
rem config set history_limit 50       # Set max items to 50
```

### Search History

```bash
# Search for pattern (shows first match content)
rem search 'error'
rem search 'error.*log'

# Show only index of first match
rem search -i 'pattern'

# Show all matching items (concatenated)
rem search -a 'TODO'

# Show indexes of all matching items
rem search -a -i 'pattern'

# Search in specific fields
rem search --title 'config'      # Search titles only
rem search --content 'password'  # Search content only

# Case-sensitive search
rem search -s 'CaseSensitive'
```

### History Management

```bash
# Clear all history (prompts for confirmation)
rem clear

# Clear without confirmation
rem clear --force
```

## Interactive TUI

The TUI provides a powerful dual-pane interface for browsing and searching history:

### Layout
- **Left Pane (25 chars)**: List view of all queue items with previews
- **Right Pane**: Full content viewer with text wrapping and search
- **Status Line**: Shows current mode, search status, and help info

### Keyboard Shortcuts

#### Global Commands
- `q` - Quit the viewer
- `z` - Toggle help screen
- `Tab` or `h`/`l` or `←`/`→` - Switch between panes

#### Left Pane (List Navigation)
- `j`/`k` or `↓`/`↑` - Move cursor down/up
- `g` - Jump to top of list
- `G` - Jump to bottom of list
- `Ctrl+d` - Page down (half page)
- `Ctrl+u` - Page up (half page)
- `d` - Delete current item (shows confirmation dialog)
- Number + `j`/`k` - Move by N items (e.g., `5j` moves down 5 items)

#### Right Pane (Content Viewing)
- `j`/`k` or `↓`/`↑` - Scroll down/up one line
- `g` - Jump to top of content
- `G` - Jump to bottom of content
- `Ctrl+d` - Scroll down half page
- `Ctrl+u` - Scroll up half page
- `Ctrl+f` - Scroll down full page
- `Ctrl+b` - Scroll up full page
- `/` - Enter search mode
- `n` - Jump to next search match
- `N` - Jump to previous search match
- Number + `j`/`k` - Scroll by N lines (e.g., `10j` scrolls down 10 lines)

#### Search Mode
- Type pattern and press `Enter` to search
- `Esc` to cancel search
- Search highlights all matches with current match emphasized
- Each item remembers its own scroll position and search state

### Configuration

Configuration is stored in the database and managed via the `rem config` command:

```bash
# View all settings
rem config list

# View specific setting
rem config get history_limit

# Update settings
rem config set history_limit 100
```

## Complete Examples

```bash
# Store workflow
echo "First item" | rem store
echo "Second item" | rem store
rem store --title "My Code" code.go
rem store -t "Config Files" file1.txt file2.txt
rem store -c  # Store current clipboard

# Access workflow
rem get                    # Browse interactively
rem get 0                  # Output most recent
rem get 1 > saved.txt      # Save second item
rem get -c 2               # Copy third item to clipboard

# Search workflow
rem search 'error.*log'        # Find first match
rem search -a 'TODO'           # Find all matches
rem search -a -i 'pattern'     # Show all match indexes
rem search --title 'config'    # Search titles only
rem search -s 'CaseSensitive'  # Case-sensitive search

# Configuration workflow
rem config set history_limit 100
rem config get history_limit
rem config list

# Database path
rem --db-path /custom/rem.db store file.txt
export REM_DB_PATH=/custom/rem.db  # Set via environment

# Maintenance
rem clear                  # Clear with confirmation
rem clear --force          # Clear without confirmation
```

## Use Cases

- **Code Development**: Store and retrieve code snippets, error messages, and API responses
- **Documentation**: Manage multiple text passages when writing docs

## Development

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/queue
go test ./internal/tui -v

# Build
go build -o rem

# Run demo (populates queue with test data)
go run ./cmd/demo/
```

## Queue Behavior (LIFO)

- **Position 0**: Top of queue - most recently added item
- **Positions 1-254**: Historical entries in reverse chronological order
- **Automatic Management**: Oldest entries removed when exceeding configured limit (default 255)
- **LIFO (Last In, First Out)**: Most recent content always at index 0

## Configuration and Storage

### Database Location

rem uses SQLite for reliable data storage. The database location follows this precedence:
1. `--db-path` CLI flag (highest priority)
2. `REM_DB_PATH` environment variable
3. Default: `~/.config/rem/rem.db` (lowest priority)

```bash
# Set custom location via environment variable
export REM_DB_PATH="$HOME/my-rem.db"
rem store < file.txt

# Or use CLI flag
rem --db-path /custom/rem.db store < file.txt
```

### Directory Structure

```
~/.config/rem/
└── rem.db                            # SQLite database with history
```

