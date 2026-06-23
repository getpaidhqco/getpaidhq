package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// newDocsCmd regenerates the markdown command reference (make docs-cli).
func newDocsCmd(root *cobra.Command) *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:    "docs",
		Short:  "Generate the markdown command reference",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return doc.GenMarkdownTree(root, dir)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "docs/cli/reference", "output directory")
	return cmd
}
