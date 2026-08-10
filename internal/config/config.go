package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// GetConfigPath resolves the OS-specific configuration path (~/.config/p2pdrop/config.json)
func GetConfigPath() (string, error) {
	baseDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not locate user config dir: %w", err)
	}

	appDir := filepath.Join(baseDir, "p2pdrop")

	// Ensure the ~/.config/p2pdrop directory exists
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", fmt.Errorf("could not create config directory: %w", err)
	}

	return filepath.Join(appDir, "config.json"), nil
}

type Config struct {
	mu sync.RWMutex `json:"-"`

	DeviceName string `json:"device_name"`
	SaveDir    string `json:"save_dir"`
	UDPPort    uint16 `json:"udp_port"`

	IsConfigured bool `json:"is_configured"`
}

// NewConfig attempts to load config from disk, falling back to safe defaults if not found.
func NewConfig() (*Config, error) {
	if configPath, err := GetConfigPath(); err == nil {
		// 1. Try to read existing config.json from disk
		if data, err := os.ReadFile(configPath); err == nil {
			var cfg Config
			if err := json.Unmarshal(data, &cfg); err == nil {
				cfg.IsConfigured = true
				return &cfg, nil
			}
		}
	}

	// 2. Fallback to OS Hostname and default settings
	name, err := os.Hostname()
	if err != nil {
		name = "P2P-Node"
	}

	return &Config{
		DeviceName:   name,
		SaveDir:      "./uploads",
		UDPPort:      9696, // Port within valid 1-65535 range
		IsConfigured: false,
	}, nil
}

// WriteConfig updates config fields in memory and persists them to config.json on disk.
func (c *Config) WriteConfig(deviceName string, saveDir string, udpPort uint16) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.DeviceName = deviceName
	c.SaveDir = saveDir
	c.UDPPort = udpPort

	c.IsConfigured = true

	// Ensure destination upload directory exists on disk
	if err := os.MkdirAll(c.SaveDir, 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Save to JSON file
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}

func (c *Config) IsReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.IsConfigured
}

// Get returns a thread-safe snapshot copy of current settings.
func (c *Config) Get() Config {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return Config{
		DeviceName:   c.DeviceName,
		SaveDir:      c.SaveDir,
		UDPPort:      c.UDPPort,
		IsConfigured: c.IsConfigured,
	}
}
