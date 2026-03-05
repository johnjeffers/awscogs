package aws

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/johnjeffers/awscogs/backend/internal/pricing"
	"github.com/johnjeffers/awscogs/backend/internal/types"
)

// cacheEntry holds a cached value with expiration
type cacheEntry[T any] struct {
	value     T
	expiresAt time.Time
}

// Discovery handles AWS resource discovery across accounts and regions
type Discovery struct {
	pricingProvider pricing.Provider
	logger          *slog.Logger

	// Cache settings
	resourceTTL time.Duration
	accountTTL  time.Duration

	// Resource discovery cache - keyed by "accountID|region|resourceType"
	resourceCache   map[string]cacheEntry[any]
	resourceCacheMu sync.RWMutex

	// Account discovery cache
	accountCache   *cacheEntry[[]Account]
	accountCacheMu sync.RWMutex

	// Region discovery cache
	regionCache   *cacheEntry[[]string]
	regionCacheMu sync.RWMutex

	// ELB usage cache - keyed by "accountID|region|window"
	usageCache   map[string]cacheEntry[map[string]elbUsageData]
	usageCacheMu sync.RWMutex

	// Semaphore for CloudWatch concurrency control
	cwSemaphore chan struct{}
}

// elbUsageData holds CloudWatch usage metrics for a single load balancer
type elbUsageData struct {
	RequestVolume       float64
	RequestMetricName   string
	BandwidthBytes      float64
	BandwidthMetricName string
	Status              string
	Error               string
}

// NewDiscovery creates a new AWS resource discovery service
func NewDiscovery(pricingProvider pricing.Provider, logger *slog.Logger, resourceTTLMinutes, accountTTLMinutes int) *Discovery {
	return &Discovery{
		pricingProvider: pricingProvider,
		logger:          logger,
		resourceTTL:     time.Duration(resourceTTLMinutes) * time.Minute,
		accountTTL:      time.Duration(accountTTLMinutes) * time.Minute,
		resourceCache:   make(map[string]cacheEntry[any]),
		usageCache:      make(map[string]cacheEntry[map[string]elbUsageData]),
		cwSemaphore:     make(chan struct{}, 10),
	}
}

// Account represents an AWS account configuration
type Account struct {
	ID      string
	Name    string
	RoleARN string
}

// resourceCacheKey creates a cache key for a specific account/region/resourceType combination
func resourceCacheKey(accountID, region, resourceType string) string {
	return accountID + "|" + region + "|" + resourceType
}

// shouldDiscover checks if a resource type should be discovered based on the filter
func shouldDiscover(resourceTypes []string, resourceType string) bool {
	if len(resourceTypes) == 0 {
		return true // No filter means discover all
	}
	for _, rt := range resourceTypes {
		if rt == resourceType {
			return true
		}
	}
	return false
}

// DiscoverResources discovers all resources across the specified accounts and regions
// resourceTypes filter: empty means all, otherwise only discover specified types (ec2, ebs, ecs, rds, eks, elb, nat, eip, secrets, publicipv4)
func (d *Discovery) DiscoverResources(ctx context.Context, accounts []Account, regions []string, resourceTypes []string) (*types.CostResponse, error) {
	var (
		allEC2        []types.EC2Instance
		allEBS        []types.EBSVolume
		allECS        []types.ECSService
		allRDS        []types.RDSInstance
		allEKS        []types.EKSCluster
		allELB        []types.LoadBalancer
		allNAT        []types.NATGateway
		allEIP        []types.ElasticIP
		allSecrets    []types.Secret
		allPublicIPv4 []types.PublicIPv4
		mu            sync.Mutex
		wg            sync.WaitGroup
		totalCost     types.CostValue
	)

	// If no accounts specified, use default credentials
	if len(accounts) == 0 {
		accounts = []Account{{}}
	}

	for _, account := range accounts {
		for _, region := range regions {
			wg.Add(1)
			go func(acc Account, reg string) {
				defer wg.Done()

				cfg, err := d.getConfigForAccount(ctx, acc, reg)
				if err != nil {
					d.logger.Error("failed to get config for account",
						"account", acc.Name,
						"region", reg,
						"error", err)
					return
				}

				// Get account ID if not set
				accountID := acc.ID
				if accountID == "" {
					accountID, err = d.getAccountID(ctx, cfg)
					if err != nil {
						d.logger.Warn("failed to get account ID", "error", err)
						accountID = "unknown"
					}
				}

				// Resolve account name: use configured name, or fetch alias, or fall back to account ID
				accountName := acc.Name
				if accountName == "" {
					accountName = d.getAccountAlias(ctx, cfg)
					if accountName == "" {
						accountName = accountID
					}
				}

				var ec2Instances []types.EC2Instance
				var ebsVolumes []types.EBSVolume
				var ecsServices []types.ECSService
				var rdsInstances []types.RDSInstance
				var eksClusters []types.EKSCluster
				var loadBalancers []types.LoadBalancer
				var natGateways []types.NATGateway
				var elasticIPs []types.ElasticIP
				var secrets []types.Secret
				var publicIPv4s []types.PublicIPv4

				// Discover EC2 instances
				if shouldDiscover(resourceTypes, "ec2") {
					ec2Instances = d.getOrDiscoverEC2(ctx, cfg, accountID, accountName, reg)
				}

				// Discover EBS volumes
				if shouldDiscover(resourceTypes, "ebs") {
					ebsVolumes = d.getOrDiscoverEBS(ctx, cfg, accountID, accountName, reg)
				}

				// Discover ECS services
				if shouldDiscover(resourceTypes, "ecs") {
					ecsServices = d.getOrDiscoverECS(ctx, cfg, accountID, accountName, reg)
				}

				// Discover RDS instances
				if shouldDiscover(resourceTypes, "rds") {
					rdsInstances = d.getOrDiscoverRDS(ctx, cfg, accountID, accountName, reg)
				}

				// Discover EKS clusters
				if shouldDiscover(resourceTypes, "eks") {
					eksClusters = d.getOrDiscoverEKS(ctx, cfg, accountID, accountName, reg)
				}

				// Discover Load Balancers
				if shouldDiscover(resourceTypes, "elb") {
					loadBalancers = d.getOrDiscoverELB(ctx, cfg, accountID, accountName, reg)
				}

				// Discover NAT Gateways
				if shouldDiscover(resourceTypes, "nat") {
					natGateways = d.getOrDiscoverNATGateways(ctx, cfg, accountID, accountName, reg)
				}

				// Discover Elastic IPs
				if shouldDiscover(resourceTypes, "eip") {
					elasticIPs = d.getOrDiscoverElasticIPs(ctx, cfg, accountID, accountName, reg)
				}

				// Discover Secrets
				if shouldDiscover(resourceTypes, "secrets") {
					secrets = d.getOrDiscoverSecrets(ctx, cfg, accountID, accountName, reg)
				}

				// Discover Public IPv4 addresses
				if shouldDiscover(resourceTypes, "publicipv4") {
					publicIPv4s = d.getOrDiscoverPublicIPv4s(ctx, cfg, accountID, accountName, reg)
				}

				mu.Lock()
				allEC2 = append(allEC2, ec2Instances...)
				allEBS = append(allEBS, ebsVolumes...)
				allECS = append(allECS, ecsServices...)
				allRDS = append(allRDS, rdsInstances...)
				allEKS = append(allEKS, eksClusters...)
				allELB = append(allELB, loadBalancers...)
				allNAT = append(allNAT, natGateways...)
				allEIP = append(allEIP, elasticIPs...)
				allSecrets = append(allSecrets, secrets...)
				allPublicIPv4 = append(allPublicIPv4, publicIPv4s...)
				mu.Unlock()
			}(account, region)
		}
	}

	wg.Wait()

	// Calculate total cost
	for _, inst := range allEC2 {
		totalCost += inst.HourlyCost
	}
	for _, vol := range allEBS {
		totalCost += vol.HourlyCost
	}
	for _, svc := range allECS {
		totalCost += svc.HourlyCost
	}
	for _, inst := range allRDS {
		totalCost += inst.HourlyCost
	}
	for _, cluster := range allEKS {
		totalCost += cluster.HourlyCost
	}
	for _, lb := range allELB {
		totalCost += lb.HourlyCost
	}
	for _, nat := range allNAT {
		totalCost += nat.HourlyCost
	}
	for _, eip := range allEIP {
		totalCost += eip.HourlyCost
	}
	for _, secret := range allSecrets {
		totalCost += secret.HourlyCost
	}
	for _, pip := range allPublicIPv4 {
		totalCost += pip.HourlyCost
	}

	// Build account and region summaries
	accountSummaries := d.buildAccountSummaries(allEC2, allEBS, allECS, allRDS, allEKS, allELB, allNAT, allEIP, allSecrets, allPublicIPv4)
	regionSummaries := d.buildRegionSummaries(allEC2, allEBS, allECS, allRDS, allEKS, allELB, allNAT, allEIP, allSecrets, allPublicIPv4)

	result := &types.CostResponse{
		TotalCost:     totalCost,
		Currency:      "USD",
		Accounts:      accountSummaries,
		Regions:       regionSummaries,
		EC2Instances:  allEC2,
		EBSVolumes:    allEBS,
		ECSServices:   allECS,
		RDSInstances:  allRDS,
		EKSClusters:   allEKS,
		LoadBalancers: allELB,
		NATGateways:   allNAT,
		ElasticIPs:    allEIP,
		Secrets:       allSecrets,
		PublicIPv4s:   allPublicIPv4,
	}

	return result, nil
}

