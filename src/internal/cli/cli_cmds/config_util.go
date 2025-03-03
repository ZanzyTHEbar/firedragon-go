package cli_cmds

import (
	"fmt"

	"strings"

	"github.com/ZanzyTHEbar/firedragon-go/internal/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// TODO: Add config file support

// NewConfig creates a command to manage server configuration
func NewConfig(params *cli.CmdParams) *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage server configuration",
		Long:  `View and modify the server configuration settings.`,
	}

	// Add subcommands for different config operations
	configCmd.AddCommand(newConfigGet(params))
	configCmd.AddCommand(newConfigSet(params))
	configCmd.AddCommand(newConfigList(params))

	return configCmd
}

// newConfigGet creates a subcommand to get a specific config value
func newConfigGet(params *cli.CmdParams) *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "Get a configuration value",
		Long:  `Retrieve a specific configuration value by key.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			key := args[0]

			// In a real implementation, this would use params.Config to get the value
			// For this example, we'll use viper directly
			if !viper.IsSet(key) {
				fmt.Printf("Config key '%s' not found\n", key)
				return
			}

			value := viper.Get(key)
			fmt.Printf("%s = %v\n", key, value)
		},
	}
}

// newConfigSet creates a subcommand to set a config value
func newConfigSet(params *cli.CmdParams) *cobra.Command {
	return &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a configuration value",
		Long:  `Set or update a configuration value by key.`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			key := args[0]
			value := args[1]

			// In a real implementation, this would use params.Config to set the value
			// and persist it to the config file
			fmt.Printf("Setting %s = %s\n", key, value)

			// Set in viper
			viper.Set(key, value)

			// Save configuration (in a real implementation)
			fmt.Println("Configuration updated")
		},
	}
}

// newConfigList creates a subcommand to list all config values
func newConfigList(params *cli.CmdParams) *cobra.Command {
	var format string

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		Long:  `Display all current configuration values.`,
		Run: func(cmd *cobra.Command, args []string) {
			// In a real implementation, this would use params.Config to get all settings
			// Example config items
			configItems := map[string]interface{}{
				"messaging.nats.url":      "nats://localhost:4222",
				"messaging.nats.user":     "perception",
				"messaging.nats.password": "********",
				"storage.type":            "file",
				"storage.path":            "/var/lib/perception/data",
				"log.level":               "info",
				"server.port":             8080,
			}

			switch strings.ToLower(format) {
			case "json":
				// In a real implementation, this would marshal to JSON
				fmt.Println("{")
				i := 0
				for k, v := range configItems {
					i++
					ending := ""
					if i < len(configItems) {
						ending = ","
					}
					fmt.Printf("  \"%s\": \"%v\"%s\n", k, v, ending)
				}
				fmt.Println("}")

			default: // plain text format
				fmt.Println("Current Configuration:")
				fmt.Println("======================")
				for k, v := range configItems {
					fmt.Printf("%s = %v\n", k, v)
				}
			}
		},
	}

	// Add flags
	listCmd.Flags().StringVarP(&format, "format", "f", "text", "Output format (text or json)")

	return listCmd
}
