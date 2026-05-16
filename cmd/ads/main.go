package main

import (
	"os"

	"github.com/dannolan/apple-ads-cli/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:], os.Stdout, os.Stderr))
}
