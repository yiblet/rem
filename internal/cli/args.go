package cli

import (
	"fmt"
)

// Args represents the top-level command structure
type Args struct {
	Store *StoreCmd `arg:"subcommand:store" help:"Push content to the stack"`
	Get   *GetCmd   `arg:"subcommand:get" help:"Access content from the stack"`
}

// StoreCmd represents the 'rem store' command (pushes to top of stack)
type StoreCmd struct {
	File      *string `arg:"positional" help:"File to read from (optional)"`
	Clipboard bool    `arg:"-c,--clipboard" help:"Read from clipboard"`
}

// GetCmd represents the 'rem get' command (accesses stack by index)
type GetCmd struct {
	Index     *int    `arg:"positional" help:"Stack index to retrieve (0=top, optional, opens TUI if not provided)"`
	File      *string `arg:"positional" help:"Output file (optional)"`
	Clipboard bool    `arg:"-c,--clipboard" help:"Copy to clipboard"`
}

// Description returns the program description
func (Args) Description() string {
	return "rem - Enhanced clipboard stack manager with persistent LIFO stack"
}

// Version returns the program version
func (Args) Version() string {
	return "rem 0.1.0"
}

// Epilogue returns additional help text
func (Args) Epilogue() string {
	return `Examples:
  # Store operations
  echo "hello" | rem store          # Store from stdin
  rem store file.txt               # Store from file
  rem store -c                     # Store from clipboard

  # Get operations
  rem get                          # Interactive TUI browser
  rem get 0                        # Output first item to stdout
  rem get -c 1                     # Copy second item to clipboard
  rem get 2 output.txt             # Save third item to file

For more information, visit: https://github.com/yiblet/rem`
}

// Validate performs validation on the parsed arguments
func (args *Args) Validate() error {
	if args.Store != nil {
		return args.Store.Validate()
	}
	if args.Get != nil {
		return args.Get.Validate()
	}
	return nil
}

// Validate validates store command arguments
func (s *StoreCmd) Validate() error {
	if s.File != nil && s.Clipboard {
		return fmt.Errorf("cannot specify both file and clipboard input")
	}
	return nil
}

// Validate validates get command arguments
func (g *GetCmd) Validate() error {
	if g.Index != nil && *g.Index < 0 {
		return fmt.Errorf("index must be non-negative")
	}
	if g.File != nil && g.Clipboard {
		return fmt.Errorf("cannot specify both file and clipboard output")
	}
	return nil
}