package main

import (
	"os"

	"github.com/duailibe/linear-cli/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