// getConfigForAccount returns an AWS config for the specified account
func (d *Discovery) getConfigForAccount(ctx context.Context, account Account, region string) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return aws.Config{}, fmt.Errorf("loading default config: %w", err)
	}

	// If a role ARN is specified, assume that role
	if account.RoleARN != "" {
		stsClient := sts.NewFromConfig(cfg)
		creds := stscreds.NewAssumeRoleProvider(stsClient, account.RoleARN)
		cfg.Credentials = aws.NewCredentialsCache(creds)
	}

	return cfg, nil
}

// getAccountID returns the AWS account ID for the given config
func (d *Discovery) getAccountID(ctx context.Context, cfg aws.Config) (string, error) {
	stsClient := sts.NewFromConfig(cfg)
	result, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return *result.Account, nil
}

// getAccountAlias returns the AWS account alias, or empty string if not set or on error
func (d *Discovery) getAccountAlias(ctx context.Context, cfg aws.Config) string {
	iamClient := iam.NewFromConfig(cfg)
	result, err := iamClient.ListAccountAliases(ctx, &iam.ListAccountAliasesInput{})
	if err != nil {
		d.logger.Debug("failed to get account alias", "error", err)
		return ""
	}
	if len(result.AccountAliases) > 0 {
		return result.AccountAliases[0]
	}
	return ""
}

// DiscoverRegions returns all enabled regions for the current account
func (d *Discovery) DiscoverRegions(ctx context.Context) ([]string, error) {
	// Check cache first
	d.regionCacheMu.RLock()
	if d.regionCache != nil && time.Now().Before(d.regionCache.expiresAt) {
		regions := d.regionCache.value
		d.regionCacheMu.RUnlock()
		d.logger.Debug("returning cached regions", "count", len(regions))
		return regions, nil
	}
	d.regionCacheMu.RUnlock()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return nil, fmt.Errorf("loading default config: %w", err)
	}

	ec2Client := ec2.NewFromConfig(cfg)
	result, err := ec2Client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(false), // Only enabled regions
	})
	if err != nil {
		return nil, fmt.Errorf("describing regions: %w", err)
	}

	regions := make([]string, 0, len(result.Regions))
	for _, r := range result.Regions {
		if r.RegionName != nil {
			regions = append(regions, *r.RegionName)
		}
	}

	// Cache the result
	d.regionCacheMu.Lock()
	d.regionCache = &cacheEntry[[]string]{
		value:     regions,
		expiresAt: time.Now().Add(d.accountTTL),
	}
	d.regionCacheMu.Unlock()

	d.logger.Info("discovered regions", "count", len(regions))
	return regions, nil
}

// DiscoverAccounts returns all accounts from AWS Organizations with the specified assume role
func (d *Discovery) DiscoverAccounts(ctx context.Context, assumeRoleName string) ([]Account, error) {
	// Check cache first
	d.accountCacheMu.RLock()
	if d.accountCache != nil && time.Now().Before(d.accountCache.expiresAt) {
		accounts := d.accountCache.value
		d.accountCacheMu.RUnlock()
		d.logger.Debug("returning cached accounts", "count", len(accounts))
		return accounts, nil
	}
	d.accountCacheMu.RUnlock()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return nil, fmt.Errorf("loading default config: %w", err)
	}

	// Get current account ID to identify the management account
	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("getting caller identity: %w", err)
	}
	currentAccountID := *identity.Account

	// Try to list accounts from Organizations
	orgClient := organizations.NewFromConfig(cfg)
	var accounts []Account

	paginator := organizations.NewListAccountsPaginator(orgClient, &organizations.ListAccountsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			// If we don't have Organizations access, fall back to current account only
			d.logger.Info("organizations access not available, using current account only", "error", err)
			return []Account{{ID: currentAccountID}}, nil
		}

		for _, acc := range page.Accounts {
			if acc.Status != "ACTIVE" {
				continue
			}

			account := Account{
				ID:   *acc.Id,
				Name: *acc.Name,
			}

			// For non-management accounts, construct the role ARN to assume
			if *acc.Id != currentAccountID && assumeRoleName != "" {
				account.RoleARN = fmt.Sprintf("arn:aws:iam::%s:role/%s", *acc.Id, assumeRoleName)
			}

			accounts = append(accounts, account)
		}
	}

	// Cache the result
	d.accountCacheMu.Lock()
	d.accountCache = &cacheEntry[[]Account]{
		value:     accounts,
		expiresAt: time.Now().Add(d.accountTTL),
	}
	d.accountCacheMu.Unlock()

	d.logger.Info("discovered accounts from organizations", "count", len(accounts))
	return accounts, nil
}

// discoverEC2 discovers EC2 instances in the specified region
func (d *Discovery) discoverEC2(ctx context.Context, cfg aws.Config, accountID, accountName, region string) ([]types.EC2Instance, error) {
	client := ec2.NewFromConfig(cfg)

	var instances []types.EC2Instance
	paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describing instances: %w", err)
		}

		for _, reservation := range page.Reservations {
			for _, inst := range reservation.Instances {
				// Skip terminated instances
				if inst.State != nil && inst.State.Name == ec2types.InstanceStateNameTerminated {
					continue
				}

				name := getEC2Name(inst.Tags)
				instanceType := string(inst.InstanceType)
				state := string(inst.State.Name)

				// Get pricing (only for running instances)
				var hourlyCost types.CostValue
				if inst.State.Name == ec2types.InstanceStateNameRunning {
					price, err := d.pricingProvider.GetEC2Price(ctx, region, instanceType)
					if err != nil {
						d.logger.Warn("failed to get EC2 price",
							"instanceType", instanceType,
							"region", region,
							"error", err)
					} else {
						hourlyCost = price
					}
				}

				instances = append(instances, types.EC2Instance{
					AccountID:    accountID,
					AccountName:  accountName,
					Region:       region,
					InstanceID:   *inst.InstanceId,
					Name:         name,
					InstanceType: instanceType,
					State:        state,
					HourlyCost:   hourlyCost,
				})
			}
		}
	}

	return instances, nil
}

