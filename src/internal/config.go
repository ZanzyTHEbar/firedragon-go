package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Firefly   FireflyConfig   `mapstructure:"firefly"`
	Ethereum  EthereumConfig  `mapstructure:"ethereum"`
	Solana    SolanaConfig    `mapstructure:"solana"`
	Sui       SuiConfig       `mapstructure:"sui"`
	Banking   BankingConfig   `mapstructure:"banking"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Service   ServiceConfig   `mapstructure:"service"`
}

// FireflyConfig contains Firefly III API configuration
type FireflyConfig struct {
	URL   string `mapstructure:"url"`
	Token string `mapstructure:"token"`
}

// EthereumConfig contains Ethereum configuration
type EthereumConfig struct {
	APIKey      string   `mapstructure:"api_key"`
	Addresses   []string `mapstructure:"addresses"`
	NetworkType string   `mapstructure:"network_type"` // mainnet, testnet, etc.
}

// SolanaConfig contains Solana configuration
type SolanaConfig struct {
	RPCEndpoint string   `mapstructure:"rpc_endpoint"`
	Addresses   []string `mapstructure:"addresses"`
	NetworkType string   `mapstructure:"network_type"` // mainnet, testnet, etc.
}

// SuiConfig contains SUI configuration
type SuiConfig struct {
	RPCEndpoint string   `mapstructure:"rpc_endpoint"`
	Addresses   []string `mapstructure:"addresses"`
	NetworkType string   `mapstructure:"network_type"` // mainnet, testnet, etc.
}

// BankingConfig contains banking provider configuration
type BankingConfig struct {
	Enable EnableBankingConfig `mapstructure:"enable"`
}

// EnableBankingConfig contains Enable Banking API configuration
type EnableBankingConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURI  string `mapstructure:"redirect_uri"`
	AccountIDs   []string `mapstructure:"account_ids"`
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	Path     string `mapstructure:"path"`
	Type     string `mapstructure:"type"` // sqlite, postgres, etc.
	FileName string `mapstructure:"filename"`
}

// ServiceConfig contains service-level configuration
type ServiceConfig struct {
	UpdateInterval      time.Duration `mapstructure:"update_interval"`
	MaxRetries         int           `mapstructure:"max_retries"`
	RetryDelay         time.Duration `mapstructure:"retry_delay"`
	LogLevel           string        `mapstructure:"log_level"`
	MetricsEnabled     bool          `mapstructure:"metrics_enabled"`
	MetricsInterval    time.Duration `mapstructure:"metrics_interval"`
}

// LoadConfig loads the application configuration from file and environment
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set default configuration values
	setDefaults(v)

	// Load configuration from file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Look for config in default locations
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.config/firedragon")
		v.AddConfigPath("/etc/firedragon")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	// Read configuration file
	if err := v.ReadInConfig(); err != nil {
		// Only return error if config file was explicitly specified
		if configPath != "" {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Load environment variables
	v.AutomaticEnv()
	v.SetEnvPrefix("FIREDRAGON")
	v.SetEnvKeyReplacer(NewEnvKeyReplacer())

	// Bind environment variables
	bindEnvVariables(v)

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Ensure required directories exist
	if err := ensureDirectories(&config); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	v.SetDefault("service.update_interval", "15m")
	v.SetDefault("service.max_retries", 3)
	v.SetDefault("service.retry_delay", "1m")
	v.SetDefault("service.log_level", "info")
	v.SetDefault("service.metrics_enabled", true)
	v.SetDefault("service.metrics_interval", "1m")
	v.SetDefault("database.type", "sqlite")
	v.SetDefault("database.filename", "firedragon.db")
}

// bindEnvVariables binds environment variables to configuration
func bindEnvVariables(v *viper.Viper) {
	// Firefly III
	v.BindEnv("firefly.url", "FIREFLY_URL")
	v.BindEnv("firefly.token", "FIREFLY_TOKEN")

	// Ethereum
	v.BindEnv("ethereum.api_key", "ETHERSCAN_API_KEY")
	v.BindEnv("ethereum.network_type", "ETH_NETWORK")

	// Enable Banking
	v.BindEnv("banking.enable.client_id", "ENABLE_CLIENT_ID")
	v.BindEnv("banking.enable.client_secret", "ENABLE_CLIENT_SECRET")
	v.BindEnv("banking.enable.redirect_uri", "ENABLE_REDIRECT_URI")
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate Firefly III configuration
	if config.Firefly.URL == "" {
		return fmt.Errorf("firefly.url is required")
	}
	if config.Firefly.Token == "" {
		return fmt.Errorf("firefly.token is required")
	}

	// Validate blockchain configuration if addresses are provided
	if len(config.Ethereum.Addresses) > 0 && config.Ethereum.APIKey == "" {
		return fmt.Errorf("ethereum.api_key is required when addresses are configured")
	}

	// Validate banking configuration if accounts are configured
	if len(config.Banking.Enable.AccountIDs) > 0 {
		if config.Banking.Enable.ClientID == "" {
			return fmt.Errorf("banking.enable.client_id is required when accounts are configured")
		}
		if config.Banking.Enable.ClientSecret == "" {
			return fmt.Errorf("banking.enable.client_secret is required when accounts are configured")
		}
		if config.Banking.Enable.RedirectURI == "" {
			return fmt.Errorf("banking.enable.redirect_uri is required when accounts are configured")
		}
	}

	return nil
}

// ensureDirectories creates required directories
func ensureDirectories(config *Config) error {
	// Create database directory if needed
	if config.Database.Path != "" {
		if err := os.MkdirAll(config.Database.Path, 0755); err != nil {
			return fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	return nil
}

// SaveConfig saves the configuration to a file
func SaveConfig(config *Config, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// NewEnvKeyReplacer returns a strings.Replacer for environment variable keys
func NewEnvKeyReplacer() *strings.Replacer {
	return strings.NewReplacer(".", "_")
}

// GetDefaultConfigPath returns the default configuration file path
func GetDefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/etc/firedragon", "config.yaml")
	}
	return filepath.Join(home, ".config", "firedragon", "config.yaml")
}

// GetConfigTemplate returns a template configuration
func GetConfigTemplate() *Config {
	return &Config{
		Firefly: FireflyConfig{
			URL:   "http://localhost:8080",
			Token: "your-token-here",
		},
		Ethereum: EthereumConfig{
			APIKey:      "your-etherscan-api-key",
			NetworkType: "mainnet",
			Addresses:   []string{"0x..."},
		},
		Solana: SolanaConfig{
			RPCEndpoint: "https://api.mainnet-beta.solana.com",
			NetworkType: "mainnet",
			Addresses:   []string{"..."},
		},
		Banking: BankingConfig{
			Enable: EnableBankingConfig{
				ClientID:     "your-client-id",
				ClientSecret: "your-client-secret",
				RedirectURI:  "http://localhost:8081/callback",
				AccountIDs:   []string{"account-id"},
			},
		},
		Service: ServiceConfig{
			UpdateInterval:   15 * time.Minute,
			MaxRetries:      3,
			RetryDelay:      time.Minute,
			LogLevel:        "info",
			MetricsEnabled:  true,
			MetricsInterval: time.Minute,
		},
	}
}
