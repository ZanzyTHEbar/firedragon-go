package internal

import (
	"log"
)

func Init() (*Config, *Logger) {
	// Load configuration with empty config file path - will be handled by viper/cobra
	cfg, err := LoadConfig("")
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// set storage path
	cfg.Server.StoragePath = "./storage"

	var logger *Logger

	// init logger
	if err := InitGlobalLogger("logs", LogLevelDebug, []Component{
		ComponentGeneral,
		ComponentHID,
		ComponentNATS,
		ComponentConfig,
		ComponentService,
		ComponentStorage,
		ComponentScreen,
		ComponentTranscript,
	}); err != nil {
		// If logger initialization fails, use the default logger
		logger = GetLogger()
		logger.SetLevel(LogLevelDebug)
		logger.Error(ComponentGeneral, "Error initializing logger: %v", err)
	}

	logger = GetLogger()

	return cfg, logger
}
