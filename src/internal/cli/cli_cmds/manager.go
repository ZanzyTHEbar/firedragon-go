package cli_cmds

import (
	"context"
	"sync"

	"github.com/ZanzyTHEbar/firedragon-go/internal"
	"github.com/ZanzyTHEbar/firedragon-go/internal/cli"

	"github.com/spf13/cobra"
)

var (
	globalManager *internal.ServiceManager
	managerOnce   sync.Once
)

// GetServiceManager returns the global ServiceManager instance
func GetServiceManager() *internal.ServiceManager {
	managerOnce.Do(func() {
		globalManager = NewServiceManager(context.Background())
	})
	return globalManager
}

func GeneratePalette(params *cli.CmdParams) []*cobra.Command {

	// Global commands
	helpCmd := NewHelp(params)
	versionCmd := NewVersion(params)

	// Utility commands

	// Return all commands
	return []*cobra.Command{
		helpCmd,
		versionCmd,
	}
}
