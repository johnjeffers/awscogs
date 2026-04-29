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
	Cache   CacheConfig   `yaml:"cache"`
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
	GovCloud         GovCloudConfig  `yaml:"govcloud"`         // GovCloud partition settings
}

// GovCloudConfig holds settings for the AWS GovCloud partition
type GovCloudConfig struct {
	Enabled          bool            `yaml:"enabled"`          // Effective GovCloud flag; requires AWSCOGS_ENABLE_GOVCLOUD
	DiscoverAccounts bool            `yaml:"discoverAccounts"` // Auto-discover GovCloud accounts from Organizations
	DiscoverRegions  bool            `yaml:"discoverRegions"`  // Auto-discover enabled GovCloud regions
	Regions          []string        `yaml:"regions"`          // Explicit GovCloud region list
	Accounts         []AccountConfig `yaml:"accounts"`         // GovCloud accounts (must have roleArn in aws-us-gov partition)
	AssumeRoleName   string          `yaml:"assumeRoleName"`   // Role name for GovCloud Organizations assume role
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

// CacheConfig holds cache settings
type CacheConfig struct {
	ResourceTTLMinutes int `yaml:"resourceTTLMinutes"` // TTL for resource discovery cache
	AccountTTLMinutes  int `yaml:"accountTTLMinutes"`  // TTL for account/region discovery cache
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
			GovCloud: GovCloudConfig{
				DiscoverRegions: true,
				AssumeRoleName:  "OrganizationAccountAccessRole",
			},
		},
		Pricing: PricingConfig{
			RefreshIntervalMinutes: 60,
			RateLimitPerSecond:     5, // Conservative default to avoid AWS throttling
		},
		Cache: CacheConfig{
			ResourceTTLMinutes: 5,  // Resource discovery cache TTL
			AccountTTLMinutes:  60, // Account/region discovery cache TTL
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

	discoverRegionsSet := false
	if discoverRegions, ok := boolEnv("AWSCOGS_DISCOVER_REGIONS"); ok {
		c.AWS.DiscoverRegions = discoverRegions
		discoverRegionsSet = true
	}

	discoverAccountsSet := false
	if discoverAccounts, ok := boolEnv("AWSCOGS_DISCOVER_ACCOUNTS"); ok {
		c.AWS.DiscoverAccounts = discoverAccounts
		discoverAccountsSet = true
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

	if resourceTTL := os.Getenv("AWSCOGS_CACHE_RESOURCE_TTL_MINUTES"); resourceTTL != "" {
		if t, err := strconv.Atoi(resourceTTL); err == nil {
			c.Cache.ResourceTTLMinutes = t
		}
	}

	if accountTTL := os.Getenv("AWSCOGS_CACHE_ACCOUNT_TTL_MINUTES"); accountTTL != "" {
		if t, err := strconv.Atoi(accountTTL); err == nil {
			c.Cache.AccountTTLMinutes = t
		}
	}

	// GovCloud environment variables
	if govEnabled, ok := boolEnv("AWSCOGS_ENABLE_GOVCLOUD"); ok {
		c.AWS.GovCloud.Enabled = govEnabled
	} else {
		// GovCloud config is inert unless explicitly enabled by environment.
		c.AWS.GovCloud.Enabled = false
	}

	if govRegions := os.Getenv("AWSCOGS_GOVCLOUD_REGIONS"); govRegions != "" {
		c.AWS.GovCloud.Regions = splitCSV(govRegions)
		c.AWS.GovCloud.DiscoverRegions = false
	}

	if govAccounts := os.Getenv("AWSCOGS_GOVCLOUD_ACCOUNTS"); govAccounts != "" {
		c.AWS.GovCloud.Accounts = parseAccountList(govAccounts)
		c.AWS.GovCloud.DiscoverAccounts = false
	}

	if govDiscoverAccounts, ok := boolEnv("AWSCOGS_GOVCLOUD_DISCOVER_ACCOUNTS"); ok {
		c.AWS.GovCloud.DiscoverAccounts = govDiscoverAccounts
	}

	if govDiscoverRegions, ok := boolEnv("AWSCOGS_GOVCLOUD_DISCOVER_REGIONS"); ok {
		c.AWS.GovCloud.DiscoverRegions = govDiscoverRegions
	}

	if govAssumeRole := os.Getenv("AWSCOGS_GOVCLOUD_ASSUME_ROLE_NAME"); govAssumeRole != "" {
		c.AWS.GovCloud.AssumeRoleName = govAssumeRole
	}

	if c.AWS.GovCloud.Enabled && len(c.AWS.Accounts) == 0 && len(c.AWS.Regions) == 0 {
		if !discoverAccountsSet {
			c.AWS.DiscoverAccounts = false
		}
		if !discoverRegionsSet {
			c.AWS.DiscoverRegions = false
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

func boolEnv(name string) (bool, bool) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return false, false
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true, true
	default:
		return false, true
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func parseAccountList(value string) []AccountConfig {
	entries := splitCSV(value)
	accounts := make([]AccountConfig, 0, len(entries))
	for _, entry := range entries {
		if name, roleARN, ok := strings.Cut(entry, "="); ok {
			accounts = append(accounts, AccountConfig{
				Name:    strings.TrimSpace(name),
				RoleARN: strings.TrimSpace(roleARN),
			})
			continue
		}

		account := AccountConfig{Name: entry}
		if strings.HasPrefix(entry, "arn:") {
			account.Name = accountNameFromRoleARN(entry)
			account.RoleARN = entry
		}
		accounts = append(accounts, account)
	}
	return accounts
}

func accountNameFromRoleARN(roleARN string) string {
	parts := strings.Split(roleARN, ":")
	if len(parts) > 4 && parts[4] != "" {
		return parts[4]
	}
	return roleARN
}
