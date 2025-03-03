package cli_cmds

import (
	"fmt"

	"github.com/ZanzyTHEbar/firedragon-go/internal/cli"

	"github.com/spf13/cobra"
)

var helpShowAll bool

// NewHelp creates a help command for the server
func NewHelp(params *cli.CmdParams) *cobra.Command {
	helpCmd := &cobra.Command{
		Use:     "detailed_help",
		Aliases: []string{"h"},
		Short:   "Display detailed help for Perception Engine Server",
		Long:    `Display detailed help information for the Perception Engine Server including command hierarchy and usage examples.`,
		Run: func(cmd *cobra.Command, args []string) {
			if helpShowAll {
				// Display all available commands and their details
				fmt.Println("Perception Engine Server - Complete Command Reference")
				fmt.Println("==============================================")
				fmt.Println("\nAvailable Commands:")

				for _, cmd := range params.Palette {
					fmt.Printf("- %s: %s\n", cmd.Use, cmd.Short)
				}
			} else {
				// Display basic help
				fmt.Println("Perception Engine Server")
				fmt.Println("======================")
				fmt.Println("\nMain Commands:")
				fmt.Println("  start       Start the server")
				fmt.Println("  stop        Stop the server")
				fmt.Println("  status      Check server status")
				fmt.Println("  config      Manage server configuration")
				fmt.Println("\nUse 'perception-server [command] --help' for more information about a command.")
				fmt.Println("Use 'perception-server detailed_help --all' to see all available commands.")
			}
		},
	}

	helpCmd.Flags().BoolVarP(&helpShowAll, "all", "a", false, "Show all commands")

	return helpCmd
}