// discoverEBS discovers EBS volumes in the specified region
func (d *Discovery) discoverEBS(ctx context.Context, cfg aws.Config, accountID, accountName, region string) ([]types.EBSVolume, error) {
	client := ec2.NewFromConfig(cfg)

	var volumes []types.EBSVolume
	paginator := ec2.NewDescribeVolumesPaginator(client, &ec2.DescribeVolumesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describing volumes: %w", err)
		}

		for _, vol := range page.Volumes {
			name := getEBSName(vol.Tags)
			volumeType := string(vol.VolumeType)
			state := string(vol.State)

			size := int32(0)
			if vol.Size != nil {
				size = *vol.Size
			}

			iops := int32(0)
			if vol.Iops != nil {
				iops = *vol.Iops
			}

			throughput := int32(0)
			if vol.Throughput != nil {
				throughput = *vol.Throughput
			}

			// Get pricing
			hourlyCost, err := d.pricingProvider.GetEBSPrice(ctx, region, volumeType, size, iops, throughput)
			if err != nil {
				d.logger.Warn("failed to get EBS price",
					"volumeType", volumeType,
					"region", region,
					"error", err)
			}

			volumes = append(volumes, types.EBSVolume{
				AccountID:   accountID,
				AccountName: accountName,
				Region:      region,
				VolumeID:    *vol.VolumeId,
				Name:        name,
				VolumeType:  volumeType,
				Size:        size,
				IOPS:        iops,
				Throughput:  throughput,
				State:       state,
				HourlyCost:  hourlyCost,
			})
		}
	}

	return volumes, nil
}

// discoverRDS discovers RDS instances in the specified region
func (d *Discovery) discoverRDS(ctx context.Context, cfg aws.Config, accountID, accountName, region string) ([]types.RDSInstance, error) {
	client := rds.NewFromConfig(cfg)

	var instances []types.RDSInstance
	paginator := rds.NewDescribeDBInstancesPaginator(client, &rds.DescribeDBInstancesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describing DB instances: %w", err)
		}

		for _, inst := range page.DBInstances {
			name := ""
			if inst.DBInstanceIdentifier != nil {
				name = *inst.DBInstanceIdentifier
			}

			engine := ""
			if inst.Engine != nil {
				engine = *inst.Engine
			}

			engineVersion := ""
			if inst.EngineVersion != nil {
				engineVersion = *inst.EngineVersion
			}

			instanceClass := ""
			if inst.DBInstanceClass != nil {
				instanceClass = *inst.DBInstanceClass
			}

			multiAZ := inst.MultiAZ != nil && *inst.MultiAZ

			storageType := ""
			if inst.StorageType != nil {
				storageType = *inst.StorageType
			}

			allocatedStorage := int32(0)
			if inst.AllocatedStorage != nil {
				allocatedStorage = *inst.AllocatedStorage
			}

			state := ""
			if inst.DBInstanceStatus != nil {
				state = *inst.DBInstanceStatus
			}

			// Get pricing for running instances (exclude stopped/deleted states)
			var hourlyCost types.CostValue
			if !isRDSNonBillableState(state) {
				price, err := d.pricingProvider.GetRDSPrice(ctx, region, instanceClass, engine, multiAZ)
				if err != nil {
					d.logger.Warn("failed to get RDS price",
						"instanceClass", instanceClass,
						"engine", engine,
						"region", region,
						"error", err)
				} else {
					hourlyCost = price
				}
			}

			instances = append(instances, types.RDSInstance{
				AccountID:        accountID,
				AccountName:      accountName,
				Region:           region,
				DBInstanceID:     *inst.DBInstanceIdentifier,
				Name:             name,
				Engine:           engine,
				EngineVersion:    engineVersion,
				InstanceClass:    instanceClass,
				MultiAZ:          multiAZ,
				StorageType:      storageType,
				AllocatedStorage: allocatedStorage,
				State:            state,
				HourlyCost:       hourlyCost,
			})
		}
	}

	return instances, nil
}

// discoverECS discovers ECS services in the specified region
func (d *Discovery) discoverECS(ctx context.Context, cfg aws.Config, accountID, accountName, region string) ([]types.ECSService, error) {
	client := ecs.NewFromConfig(cfg)

	var services []types.ECSService

	// List all clusters
	clusterPaginator := ecs.NewListClustersPaginator(client, &ecs.ListClustersInput{})
	for clusterPaginator.HasMorePages() {
		clusterPage, err := clusterPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing clusters: %w", err)
		}

		for _, clusterArn := range clusterPage.ClusterArns {
			// Extract cluster name from ARN
			clusterName := clusterArn
			if idx := len("arn:aws:ecs:" + region + ":" + accountID + ":cluster/"); idx < len(clusterArn) {
				clusterName = clusterArn[idx:]
			}

			// List services in this cluster
			servicePaginator := ecs.NewListServicesPaginator(client, &ecs.ListServicesInput{
				Cluster: &clusterArn,
			})

			for servicePaginator.HasMorePages() {
				servicePage, err := servicePaginator.NextPage(ctx)
				if err != nil {
					d.logger.Warn("failed to list services in cluster",
						"cluster", clusterName,
						"error", err)
					break
				}

				if len(servicePage.ServiceArns) == 0 {
					continue
				}

				// Describe services to get details
				describeOutput, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
					Cluster:  &clusterArn,
					Services: servicePage.ServiceArns,
				})
				if err != nil {
					d.logger.Warn("failed to describe services",
						"cluster", clusterName,
						"error", err)
					continue
				}

				for _, svc := range describeOutput.Services {
					serviceName := ""
					if svc.ServiceName != nil {
						serviceName = *svc.ServiceName
					}

					launchType := "EC2"
					if svc.LaunchType != "" {
						launchType = string(svc.LaunchType)
					}

					state := "ACTIVE"
					if svc.Status != nil {
						state = *svc.Status
					}

					desiredCount := int32(0)
					if svc.DesiredCount != 0 {
						desiredCount = svc.DesiredCount
					}

					runningCount := int32(0)
					if svc.RunningCount != 0 {
						runningCount = svc.RunningCount
					}

					// Get pricing for Fargate services
					var hourlyCost types.CostValue
					if launchType == "FARGATE" && runningCount > 0 {
						price, err := d.pricingProvider.GetECSPrice(ctx, region, launchType, runningCount)
						if err != nil {
							d.logger.Warn("failed to get ECS price",
								"service", serviceName,
								"region", region,
								"error", err)
						} else {
							hourlyCost = price
						}
					}

					services = append(services, types.ECSService{
						AccountID:    accountID,
						AccountName:  accountName,
						Region:       region,
						ClusterName:  clusterName,
						ServiceName:  serviceName,
						LaunchType:   launchType,
						DesiredCount: desiredCount,
						RunningCount: runningCount,
						State:        state,
						HourlyCost:   hourlyCost,
					})
				}
			}
		}
	}

	return services, nil
}

// discoverEKS discovers EKS clusters in the specified region
func (d *Discovery) discoverEKS(ctx context.Context, cfg aws.Config, accountID, accountName, region string) ([]types.EKSCluster, error) {
	client := eks.NewFromConfig(cfg)

	var clusters []types.EKSCluster

	// List all clusters
	listInput := &eks.ListClustersInput{}
	paginator := eks.NewListClustersPaginator(client, listInput)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing clusters: %w", err)
		}

		for _, clusterName := range page.Clusters {
			// Describe each cluster to get details
			describeOutput, err := client.DescribeCluster(ctx, &eks.DescribeClusterInput{
				Name: aws.String(clusterName),
			})
			if err != nil {
				d.logger.Warn("failed to describe EKS cluster",
					"cluster", clusterName,
					"error", err)
				continue
			}

			cluster := describeOutput.Cluster
			if cluster == nil {
				continue
			}

			status := ""
			if cluster.Status != "" {
				status = string(cluster.Status)
			}

			version := ""
			if cluster.Version != nil {
				version = *cluster.Version
			}

			platform := "linux"
			if cluster.PlatformVersion != nil && *cluster.PlatformVersion != "" {
				platform = *cluster.PlatformVersion
			}

			// Get pricing for active clusters
			var hourlyCost types.CostValue
			if status == "ACTIVE" {
				price, err := d.pricingProvider.GetEKSPrice(ctx, region)
				if err != nil {
					d.logger.Warn("failed to get EKS price",
						"cluster", clusterName,
						"region", region,
						"error", err)
				} else {
					hourlyCost = price
				}
			}

			clusters = append(clusters, types.EKSCluster{
				AccountID:   accountID,
				AccountName: accountName,
				Region:      region,
				ClusterName: clusterName,
				Status:      status,
				Version:     version,
				Platform:    platform,
				HourlyCost:  hourlyCost,
			})
		}
	}

	return clusters, nil
}

