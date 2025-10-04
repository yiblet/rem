package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/yiblet/rem/internal/queue"
	"github.com/yiblet/rem/internal/store/memstore"
	"github.com/yiblet/rem/internal/tui"
)

func main() {
	fmt.Println("Testing TUI Border Fix")
	fmt.Println("=========================")

	// Create in-memory store and queue manager
	store := memstore.NewMemoryStore()
	qm, err := queue.NewQueueManager(store)
	if err != nil {
		log.Fatalf("Error creating queue manager: %v", err)
	}
	defer qm.Close()

	// Add some test items if queue is empty
	items, err := qm.List()
	if err != nil {
		log.Fatalf("Error listing queue items: %v", err)
	}

	if len(items) == 0 {
		// Add test items
		testContent := []string{
			"Hello, World! This is test content.",
			"package main\n\nfunc main() {\n    println(\"test\")\n}",
			"Some more test content for the TUI.",
		}
		for _, content := range testContent {
			_, err := qm.Enqueue(strings.NewReader(content), "")
			if err != nil {
				log.Printf("Error enqueuing item: %v", err)
			}
		}
		// Refresh items list
		items, err = qm.List()
		if err != nil {
			log.Fatalf("Error listing queue items: %v", err)
		}
	}

	// Convert queue items to TUI items
	var tuiItems []*tui.StackItem
	for _, item := range items[:min(5, len(items))] { // Take first 5 items
		contentReader, err := qm.GetContent(item.ID)
		if err != nil {
			fmt.Printf("Error getting content reader: %v\n", err)
			continue
		}

		// Capture item ID for delete function
		itemID := item.ID

		tuiItem := &tui.StackItem{
			ID:       fmt.Sprintf("%d", item.ID),
			Content:  contentReader,
			Preview:  item.Title,
			ViewPos:  0,
			IsBinary: item.IsBinary,
			Size:     item.Size,
			SHA256:   item.SHA256,
			DeleteFunc: func() error {
				// Find index and delete
				allItems, err := qm.List()
				if err != nil {
					return err
				}
				for idx, itm := range allItems {
					if itm.ID == itemID {
						return qm.Delete(idx)
					}
				}
				return fmt.Errorf("item not found")
			},
		}
		tuiItems = append(tuiItems, tuiItem)
	}

	// Create model with specific dimensions
	model := tui.NewModel(tuiItems)
	model.UpdateMockSize(120, 20)

	// Render the view
	view := model.View()
	lines := strings.Split(view, "\n")

	fmt.Printf("Rendered TUI view (%d lines):\n", len(lines))
	fmt.Println(strings.Repeat("=", 120))

	for i, line := range lines[:min(15, len(lines))] {
		fmt.Printf("Line %2d: %s\n", i, line)
	}

	fmt.Println(strings.Repeat("=", 120))

	// Check for border integrity
	var borderCheckLine string
	for i, line := range lines {
		if i > 2 && i < len(lines)-3 && len(line) > 25 && strings.Contains(line, "│") {
			borderCheckLine = line
			break
		}
	}

	if borderCheckLine != "" {
		fmt.Printf("Border analysis: %s\n", borderCheckLine)
		fmt.Printf("Line length: %d\n", len(borderCheckLine))

		// Find all '│' characters
		borderPositions := []int{}
		for i, char := range borderCheckLine {
			if char == '│' {
				borderPositions = append(borderPositions, i)
			}
		}

		fmt.Printf("Found border characters (│) at positions: %v\n", borderPositions)

		if len(borderPositions) >= 2 {
			fmt.Printf("Both left pane borders are present!\n")
			fmt.Printf("   Left border at position: %d\n", borderPositions[0])
			fmt.Printf("   Right border at position: %d\n", borderPositions[len(borderPositions)-1])
		} else {
			fmt.Printf("Missing border characters\n")
		}
	} else {
		fmt.Printf("Could not find a line with borders to analyze\n")
	}

	fmt.Println("\nBorder fix verification complete!")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
