// Package commands holds the gphq CLI command tree.
package commands

import (
	"io"

	"github.com/spf13/cobra"

	"getpaidhq/internal/cli/client"
)

// App carries the resolved configuration and shared dependencies into
// every command.
type App struct {
	Out, ErrOut io.Writer
	Client      *client.Client
	Output      string // "table" or "json"
}

// AddAll registers every resource group on the root command. Each task in
// the implementation plan appends its constructor here.
func AddAll(root *cobra.Command, app *App) {
	root.AddCommand(newHealthCmd(app), newCustomersCmd(app))
}
