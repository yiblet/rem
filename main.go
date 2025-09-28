package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yiblet/rem/internal/tui"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "debug" {
		// Keep the old debug functionality for testing
		fmt.Println("Use the interactive mode instead: just run 'go run main.go'")
		return
	}

	// Create dummy items with proper io.ReadSeekCloser content
	items := []*tui.QueueItem{
		{
			Content: tui.NewStringReadSeekCloser("Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."),
			Preview: "Lorem ipsum dolor si...",
			ViewPos: 0,
		},
		{
			Content: tui.NewStringReadSeekCloser("package main\n\nimport \"fmt\"\n\nfunc fibonacci(n int) int {\n    if n <= 1 {\n        return n\n    }\n    return fibonacci(n-1) + fibonacci(n-2)\n}\n\nfunc main() {\n    for i := 0; i < 15; i++ {\n        fmt.Printf(\"Fibonacci of %d: %d\\n\", i, fibonacci(i))\n    }\n}"),
			Preview: "package main  import...",
			ViewPos: 0,
		},
		{
			Content: tui.NewStringReadSeekCloser("#!/bin/bash\n\necho \"Starting backup process...\"\nfor file in *.txt; do\n    if [ -f \"$file\" ]; then\n        cp \"$file\" \"backup_$file\"\n        echo \"Backed up: $file\"\n    fi\ndone\necho \"Backup complete!\"\n\n# Clean up old backups\nfind . -name \"backup_*\" -mtime +30 -delete\necho \"Cleaned up old backups\""),
			Preview: "#!/bin/bash  echo \"S...",
			ViewPos: 0,
		},
		{
			Content: tui.NewStringReadSeekCloser("SELECT users.name, COUNT(orders.id) as order_count\nFROM users\nLEFT JOIN orders ON users.id = orders.user_id\nWHERE users.created_at > '2023-01-01'\nGROUP BY users.id, users.name\nHAVING COUNT(orders.id) > 5\nORDER BY order_count DESC\nLIMIT 10;\n\n-- This query finds users with the most orders\n-- created after January 1st, 2023"),
			Preview: "SELECT users.name, C...",
			ViewPos: 0,
		},
		{
			Content: tui.NewStringReadSeekCloser("{\n  \"name\": \"my-awesome-project\",\n  \"version\": \"1.0.0\",\n  \"description\": \"A revolutionary new tool that will change the way you work\",\n  \"main\": \"index.js\",\n  \"scripts\": {\n    \"start\": \"node index.js\",\n    \"test\": \"jest --coverage\",\n    \"build\": \"webpack --mode=production\",\n    \"dev\": \"webpack --mode=development --watch\"\n  },\n  \"dependencies\": {\n    \"express\": \"^4.18.0\",\n    \"lodash\": \"^4.17.21\",\n    \"axios\": \"^1.6.0\"\n  }\n}"),
			Preview: "{   \"name\": \"my-awes...",
			ViewPos: 0,
		},
	}

	model := tui.NewModel(items)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}