package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yiblet/rem/internal/clipboard"
	"github.com/yiblet/rem/internal/clipboard/sysboard"
	"github.com/yiblet/rem/internal/queue"
	"github.com/yiblet/rem/internal/store"
	"github.com/yiblet/rem/internal/store/dbstore"
	"github.com/yiblet/rem/internal/tui"
)

// CLI handles the command-line interface
type CLI struct {
	queueManager *queue.QueueManager
	store        store.Store
	clipboard    clipboard.Clipboard
}

// New creates a new CLI instance
func New() (*CLI, error) {
	return NewWithArgs(nil)
}

// NewWithArgs creates a new CLI instance with custom arguments for database path
func NewWithArgs(args *Args) (*CLI, error) {
	// Determine database path (precedence: flag > env var > default)
	var dbPath string
	if args != nil && args.DBPath != nil {
		dbPath = *args.DBPath
	} else {
		// Use default path
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		dbPath = filepath.Join(homeDir, ".config", "rem", "rem.db")
	}

	// Ensure directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Create SQLite store
	sqliteStore, err := dbstore.NewSQLiteStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create database store: %w", err)
	}

	// Load history limit from config store
	historyLimit := queue.DefaultMaxQueueSize
	if limitStr, err := sqliteStore.Config().Get("history_limit"); err == nil {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			historyLimit = limit
		}
	}

	// Create queue manager with store
	qm, err := queue.NewQueueManagerWithConfig(sqliteStore, historyLimit)
	if err != nil {
		sqliteStore.Close()
		return nil, fmt.Errorf("failed to create queue manager: %w", err)
	}

	// Create system clipboard
	clip := sysboard.New()

	return &CLI{
		queueManager: qm,
		store:        sqliteStore,
		clipboard:    clip,
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
	// Get title if provided
	var title string
	if cmd.Title != nil {
		title = *cmd.Title
	}

	switch {
	case cmd.Clipboard:
		// Read from clipboard
		content, err := c.readFromClipboard()
		if err != nil {
			return fmt.Errorf("failed to read content: %w", err)
		}
		item, err := c.queueManager.Enqueue(content, title)
		if err != nil {
			return fmt.Errorf("failed to store content: %w", err)
		}
		fmt.Printf("Stored: %s\n", item.Title)
		return nil

	case len(cmd.Files) > 0:
		// Read from files
		for _, filename := range cmd.Files {
			content, err := c.readFromFile(filename)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", filename, err)
			}
			item, err := c.queueManager.Enqueue(content, title)
			content.Close() // Close file handle after enqueue
			if err != nil {
				return fmt.Errorf("failed to store content from %s: %w", filename, err)
			}
			fmt.Printf("Stored from %s: %s\n", filename, item.Title)
		}
		return nil

	default:
		// Read from stdin
		content, err := c.readFromStdin()
		if err != nil {
			return fmt.Errorf("failed to read content: %w", err)
		}
		item, err := c.queueManager.Enqueue(content, title)
		if err != nil {
			return fmt.Errorf("failed to store content: %w", err)
		}
		fmt.Printf("Stored: %s\n", item.Title)
		return nil
	}
}

// executeGet handles the 'rem get' command
func (c *CLI) executeGet(cmd *GetCmd) error {
	if cmd.Index == nil {
		// No index specified, launch TUI
		return c.launchTUI()
	}

	index := *cmd.Index

	// Get item from queue
	item, err := c.queueManager.Get(index)
	if err != nil {
		return fmt.Errorf("failed to get item at index %d: %w", index, err)
	}

	// Get content reader using ID
	reader, err := c.queueManager.GetContent(item.ID)
	if err != nil {
		return fmt.Errorf("failed to read content: %w", err)
	}
	defer reader.Close()

	switch {
	case cmd.Clipboard:
		// Copy to clipboard - stream directly without reading into memory
		return c.writeToClipboard(reader, item.Title)
	case cmd.File != nil:
		// Stream to file
		outFile, err := os.Create(*cmd.File)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, reader)
		if err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}

		// Display title
		fmt.Printf("Written to %s: %s\n", *cmd.File, item.Title)
		return nil
	default:
		// Stream to stdout
		_, err = io.Copy(os.Stdout, reader)
		return err
	}
}

// executeConfig handles the 'rem config' command
func (c *CLI) executeConfig(cmd *ConfigCmd) error {
	switch {
	case cmd.Get != nil:
		return c.executeConfigGet(cmd.Get)
	case cmd.Set != nil:
		return c.executeConfigSet(cmd.Set)
	case cmd.List != nil:
		return c.executeConfigList(cmd.List)
	default:
		return fmt.Errorf("no config subcommand specified")
	}
}

// executeConfigGet handles the 'rem config get' command
func (c *CLI) executeConfigGet(cmd *ConfigGetCmd) error {
	value, err := c.store.Config().Get(cmd.Key)
	if err != nil {
		return fmt.Errorf("failed to get config value: %w", err)
	}

	fmt.Printf("%s\n", value)
	return nil
}

// executeConfigSet handles the 'rem config set' command
func (c *CLI) executeConfigSet(cmd *ConfigSetCmd) error {
	// Validate the value based on the key
	switch cmd.Key {
	case "history_limit":
		// Validate it's a positive integer
		if limit, err := strconv.Atoi(cmd.Value); err != nil || limit <= 0 {
			return fmt.Errorf("history_limit must be a positive integer")
		}
	case "show_binary":
		// Validate it's a boolean
		if cmd.Value != "true" && cmd.Value != "false" {
			return fmt.Errorf("show_binary must be 'true' or 'false'")
		}
	}

	if err := c.store.Config().Set(cmd.Key, cmd.Value); err != nil {
		return fmt.Errorf("failed to set config value: %w", err)
	}

	fmt.Printf("Set %s = %s\n", cmd.Key, cmd.Value)
	return nil
}

