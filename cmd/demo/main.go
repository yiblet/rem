package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/yiblet/rem/internal/queue"
	"github.com/yiblet/rem/internal/remfs"
)

func main() {
	fmt.Println("rem Stack Manager Demo")

	// Create filesystem and stack manager
	remFS, err := remfs.New()
	if err != nil {
		log.Fatalf("Failed to create rem filesystem: %v", err)
	}

	sm, err := queue.NewStackManager(remFS)
	if err != nil {
		log.Fatalf("Failed to create stack manager: %v", err)
	}

	// Show initial state
	size, err := sm.Size()
	if err != nil {
		log.Fatalf("Failed to get size: %v", err)
	}
	fmt.Printf("Initial stack size: %d\n\n", size)

	// Add some test content
	testContent := []string{
		"Hello, World! This is the first item in our stack.",
		"package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, Go!\")\n}",
		"#!/bin/bash\necho \"Starting script...\"\nfor i in {1..5}; do\n    echo \"Processing $i\"\ndone",
		"SELECT * FROM users WHERE created_at > '2023-01-01' ORDER BY created_at DESC LIMIT 10;",
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
	}

	fmt.Println("Adding items to stack:")
	for i, content := range testContent {
		item, err := sm.Push(strings.NewReader(content))
		if err != nil {
			log.Printf("Failed to push item %d: %v", i, err)
			continue
		}
		fmt.Printf("%d. %s\n", i+1, item.Preview)
	}

	// Show final state
	size, err = sm.Size()
	if err != nil {
		log.Fatalf("Failed to get final size: %v", err)
	}
	fmt.Printf("\nFinal stack size: %d\n\n", size)

	// List all items
	fmt.Println("Stack contents (newest first - LIFO):")
	items, err := sm.List()
	if err != nil {
		log.Fatalf("Failed to list items: %v", err)
	}

	for i, item := range items {
		fmt.Printf("%d. [%s] %s\n", i, item.Timestamp.Format("15:04:05"), item.Preview)
	}

	// Demonstrate getting specific item
	if len(items) > 0 {
		fmt.Printf("\nContent of item 0 (newest):\n")
		reader, err := items[0].GetContentReader(sm.FileSystem())
		if err != nil {
			log.Printf("Failed to get content reader: %v", err)
		} else {
			defer reader.Close()

			// Read first 200 bytes
			buffer := make([]byte, 200)
			n, err := reader.Read(buffer)
			if err != nil && err.Error() != "EOF" {
				log.Printf("Failed to read content: %v", err)
			} else {
				fmt.Printf("%s\n", string(buffer[:n]))
			}
		}
	}

	fmt.Printf("\nDemo complete! Stack stored in: ~/.config/rem/content/\n")
}