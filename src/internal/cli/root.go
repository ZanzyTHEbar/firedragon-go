package cli

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// RootCMD wraps the root cobra.Command
type RootCMD struct {
	Root *cobra.Command
}

// NewRootCMD creates a new RootCMD with the given parameters
func NewRootCMD(params *CmdParams) *RootCMD {
	return &RootCMD{
		Root: NewRoot(params),
	}
}

// NewRoot creates and configures the root command
func NewRoot(params *CmdParams) *cobra.Command {
	// rootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:     params.Use,
		Aliases: []string{params.Alias},
		Short:   params.Short,
		Long:    params.Long,
	}

	// Validate palette
	if params.Palette == nil {
		params.Palette = []*cobra.Command{}
	}

	// Add commands to the root
	rootCmd.AddCommand(params.Palette...)

	// Define persistent flags for the root command
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is config.toml)")

	// Bind configuration
	cobra.OnInitialize(func() {
		if cfgFile != "" {
			viper.SetConfigFile(cfgFile)
		} else {
			viper.SetConfigName("config")
			viper.AddConfigPath(".")
		}

		viper.AutomaticEnv() // read in environment variables that match

		// If a config file is found, read it in
		if err := viper.ReadInConfig(); err == nil {
			slog.Info(fmt.Sprintf("Using config file: %s", viper.ConfigFileUsed()))
		}
	})

	return rootCmd
}
