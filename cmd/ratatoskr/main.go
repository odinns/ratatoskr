package main

import (
	"fmt"
	"os"

	"github.com/odinn/ratatoskr/internal/cli"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
