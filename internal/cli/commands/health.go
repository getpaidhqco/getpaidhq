package commands

import (
	"net/http"

	"github.com/spf13/cobra"
)

func newHealthCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "health",
		Short:   "Check API server health",
		Long:    "Calls the unauthenticated /api/health endpoint and prints the result.",
		Example: "  gphq health",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/health", nil, nil)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	return annotate(cmd, "GET", "/api/health")
}
