package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server ServerConfig
	Client ClientConfig
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	Addr  string
	File  string
	Delay int
	Stun  string
}

// ClientConfig represents the client configuration
type ClientConfig struct {
	Server string
	Output string
	Stun   string
}

// LoadConfig loads the configuration from the specified file
func LoadConfig(configFile string) (*Config, error) {
	v := viper.New()

	// Set default configuration values
	setDefaults(v)

	if configFile != "" {
		// Use config file from the flag
		v.SetConfigFile(configFile)
	} else {
		// Search for config in current directory with name "config" (without extension)
		v.AddConfigPath(".")
		v.SetConfigName("config")
	}

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		if configFile != "" && os.IsNotExist(err) {
			// Specific config file was provided but doesn't exist, using defaults
		} else if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error was produced
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, using defaults
	} else {
		fmt.Println("Using config file:", v.ConfigFileUsed())
	}

	// Parse the config
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	return &config, nil
}

// SaveConfig saves the configuration to the specified file
func SaveConfig(config *Config, configFile string) error {
	v := viper.New()

	// Set the config values
	v.Set("server.addr", config.Server.Addr)
	v.Set("server.file", config.Server.File)
	v.Set("server.delay", config.Server.Delay)
	v.Set("server.stun", config.Server.Stun)
	v.Set("client.server", config.Client.Server)
	v.Set("client.output", config.Client.Output)
	v.Set("client.stun", config.Client.Stun)

	// Create the directory if it doesn't exist
	dir := filepath.Dir(configFile)
	if dir != "." && dir != "/" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creating config directory: %w", err)
		}
	}

	// Write the config file
	v.SetConfigFile(configFile)
	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

// setDefaults sets the default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.addr", ":8080")
	v.SetDefault("server.file", "sample.txt")
	v.SetDefault("server.delay", 1000)
	v.SetDefault("server.stun", "")

	// Client defaults
	v.SetDefault("client.server", "http://localhost:8080/offer")
	v.SetDefault("client.output", "")
	v.SetDefault("client.stun", "")
}
