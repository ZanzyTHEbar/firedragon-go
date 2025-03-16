package cli_cmds

import (
	"sync"

	"github.com/ZanzyTHEbar/firedragon-go/interfaces"
	"github.com/ZanzyTHEbar/firedragon-go/internal"
	"github.com/ZanzyTHEbar/firedragon-go/internal/cli"
	"github.com/ZanzyTHEbar/firedragon-go/services"
	"github.com/spf13/cobra"
)

var (
	globalManager *interfaces.ServiceManager
	managerOnce   sync.Once
)

// GetServiceManager returns the global ServiceManager instance
func GetServiceManager() interfaces.ServiceManager {
	managerOnce.Do(func() {
		logger := internal.GetLogger()
		config := internal.GetConfig()
		globalManager = services.NewActorServiceManager(config, logger)
		if err := globalManager.Initialize(); err != nil {
			logger.Fatal(internal.ComponentService, "Failed to initialize service manager: %v", err)
		}
	})
	return globalManager
}

func GeneratePalette(params *cli.CmdParams) []*cobra.Command {
	// Global commands
	helpCmd := NewHelp(params)
	versionCmd := NewVersion(params)
	servicesCmd := NewServices(params)

	// Return all commands
	return []*cobra.Command{
		helpCmd,
		versionCmd,
		servicesCmd,
	}
}
