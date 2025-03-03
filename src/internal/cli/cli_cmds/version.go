package cli_cmds

import (
	"fmt"
	"github.com/ZanzyTHEbar/firedragon-go/internal"
	"github.com/ZanzyTHEbar/firedragon-go/internal/cli"

	"github.com/spf13/cobra"
)

// NewVersion creates a version command for the server
func NewVersion(params *cli.CmdParams) *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version of Perception Engine Server",
		Long:  `Print the version information for Perception Engine Server including build details.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Perception Engine Server")
			fmt.Println("=======================")
			fmt.Printf("%s\n", internal.VersionInfo())
		},
	}

	return versionCmd
}