// discoverELB discovers Elastic Load Balancers (ALB, NLB, and CLB) in the specified region
func (d *Discovery) discoverELB(ctx context.Context, cfg aws.Config, accountID, accountName, region string) ([]types.LoadBalancer, error) {
	var loadBalancers []types.LoadBalancer

	// Discover ALB and NLB using v2 API
	v2Client := elasticloadbalancingv2.NewFromConfig(cfg)
	v2Paginator := elasticloadbalancingv2.NewDescribeLoadBalancersPaginator(v2Client, &elasticloadbalancingv2.DescribeLoadBalancersInput{})

	for v2Paginator.HasMorePages() {
		page, err := v2Paginator.NextPage(ctx)
		if err != nil {
			d.logger.Warn("failed to describe v2 load balancers",
				"region", region,
				"error", err)
			break
		}

		for _, lb := range page.LoadBalancers {
			name := ""
			if lb.LoadBalancerName != nil {
				name = *lb.LoadBalancerName
			}

			arn := ""
			if lb.LoadBalancerArn != nil {
				arn = *lb.LoadBalancerArn
			}

			lbType := "application"
			if lb.Type != "" {
				lbType = string(lb.Type)
			}

			scheme := ""
			if lb.Scheme != "" {
				scheme = string(lb.Scheme)
			}

			state := ""
			if lb.State != nil && lb.State.Code != "" {
				state = string(lb.State.Code)
			}

			// Get pricing for active load balancers
			var hourlyCost types.CostValue
			if state == "active" {
				price, err := d.pricingProvider.GetELBPrice(ctx, region, lbType)
				if err != nil {
					d.logger.Warn("failed to get ELB price",
						"name", name,
						"type", lbType,
						"region", region,
						"error", err)
				} else {
					hourlyCost = price
				}
			}

			loadBalancers = append(loadBalancers, types.LoadBalancer{
				AccountID:   accountID,
				AccountName: accountName,
				Region:      region,
				Name:        name,
				ARN:         arn,
				Type:        lbType,
				Scheme:      scheme,
				State:       state,
				HourlyCost:  hourlyCost,
			})
		}
	}

	// Discover Classic Load Balancers using v1 API
	v1Client := elasticloadbalancing.NewFromConfig(cfg)
	v1Output, err := v1Client.DescribeLoadBalancers(ctx, &elasticloadbalancing.DescribeLoadBalancersInput{})
	if err != nil {
		d.logger.Warn("failed to describe classic load balancers",
			"region", region,
			"error", err)
	} else {
		for _, lb := range v1Output.LoadBalancerDescriptions {
			name := ""
			if lb.LoadBalancerName != nil {
				name = *lb.LoadBalancerName
			}

			scheme := "internet-facing"
			if lb.Scheme != nil {
				scheme = *lb.Scheme
			}

			// Get pricing for classic load balancers
			price, err := d.pricingProvider.GetELBPrice(ctx, region, "classic")
			var hourlyCost types.CostValue
			if err != nil {
				d.logger.Warn("failed to get CLB price",
					"name", name,
					"region", region,
					"error", err)
			} else {
				hourlyCost = price
			}

			loadBalancers = append(loadBalancers, types.LoadBalancer{
				AccountID:   accountID,
				AccountName: accountName,
				Region:      region,
				Name:        name,
				ARN:         "", // CLB doesn't have ARN in the same way
				Type:        "classic",
				Scheme:      scheme,
				State:       "active", // CLB doesn't have state in the same way
				HourlyCost:  hourlyCost,
			})
		}
	}

	return loadBalancers, nil
}

// discoverNATGateways discovers NAT Gateways in the specified region
func (d *Discovery) discoverNATGateways(ctx context.Context, cfg aws.Config, accountID, accountName, region string) ([]types.NATGateway, error) {
	client := ec2.NewFromConfig(cfg)

	var gateways []types.NATGateway
	paginator := ec2.NewDescribeNatGatewaysPaginator(client, &ec2.DescribeNatGatewaysInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describing NAT gateways: %w", err)
		}

		for _, nat := range page.NatGateways {
			// Skip deleted NAT gateways
			state := string(nat.State)
			if state == "deleted" || state == "deleting" {
				continue
			}

			id := ""
			if nat.NatGatewayId != nil {
				id = *nat.NatGatewayId
			}

			name := getNATGatewayName(nat.Tags)

			natType := "public"
			if nat.ConnectivityType != "" {
				natType = string(nat.ConnectivityType)
			}

			vpcID := ""
			if nat.VpcId != nil {
				vpcID = *nat.VpcId
			}

			subnetID := ""
			if nat.SubnetId != nil {
				subnetID = *nat.SubnetId
			}

			// Get pricing for available NAT gateways
			var hourlyCost types.CostValue
			if state == "available" {
				price, err := d.pricingProvider.GetNATGatewayPrice(ctx, region)
				if err != nil {
					d.logger.Warn("failed to get NAT Gateway price",
						"id", id,
						"region", region,
						"error", err)
				} else {
					hourlyCost = price
				}
			}

			gateways = append(gateways, types.NATGateway{
				AccountID:   accountID,
				AccountName: accountName,
				Region:      region,
				ID:          id,
				Name:        name,
				State:       state,
				Type:        natType,
				VPCID:       vpcID,
				SubnetID:    subnetID,
				HourlyCost:  hourlyCost,
			})
		}
	}

	return gateways, nil
}

// discoverElasticIPs discovers Elastic IP addresses in the specified region
func (d *Discovery) discoverElasticIPs(ctx context.Context, cfg aws.Config, accountID, accountName, region string) ([]types.ElasticIP, error) {
	client := ec2.NewFromConfig(cfg)

	output, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{})
	if err != nil {
		return nil, fmt.Errorf("describing Elastic IPs: %w", err)
	}

	var elasticIPs []types.ElasticIP

	for _, addr := range output.Addresses {
		allocationID := ""
		if addr.AllocationId != nil {
			allocationID = *addr.AllocationId
		}

		publicIP := ""
		if addr.PublicIp != nil {
			publicIP = *addr.PublicIp
		}

		name := getElasticIPName(addr.Tags)

		associationID := ""
		if addr.AssociationId != nil {
			associationID = *addr.AssociationId
		}

		instanceID := ""
		if addr.InstanceId != nil {
			instanceID = *addr.InstanceId
		}

		isAssociated := associationID != ""

		// Get pricing - only unassociated EIPs cost money
		price, err := d.pricingProvider.GetElasticIPPrice(ctx, region, isAssociated)
		var hourlyCost types.CostValue
		if err != nil {
			d.logger.Warn("failed to get Elastic IP price",
				"allocationId", allocationID,
				"region", region,
				"error", err)
		} else {
			hourlyCost = price
		}

		elasticIPs = append(elasticIPs, types.ElasticIP{
			AccountID:     accountID,
			AccountName:   accountName,
			Region:        region,
			AllocationID:  allocationID,
			PublicIP:      publicIP,
			Name:          name,
			AssociationID: associationID,
			InstanceID:    instanceID,
			IsAssociated:  isAssociated,
			HourlyCost:    hourlyCost,
		})
	}

	return elasticIPs, nil
}

// discoverSecrets discovers Secrets Manager secrets in the specified region
func (d *Discovery) discoverSecrets(ctx context.Context, cfg aws.Config, accountID, accountName, region string) ([]types.Secret, error) {
	client := secretsmanager.NewFromConfig(cfg)

	var secrets []types.Secret
	paginator := secretsmanager.NewListSecretsPaginator(client, &secretsmanager.ListSecretsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing secrets: %w", err)
		}

		for _, secret := range page.SecretList {
			name := ""
			if secret.Name != nil {
				name = *secret.Name
			}

			arn := ""
			if secret.ARN != nil {
				arn = *secret.ARN
			}

			description := ""
			if secret.Description != nil {
				description = *secret.Description
			}

			// Get pricing
			price, err := d.pricingProvider.GetSecretPrice(ctx, region)
			var hourlyCost types.CostValue
			if err != nil {
				d.logger.Warn("failed to get Secret price",
					"name", name,
					"region", region,
					"error", err)
			} else {
				hourlyCost = price
			}

			secrets = append(secrets, types.Secret{
				AccountID:   accountID,
				AccountName: accountName,
				Region:      region,
				Name:        name,
				ARN:         arn,
				Description: description,
				HourlyCost:  hourlyCost,
			})
		}
	}

	return secrets, nil
}

