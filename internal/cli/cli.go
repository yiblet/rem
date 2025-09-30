package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yiblet/rem/internal/config"
	"github.com/yiblet/rem/internal/queue"
	"github.com/yiblet/rem/internal/remfs"
	"github.com/yiblet/rem/internal/tui"
	"golang.design/x/clipboard"
)

// CLI handles the command-line interface
type CLI struct {
	stackManager *queue.StackManager
	filesystem   queue.FileSystem
}

// New creates a new CLI instance
func New() (*CLI, error) {
	return NewWithArgs(nil)
}

// NewWithArgs creates a new CLI instance with custom arguments for history location
func NewWithArgs(args *Args) (*CLI, error) {
	var historyPath string
	if args != nil && args.History != nil {
		historyPath = *args.History
	}

	// Create filesystem with custom history location
	remFS, err := remfs.NewWithHistoryPath(historyPath)
	if err != nil {
		return nil, fmt.Errorf("error creating rem filesystem: %w", err)
	}

	// Load configuration to get history limit
	configManager, err := config.NewConfigManager()
	if err != nil {
		return nil, fmt.Errorf("error creating config manager: %w", err)
	}

	cfg, err := configManager.Load()
	if err != nil {
		return nil, fmt.Errorf("error loading configuration: %w", err)
	}

	// Create stack manager with configured history limit
	sm, err := queue.NewStackManagerWithConfig(remFS, cfg.HistoryLimit)
	if err != nil {
		return nil, fmt.Errorf("error creating queue manager: %w", err)
	}

	return &CLI{
		stackManager: sm,
		filesystem:   remFS,
	}, nil
}

// Execute runs the CLI command based on parsed arguments
func (c *CLI) Execute(args *Args) error {
	if err := args.Validate(); err != nil {
		return err
	}

	switch {
	case args.Store != nil:
		return c.executeStore(args.Store)
	case args.Get != nil:
		return c.executeGet(args.Get)
	case args.Config != nil:
		return c.executeConfig(args.Config)
	case args.Clear != nil:
		return c.executeClear(args.Clear)
	case args.Search != nil:
		return c.executeSearch(args.Search)
	default:
		// Default behavior: launch TUI
		return c.launchTUI()
	}
}

// executeStore handles the 'rem store' command
func (c *CLI) executeStore(cmd *StoreCmd) error {
	var content io.ReadSeeker
	var err error

	switch {
	case cmd.Clipboard:
		// Read from clipboard
		content, err = c.readFromClipboard()
	case cmd.File != nil:
		// Read from file
		content, err = c.readFromFile(*cmd.File)
	default:
		// Read from stdin
		content, err = c.readFromStdin()
	}

	if err != nil {
		return fmt.Errorf("failed to read content: %w", err)
	}

	// Store content in stack
	item, err := c.stackManager.Push(content)
	if err != nil {
		return fmt.Errorf("failed to store content: %w", err)
	}

	fmt.Printf("Stored: %s\n", item.Preview)
	return nil
}

// executeGet handles the 'rem get' command
func (c *CLI) executeGet(cmd *GetCmd) error {
	if cmd.Index == nil {
		// No index specified, launch TUI
		return c.launchTUI()
	}

	index := *cmd.Index

	// Get item from stack
	item, err := c.stackManager.Get(index)
	if err != nil {
		return fmt.Errorf("failed to get item at index %d: %w", index, err)
	}

	// Get content reader
	reader, err := item.GetContentReader(c.filesystem)
	if err != nil {
		return fmt.Errorf("failed to read content: %w", err)
	}
	defer reader.Close()

	// Read all content
	content, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read content: %w", err)
	}

	switch {
	case cmd.Clipboard:
		// Copy to clipboard
		return c.writeToClipboard(content)
	case cmd.File != nil:
		// Write to file
		return c.writeToFile(*cmd.File, content)
	default:
		// Write to stdout
		_, err = os.Stdout.Write(content)
		return err
	}
}

// executeConfig handles the 'rem config' command
func (c *CLI) executeConfig(cmd *ConfigCmd) error {
	configManager, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	switch {
	case cmd.Get != nil:
		return c.executeConfigGet(configManager, cmd.Get)
	case cmd.Set != nil:
		return c.executeConfigSet(configManager, cmd.Set)
	case cmd.List != nil:
		return c.executeConfigList(configManager, cmd.List)
	default:
		return fmt.Errorf("no config subcommand specified")
	}
}

// executeConfigGet handles the 'rem config get' command
func (c *CLI) executeConfigGet(configManager *config.ConfigManager, cmd *ConfigGetCmd) error {
	value, err := configManager.Get(cmd.Key)
	if err != nil {
		return fmt.Errorf("failed to get config value: %w", err)
	}

	fmt.Printf("%s\n", value)
	return nil
}

// executeConfigSet handles the 'rem config set' command
func (c *CLI) executeConfigSet(configManager *config.ConfigManager, cmd *ConfigSetCmd) error {
	if err := configManager.Update(cmd.Key, cmd.Value); err != nil {
		return fmt.Errorf("failed to set config value: %w", err)
	}

	fmt.Printf("Set %s = %s\n", cmd.Key, cmd.Value)
	return nil
}

