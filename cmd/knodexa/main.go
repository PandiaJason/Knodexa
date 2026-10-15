package main

import (
	"fmt"
	"os"

	"knodexa/internal/cli"
	"knodexa/internal/mcp"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error determining current working directory: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 && os.Args[1] == "mcp" {
		if err := mcp.RunServer(cwd); err != nil {
			fmt.Fprintf(os.Stderr, "MCP server failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	app, err := cli.NewApp(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize Knodexa: %v\n", err)
		os.Exit(1)
	}

	args := os.Args[1:]
	if err := app.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
