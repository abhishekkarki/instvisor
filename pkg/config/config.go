package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Collection CollectionConfig `yaml:"collection"`
	Storage    StorageConfig    `yaml:"storage"`
	Collectors CollectorsConfig `yaml:"collectors"`
	Server     ServerConfig     `yaml:"server"`
}

type CollectionConfig struct {
	Interval      time.Duration `yaml:"interval"`
	RetentionDays int           `yaml:"retention_days"`
}

type StorageConfig struct {
	Path string `yaml:"path"`
}

type CollectorsConfig struct {
	CPU       CPUCollectorConfig       `yaml:"cpu"`
	Memory    MemoryCollectorConfig    `yaml:"memory"`
	Disk      DiskCollectorConfig      `yaml:"disk"`
	Network   NetworkCollectorConfig   `yaml:"network"`
	Process   ProcessCollectorConfig   `yaml:"process"`
	Container ContainerCollectorConfig `yaml:"container"`
}

type CPUCollectorConfig struct {
	Enabled bool `yaml:"enabled"`
	PerCore bool `yaml:"per_core"`
}

type MemoryCollectorConfig struct {
	Enabled     bool `yaml:"enabled"`
	IncludeSwap bool `yaml:"include_swap"`
}

type DiskCollectorConfig struct {
	Enabled bool     `yaml:"enabled"`
	Devices []string `yaml:"devices"`
}

type NetworkCollectorConfig struct {
	Enabled    bool     `yaml:"enabled"`
	Interfaces []string `yaml:"interfaces"`
}

type ProcessCollectorConfig struct {
	Enabled bool `yaml:"enabled"`
	TopN    int  `yaml:"top_n"`
}

type ServerConfig struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}

type ContainerCollectorConfig struct {
	Enabled bool `yaml:"enabled"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if config.Collection.Interval == 0 {
		config.Collection.Interval = 15 * time.Second
	}
	if config.Collection.RetentionDays == 0 {
		config.Collection.RetentionDays = 90
	}
	if config.Storage.Path == "" {
		config.Storage.Path = "/var/lib/instvisor/metrics.db"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 9090
	}

	return &config, nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Collection: CollectionConfig{
			Interval:      15 * time.Second,
			RetentionDays: 90,
		},
		Storage: StorageConfig{
			Path: "/var/lib/instvisor/metrics.db",
		},
		Collectors: CollectorsConfig{
			CPU: CPUCollectorConfig{
				Enabled: true,
				PerCore: true,
			},
			Memory: MemoryCollectorConfig{
				Enabled:     true,
				IncludeSwap: true,
			},
			Disk: DiskCollectorConfig{
				Enabled: true,
				Devices: []string{},
			},
			Network: NetworkCollectorConfig{
				Enabled:    true,
				Interfaces: []string{},
			},
			Process: ProcessCollectorConfig{
				Enabled: true,
				TopN:    10,
			},
			Container: ContainerCollectorConfig{
				Enabled: true,
			},
		},
		Server: ServerConfig{
			Enabled: false,
			Port:    9090,
		},
	}
}
