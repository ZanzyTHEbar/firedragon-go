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

// Config represents the server configuration
type Config struct {
	// NATS configuration
	NATS struct {
		ServerURL      string   `mapstructure:"server_url"`
		Username       string   `mapstructure:"username"`
		Password       string   `mapstructure:"password"`
		Token          string   `mapstructure:"token"`
		StreamName     string   `mapstructure:"stream_name"`
		Subjects       []string `mapstructure:"subjects"`
		CommandChannel string   `mapstructure:"command_channel"`
	} `mapstructure:"nats"`

	// Client configuration
	Client struct {
		Screen struct {
			Interval int    `mapstructure:"interval"` // Interval in seconds, 0 means disabled
			Quality  string `mapstructure:"quality"`  // high, medium, low
			Format   string `mapstructure:"format"`   // png, jpg
		} `mapstructure:"screen"`
		HID struct {
			BufferSize int `mapstructure:"buffer_size"` // Event buffer size
		} `mapstructure:"hid"`
		Transcript struct {
			Language     string `mapstructure:"language"`      // Default language code
			BufferSize   int    `mapstructure:"buffer_size"`   // Event buffer size
			AutoStart    bool   `mapstructure:"auto_start"`    // Start transcription on launch
			OutputFormat string `mapstructure:"output_format"` // text, json
		} `mapstructure:"transcript"`
	} `mapstructure:"client"`

	// Server configuration
	Server struct {
		LogLevel        string        `mapstructure:"log_level"`
		Port            int           `mapstructure:"port"`
		StoragePath     string        `mapstructure:"storage_path"`
		RetentionPeriod time.Duration `mapstructure:"retention_period"`
	} `mapstructure:"server"`

	Debug bool `mapstructure:"debug"`
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
		// Look for config in the current directory and in /etc/perception-engine/
		v.AddConfigPath(".")
		v.AddConfigPath("/etc/perception-engine/")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		// It's okay if the config file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	// Override with environment variables prefixed with PE_
	v.SetEnvPrefix("PE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Parse the configuration
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Special handling for environment variables that don't follow the pattern
	if os.Getenv("NATS_USERNAME") != "" {
		config.NATS.Username = os.Getenv("NATS_USERNAME")
	}
	if os.Getenv("NATS_PASSWORD") != "" {
		config.NATS.Password = os.Getenv("NATS_PASSWORD")
	}
	if os.Getenv("NATS_TOKEN") != "" {
		config.NATS.Token = os.Getenv("NATS_TOKEN")
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
	// NATS defaults
	v.SetDefault("nats.server_url", "nats://localhost:4222")
	v.SetDefault("nats.stream_name", "perception-events")
	v.SetDefault("nats.subjects", []string{
		"local.*.client.data.*",
		"local.*.browser.data.*",
	})
	v.SetDefault("nats.command_channel", "perception-commands")

	// Server defaults
	v.SetDefault("server.log_level", "debug")
	v.SetDefault("server.port", 4222)
	v.SetDefault("server.storage_path", filepath.Join(".", "data"))
	v.SetDefault("server.retention_period", 24*time.Hour)
}
