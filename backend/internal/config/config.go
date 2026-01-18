package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	AWS     AWSConfig     `yaml:"aws"`
	Pricing PricingConfig `yaml:"pricing"`
	Log     LogConfig     `yaml:"log"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Port int `yaml:"port"`
}

// AWSConfig holds AWS account and region settings
type AWSConfig struct {
	DiscoverAccounts bool            `yaml:"discoverAccounts"` // Auto-discover accounts from Organizations
	DiscoverRegions  bool            `yaml:"discoverRegions"`  // Auto-discover enabled regions
	AssumeRoleName   string          `yaml:"assumeRoleName"`   // Role name to assume into each account
	Accounts         []AccountConfig `yaml:"accounts"`         // Manual account list (used if discoverAccounts is false)
	Regions          []string        `yaml:"regions"`          // Manual region list (used if discoverRegions is false)
}

// AccountConfig defines how to connect to a specific AWS account
type AccountConfig struct {
	Name    string `yaml:"name"`
	RoleARN string `yaml:"roleArn,omitempty"`
}

// PricingConfig holds AWS pricing settings
type PricingConfig struct {
	RefreshIntervalMinutes int `yaml:"refreshIntervalMinutes"`
	RateLimitPerSecond     int `yaml:"rateLimitPerSecond"` // Max pricing API calls per second (0 = unlimited)
}

// LogConfig holds logging settings
type LogConfig struct {
	Level string `yaml:"level"`
}

// DefaultConfig returns configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 8080,
		},
		AWS: AWSConfig{
			DiscoverAccounts: true,
			DiscoverRegions:  true,
			AssumeRoleName:   "OrganizationAccountAccessRole",
		},
		Pricing: PricingConfig{
			RefreshIntervalMinutes: 60,
			RateLimitPerSecond:     5, // Conservative default to avoid AWS throttling
		},
		Log: LogConfig{
			Level: "info",
		},
	}
}

// Load reads configuration from file and environment
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	// Load from file if provided
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parsing config file: %w", err)
		}
	}

	// Override with environment variables
	cfg.loadFromEnv()

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// loadFromEnv overrides config values from environment variables
func (c *Config) loadFromEnv() {
	if port := os.Getenv("AWSCOGS_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.Server.Port = p
		}
	}

	if level := os.Getenv("AWSCOGS_LOG_LEVEL"); level != "" {
		c.Log.Level = level
	}

	if regions := os.Getenv("AWSCOGS_REGIONS"); regions != "" {
		c.AWS.Regions = strings.Split(regions, ",")
		c.AWS.DiscoverRegions = false // Disable discovery if explicit regions set
	}

	if discoverRegions := os.Getenv("AWSCOGS_DISCOVER_REGIONS"); discoverRegions != "" {
		c.AWS.DiscoverRegions = discoverRegions == "true" || discoverRegions == "1"
	}

	if discoverAccounts := os.Getenv("AWSCOGS_DISCOVER_ACCOUNTS"); discoverAccounts != "" {
		c.AWS.DiscoverAccounts = discoverAccounts == "true" || discoverAccounts == "1"
	}

	if assumeRole := os.Getenv("AWSCOGS_ASSUME_ROLE_NAME"); assumeRole != "" {
		c.AWS.AssumeRoleName = assumeRole
	}

	if interval := os.Getenv("AWSCOGS_PRICING_REFRESH_MINUTES"); interval != "" {
		if i, err := strconv.Atoi(interval); err == nil {
			c.Pricing.RefreshIntervalMinutes = i
		}
	}

	if rateLimit := os.Getenv("AWSCOGS_PRICING_RATE_LIMIT"); rateLimit != "" {
		if r, err := strconv.Atoi(rateLimit); err == nil {
			c.Pricing.RateLimitPerSecond = r
		}
	}
}

// Validate checks the configuration for errors
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Server.Port)
	}

	if c.Pricing.RefreshIntervalMinutes < 1 {
		return fmt.Errorf("pricing refresh interval must be at least 1 minute")
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Log.Level] {
		return fmt.Errorf("invalid log level: %s", c.Log.Level)
	}

	return nil
}