// executeConfigList handles the 'rem config list' command
func (c *CLI) executeConfigList(cmd *ConfigListCmd) error {
	values, err := c.store.Config().List()
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
	// Get items from queue
	queueItems, err := c.queueManager.List()
	if err != nil {
		return fmt.Errorf("error listing queue items: %w", err)
	}

	// Convert queue items to TUI items
	var tuiItems []*tui.StackItem
	for _, item := range queueItems {
		// Get content reader using ID
		contentReader, err := c.queueManager.GetContent(item.ID)
		if err != nil {
			fmt.Printf("Warning: Error getting content reader for item %d: %v\n", item.ID, err)
			continue
		}

		// Capture ID for closure
		itemID := item.ID

		tuiItem := &tui.StackItem{
			ID:       fmt.Sprintf("%d", itemID),
			Content:  contentReader,
			Preview:  item.Title, // Use title as preview
			ViewPos:  0,
			IsBinary: item.IsBinary,
			Size:     item.Size,
			SHA256:   item.SHA256,
			DeleteFunc: func() error {
				// Delete by finding index of item with this ID
				items, err := c.queueManager.List()
				if err != nil {
					return err
				}
				for idx, itm := range items {
					if itm.ID == itemID {
						return c.queueManager.Delete(idx)
					}
				}
				return fmt.Errorf("item %d not found", itemID)
			},
		}
		tuiItems = append(tuiItems, tuiItem)
	}

	// If no items in queue, show a helpful message
	if len(tuiItems) == 0 {
		fmt.Println("Queue is empty!")
		fmt.Println()
		fmt.Println("To add items to the queue:")
		fmt.Printf("  echo \"Hello World\" | rem store\n")
		fmt.Printf("  rem store filename.txt\n")
		fmt.Printf("  rem store -c  # from clipboard\n")
		return nil
	}

	model := tui.NewModel(tuiItems, c.clipboard)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// readFromClipboard reads content from system clipboard
func (c *CLI) readFromClipboard() (io.ReadSeeker, error) {
	reader, err := c.clipboard.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read clipboard: %w", err)
	}
	defer reader.Close()

	// Read all content into memory to create a ReadSeeker
	// This is necessary because we need Seek capability for the queue manager
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read clipboard content: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("clipboard is empty")
	}

	return strings.NewReader(string(data)), nil
}

// readFromFile reads content from a file
// Returns the file handle directly for streaming - caller must close it
func (c *CLI) readFromFile(filename string) (*os.File, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	// Caller is responsible for closing the file
	return file, nil
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

// writeToClipboard writes content to system clipboard from a reader
func (c *CLI) writeToClipboard(r io.Reader, preview string) error {
	if err := c.clipboard.Write(r); err != nil {
		return fmt.Errorf("failed to write to clipboard: %w", err)
	}

	fmt.Printf("Copied to clipboard: %s\n", c.truncatePreview(preview))
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
	// Get current queue size
	items, err := c.queueManager.List()
	if err != nil {
		return fmt.Errorf("failed to list items: %w", err)
	}

	if len(items) == 0 {
		fmt.Println("Queue is already empty.")
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

	// Clear the queue
	if err := c.queueManager.Clear(); err != nil {
		return fmt.Errorf("failed to clear history: %w", err)
	}

	fmt.Printf("Cleared %d item(s) from history.\n", len(items))
	return nil
}

// executeSearch handles the 'rem search' command
func (c *CLI) executeSearch(cmd *SearchCmd) error {
	// Build search query
	searchQuery := &store.SearchQuery{
		Pattern:       cmd.Pattern,
		SearchTitle:   cmd.SearchTitle,
		SearchContent: cmd.SearchContent,
		CaseSensitive: cmd.CaseSensitive,
		Limit:         0, // No limit
	}

	// If AllMatches is false, limit to 1 result
	if !cmd.AllMatches {
		searchQuery.Limit = 1
	}

	// Perform search using store
	results, err := c.store.History().Search(searchQuery)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		return fmt.Errorf("no matches found for pattern: %s", cmd.Pattern)
	}

	// Get all items to find indexes of matched items
	allItems, err := c.queueManager.List()
	if err != nil {
		return fmt.Errorf("failed to list items: %w", err)
	}

	// Create a map of ID to index
	idToIndex := make(map[uint]int)
	for idx, item := range allItems {
		idToIndex[item.ID] = idx
	}

	// Output results
	for i, result := range results {
		index, ok := idToIndex[result.ID]
		if !ok {
			// Item was deleted between search and now
			continue
		}

		if cmd.IndexOnly {
			fmt.Printf("%d\n", index)
		} else {
			if i > 0 {
				fmt.Println()
			}
			// Get content reader using ID
			reader, err := c.queueManager.GetContent(result.ID)
			if err != nil {
				return fmt.Errorf("failed to read content for match %d: %w", i, err)
			}
			if _, err := io.Copy(os.Stdout, reader); err != nil {
				reader.Close()
				return fmt.Errorf("failed to write content for match %d: %w", i, err)
			}
			reader.Close()
		}
	}

	return nil
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