// discoverPublicIPv4s discovers public IPv4 addresses on EC2 instances in the specified region
// These are auto-assigned public IPs, not Elastic IPs (which are tracked separately)
func (d *Discovery) discoverPublicIPv4s(ctx context.Context, cfg aws.Config, accountID, accountName, region string) ([]types.PublicIPv4, error) {
	client := ec2.NewFromConfig(cfg)

	var publicIPs []types.PublicIPv4
	paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})

	// First, get a set of Elastic IPs to exclude them
	eipOutput, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{})
	elasticIPs := make(map[string]bool)
	if err == nil {
		for _, addr := range eipOutput.Addresses {
			if addr.PublicIp != nil {
				elasticIPs[*addr.PublicIp] = true
			}
		}
	}

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describing instances for public IPs: %w", err)
		}

		for _, reservation := range page.Reservations {
			for _, inst := range reservation.Instances {
				// Skip non-running instances (no public IP charge when stopped)
				if inst.State == nil || inst.State.Name != ec2types.InstanceStateNameRunning {
					continue
				}

				// Skip instances without public IP
				if inst.PublicIpAddress == nil || *inst.PublicIpAddress == "" {
					continue
				}

				publicIP := *inst.PublicIpAddress

				// Skip Elastic IPs (tracked separately)
				if elasticIPs[publicIP] {
					continue
				}

				instanceID := ""
				if inst.InstanceId != nil {
					instanceID = *inst.InstanceId
				}

				instanceName := getEC2Name(inst.Tags)

				// Get pricing
				price, err := d.pricingProvider.GetPublicIPv4Price(ctx, region)
				var hourlyCost types.CostValue
				if err != nil {
					d.logger.Warn("failed to get public IPv4 price",
						"publicIp", publicIP,
						"region", region,
						"error", err)
				} else {
					hourlyCost = price
				}

				publicIPs = append(publicIPs, types.PublicIPv4{
					AccountID:    accountID,
					AccountName:  accountName,
					Region:       region,
					PublicIP:     publicIP,
					InstanceID:   instanceID,
					InstanceName: instanceName,
					HourlyCost:   hourlyCost,
				})
			}
		}
	}

	return publicIPs, nil
}

// getEC2Name extracts the Name tag from EC2 instance tags
func getEC2Name(tags []ec2types.Tag) string {
	for _, tag := range tags {
		if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
			return *tag.Value
		}
	}
	return ""
}

// getEBSName extracts the Name tag from EBS volume tags
func getEBSName(tags []ec2types.Tag) string {
	for _, tag := range tags {
		if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
			return *tag.Value
		}
	}
	return ""
}

// getNATGatewayName extracts the Name tag from NAT Gateway tags
func getNATGatewayName(tags []ec2types.Tag) string {
	for _, tag := range tags {
		if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
			return *tag.Value
		}
	}
	return ""
}

// getElasticIPName extracts the Name tag from Elastic IP tags
func getElasticIPName(tags []ec2types.Tag) string {
	for _, tag := range tags {
		if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
			return *tag.Value
		}
	}
	return ""
}

// isRDSNonBillableState returns true if the RDS instance state is non-billable
func isRDSNonBillableState(state string) bool {
	switch state {
	case "stopped", "stopping", "deleted", "deleting", "failed",
		"inaccessible-encryption-credentials", "incompatible-network",
		"incompatible-restore", "insufficient-capacity":
		return true
	}
	return false
}

// getOrDiscoverEC2 returns cached EC2 instances or discovers them
func (d *Discovery) getOrDiscoverEC2(ctx context.Context, cfg aws.Config, accountID, accountName, region string) []types.EC2Instance {
	cacheKey := resourceCacheKey(accountID, region, "ec2")

	d.resourceCacheMu.RLock()
	if entry, ok := d.resourceCache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		d.resourceCacheMu.RUnlock()
		d.logger.Debug("cache hit", "key", cacheKey)
		return entry.value.([]types.EC2Instance)
	}
	d.resourceCacheMu.RUnlock()

	result, err := d.discoverEC2(ctx, cfg, accountID, accountName, region)
	if err != nil {
		d.logger.Error("failed to discover EC2 instances", "account", accountName, "region", region, "error", err)
		return nil
	}

	d.resourceCacheMu.Lock()
	d.resourceCache[cacheKey] = cacheEntry[any]{value: result, expiresAt: time.Now().Add(d.resourceTTL)}
	d.resourceCacheMu.Unlock()
	d.logger.Debug("cached", "key", cacheKey)

	return result
}

// getOrDiscoverEBS returns cached EBS volumes or discovers them
func (d *Discovery) getOrDiscoverEBS(ctx context.Context, cfg aws.Config, accountID, accountName, region string) []types.EBSVolume {
	cacheKey := resourceCacheKey(accountID, region, "ebs")

	d.resourceCacheMu.RLock()
	if entry, ok := d.resourceCache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		d.resourceCacheMu.RUnlock()
		d.logger.Debug("cache hit", "key", cacheKey)
		return entry.value.([]types.EBSVolume)
	}
	d.resourceCacheMu.RUnlock()

	result, err := d.discoverEBS(ctx, cfg, accountID, accountName, region)
	if err != nil {
		d.logger.Error("failed to discover EBS volumes", "account", accountName, "region", region, "error", err)
		return nil
	}

	d.resourceCacheMu.Lock()
	d.resourceCache[cacheKey] = cacheEntry[any]{value: result, expiresAt: time.Now().Add(d.resourceTTL)}
	d.resourceCacheMu.Unlock()
	d.logger.Debug("cached", "key", cacheKey)

	return result
}

// getOrDiscoverECS returns cached ECS services or discovers them
func (d *Discovery) getOrDiscoverECS(ctx context.Context, cfg aws.Config, accountID, accountName, region string) []types.ECSService {
	cacheKey := resourceCacheKey(accountID, region, "ecs")

	d.resourceCacheMu.RLock()
	if entry, ok := d.resourceCache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		d.resourceCacheMu.RUnlock()
		d.logger.Debug("cache hit", "key", cacheKey)
		return entry.value.([]types.ECSService)
	}
	d.resourceCacheMu.RUnlock()

	result, err := d.discoverECS(ctx, cfg, accountID, accountName, region)
	if err != nil {
		d.logger.Error("failed to discover ECS services", "account", accountName, "region", region, "error", err)
		return nil
	}

	d.resourceCacheMu.Lock()
	d.resourceCache[cacheKey] = cacheEntry[any]{value: result, expiresAt: time.Now().Add(d.resourceTTL)}
	d.resourceCacheMu.Unlock()
	d.logger.Debug("cached", "key", cacheKey)

	return result
}

// getOrDiscoverRDS returns cached RDS instances or discovers them
func (d *Discovery) getOrDiscoverRDS(ctx context.Context, cfg aws.Config, accountID, accountName, region string) []types.RDSInstance {
	cacheKey := resourceCacheKey(accountID, region, "rds")

	d.resourceCacheMu.RLock()
	if entry, ok := d.resourceCache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		d.resourceCacheMu.RUnlock()
		d.logger.Debug("cache hit", "key", cacheKey)
		return entry.value.([]types.RDSInstance)
	}
	d.resourceCacheMu.RUnlock()

	result, err := d.discoverRDS(ctx, cfg, accountID, accountName, region)
	if err != nil {
		d.logger.Error("failed to discover RDS instances", "account", accountName, "region", region, "error", err)
		return nil
	}

	d.resourceCacheMu.Lock()
	d.resourceCache[cacheKey] = cacheEntry[any]{value: result, expiresAt: time.Now().Add(d.resourceTTL)}
	d.resourceCacheMu.Unlock()
	d.logger.Debug("cached", "key", cacheKey)

	return result
}

