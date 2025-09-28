package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yiblet/rem/internal/queue"
	"github.com/yiblet/rem/internal/remfs"
	"github.com/yiblet/rem/internal/tui"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "debug" {
		// Keep the old debug functionality for testing
		fmt.Println("Use the interactive mode instead: just run 'go run main.go'")
		return
	}

	// Create filesystem rooted at rem config directory
	remFS, err := remfs.New()
	if err != nil {
		fmt.Printf("Error creating rem filesystem: %v\n", err)
		os.Exit(1)
	}

	// Create queue manager
	qm, err := queue.NewQueueManager(remFS)
	if err != nil {
		fmt.Printf("Error creating queue manager: %v\n", err)
		os.Exit(1)
	}

	// Get items from queue
	queueItems, err := qm.List()
	if err != nil {
		fmt.Printf("Error listing queue items: %v\n", err)
		os.Exit(1)
	}

	// Convert queue items to TUI items
	var tuiItems []*tui.QueueItem
	for _, qItem := range queueItems {
		// Get content reader
		contentReader, err := qItem.GetContentReader(qm.FileSystem())
		if err != nil {
			fmt.Printf("Error getting content reader: %v\n", err)
			continue
		}

		tuiItem := &tui.QueueItem{
			Content: contentReader,
			Preview: qItem.Preview,
			ViewPos: 0,
		}
		tuiItems = append(tuiItems, tuiItem)
	}

	// If no items in queue, show a helpful message
	if len(tuiItems) == 0 {
		fmt.Println("📭 Queue is empty!")
		fmt.Println()
		fmt.Println("To add items to the queue:")
		fmt.Printf("  echo \"Hello World\" | rem store\n")
		fmt.Printf("  rem store filename.txt\n")
		fmt.Printf("  rem store -c  # from clipboard\n")
		fmt.Println()
		fmt.Println("For now, run the demo to add some content:")
		fmt.Printf("  go run ./cmd/demo/\n")
		return
	}

	model := tui.NewModel(tuiItems)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}

