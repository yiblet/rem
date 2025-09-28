# rem - Enhanced Clipboard Stack Manager

A powerful clipboard management tool that extends `pbcopy` and `pbpaste` with a persistent stack of clipboard history.

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
rem view
```

The interactive viewer displays:
- **Left Pane**: Numbered list of clipboard entries showing the beginning of each entry
- **Right Pane**: Full content of the selected entry with paging support

### Access Specific Entry

```bash
# Output the nth entry from the stack (0-indexed)
rem view 0    # Outputs the most recent entry (same as pbpaste)
rem view 1    # Outputs the second most recent entry
rem view 5    # Outputs the 6th most recent entry
```

### Add to Stack

```bash
# Pipe content directly into the stack
echo "test content" | rem in

# Add current clipboard content to stack
rem clip
```

## Stack Behavior

- **Position 0**: Always contains the current clipboard content (equivalent to `pbpaste`)
- **Positions 1-19**: Historical clipboard entries, with newer entries pushing older ones down
- **Automatic Management**: Oldest entries are automatically removed when the stack exceeds 20 items

## Configuration

The clipboard history is stored in a local configuration folder, typically located at:
- macOS: `~/.config/rem/`
- Linux: `~/.config/rem/`

## Examples

```bash
# Copy something to clipboard
echo "Hello World" | pbcopy

# Add it to rem stack
rem clip

# Copy something else
echo "Second entry" | pbcopy

# View the stack interactively
rem view

# Get the previous entry without interactive mode
rem view 1  # Outputs "Hello World"

# Pipe new content directly
echo "Direct input" | rem in

# Now position 0 has "Direct input", position 1 has "Second entry"
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