// getOrDiscoverEKS returns cached EKS clusters or discovers them
func (d *Discovery) getOrDiscoverEKS(ctx context.Context, cfg aws.Config, accountID, accountName, region string) []types.EKSCluster {
	cacheKey := resourceCacheKey(accountID, region, "eks")

	d.resourceCacheMu.RLock()
	if entry, ok := d.resourceCache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		d.resourceCacheMu.RUnlock()
		d.logger.Debug("cache hit", "key", cacheKey)
		return entry.value.([]types.EKSCluster)
	}
	d.resourceCacheMu.RUnlock()

	result, err := d.discoverEKS(ctx, cfg, accountID, accountName, region)
	if err != nil {
		d.logger.Error("failed to discover EKS clusters", "account", accountName, "region", region, "error", err)
		return nil
	}

	d.resourceCacheMu.Lock()
	d.resourceCache[cacheKey] = cacheEntry[any]{value: result, expiresAt: time.Now().Add(d.resourceTTL)}
	d.resourceCacheMu.Unlock()
	d.logger.Debug("cached", "key", cacheKey)

	return result
}

// getOrDiscoverELB returns cached load balancers or discovers them
func (d *Discovery) getOrDiscoverELB(ctx context.Context, cfg aws.Config, accountID, accountName, region string) []types.LoadBalancer {
	cacheKey := resourceCacheKey(accountID, region, "elb")

	d.resourceCacheMu.RLock()
	if entry, ok := d.resourceCache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		d.resourceCacheMu.RUnlock()
		d.logger.Debug("cache hit", "key", cacheKey)
		return entry.value.([]types.LoadBalancer)
	}
	d.resourceCacheMu.RUnlock()

	result, err := d.discoverELB(ctx, cfg, accountID, accountName, region)
	if err != nil {
		d.logger.Error("failed to discover load balancers", "account", accountName, "region", region, "error", err)
		return nil
	}

	d.resourceCacheMu.Lock()
	d.resourceCache[cacheKey] = cacheEntry[any]{value: result, expiresAt: time.Now().Add(d.resourceTTL)}
	d.resourceCacheMu.Unlock()
	d.logger.Debug("cached", "key", cacheKey)

	return result
}

// getOrDiscoverNATGateways returns cached NAT gateways or discovers them
func (d *Discovery) getOrDiscoverNATGateways(ctx context.Context, cfg aws.Config, accountID, accountName, region string) []types.NATGateway {
	cacheKey := resourceCacheKey(accountID, region, "nat")

	d.resourceCacheMu.RLock()
	if entry, ok := d.resourceCache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		d.resourceCacheMu.RUnlock()
		d.logger.Debug("cache hit", "key", cacheKey)
		return entry.value.([]types.NATGateway)
	}
	d.resourceCacheMu.RUnlock()

	result, err := d.discoverNATGateways(ctx, cfg, accountID, accountName, region)
	if err != nil {
		d.logger.Error("failed to discover NAT gateways", "account", accountName, "region", region, "error", err)
		return nil
	}

	d.resourceCacheMu.Lock()
	d.resourceCache[cacheKey] = cacheEntry[any]{value: result, expiresAt: time.Now().Add(d.resourceTTL)}
	d.resourceCacheMu.Unlock()
	d.logger.Debug("cached", "key", cacheKey)

	return result
}

// getOrDiscoverElasticIPs returns cached Elastic IPs or discovers them
func (d *Discovery) getOrDiscoverElasticIPs(ctx context.Context, cfg aws.Config, accountID, accountName, region string) []types.ElasticIP {
	cacheKey := resourceCacheKey(accountID, region, "eip")

	d.resourceCacheMu.RLock()
	if entry, ok := d.resourceCache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		d.resourceCacheMu.RUnlock()
		d.logger.Debug("cache hit", "key", cacheKey)
		return entry.value.([]types.ElasticIP)
	}
	d.resourceCacheMu.RUnlock()

	result, err := d.discoverElasticIPs(ctx, cfg, accountID, accountName, region)
	if err != nil {
		d.logger.Error("failed to discover Elastic IPs", "account", accountName, "region", region, "error", err)
		return nil
	}

	d.resourceCacheMu.Lock()
	d.resourceCache[cacheKey] = cacheEntry[any]{value: result, expiresAt: time.Now().Add(d.resourceTTL)}
	d.resourceCacheMu.Unlock()
	d.logger.Debug("cached", "key", cacheKey)

	return result
}

// getOrDiscoverSecrets returns cached secrets or discovers them
func (d *Discovery) getOrDiscoverSecrets(ctx context.Context, cfg aws.Config, accountID, accountName, region string) []types.Secret {
	cacheKey := resourceCacheKey(accountID, region, "secrets")

	d.resourceCacheMu.RLock()
	if entry, ok := d.resourceCache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		d.resourceCacheMu.RUnlock()
		d.logger.Debug("cache hit", "key", cacheKey)
		return entry.value.([]types.Secret)
	}
	d.resourceCacheMu.RUnlock()

	result, err := d.discoverSecrets(ctx, cfg, accountID, accountName, region)
	if err != nil {
		d.logger.Error("failed to discover secrets", "account", accountName, "region", region, "error", err)
		return nil
	}

	d.resourceCacheMu.Lock()
	d.resourceCache[cacheKey] = cacheEntry[any]{value: result, expiresAt: time.Now().Add(d.resourceTTL)}
	d.resourceCacheMu.Unlock()
	d.logger.Debug("cached", "key", cacheKey)

	return result
}

// getOrDiscoverPublicIPv4s returns cached public IPv4 addresses or discovers them
func (d *Discovery) getOrDiscoverPublicIPv4s(ctx context.Context, cfg aws.Config, accountID, accountName, region string) []types.PublicIPv4 {
	cacheKey := resourceCacheKey(accountID, region, "publicipv4")

	d.resourceCacheMu.RLock()
	if entry, ok := d.resourceCache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		d.resourceCacheMu.RUnlock()
		d.logger.Debug("cache hit", "key", cacheKey)
		return entry.value.([]types.PublicIPv4)
	}
	d.resourceCacheMu.RUnlock()

	result, err := d.discoverPublicIPv4s(ctx, cfg, accountID, accountName, region)
	if err != nil {
		d.logger.Error("failed to discover public IPv4 addresses", "account", accountName, "region", region, "error", err)
		return nil
	}

	d.resourceCacheMu.Lock()
	d.resourceCache[cacheKey] = cacheEntry[any]{value: result, expiresAt: time.Now().Add(d.resourceTTL)}
	d.resourceCacheMu.Unlock()
	d.logger.Debug("cached", "key", cacheKey)

	return result
}

// buildAccountSummaries builds account-level cost summaries
func (d *Discovery) buildAccountSummaries(ec2 []types.EC2Instance, ebs []types.EBSVolume, ecs []types.ECSService, rds []types.RDSInstance, eks []types.EKSCluster, elb []types.LoadBalancer, nat []types.NATGateway, eip []types.ElasticIP, secrets []types.Secret, publicIPv4 []types.PublicIPv4) []types.AccountSummary {
	summaries := make(map[string]*types.AccountSummary)

	for _, inst := range ec2 {
		key := inst.AccountID
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.AccountSummary{
				AccountID:   inst.AccountID,
				AccountName: inst.AccountName,
			}
		}
		summaries[key].EC2Count++
		summaries[key].TotalCost += inst.HourlyCost
	}

	for _, vol := range ebs {
		key := vol.AccountID
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.AccountSummary{
				AccountID:   vol.AccountID,
				AccountName: vol.AccountName,
			}
		}
		summaries[key].EBSCount++
		summaries[key].TotalCost += vol.HourlyCost
	}

	for _, svc := range ecs {
		key := svc.AccountID
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.AccountSummary{
				AccountID:   svc.AccountID,
				AccountName: svc.AccountName,
			}
		}
		summaries[key].ECSCount++
		summaries[key].TotalCost += svc.HourlyCost
	}

	for _, inst := range rds {
		key := inst.AccountID
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.AccountSummary{
				AccountID:   inst.AccountID,
				AccountName: inst.AccountName,
			}
		}
		summaries[key].RDSCount++
		summaries[key].TotalCost += inst.HourlyCost
	}

	for _, cluster := range eks {
		key := cluster.AccountID
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.AccountSummary{
				AccountID:   cluster.AccountID,
				AccountName: cluster.AccountName,
			}
		}
		summaries[key].EKSCount++
		summaries[key].TotalCost += cluster.HourlyCost
	}

	for _, lb := range elb {
		key := lb.AccountID
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.AccountSummary{
				AccountID:   lb.AccountID,
				AccountName: lb.AccountName,
			}
		}
		summaries[key].ELBCount++
		summaries[key].TotalCost += lb.HourlyCost
	}

	for _, gw := range nat {
		key := gw.AccountID
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.AccountSummary{
				AccountID:   gw.AccountID,
				AccountName: gw.AccountName,
			}
		}
		summaries[key].NATCount++
		summaries[key].TotalCost += gw.HourlyCost
	}

	for _, ip := range eip {
		key := ip.AccountID
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.AccountSummary{
				AccountID:   ip.AccountID,
				AccountName: ip.AccountName,
			}
		}
		summaries[key].EIPCount++
		summaries[key].TotalCost += ip.HourlyCost
	}

	for _, secret := range secrets {
		key := secret.AccountID
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.AccountSummary{
				AccountID:   secret.AccountID,
				AccountName: secret.AccountName,
			}
		}
		summaries[key].SecretCount++
		summaries[key].TotalCost += secret.HourlyCost
	}

	for _, pip := range publicIPv4 {
		key := pip.AccountID
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.AccountSummary{
				AccountID:   pip.AccountID,
				AccountName: pip.AccountName,
			}
		}
		summaries[key].PublicIPv4Count++
		summaries[key].TotalCost += pip.HourlyCost
	}

	result := make([]types.AccountSummary, 0, len(summaries))
	for _, s := range summaries {
		result = append(result, *s)
	}
	return result
}

