package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/johnjeffers/awscogs/backend/internal/aws"
	"github.com/johnjeffers/awscogs/backend/internal/config"
	"github.com/johnjeffers/awscogs/backend/internal/types"
)

// CostsHandler handles cost-related requests
type CostsHandler struct {
	config    *config.Config
	discovery *aws.Discovery
}

// NewCostsHandler creates a new costs handler
func NewCostsHandler(cfg *config.Config, discovery *aws.Discovery) *CostsHandler {
	return &CostsHandler{
		config:    cfg,
		discovery: discovery,
	}
}

// GetCosts returns all cost data
func (h *CostsHandler) GetCosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse filters from query params
	accountFilter := parseArrayParam(r, "account")
	regionFilter := parseArrayParam(r, "region")
	resourceFilter := parseArrayParam(r, "resource")

	// Get regions (discover or use config)
	regions, err := h.getRegions(ctx, regionFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get accounts (discover or use config)
	accounts, err := h.getAccounts(ctx, accountFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Discover resources
	response, err := h.discovery.DiscoverResources(ctx, accounts, regions, resourceFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response.Timestamp = time.Now().UTC().Format(time.RFC3339)
	response.Filters = types.AppliedFilters{
		Accounts:      accountFilter,
		Regions:       regionFilter,
		ResourceTypes: resourceFilter,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAccountCosts returns account-level cost summaries
func (h *CostsHandler) GetAccountCosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountFilter := parseArrayParam(r, "account")
	regionFilter := parseArrayParam(r, "region")

	regions, err := h.getRegions(ctx, regionFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accounts, err := h.getAccounts(ctx, accountFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response, err := h.discovery.DiscoverResources(ctx, accounts, regions, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return only account summaries
	result := &types.CostResponse{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		TotalCost: response.TotalCost,
		Currency:  "USD",
		Accounts:  response.Accounts,
		Filters: types.AppliedFilters{
			Accounts: accountFilter,
			Regions:  regionFilter,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetRegionCosts returns region-level cost summaries
func (h *CostsHandler) GetRegionCosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountFilter := parseArrayParam(r, "account")
	regionFilter := parseArrayParam(r, "region")

	regions, err := h.getRegions(ctx, regionFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accounts, err := h.getAccounts(ctx, accountFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response, err := h.discovery.DiscoverResources(ctx, accounts, regions, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return only region summaries
	result := &types.CostResponse{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		TotalCost: response.TotalCost,
		Currency:  "USD",
		Regions:   response.Regions,
		Filters: types.AppliedFilters{
			Accounts: accountFilter,
			Regions:  regionFilter,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetEC2Costs returns EC2 instance costs
func (h *CostsHandler) GetEC2Costs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountFilter := parseArrayParam(r, "account")
	regionFilter := parseArrayParam(r, "region")

	regions, err := h.getRegions(ctx, regionFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accounts, err := h.getAccounts(ctx, accountFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response, err := h.discovery.DiscoverResources(ctx, accounts, regions, []string{"ec2"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate EC2-only total cost
	var ec2Total types.CostValue
	for _, inst := range response.EC2Instances {
		ec2Total += inst.HourlyCost
	}

	result := &types.CostResponse{
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		TotalCost:    ec2Total,
		Currency:     "USD",
		EC2Instances: response.EC2Instances,
		Filters: types.AppliedFilters{
			Accounts:      accountFilter,
			Regions:       regionFilter,
			ResourceTypes: []string{"ec2"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetEBSCosts returns EBS volume costs
func (h *CostsHandler) GetEBSCosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountFilter := parseArrayParam(r, "account")
	regionFilter := parseArrayParam(r, "region")

	regions, err := h.getRegions(ctx, regionFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accounts, err := h.getAccounts(ctx, accountFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response, err := h.discovery.DiscoverResources(ctx, accounts, regions, []string{"ebs"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate EBS-only total cost
	var ebsTotal types.CostValue
	for _, vol := range response.EBSVolumes {
		ebsTotal += vol.HourlyCost
	}

	result := &types.CostResponse{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		TotalCost:  ebsTotal,
		Currency:   "USD",
		EBSVolumes: response.EBSVolumes,
		Filters: types.AppliedFilters{
			Accounts:      accountFilter,
			Regions:       regionFilter,
			ResourceTypes: []string{"ebs"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetRDSCosts returns RDS instance costs
func (h *CostsHandler) GetRDSCosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountFilter := parseArrayParam(r, "account")
	regionFilter := parseArrayParam(r, "region")

	regions, err := h.getRegions(ctx, regionFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accounts, err := h.getAccounts(ctx, accountFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response, err := h.discovery.DiscoverResources(ctx, accounts, regions, []string{"rds"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate RDS-only total cost
	var rdsTotal types.CostValue
	for _, inst := range response.RDSInstances {
		rdsTotal += inst.HourlyCost
	}

	result := &types.CostResponse{
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		TotalCost:    rdsTotal,
		Currency:     "USD",
		RDSInstances: response.RDSInstances,
		Filters: types.AppliedFilters{
			Accounts:      accountFilter,
			Regions:       regionFilter,
			ResourceTypes: []string{"rds"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetECSCosts returns ECS service costs
func (h *CostsHandler) GetECSCosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountFilter := parseArrayParam(r, "account")
	regionFilter := parseArrayParam(r, "region")

	regions, err := h.getRegions(ctx, regionFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accounts, err := h.getAccounts(ctx, accountFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response, err := h.discovery.DiscoverResources(ctx, accounts, regions, []string{"ecs"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate ECS-only total cost
	var ecsTotal types.CostValue
	for _, svc := range response.ECSServices {
		ecsTotal += svc.HourlyCost
	}

	result := &types.CostResponse{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		TotalCost:   ecsTotal,
		Currency:    "USD",
		ECSServices: response.ECSServices,
		Filters: types.AppliedFilters{
			Accounts:      accountFilter,
			Regions:       regionFilter,
			ResourceTypes: []string{"ecs"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetEKSCosts returns EKS cluster costs
func (h *CostsHandler) GetEKSCosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountFilter := parseArrayParam(r, "account")
	regionFilter := parseArrayParam(r, "region")

	regions, err := h.getRegions(ctx, regionFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accounts, err := h.getAccounts(ctx, accountFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response, err := h.discovery.DiscoverResources(ctx, accounts, regions, []string{"eks"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate EKS-only total cost
	var eksTotal types.CostValue
	for _, cluster := range response.EKSClusters {
		eksTotal += cluster.HourlyCost
	}

	result := &types.CostResponse{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		TotalCost:   eksTotal,
		Currency:    "USD",
		EKSClusters: response.EKSClusters,
		Filters: types.AppliedFilters{
			Accounts:      accountFilter,
			Regions:       regionFilter,
			ResourceTypes: []string{"eks"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetELBCosts returns Elastic Load Balancer costs
func (h *CostsHandler) GetELBCosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountFilter := parseArrayParam(r, "account")
	regionFilter := parseArrayParam(r, "region")

	regions, err := h.getRegions(ctx, regionFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accounts, err := h.getAccounts(ctx, accountFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response, err := h.discovery.DiscoverResources(ctx, accounts, regions, []string{"elb"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate ELB-only total cost
	var elbTotal types.CostValue
	for _, lb := range response.LoadBalancers {
		elbTotal += lb.HourlyCost
	}

	result := &types.CostResponse{
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		TotalCost:     elbTotal,
		Currency:      "USD",
		LoadBalancers: response.LoadBalancers,
		Filters: types.AppliedFilters{
			Accounts:      accountFilter,
			Regions:       regionFilter,
			ResourceTypes: []string{"elb"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetNATGatewayCosts returns NAT Gateway costs
func (h *CostsHandler) GetNATGatewayCosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountFilter := parseArrayParam(r, "account")
	regionFilter := parseArrayParam(r, "region")

	regions, err := h.getRegions(ctx, regionFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accounts, err := h.getAccounts(ctx, accountFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response, err := h.discovery.DiscoverResources(ctx, accounts, regions, []string{"nat"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate NAT Gateway-only total cost
	var natTotal types.CostValue
	for _, nat := range response.NATGateways {
		natTotal += nat.HourlyCost
	}

	result := &types.CostResponse{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		TotalCost:   natTotal,
		Currency:    "USD",
		NATGateways: response.NATGateways,
		Filters: types.AppliedFilters{
			Accounts:      accountFilter,
			Regions:       regionFilter,
			ResourceTypes: []string{"nat"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetElasticIPCosts returns Elastic IP costs
func (h *CostsHandler) GetElasticIPCosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountFilter := parseArrayParam(r, "account")
	regionFilter := parseArrayParam(r, "region")

	regions, err := h.getRegions(ctx, regionFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accounts, err := h.getAccounts(ctx, accountFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response, err := h.discovery.DiscoverResources(ctx, accounts, regions, []string{"eip"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate EIP-only total cost
	var eipTotal types.CostValue
	for _, eip := range response.ElasticIPs {
		eipTotal += eip.HourlyCost
	}

	result := &types.CostResponse{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		TotalCost:  eipTotal,
		Currency:   "USD",
		ElasticIPs: response.ElasticIPs,
		Filters: types.AppliedFilters{
			Accounts:      accountFilter,
			Regions:       regionFilter,
			ResourceTypes: []string{"eip"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetSecretsCosts returns Secrets Manager costs
func (h *CostsHandler) GetSecretsCosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountFilter := parseArrayParam(r, "account")
	regionFilter := parseArrayParam(r, "region")

	regions, err := h.getRegions(ctx, regionFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accounts, err := h.getAccounts(ctx, accountFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response, err := h.discovery.DiscoverResources(ctx, accounts, regions, []string{"secrets"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate Secrets-only total cost
	var secretsTotal types.CostValue
	for _, secret := range response.Secrets {
		secretsTotal += secret.HourlyCost
	}

	result := &types.CostResponse{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		TotalCost: secretsTotal,
		Currency:  "USD",
		Secrets:   response.Secrets,
		Filters: types.AppliedFilters{
			Accounts:      accountFilter,
			Regions:       regionFilter,
			ResourceTypes: []string{"secrets"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetPublicIPv4Costs returns Public IPv4 address costs
func (h *CostsHandler) GetPublicIPv4Costs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountFilter := parseArrayParam(r, "account")
	regionFilter := parseArrayParam(r, "region")

	regions, err := h.getRegions(ctx, regionFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accounts, err := h.getAccounts(ctx, accountFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response, err := h.discovery.DiscoverResources(ctx, accounts, regions, []string{"publicipv4"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate Public IPv4-only total cost
	var publicIPv4Total types.CostValue
	for _, pip := range response.PublicIPv4s {
		publicIPv4Total += pip.HourlyCost
	}

	result := &types.CostResponse{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		TotalCost:   publicIPv4Total,
		Currency:    "USD",
		PublicIPv4s: response.PublicIPv4s,
		Filters: types.AppliedFilters{
			Accounts:      accountFilter,
			Regions:       regionFilter,
			ResourceTypes: []string{"publicipv4"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// getRegions returns regions to query - either from filter, discovery, or config
func (h *CostsHandler) getRegions(ctx context.Context, filter []string) ([]string, error) {
	// If filter specified, use that
	if len(filter) > 0 {
		return filter, nil
	}

	// If discovery enabled, discover regions
	if h.config.AWS.DiscoverRegions {
		return h.discovery.DiscoverRegions(ctx)
	}

	// Fall back to configured regions
	if len(h.config.AWS.Regions) > 0 {
		return h.config.AWS.Regions, nil
	}

	// Default fallback
	return []string{"us-east-1"}, nil
}

// getAccounts returns accounts to query - either from filter, discovery, or config
func (h *CostsHandler) getAccounts(ctx context.Context, filter []string) ([]aws.Account, error) {
	// If discovery enabled, discover accounts
	if h.config.AWS.DiscoverAccounts {
		accounts, err := h.discovery.DiscoverAccounts(ctx, h.config.AWS.AssumeRoleName)
		if err != nil {
			return nil, err
		}

		// If filter specified, filter the discovered accounts
		if len(filter) > 0 {
			filterSet := make(map[string]bool)
			for _, name := range filter {
				filterSet[name] = true
			}

			var filtered []aws.Account
			for _, acc := range accounts {
				if filterSet[acc.Name] || filterSet[acc.ID] {
					filtered = append(filtered, acc)
				}
			}
			return filtered, nil
		}

		return accounts, nil
	}

	// Use manually configured accounts
	if len(h.config.AWS.Accounts) > 0 {
		accounts := make([]aws.Account, 0, len(h.config.AWS.Accounts))
		for _, acc := range h.config.AWS.Accounts {
			accounts = append(accounts, aws.Account{
				Name:    acc.Name,
				RoleARN: acc.RoleARN,
			})
		}

		// If filter specified, filter the configured accounts
		if len(filter) > 0 {
			filterSet := make(map[string]bool)
			for _, name := range filter {
				filterSet[name] = true
			}

			var filtered []aws.Account
			for _, acc := range accounts {
				if filterSet[acc.Name] {
					filtered = append(filtered, acc)
				}
			}
			return filtered, nil
		}

		return accounts, nil
	}

	// Default: use current credentials (empty account list triggers default behavior)
	return nil, nil
}

// parseArrayParam parses a comma-separated query parameter into a slice
func parseArrayParam(r *http.Request, key string) []string {
	value := r.URL.Query().Get(key)
	if value == "" {
		return nil
	}
	return strings.Split(value, ",")
}
