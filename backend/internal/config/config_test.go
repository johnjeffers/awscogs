package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGovCloudRequiresEnableEnvGate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("aws:\n  govcloud:\n    enabled: true\n    regions:\n      - us-gov-west-1\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AWS.GovCloud.Enabled {
		t.Fatal("GovCloud should stay disabled without AWSCOGS_ENABLE_GOVCLOUD")
	}
}

func TestEnableGovCloudDisablesCommercialDefaultsWhenUnconfigured(t *testing.T) {
	t.Setenv("AWSCOGS_ENABLE_GOVCLOUD", "true")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.AWS.GovCloud.Enabled {
		t.Fatal("GovCloud should be enabled")
	}
	if cfg.AWS.DiscoverAccounts {
		t.Fatal("commercial account discovery should be disabled for GovCloud-only config")
	}
	if cfg.AWS.DiscoverRegions {
		t.Fatal("commercial region discovery should be disabled for GovCloud-only config")
	}
	if !cfg.AWS.GovCloud.DiscoverRegions {
		t.Fatal("GovCloud region discovery should default to enabled")
	}
}

func TestGovCloudAccountsFromEnv(t *testing.T) {
	t.Setenv("AWSCOGS_ENABLE_GOVCLOUD", "1")
	t.Setenv("AWSCOGS_GOVCLOUD_ACCOUNTS", "prod=arn:aws-us-gov:iam::123456789012:role/Audit,arn:aws-us-gov:iam::210987654321:role/Audit")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := len(cfg.AWS.GovCloud.Accounts); got != 2 {
		t.Fatalf("expected 2 GovCloud accounts, got %d", got)
	}
	if cfg.AWS.GovCloud.DiscoverAccounts {
		t.Fatal("explicit GovCloud accounts should disable GovCloud account discovery")
	}
	if cfg.AWS.GovCloud.Accounts[0].Name != "prod" {
		t.Fatalf("first account name = %q", cfg.AWS.GovCloud.Accounts[0].Name)
	}
	if cfg.AWS.GovCloud.Accounts[0].RoleARN != "arn:aws-us-gov:iam::123456789012:role/Audit" {
		t.Fatalf("first account role = %q", cfg.AWS.GovCloud.Accounts[0].RoleARN)
	}
	if cfg.AWS.GovCloud.Accounts[1].Name != "210987654321" {
		t.Fatalf("bare ARN account name = %q", cfg.AWS.GovCloud.Accounts[1].Name)
	}
}