// executeConfigList handles the 'rem config list' command
func (c *CLI) executeConfigList(configManager *config.ConfigManager, cmd *ConfigListCmd) error {
	values, err := configManager.List()
	if err != nil {
		return fmt.Errorf("failed to list config values: %w", err)
	}

	fmt.Printf("Current configuration:\n")
	for key, value := range values {
		fmt.Printf("  %s = %s\n", key, value)
	}
	return nil
}

// launchTUI starts the interactive TUI
func (c *CLI) launchTUI() error {
	// Get items from stack
	stackItems, err := c.stackManager.List()
	if err != nil {
		return fmt.Errorf("error listing queue items: %w", err)
	}

	// Convert stack items to TUI items
	var tuiItems []*tui.StackItem
	for _, sItem := range stackItems {
		// Get content reader
		contentReader, err := sItem.GetContentReader(c.stackManager.FileSystem())
		if err != nil {
			fmt.Printf("Warning: Error getting content reader: %v\n", err)
			continue
		}

		tuiItem := &tui.StackItem{
			Content: contentReader,
			Preview: sItem.Preview,
			ViewPos: 0,
		}
		tuiItems = append(tuiItems, tuiItem)
	}

	// If no items in stack, show a helpful message
	if len(tuiItems) == 0 {
		fmt.Println("Stack is empty!")
		fmt.Println()
		fmt.Println("To add items to the stack:")
		fmt.Printf("  echo \"Hello World\" | rem store\n")
		fmt.Printf("  rem store filename.txt\n")
		fmt.Printf("  rem store -c  # from clipboard\n")
		return nil
	}

	model := tui.NewModel(tuiItems)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// readFromClipboard reads content from system clipboard
func (c *CLI) readFromClipboard() (io.ReadSeeker, error) {
	err := clipboard.Init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize clipboard: %w", err)
	}

	data := clipboard.Read(clipboard.FmtText)
	if len(data) == 0 {
		return nil, fmt.Errorf("clipboard is empty")
	}

	return strings.NewReader(string(data)), nil
}

// readFromFile reads content from a file
func (c *CLI) readFromFile(filename string) (io.ReadSeeker, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return strings.NewReader(string(data)), nil
}

// readFromStdin reads content from stdin
func (c *CLI) readFromStdin() (io.ReadSeeker, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no input provided")
	}

	return strings.NewReader(string(data)), nil
}

// writeToClipboard writes content to system clipboard
func (c *CLI) writeToClipboard(content []byte) error {
	err := clipboard.Init()
	if err != nil {
		return fmt.Errorf("failed to initialize clipboard: %w", err)
	}

	clipboard.Write(clipboard.FmtText, content)
	fmt.Printf("Copied to clipboard: %s\n", c.truncatePreview(string(content)))
	return nil
}

// writeToFile writes content to a file
func (c *CLI) writeToFile(filename string, content []byte) error {
	err := os.WriteFile(filename, content, 0644)
	if err != nil {
		return err
	}

	fmt.Printf("Written to %s: %s\n", filename, c.truncatePreview(string(content)))
	return nil
}

// executeClear handles the 'rem clear' command
func (c *CLI) executeClear(cmd *ClearCmd) error {
	// Get current stack size
	items, err := c.stackManager.List()
	if err != nil {
		return fmt.Errorf("failed to list items: %w", err)
	}

	if len(items) == 0 {
		fmt.Println("Stack is already empty.")
		return nil
	}

	// Prompt for confirmation unless --force is used
	if !cmd.Force {
		fmt.Printf("This will delete %d item(s) from history. Continue? [y/N]: ", len(items))
		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Clear the stack by deleting all history files
	if err := c.stackManager.Clear(); err != nil {
		return fmt.Errorf("failed to clear history: %w", err)
	}

	fmt.Printf("Cleared %d item(s) from history.\n", len(items))
	return nil
}

// executeSearch handles the 'rem search' command
func (c *CLI) executeSearch(cmd *SearchCmd) error {
	// Get all items from stack
	items, err := c.stackManager.List()
	if err != nil {
		return fmt.Errorf("failed to list items: %w", err)
	}

	if len(items) == 0 {
		return fmt.Errorf("stack is empty")
	}

	// Search for pattern in items
	matches, err := c.stackManager.Search(cmd.Pattern)
	if err != nil {
		return fmt.Errorf("failed to search: %w", err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("no matches found for pattern: %s", cmd.Pattern)
	}

	// Handle different output modes
	if cmd.IndexOnly {
		// Output only the index of the first match
		fmt.Printf("%d\n", matches[0].Index)
		return nil
	}

	if cmd.AllMatches {
		// Show all matches
		for _, match := range matches {
			fmt.Printf("Index %d: %s\n", match.Index, match.Item.Preview)
		}
		return nil
	}

	// Default: output content of first match
	firstMatch := matches[0]
	reader, err := firstMatch.Item.GetContentReader(c.filesystem)
	if err != nil {
		return fmt.Errorf("failed to read content: %w", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read content: %w", err)
	}

	_, err = os.Stdout.Write(content)
	return err
}

// truncatePreview creates a truncated preview of content for display
func (c *CLI) truncatePreview(content string) string {
	const maxLength = 80

	// Replace newlines with spaces for preview
	preview := strings.ReplaceAll(content, "\n", " ")
	preview = strings.TrimSpace(preview)

	if len(preview) <= maxLength {
		return preview
	}

	return preview[:maxLength-3] + "..."
}
