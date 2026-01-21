package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/johnjeffers/awscogs/backend/internal/aws"
	"github.com/johnjeffers/awscogs/backend/internal/config"
	"github.com/johnjeffers/awscogs/backend/internal/version"
)

// ConfigHandler handles configuration requests
type ConfigHandler struct {
	config    *config.Config
	discovery *aws.Discovery
}

// NewConfigHandler creates a new config handler
func NewConfigHandler(cfg *config.Config, discovery *aws.Discovery) *ConfigHandler {
	return &ConfigHandler{
		config:    cfg,
		discovery: discovery,
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
	var err error

	if h.config.AWS.DiscoverRegions {
		regions, err = h.discovery.DiscoverRegions(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if len(h.config.AWS.Regions) > 0 {
		regions = h.config.AWS.Regions
	} else {
		regions = []string{"us-east-1"}
	}

	// Get available accounts
	var accounts []AccountInfo

	if h.config.AWS.DiscoverAccounts {
		discoveredAccounts, err := h.discovery.DiscoverAccounts(ctx, h.config.AWS.AssumeRoleName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		accounts = make([]AccountInfo, len(discoveredAccounts))
		for i, acc := range discoveredAccounts {
			accounts[i] = AccountInfo{
				ID:   acc.ID,
				Name: acc.Name,
			}
		}
	} else if len(h.config.AWS.Accounts) > 0 {
		accounts = make([]AccountInfo, len(h.config.AWS.Accounts))
		for i, acc := range h.config.AWS.Accounts {
			accounts[i] = AccountInfo{Name: acc.Name}
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
	json.NewEncoder(w).Encode(response)
}
