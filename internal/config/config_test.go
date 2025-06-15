package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Test loading default configuration
	t.Run("Default config", func(t *testing.T) {
		config, err := LoadConfig("")
		if err != nil {
			t.Errorf("LoadConfig returned error: %v", err)
		}

		// Check default values
		if config.Server.Addr != ":8080" {
			t.Errorf("Expected server.addr to be ':8080', got '%s'", config.Server.Addr)
		}
		if config.Server.File != "sample.txt" {
			t.Errorf("Expected server.file to be 'sample.txt', got '%s'", config.Server.File)
		}
		if config.Server.Delay != 1000 {
			t.Errorf("Expected server.delay to be 1000, got %d", config.Server.Delay)
		}
		if config.Server.Stun != "" {
			t.Errorf("Expected server.stun to be empty, got '%s'", config.Server.Stun)
		}
		if config.Client.Server != "http://localhost:8080/offer" {
			t.Errorf("Expected client.server to be 'http://localhost:8080/offer', got '%s'", config.Client.Server)
		}
		if config.Client.Output != "" {
			t.Errorf("Expected client.output to be empty, got '%s'", config.Client.Output)
		}
		if config.Client.Stun != "" {
			t.Errorf("Expected client.stun to be empty, got '%s'", config.Client.Stun)
		}
	})

	// Test loading configuration from a file
	t.Run("Load from file", func(t *testing.T) {
		// Create a temporary config file
		tmpDir, err := os.MkdirTemp("", "config-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		configFile := filepath.Join(tmpDir, "config.yaml")
		configContent := `
server:
  addr: ":9090"
  file: "test.txt"
  delay: 500
  stun: "stun:stun.l.google.com:19302"
client:
  server: "http://localhost:9090/offer"
  output: "output.txt"
  stun: "stun:stun.l.google.com:19302"
`
		if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		// Load the config
		config, err := LoadConfig(configFile)
		if err != nil {
			t.Errorf("LoadConfig returned error: %v", err)
		}

		// Check values from the file
		if config.Server.Addr != ":9090" {
			t.Errorf("Expected server.addr to be ':9090', got '%s'", config.Server.Addr)
		}
		if config.Server.File != "test.txt" {
			t.Errorf("Expected server.file to be 'test.txt', got '%s'", config.Server.File)
		}
		if config.Server.Delay != 500 {
			t.Errorf("Expected server.delay to be 500, got %d", config.Server.Delay)
		}
		if config.Server.Stun != "stun:stun.l.google.com:19302" {
			t.Errorf("Expected server.stun to be 'stun:stun.l.google.com:19302', got '%s'", config.Server.Stun)
		}
		if config.Client.Server != "http://localhost:9090/offer" {
			t.Errorf("Expected client.server to be 'http://localhost:9090/offer', got '%s'", config.Client.Server)
		}
		if config.Client.Output != "output.txt" {
			t.Errorf("Expected client.output to be 'output.txt', got '%s'", config.Client.Output)
		}
		if config.Client.Stun != "stun:stun.l.google.com:19302" {
			t.Errorf("Expected client.stun to be 'stun:stun.l.google.com:19302', got '%s'", config.Client.Stun)
		}
	})

	// Test loading from a non-existent file (should use defaults)
	t.Run("Non-existent file", func(t *testing.T) {
		config, err := LoadConfig("non-existent-file.yaml")
		if err != nil {
			t.Errorf("LoadConfig returned error: %v", err)
		}

		// Check default values
		if config.Server.Addr != ":8080" {
			t.Errorf("Expected server.addr to be ':8080', got '%s'", config.Server.Addr)
		}
	})

	// Test loading from an invalid file
	t.Run("Invalid file", func(t *testing.T) {
		// Create a temporary invalid config file
		tmpDir, err := os.MkdirTemp("", "config-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		configFile := filepath.Join(tmpDir, "config.yaml")
		configContent := `
server:
  addr: ":9090"
  file: "test.txt"
  delay: "not-a-number" # This should cause an error
`
		if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		// Load the config
		_, err = LoadConfig(configFile)
		if err == nil {
			t.Error("LoadConfig should have returned an error for invalid file")
		}
	})
}

func TestSaveConfig(t *testing.T) {
	// Test saving configuration to a file
	t.Run("Save to file", func(t *testing.T) {
		// Create a temporary directory
		tmpDir, err := os.MkdirTemp("", "config-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		configFile := filepath.Join(tmpDir, "config.yaml")

		// Create a config to save
		config := &Config{
			Server: ServerConfig{
				Addr:  ":9090",
				File:  "test.txt",
				Delay: 500,
				Stun:  "stun:stun.l.google.com:19302",
			},
			Client: ClientConfig{
				Server: "http://localhost:9090/offer",
				Output: "output.txt",
				Stun:   "stun:stun.l.google.com:19302",
			},
		}

		// Save the config
		err = SaveConfig(config, configFile)
		if err != nil {
			t.Errorf("SaveConfig returned error: %v", err)
		}

		// Check that the file exists
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			t.Errorf("Config file was not created: %v", err)
		}

		// Load the config back and check values
		loadedConfig, err := LoadConfig(configFile)
		if err != nil {
			t.Errorf("LoadConfig returned error: %v", err)
		}

		// Check values
		if loadedConfig.Server.Addr != config.Server.Addr {
			t.Errorf("Expected server.addr to be '%s', got '%s'", config.Server.Addr, loadedConfig.Server.Addr)
		}
		if loadedConfig.Server.File != config.Server.File {
			t.Errorf("Expected server.file to be '%s', got '%s'", config.Server.File, loadedConfig.Server.File)
		}
		if loadedConfig.Server.Delay != config.Server.Delay {
			t.Errorf("Expected server.delay to be %d, got %d", config.Server.Delay, loadedConfig.Server.Delay)
		}
		if loadedConfig.Server.Stun != config.Server.Stun {
			t.Errorf("Expected server.stun to be '%s', got '%s'", config.Server.Stun, loadedConfig.Server.Stun)
		}
		if loadedConfig.Client.Server != config.Client.Server {
			t.Errorf("Expected client.server to be '%s', got '%s'", config.Client.Server, loadedConfig.Client.Server)
		}
		if loadedConfig.Client.Output != config.Client.Output {
			t.Errorf("Expected client.output to be '%s', got '%s'", config.Client.Output, loadedConfig.Client.Output)
		}
		if loadedConfig.Client.Stun != config.Client.Stun {
			t.Errorf("Expected client.stun to be '%s', got '%s'", config.Client.Stun, loadedConfig.Client.Stun)
		}
	})

	// Test saving to a directory that doesn't exist (should create it)
	t.Run("Save to non-existent directory", func(t *testing.T) {
		// Create a temporary directory
		tmpDir, err := os.MkdirTemp("", "config-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		configFile := filepath.Join(tmpDir, "subdir", "config.yaml")

		// Create a config to save
		config := &Config{
			Server: ServerConfig{
				Addr:  ":9090",
				File:  "test.txt",
				Delay: 500,
				Stun:  "",
			},
			Client: ClientConfig{
				Server: "http://localhost:9090/offer",
				Output: "",
				Stun:   "",
			},
		}

		// Save the config
		err = SaveConfig(config, configFile)
		if err != nil {
			t.Errorf("SaveConfig returned error: %v", err)
		}

		// Check that the file exists
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			t.Errorf("Config file was not created: %v", err)
		}
	})
}