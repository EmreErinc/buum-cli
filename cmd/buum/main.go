package main

import (
	"os"

	"github.com/emreerinc/buum/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
