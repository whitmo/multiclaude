package main

import (
	"fmt"
	"os"

	"github.com/dlorenc/multiclaude/internal/cli"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	c, err := cli.New()
	if err != nil {
		return err
	}

	return c.Execute(os.Args[1:])
}