// buildRegionSummaries builds region-level cost summaries
func (d *Discovery) buildRegionSummaries(ec2 []types.EC2Instance, ebs []types.EBSVolume, ecs []types.ECSService, rds []types.RDSInstance, eks []types.EKSCluster, elb []types.LoadBalancer, nat []types.NATGateway, eip []types.ElasticIP, secrets []types.Secret, publicIPv4 []types.PublicIPv4) []types.RegionSummary {
	summaries := make(map[string]*types.RegionSummary)

	for _, inst := range ec2 {
		key := inst.Region
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.RegionSummary{Region: key}
		}
		summaries[key].EC2Count++
		summaries[key].TotalCost += inst.HourlyCost
	}

	for _, vol := range ebs {
		key := vol.Region
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.RegionSummary{Region: key}
		}
		summaries[key].EBSCount++
		summaries[key].TotalCost += vol.HourlyCost
	}

	for _, svc := range ecs {
		key := svc.Region
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.RegionSummary{Region: key}
		}
		summaries[key].ECSCount++
		summaries[key].TotalCost += svc.HourlyCost
	}

	for _, inst := range rds {
		key := inst.Region
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.RegionSummary{Region: key}
		}
		summaries[key].RDSCount++
		summaries[key].TotalCost += inst.HourlyCost
	}

	for _, cluster := range eks {
		key := cluster.Region
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.RegionSummary{Region: key}
		}
		summaries[key].EKSCount++
		summaries[key].TotalCost += cluster.HourlyCost
	}

	for _, lb := range elb {
		key := lb.Region
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.RegionSummary{Region: key}
		}
		summaries[key].ELBCount++
		summaries[key].TotalCost += lb.HourlyCost
	}

	for _, gw := range nat {
		key := gw.Region
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.RegionSummary{Region: key}
		}
		summaries[key].NATCount++
		summaries[key].TotalCost += gw.HourlyCost
	}

	for _, ip := range eip {
		key := ip.Region
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.RegionSummary{Region: key}
		}
		summaries[key].EIPCount++
		summaries[key].TotalCost += ip.HourlyCost
	}

	for _, secret := range secrets {
		key := secret.Region
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.RegionSummary{Region: key}
		}
		summaries[key].SecretCount++
		summaries[key].TotalCost += secret.HourlyCost
	}

	for _, pip := range publicIPv4 {
		key := pip.Region
		if _, ok := summaries[key]; !ok {
			summaries[key] = &types.RegionSummary{Region: key}
		}
		summaries[key].PublicIPv4Count++
		summaries[key].TotalCost += pip.HourlyCost
	}

	result := make([]types.RegionSummary, 0, len(summaries))
	for _, s := range summaries {
		result = append(result, *s)
	}
	return result
}

// elbMetricMeta holds CloudWatch metric metadata for a load balancer type
type elbMetricMeta struct {
	namespace           string
	dimensionName       string
	dimensionValue      string
	volumeMetric        string
	bandwidthMetric     string
}

// getELBMetricMeta returns CloudWatch metric metadata for a load balancer
func getELBMetricMeta(lb types.LoadBalancer) elbMetricMeta {
	switch lb.Type {
	case "network":
		// NLB: extract resource portion from ARN (net/<name>/<id>)
		return elbMetricMeta{
			namespace:       "AWS/NetworkELB",
			dimensionName:   "LoadBalancer",
			dimensionValue:  extractLBResource(lb.ARN),
			volumeMetric:    "NewFlowCount",
			bandwidthMetric: "ProcessedBytes",
		}
	case "classic":
		// CLB: use load balancer name as dimension
		return elbMetricMeta{
			namespace:       "AWS/ELB",
			dimensionName:   "LoadBalancerName",
			dimensionValue:  lb.Name,
			volumeMetric:    "RequestCount",
			bandwidthMetric: "EstimatedProcessedBytes",
		}
	default:
		// ALB: extract resource portion from ARN (app/<name>/<id>)
		return elbMetricMeta{
			namespace:       "AWS/ApplicationELB",
			dimensionName:   "LoadBalancer",
			dimensionValue:  extractLBResource(lb.ARN),
			volumeMetric:    "RequestCount",
			bandwidthMetric: "ProcessedBytes",
		}
	}
}

// extractLBResource extracts the resource portion from an ELBv2 ARN
// e.g., "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/abc123" -> "app/my-alb/abc123"
func extractLBResource(arn string) string {
	const prefix = "loadbalancer/"
	if idx := strings.LastIndex(arn, prefix); idx >= 0 {
		return arn[idx+len(prefix):]
	}
	return arn
}

// parseUsageWindow returns the duration and CloudWatch period for a usage window string
func parseUsageWindow(window string) (duration time.Duration, period int32, err error) {
	switch window {
	case "1h":
		return 1 * time.Hour, 300, nil
	case "24h":
		return 24 * time.Hour, 3600, nil
	default:
		return 0, 0, fmt.Errorf("invalid usage window: %q (must be 1h or 24h)", window)
	}
}

// usageCacheKey creates a cache key for ELB usage data
func usageCacheKey(accountID, region, window string) string {
	return accountID + "|" + region + "|" + window
}

// usageCacheTTL returns the cache TTL for a given usage window
func usageCacheTTL(window string) time.Duration {
	switch window {
	case "24h":
		return 10 * time.Minute
	default:
		return 3 * time.Minute
	}
}

