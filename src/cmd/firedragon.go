package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/ZanzyTHEbar/firedragon-go/internal"
	"github.com/ZanzyTHEbar/firedragon-go/internal/cli"
	"github.com/ZanzyTHEbar/firedragon-go/internal/cli/cli_cmds"
)

func main() {
	cfg, log := internal.Init()
	
	if err := run(cfg, log); err != nil {
		log.Fatal(internal.ComponentGeneral, "Error running firedragon: %v", err)
	}
}

func run(cfg *internal.Config, logger *internal.Logger) error {
	// Setup the Root Command with access to services
	rootParams := &cli.CmdParams{
		Config:  cfg,
		Logger:  logger,
		Palette: nil,
		Use:     "firedragon",
		Alias:   "fd",
		Short:   "Firedragon - Crypto Wallet and Bank Account Transaction Importer for Firefly III",
		Long:    "Firedragon imports transactions from cryptocurrency wallets and bank accounts into Firefly III for unified financial tracking.",
	}
	
	// Generate command palette
	palette := cli_cmds.GeneratePalette(rootParams)
	rootParams.Palette = palette
	
	// Create root command
	rootCmd := cli.NewRootCMD(rootParams)

	// Setup signal handling for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Execute root command (blocks until command completes or interrupted)
	go func() {
		if err := rootCmd.Root.Execute(); err != nil {
			logger.Error(internal.ComponentGeneral, "Error executing root command: %v", err)
			signalChan <- syscall.SIGTERM
		}
	}()

	// Wait for signal
	<-signalChan
	logger.Info(internal.ComponentGeneral, "Shutting down gracefully...")
	
	// Perform cleanup - can add additional shutdown logic here
	
	return nil
}
