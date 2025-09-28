# rem - Enhanced Clipboard Stack Manager

A powerful clipboard management tool that extends `pbcopy` and `pbpaste` with a persistent LIFO stack of clipboard history.

## Overview

`rem` maintains a configurable history of clipboard entries, allowing you to access and manage the last 20 items copied to your clipboard. It provides an intuitive CLI interface with dual-pane viewing and seamless integration with your existing clipboard workflow.

## Features

- **Persistent Stack**: Maintains the last 20 clipboard entries in a local config folder
- **Interactive Viewer**: Dual-pane CLI interface for browsing clipboard history
- **Seamless Integration**: Works alongside existing `pbcopy`/`pbpaste` workflows
- **Multiple Input Methods**: Accept input from pipes, clipboard, or direct commands

## Installation

```bash
# Installation instructions will be added once implementation is complete
```

## Usage

### View Clipboard Stack

```bash
# Launch interactive dual-pane viewer
rem get
# or simply
rem
```

The interactive viewer displays:
- **Left Pane**: Numbered list of clipboard entries showing the beginning of each entry
- **Right Pane**: Full content of the selected entry with paging support

### Access Specific Entry

```bash
# Output the nth entry from the stack (0-indexed)
rem get 0     # Outputs the most recent entry (top of stack)
rem get 1     # Outputs the second most recent entry
rem get 5     # Outputs the 6th most recent entry

# Copy to clipboard
rem get -c 0  # Copy most recent entry to clipboard
rem get -c 2  # Copy third most recent entry to clipboard

# Save to file
rem get 0 output.txt  # Save most recent entry to file
```

### Add to Stack

```bash
# Push content from stdin to top of stack
echo "test content" | rem store

# Push file content to stack
rem store filename.txt

# Push current clipboard content to stack
rem store -c
```

## Stack Behavior (LIFO)

- **Position 0**: Top of stack - most recently added item
- **Positions 1-19**: Historical entries in reverse chronological order (newer items push older ones down)
- **Automatic Management**: Oldest entries are automatically removed when the stack exceeds 20 items
- **LIFO (Last In, First Out)**: Most recent content is always accessible at index 0

## Configuration

The clipboard history is stored in a local configuration folder, typically located at:
- macOS: `~/.config/rem/`
- Linux: `~/.config/rem/`

## Examples

```bash
# Push content to stack from stdin
echo "Hello World" | rem store

# Push another item
echo "Second entry" | rem store

# View the stack interactively
rem get
# or simply: rem

# Get specific entries without interactive mode
rem get 0     # Outputs "Second entry" (most recent)
rem get 1     # Outputs "Hello World" (previous)

# Push content from clipboard
rem store -c

# Push content from file
rem store myfile.txt

# Copy stack item to clipboard
rem get -c 0  # Copy top of stack to clipboard
```

## Use Cases

- **Code Development**: Keep track of multiple code snippets while working
- **Documentation**: Manage multiple text passages when writing
- **Data Analysis**: Maintain a history of copied data or commands
- **General Productivity**: Never lose that important text you copied earlier

## Contributing

[Contributing guidelines will be added]

## License

[License information will be added]