// EnrichELBUsage enriches a slice of load balancers with CloudWatch usage metrics.
// It groups LBs by account+region, checks the usage cache, and fetches from CloudWatch as needed.
func (d *Discovery) EnrichELBUsage(ctx context.Context, loadBalancers []types.LoadBalancer, window string, accounts []Account) {
	windowDuration, period, err := parseUsageWindow(window)
	if err != nil {
		for i := range loadBalancers {
			loadBalancers[i].UsageStatus = types.UsageStatusUnavailable
			loadBalancers[i].UsageError = err.Error()
		}
		return
	}

	now := time.Now().UTC()
	usageEnd := now
	usageStart := now.Add(-windowDuration)

	d.logger.Info("enriching ELB usage",
		"lbCount", len(loadBalancers),
		"window", window,
		"accountCount", len(accounts))

	// Build account lookup by ID and name for role ARN resolution
	accountByID := make(map[string]Account)
	for _, acc := range accounts {
		if acc.ID != "" {
			accountByID[acc.ID] = acc
		}
		if acc.Name != "" {
			accountByID[acc.Name] = acc
		}
	}

	// Group LBs by account+region for batched queries
	type groupKey struct{ accountID, region string }
	groups := make(map[groupKey][]int) // value = indices into loadBalancers
	for i, lb := range loadBalancers {
		key := groupKey{lb.AccountID, lb.Region}
		groups[key] = append(groups[key], i)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for gk, indices := range groups {
		cacheKey := usageCacheKey(gk.accountID, gk.region, window)

		// Check cache
		d.usageCacheMu.RLock()
		entry, cached := d.usageCache[cacheKey]
		d.usageCacheMu.RUnlock()

		if cached && time.Now().Before(entry.expiresAt) {
			// Apply cached data
			for _, i := range indices {
				lb := &loadBalancers[i]
				lbID := lb.ARN
				if lbID == "" {
					lbID = lb.Name
				}
				if usage, ok := entry.value[lbID]; ok {
					lb.UsageWindow = window
					lb.UsageStart = usageStart.Format(time.RFC3339)
					lb.UsageEnd = usageEnd.Format(time.RFC3339)
					lb.RequestVolume = usage.RequestVolume
					lb.RequestMetricName = usage.RequestMetricName
					lb.BandwidthBytes = usage.BandwidthBytes
					lb.BandwidthMetricName = usage.BandwidthMetricName
					lb.UsageStatus = usage.Status
					lb.UsageError = usage.Error
				}
			}
			continue
		}

		// Fetch from CloudWatch for this account+region group
		wg.Add(1)
		go func(gk groupKey, indices []int) {
			defer wg.Done()

			// Acquire semaphore
			d.cwSemaphore <- struct{}{}
			defer func() { <-d.cwSemaphore }()

			// Get a config for this account+region, resolving role ARN from accounts list
			acc, ok := accountByID[gk.accountID]
			if !ok && len(indices) > 0 {
				// Try lookup by account name
				acc, ok = accountByID[loadBalancers[indices[0]].AccountName]
			}
			if !ok {
				acc = Account{ID: gk.accountID}
				if len(indices) > 0 {
					acc.Name = loadBalancers[indices[0]].AccountName
				}
			}
			d.logger.Info("resolved account for CloudWatch",
				"accountID", gk.accountID,
				"region", gk.region,
				"resolvedName", acc.Name,
				"hasRoleARN", acc.RoleARN != "",
				"lbCount", len(indices))

			cfg, err := d.getConfigForAccount(ctx, acc, gk.region)
			if err != nil {
				mu.Lock()
				for _, i := range indices {
					loadBalancers[i].UsageWindow = window
					loadBalancers[i].UsageStart = usageStart.Format(time.RFC3339)
					loadBalancers[i].UsageEnd = usageEnd.Format(time.RFC3339)
					loadBalancers[i].UsageStatus = types.UsageStatusUnavailable
					loadBalancers[i].UsageError = "failed to get AWS config: " + err.Error()
				}
				mu.Unlock()
				return
			}

			cwClient := cloudwatch.NewFromConfig(cfg)
			usageMap := make(map[string]elbUsageData)

			for _, i := range indices {
				lb := loadBalancers[i]
				meta := getELBMetricMeta(lb)

				lbID := lb.ARN
				if lbID == "" {
					lbID = lb.Name
				}

				usage := d.fetchLBUsage(ctx, cwClient, meta, usageStart, usageEnd, period)
				usageMap[lbID] = usage

				mu.Lock()
				loadBalancers[i].UsageWindow = window
				loadBalancers[i].UsageStart = usageStart.Format(time.RFC3339)
				loadBalancers[i].UsageEnd = usageEnd.Format(time.RFC3339)
				loadBalancers[i].RequestVolume = usage.RequestVolume
				loadBalancers[i].RequestMetricName = usage.RequestMetricName
				loadBalancers[i].BandwidthBytes = usage.BandwidthBytes
				loadBalancers[i].BandwidthMetricName = usage.BandwidthMetricName
				loadBalancers[i].UsageStatus = usage.Status
				loadBalancers[i].UsageError = usage.Error
				mu.Unlock()
			}

			// Cache results
			d.usageCacheMu.Lock()
			d.usageCache[cacheKey] = cacheEntry[map[string]elbUsageData]{
				value:     usageMap,
				expiresAt: time.Now().Add(usageCacheTTL(window)),
			}
			d.usageCacheMu.Unlock()
		}(gk, indices)
	}

	wg.Wait()
}

// fetchLBUsage fetches CloudWatch metrics for a single load balancer
func (d *Discovery) fetchLBUsage(ctx context.Context, client *cloudwatch.Client, meta elbMetricMeta, start, end time.Time, period int32) elbUsageData {
	dimension := cwtypes.Dimension{
		Name:  aws.String(meta.dimensionName),
		Value: aws.String(meta.dimensionValue),
	}

	input := &cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(start),
		EndTime:   aws.Time(end),
		ScanBy:    cwtypes.ScanByTimestampDescending,
		MetricDataQueries: []cwtypes.MetricDataQuery{
			{
				Id: aws.String("volume"),
				MetricStat: &cwtypes.MetricStat{
					Metric: &cwtypes.Metric{
						Namespace:  aws.String(meta.namespace),
						MetricName: aws.String(meta.volumeMetric),
						Dimensions: []cwtypes.Dimension{dimension},
					},
					Period: aws.Int32(period),
					Stat:   aws.String("Sum"),
				},
			},
			{
				Id: aws.String("bandwidth"),
				MetricStat: &cwtypes.MetricStat{
					Metric: &cwtypes.Metric{
						Namespace:  aws.String(meta.namespace),
						MetricName: aws.String(meta.bandwidthMetric),
						Dimensions: []cwtypes.Dimension{dimension},
					},
					Period: aws.Int32(period),
					Stat:   aws.String("Sum"),
				},
			},
		},
	}

	d.logger.Info("fetching CloudWatch metrics",
		"namespace", meta.namespace,
		"dimensionName", meta.dimensionName,
		"dimensionValue", meta.dimensionValue,
		"volumeMetric", meta.volumeMetric,
		"bandwidthMetric", meta.bandwidthMetric,
		"start", start.Format(time.RFC3339),
		"end", end.Format(time.RFC3339),
		"period", period)

	output, err := client.GetMetricData(ctx, input)
	if err != nil {
		d.logger.Warn("failed to get CloudWatch metrics",
			"namespace", meta.namespace,
			"dimensionValue", meta.dimensionValue,
			"error", err)
		return elbUsageData{
			RequestMetricName:   meta.volumeMetric,
			BandwidthMetricName: meta.bandwidthMetric,
			Status:              types.UsageStatusUnavailable,
			Error:               err.Error(),
		}
	}

	var volumeSum, bandwidthSum float64
	hasData := false

	for _, result := range output.MetricDataResults {
		if result.Id == nil {
			continue
		}
		d.logger.Info("CloudWatch metric result",
			"id", *result.Id,
			"statusCode", result.StatusCode,
			"datapointCount", len(result.Values),
			"messageCount", len(result.Messages))
		for _, msg := range result.Messages {
			d.logger.Warn("CloudWatch message",
				"id", *result.Id,
				"code", aws.ToString(msg.Code),
				"value", aws.ToString(msg.Value))
		}
		if result.StatusCode == cwtypes.StatusCodeInternalError {
			d.logger.Warn("CloudWatch internal error for metric", "id", *result.Id)
			continue
		}
		for _, v := range result.Values {
			hasData = true
			switch *result.Id {
			case "volume":
				volumeSum += v
			case "bandwidth":
				bandwidthSum += v
			}
		}
	}

	d.logger.Info("CloudWatch usage result",
		"dimensionValue", meta.dimensionValue,
		"volumeSum", volumeSum,
		"bandwidthSum", bandwidthSum,
		"hasData", hasData)

	status := types.UsageStatusOK
	usageErr := ""
	if !hasData {
		status = types.UsageStatusPartial
		usageErr = "no datapoints in window"
	}

	return elbUsageData{
		RequestVolume:       volumeSum,
		RequestMetricName:   meta.volumeMetric,
		BandwidthBytes:      bandwidthSum,
		BandwidthMetricName: meta.bandwidthMetric,
		Status:              status,
		Error:               usageErr,
	}
}
