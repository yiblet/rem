package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/yiblet/rem/internal/queue"
	"github.com/yiblet/rem/internal/remfs"
	"github.com/yiblet/rem/internal/tui"
)

func main() {
	fmt.Println("Testing TUI Border Fix")
	fmt.Println("=========================")

	// Create filesystem and stack manager
	remFS, err := remfs.New()
	if err != nil {
		log.Fatalf("Error creating rem filesystem: %v", err)
	}

	sm, err := queue.NewStackManager(remFS)
	if err != nil {
		log.Fatalf("Error creating stack manager: %v", err)
	}

	// Get items from stack
	stackItems, err := sm.List()
	if err != nil {
		log.Fatalf("Error listing stack items: %v", err)
	}

	if len(stackItems) == 0 {
		fmt.Println("No items in stack. Run 'go run ./cmd/demo/' first.")
		return
	}

	// Convert stack items to TUI items
	var tuiItems []*tui.StackItem
	for _, sItem := range stackItems[:min(5, len(stackItems))] { // Take first 5 items
		contentReader, err := sItem.GetContentReader(sm.FileSystem())
		if err != nil {
			fmt.Printf("Error getting content reader: %v\n", err)
			continue
		}

		tuiItem := &tui.StackItem{
			Content: contentReader,
			Preview: sItem.Preview,
			ViewPos: 0,
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
