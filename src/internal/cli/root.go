/*
Copyright © 2024 FinalRoundAI

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is config.yaml)")

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
