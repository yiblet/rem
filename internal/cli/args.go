package cli

import (
	"fmt"
)

// Args represents the top-level command structure
type Args struct {
	Store   *StoreCmd  `arg:"subcommand:store" help:"Push content to the queue"`
	Get     *GetCmd    `arg:"subcommand:get" help:"Access content from the queue"`
	Config  *ConfigCmd `arg:"subcommand:config" help:"Manage rem configuration"`
	Clear   *ClearCmd  `arg:"subcommand:clear" help:"Clear all history from the queue"`
	Search  *SearchCmd `arg:"subcommand:search" help:"Search history for content matching a regex pattern"`
	History *string    `arg:"--history,env:REM_HISTORY" help:"Custom history directory location (overrides default ~/.config/rem/history/)"`
}

// StoreCmd represents the 'rem store' command (pushes to top of queue)
type StoreCmd struct {
	Files     []string `arg:"positional" help:"Files to read from (optional)"`
	Clipboard bool     `arg:"-c,--clipboard" help:"Read from clipboard"`
}

// GetCmd represents the 'rem get' command (accesses queue by index)
type GetCmd struct {
	Index     *int    `arg:"positional" help:"Queue index to retrieve (0=top, optional, opens TUI if not provided)"`
	File      *string `arg:"positional" help:"Output file (optional)"`
	Clipboard bool    `arg:"-c,--clipboard" help:"Copy to clipboard"`
}

// ConfigCmd represents the 'rem config' command (manages configuration)
type ConfigCmd struct {
	Get  *ConfigGetCmd  `arg:"subcommand:get" help:"Get configuration value"`
	Set  *ConfigSetCmd  `arg:"subcommand:set" help:"Set configuration value"`
	List *ConfigListCmd `arg:"subcommand:list" help:"List all configuration values"`
}

// ConfigGetCmd represents the 'rem config get' command
type ConfigGetCmd struct {
	Key string `arg:"positional,required" help:"Configuration key to get (history-limit, show-binary, history-location)"`
}

// ConfigSetCmd represents the 'rem config set' command
type ConfigSetCmd struct {
	Key   string `arg:"positional,required" help:"Configuration key to set (history-limit, show-binary, history-location)"`
	Value string `arg:"positional,required" help:"Configuration value to set"`
}

// ConfigListCmd represents the 'rem config list' command
type ConfigListCmd struct {
}

// ClearCmd represents the 'rem clear' command (clears all history)
type ClearCmd struct {
	Force bool `arg:"-f,--force" help:"Skip confirmation prompt"`
}

// SearchCmd represents the 'rem search' command (searches history)
type SearchCmd struct {
	Pattern    string `arg:"positional,required" help:"Regex pattern to search for"`
	IndexOnly  bool   `arg:"-i,--index-only" help:"Output only the index of the first match"`
	AllMatches bool   `arg:"-a,--all" help:"Show all matching items (not just the first)"`
}

// Description returns the program description
func (Args) Description() string {
	return "rem - Enhanced clipboard queue manager with persistent LIFO queue"
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
  rem store file1.txt file2.txt    # Store multiple files
  rem store -c                     # Store from clipboard

  # Get operations
  rem get                          # Interactive TUI browser
  rem get 0                        # Output first item to stdout
  rem get -c 1                     # Copy second item to clipboard
  rem get 2 output.txt             # Save third item to file

  # Configuration operations
  rem config list                  # List all configuration values
  rem config get history-limit     # Get specific configuration value
  rem config set history-limit 50  # Set configuration value

  # History management
  rem clear                        # Clear all history (with confirmation)
  rem clear --force                # Clear all history without confirmation
  rem search 'error.*log'          # Search for regex pattern in history
  rem search -i 'pattern'          # Output only the index of first match
  rem search -a 'pattern'          # Show all matching items

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
	if args.Config != nil {
		return args.Config.Validate()
	}
	if args.Clear != nil {
		return args.Clear.Validate()
	}
	if args.Search != nil {
		return args.Search.Validate()
	}
	return nil
}

// Validate validates store command arguments
func (s *StoreCmd) Validate() error {
	if len(s.Files) > 0 && s.Clipboard {
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

// Validate validates config command arguments
func (c *ConfigCmd) Validate() error {
	// Exactly one subcommand must be provided
	subCmdCount := 0
	if c.Get != nil {
		subCmdCount++
	}
	if c.Set != nil {
		subCmdCount++
	}
	if c.List != nil {
		subCmdCount++
	}

	if subCmdCount == 0 {
		return fmt.Errorf("config command requires a subcommand: get, set, or list")
	}
	if subCmdCount > 1 {
		return fmt.Errorf("config command accepts only one subcommand")
	}

	// Validate specific subcommands
	if c.Get != nil {
		return c.Get.Validate()
	}
	if c.Set != nil {
		return c.Set.Validate()
	}
	if c.List != nil {
		return c.List.Validate()
	}

	return nil
}

// Validate validates config get command arguments
func (g *ConfigGetCmd) Validate() error {
	validKeys := []string{"history-limit", "show-binary", "history-location"}
	for _, validKey := range validKeys {
		if g.Key == validKey {
			return nil
		}
	}
	return fmt.Errorf("invalid configuration key '%s', valid keys are: %v", g.Key, validKeys)
}

// Validate validates config set command arguments
func (s *ConfigSetCmd) Validate() error {
	validKeys := []string{"history-limit", "show-binary", "history-location"}
	for _, validKey := range validKeys {
		if s.Key == validKey {
			return nil
		}
	}
	return fmt.Errorf("invalid configuration key '%s', valid keys are: %v", s.Key, validKeys)
}

// Validate validates config list command arguments
func (l *ConfigListCmd) Validate() error {
	// No validation needed for list command
	return nil
}

// Validate validates clear command arguments
func (c *ClearCmd) Validate() error {
	// No specific validation needed for clear command
	return nil
}

// Validate validates search command arguments
func (s *SearchCmd) Validate() error {
	if s.Pattern == "" {
		return fmt.Errorf("search pattern cannot be empty")
	}
	return nil
}
