package main

import (
	"fmt"

	"github.com/ZanzyTHEbar/firedragon-go/internal"
	"github.com/ZanzyTHEbar/firedragon-go/internal/cli"
	"github.com/ZanzyTHEbar/firedragon-go/internal/cli/cli_cmds"
	"github.com/ZanzyTHEbar/firedragon-go/internal/nats_common"
)

func main() {
	cfg, log := internal.Init()

	// Initialize with empty NATS config - will be populated from flags in start command
	natsConfig := nats_common.NATSConfig{
		StreamName: cfg.NATS.StreamName,
		Subjects:   cfg.NATS.Subjects,
	}

	if err := run(cfg, log, natsConfig); err != nil {
		log.Fatal(internal.ComponentGeneral, "Error running client: %v", err)
	}
}

func run(cfg *internal.Config, logger *internal.Logger, natsConfig nats_common.NATSConfig) error {
	// Setup the Root Command with access to services
	rootParams := &cli.CmdParams{
		Config:  cfg,
		Logger:  logger,
		Palette: nil,
		Use:     "perception_engine_client",
		Alias:   "pec",
		Short:   "Perception Engine Client",
		Long:    "Perception Engine Client - Capture and process perception events",
	}

	// Generate command palette
	palette := cli_cmds.GeneratePalette(rootParams)
	rootParams.Palette = palette

	// Create root command
	rootCmd := cli.NewRootCMD(rootParams)

	// Execute root command
	if err := rootCmd.Root.Execute(); err != nil {
		return fmt.Errorf("error executing root command: %v", err)
	}

	return nil
}
