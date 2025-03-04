package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the Firedragon configuration
type Config struct {
	// Firefly III configuration
	Firefly struct {
		URL   string `mapstructure:"url"`   // Firefly III API URL
		Token string `mapstructure:"token"` // Firefly III API token
	} `mapstructure:"firefly"`

	// Wallets configuration - map of wallet chain to address
	Wallets map[string]string `mapstructure:"wallets"` // e.g., "ethereum": "0xAddress"

	// BankAccounts configuration
	BankAccounts []BankAccountConfig `mapstructure:"bank_accounts"`

	// Interval for background operations
	Interval string `mapstructure:"interval"` // e.g., "15m" for background task frequency

	// Database configuration
	Database struct {
		Path string `mapstructure:"path"` // Path to SQLite database file
	} `mapstructure:"database"`

	// Debug mode
	Debug bool `mapstructure:"debug"`
}

// BankAccountConfig represents configuration for a bank account
type BankAccountConfig struct {
	Name        string            `mapstructure:"name"`        // Account name
	Provider    string            `mapstructure:"provider"`    // e.g., "enable_banking"
	Credentials map[string]string `mapstructure:"-"`           // Loaded from env vars (e.g., ENABLE_CLIENT_ID)
	Currencies  map[string]string `mapstructure:"currencies"`  // Currency to Firefly account ID
	Limit       int               `mapstructure:"limit"`       // Default transaction limit
	FromDate    string            `mapstructure:"from_date"`   // Optional start date (e.g., "2023-01-01")
	ToDate      string            `mapstructure:"to_date"`     // Optional end date (e.g., "2023-12-31")
}

// LoadConfig loads the configuration from various sources
func LoadConfig(configFile string) (*Config, error) {
	v := viper.New()
	
	// Set default values
	setDefaultConfig(v)
	
	// Read configuration from file if provided
	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		// Look for config in the current directory and in /etc/firedragon/
		v.AddConfigPath(".")
		v.AddConfigPath("/etc/firedragon/")
		v.SetConfigName("config")
		v.SetConfigType("json") // Using JSON as specified in the design doc
	}
	
	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		// It's okay if the config file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}
	
	// Override with environment variables prefixed with FIREDRAGON_
	v.SetEnvPrefix("FIREDRAGON")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	
	// Parse the configuration
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}
	
	// Load credentials from environment variables
	for i := range config.BankAccounts {
		config.BankAccounts[i].Credentials = make(map[string]string)
		
		// Handle Enable Banking credentials if provider is 'enable_banking'
		if config.BankAccounts[i].Provider == "enable_banking" {
			config.BankAccounts[i].Credentials["client_id"] = os.Getenv("ENABLE_CLIENT_ID")
			config.BankAccounts[i].Credentials["client_secret"] = os.Getenv("ENABLE_CLIENT_SECRET")
		}
		
		// Handle other credential types as needed
	}
	
	// Load blockchain API keys from environment variables
	if _, ok := config.Wallets["ethereum"]; ok {
		// We have an Ethereum wallet, so we need the Etherscan API key
		etherScanApiKey := os.Getenv("ETHERSCAN_API_KEY")
		if etherScanApiKey == "" {
			return nil, fmt.Errorf("ETHERSCAN_API_KEY environment variable not set")
		}
	}
	
	return &config, nil
}

// SaveConfig saves the configuration to disk
func SaveConfig(config *Config) error {
	v := viper.New()
	v.SetConfigFile(viper.ConfigFileUsed())
	
	// Marshal the config to map
	configMap := make(map[string]interface{})
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := json.Unmarshal(data, &configMap); err != nil {
		return fmt.Errorf("failed to unmarshal config to map: %w", err)
	}
	
	// Set all values in viper
	for k, v := range configMap {
		viper.Set(k, v)
	}
	
	// Save the config file
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	
	return nil
}

// setDefaultConfig sets default configuration values
func setDefaultConfig(v *viper.Viper) {
	// Firefly III defaults
	v.SetDefault("firefly.url", "http://localhost:8080")
	
	// Database defaults
	v.SetDefault("database.path", filepath.Join(".", "data", "firedragon.db"))
	
	// Background task interval default
	v.SetDefault("interval", "15m")
	
	// Debug mode default
	v.SetDefault("debug", false)
}
