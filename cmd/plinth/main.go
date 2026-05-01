// Package main is the entry point for the plinth CLI.
package main

import (
	"os"

	"github.com/plinth-dev/cli/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
