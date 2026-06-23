package cli

import (
	"github.com/spf13/cobra"

	"github.com/getpaidhqco/getpaidhq/cli/internal/apigen"
)

func newHealthCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "health",
		Short:   "Check API server health",
		Long:    "Calls the unauthenticated /api/health endpoint and prints the result.",
		Example: "  gphq health",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := app.API.GetHealth(cmd.Context(), apigen.GetHealthParams{})
			h, err := expectOK[*apigen.HealthResponse](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, h)
		},
	}
	return annotate(cmd, "GET", "/api/health")
}
