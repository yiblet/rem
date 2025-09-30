package main

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/yiblet/rem/internal/cli"
)

func main() {
	// Parse command-line arguments
	var args cli.Args
	parser := arg.MustParse(&args)

	// If no subcommand provided, show help or launch TUI
	if args.Store == nil && args.Get == nil && args.Config == nil && args.Clear == nil && args.Search == nil {
		// Default behavior: launch TUI (same as 'rem get')
		args.Get = &cli.GetCmd{}
	}

	// Create CLI instance with args for history location support
	cliHandler, err := cli.NewWithArgs(&args)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Execute the command
	if err := cliHandler.Execute(&args); err != nil {
		fmt.Printf("Error: %v\n", err)

		// If it's an argument validation error, show usage
		if args.Store != nil || args.Get != nil || args.Config != nil || args.Clear != nil || args.Search != nil {
			fmt.Println()
			parser.WriteUsage(os.Stderr)
		}
		os.Exit(1)
	}
}
