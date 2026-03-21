package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/johnjeffers/awscogs/backend/internal/aws"
	"github.com/johnjeffers/awscogs/backend/internal/config"
	"github.com/johnjeffers/awscogs/backend/internal/version"
)

// ConfigHandler handles configuration requests
type ConfigHandler struct {
	config    *config.Config
	discovery *aws.Discovery
	logger    *slog.Logger
}

// NewConfigHandler creates a new config handler
func NewConfigHandler(cfg *config.Config, discovery *aws.Discovery, logger *slog.Logger) *ConfigHandler {
	return &ConfigHandler{
		config:    cfg,
		discovery: discovery,
		logger:    logger,
	}
}

// VersionInfo provides application version information
type VersionInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildTime string `json:"buildTime"`
}

// ConfigResponse is the response for configuration
type ConfigResponse struct {
	Accounts []AccountInfo `json:"accounts"`
	Regions  []string      `json:"regions"`
	Version  VersionInfo   `json:"version"`
}

// AccountInfo provides account information
type AccountInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetConfig returns available accounts and regions
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get available regions
	var regions []string

	if h.config.AWS.DiscoverRegions {
		discovered, err := h.discovery.DiscoverRegions(ctx)
		if err != nil {
			h.logger.Error("failed to discover regions", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		regions = append(regions, discovered...)
	} else if len(h.config.AWS.Regions) > 0 {
		regions = append(regions, h.config.AWS.Regions...)
	}

	// Append GovCloud regions
	if h.config.AWS.GovCloud.Enabled {
		if h.config.AWS.GovCloud.DiscoverRegions && len(h.config.AWS.GovCloud.Accounts) > 0 {
			firstAccount := aws.Account{
				Name:      h.config.AWS.GovCloud.Accounts[0].Name,
				RoleARN:   h.config.AWS.GovCloud.Accounts[0].RoleARN,
				Partition: "aws-us-gov",
			}
			govRegions, err := h.discovery.DiscoverGovCloudRegions(ctx, firstAccount)
			if err != nil {
				h.logger.Error("failed to discover govcloud regions", "error", err)
			} else {
				regions = append(regions, govRegions...)
			}
		} else if len(h.config.AWS.GovCloud.Regions) > 0 {
			regions = append(regions, h.config.AWS.GovCloud.Regions...)
		}
	}

	if len(regions) == 0 {
		regions = []string{"us-east-1"}
	}

	// Get available accounts
	var accounts []AccountInfo

	if h.config.AWS.DiscoverAccounts {
		discoveredAccounts, err := h.discovery.DiscoverAccounts(ctx, h.config.AWS.AssumeRoleName)
		if err != nil {
			h.logger.Error("failed to discover accounts", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		for _, acc := range discoveredAccounts {
			accounts = append(accounts, AccountInfo{
				ID:   acc.ID,
				Name: acc.Name,
			})
		}
	} else if len(h.config.AWS.Accounts) > 0 {
		for _, acc := range h.config.AWS.Accounts {
			accounts = append(accounts, AccountInfo{Name: acc.Name})
		}
	}

	// Append GovCloud accounts
	if h.config.AWS.GovCloud.Enabled {
		for _, acc := range h.config.AWS.GovCloud.Accounts {
			accounts = append(accounts, AccountInfo{Name: acc.Name})
		}
	}

	response := ConfigResponse{
		Accounts: accounts,
		Regions:  regions,
		Version: VersionInfo{
			Version:   version.Version,
			GitCommit: version.GitCommit,
			BuildTime: version.BuildTime,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}
