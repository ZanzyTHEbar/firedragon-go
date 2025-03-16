package cli_cmds

import (
	"fmt"

	"github.com/ZanzyTHEbar/firedragon-go/internal"
	"github.com/ZanzyTHEbar/firedragon-go/internal/cli"
	"github.com/spf13/cobra"
)

// NewServices creates a command to manage services
func NewServices(params *cli.CmdParams) *cobra.Command {
	servicesCmd := &cobra.Command{
		Use:   "services",
		Short: "Manage running services",
		Long:  `View and control the running services.`,
	}

	// Add subcommands for different service operations
	servicesCmd.AddCommand(newServicesList(params))
	servicesCmd.AddCommand(newServicesStart(params))
	servicesCmd.AddCommand(newServicesStop(params))

	return servicesCmd
}

// newServicesList creates a subcommand to list all services
func newServicesList(params *cli.CmdParams) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all services",
		Long:  `Display status information for all registered services.`,
		Run: func(cmd *cobra.Command, args []string) {
			manager := GetServiceManager()
			services := manager.GetAllServicesInfo()

			fmt.Println("Services:")
			for _, service := range services {
				fmt.Printf("- %s: %s (Started: %s)\n",
					service.Name,
					service.Status,
					service.StartTime.Format("2006-01-02 15:04:05"))

				if service.LastError != nil {
					fmt.Printf("  Last Error: %v\n", service.LastError)
				}
				if service.CustomStats != nil {
					fmt.Printf("  Stats: %v\n", service.CustomStats)
				}
			}
		},
	}
}

// newServicesStart creates a subcommand to start services
func newServicesStart(params *cli.CmdParams) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "start [service-name]",
		Short: "Start one or all services",
		Long:  `Start a specific service by name, or all services if --all is specified.`,
		Run: func(cmd *cobra.Command, args []string) {
			manager := GetServiceManager()

			if all {
				if err := manager.StartAll(); err != nil {
					params.Logger.Error(internal.ComponentCLI, "Failed to start all services: %v", err)
					return
				}
				params.Logger.Info(internal.ComponentCLI, "All services started")
				return
			}

			if len(args) != 1 {
				params.Logger.Error(internal.ComponentCLI, "Service name required")
				return
			}
			if err := manager.StartService(args[0]); err != nil {
				params.Logger.Error(internal.ComponentCLI, "Failed to start service %s: %v", args[0], err)
				return
			}

			if err := manager.StartService(args[0]); err != nil {
				params.Logger.Error(internal.ComponentCLI, "Failed to start service %s: %v", args[0], err)
				return
			}
			params.Logger.Info(internal.ComponentCLI, "Service %s started", args[0])
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Start all services")
	return cmd
}

// newServicesStop creates a subcommand to stop services
func newServicesStop(params *cli.CmdParams) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "stop [service-name]",
		Short: "Stop one or all services",
		Long:  `Stop a specific service by name, or all services if --all is specified.`,
		Run: func(cmd *cobra.Command, args []string) {
			manager := GetServiceManager()

			if all {
				if err := manager.StopAll(); err != nil {
					params.Logger.Error(internal.ComponentCLI, "Failed to stop all services: %v", err)
					return
				}
				params.Logger.Info(internal.ComponentCLI, "All services stopped")
				return
			}

			if len(args) != 1 {
				params.Logger.Error(internal.ComponentCLI, "Service name required")
				return
			}

			if err := manager.StopService(args[0]); err != nil {
				params.Logger.Error(internal.ComponentCLI, "Failed to stop service %s: %v", args[0], err)
				return
			}
			params.Logger.Info(internal.ComponentCLI, "Service %s stopped", args[0])
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Stop all services")
	return cmd
}
