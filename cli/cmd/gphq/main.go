// Command gphq is the command-line client for the GetPaidHQ API. It talks to
// the API through the OpenAPI-generated client and depends on nothing in the
// server module.
package main

import (
	"os"

	"github.com/getpaidhqco/getpaidhq/cli/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
