package bootstrap

import (
	"github.com/spf13/cobra"
	"payloop/internal/application/bootstrap/commands"
)

// This is a command runner or cli for api architecture in golang.
// Using this we can use underlying dependency injection container for running scripts.
// Main advantage is that, we can use same services, repository, infrastructure present in the application itself

var rootCmd = &cobra.Command{
	Use:   "payloop",
	Short: "Smart recurring payment processing framework",
	Long: `
Payloop is a smart recurring payment processing framework.
`,
	TraverseChildren: true,
}

// App root of application
type App struct {
	*cobra.Command
}

func NewApp() App {
	cmd := App{
		Command: rootCmd,
	}
	cmd.AddCommand(commands.GetSubCommands(CommonModules)...)
	return cmd
}

var RootApp = NewApp()
