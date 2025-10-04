package main

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/yiblet/rem/internal/queue"
	"github.com/yiblet/rem/internal/store/memstore"
)

func main() {
	fmt.Println("rem Queue Manager Demo")

	// Create in-memory store and queue manager
	store := memstore.NewMemoryStore()
	qm, err := queue.NewQueueManager(store)
	if err != nil {
		log.Fatalf("Failed to create queue manager: %v", err)
	}
	defer qm.Close()

	// Show initial state
	items, err := qm.List()
	if err != nil {
		log.Fatalf("Failed to get initial items: %v", err)
	}
	fmt.Printf("Initial queue size: %d\n\n", len(items))

	// Add some test content
	testContent := []string{
		"Hello, World! This is the first item in our queue.",
		"package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, Go!\")\n}",
		"#!/bin/bash\necho \"Starting script...\"\nfor i in {1..5}; do\n    echo \"Processing $i\"\ndone",
		"SELECT * FROM users WHERE created_at > '2023-01-01' ORDER BY created_at DESC LIMIT 10;",
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
	}

	fmt.Println("Adding items to queue:")
	for i, content := range testContent {
		item, err := qm.Enqueue(strings.NewReader(content), "")
		if err != nil {
			log.Printf("Failed to enqueue item %d: %v", i, err)
			continue
		}
		fmt.Printf("%d. %s\n", i+1, item.Title)
	}

	// Show final state
	items, err = qm.List()
	if err != nil {
		log.Fatalf("Failed to get final items: %v", err)
	}
	fmt.Printf("\nFinal queue size: %d\n\n", len(items))

	// List all items
	fmt.Println("Queue contents (newest first - LIFO):")
	for i, item := range items {
		fmt.Printf("%d. [%s] %s\n", i, item.Timestamp.Format("15:04:05"), item.Title)
	}

	// Demonstrate getting specific item
	if len(items) > 0 {
		fmt.Printf("\nContent of item 0 (newest):\n")
		reader, err := qm.GetContent(items[0].ID)
		if err != nil {
			log.Printf("Failed to get content reader: %v", err)
		} else {
			defer reader.Close()

			// Read first 200 bytes
			buffer := make([]byte, 200)
			n, err := reader.Read(buffer)
			if err != nil && err != io.EOF {
				log.Printf("Failed to read content: %v", err)
			} else {
				fmt.Printf("%s\n", string(buffer[:n]))
			}
		}
	}

	fmt.Printf("\nDemo complete! (Using in-memory store)\n")
}
