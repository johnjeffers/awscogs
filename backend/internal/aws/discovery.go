package aws

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/johnjeffers/infra-utilities/awscogs/backend/internal/pricing"
	"github.com/johnjeffers/infra-utilities/awscogs/backend/internal/types"
)

// Discovery handles AWS resource discovery across accounts and regions
type Discovery struct {
	pricingProvider pricing.Provider
	logger          *slog.Logger
}

// NewDiscovery creates a new AWS resource discovery service
func NewDiscovery(pricingProvider pricing.Provider, logger *slog.Logger) *Discovery {
	return &Discovery{
		pricingProvider: pricingProvider,
		logger:          logger,
	}
}

// Account represents an AWS account configuration
type Account struct {
	ID      string
	Name    string
	RoleARN string
}

// DiscoverResources discovers all resources across the specified accounts and regions
func (d *Discovery) DiscoverResources(ctx context.Context, accounts []Account, regions []string) (*types.CostResponse, error) {
	var (
		allEC2    []types.EC2Instance
		allEBS    []types.EBSVolume
		allECS    []types.ECSService
		allRDS    []types.RDSInstance
		allEKS    []types.EKSCluster
		allELB    []types.LoadBalancer
		mu        sync.Mutex
		wg        sync.WaitGroup
		totalCost types.CostValue
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

				// Discover EC2 instances
				ec2Instances, err := d.discoverEC2(ctx, cfg, accountID, accountName, reg)
				if err != nil {
					d.logger.Error("failed to discover EC2 instances",
						"account", acc.Name,
						"region", reg,
						"error", err)
				}

				// Discover EBS volumes
				ebsVolumes, err := d.discoverEBS(ctx, cfg, accountID, accountName, reg)
				if err != nil {
					d.logger.Error("failed to discover EBS volumes",
						"account", acc.Name,
						"region", reg,
						"error", err)
				}

				// Discover ECS services
				ecsServices, err := d.discoverECS(ctx, cfg, accountID, accountName, reg)
				if err != nil {
					d.logger.Error("failed to discover ECS services",
						"account", acc.Name,
						"region", reg,
						"error", err)
				}

				// Discover RDS instances
				rdsInstances, err := d.discoverRDS(ctx, cfg, accountID, accountName, reg)
				if err != nil {
					d.logger.Error("failed to discover RDS instances",
						"account", acc.Name,
						"region", reg,
						"error", err)
				}

				// Discover EKS clusters
				eksClusters, err := d.discoverEKS(ctx, cfg, accountID, accountName, reg)
				if err != nil {
					d.logger.Error("failed to discover EKS clusters",
						"account", acc.Name,
						"region", reg,
						"error", err)
				}

				// Discover Load Balancers
				loadBalancers, err := d.discoverELB(ctx, cfg, accountID, accountName, reg)
				if err != nil {
					d.logger.Error("failed to discover load balancers",
						"account", acc.Name,
						"region", reg,
						"error", err)
				}

				mu.Lock()
				allEC2 = append(allEC2, ec2Instances...)
				allEBS = append(allEBS, ebsVolumes...)
				allECS = append(allECS, ecsServices...)
				allRDS = append(allRDS, rdsInstances...)
				allEKS = append(allEKS, eksClusters...)
				allELB = append(allELB, loadBalancers...)
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

	// Build account and region summaries
	accountSummaries := d.buildAccountSummaries(allEC2, allEBS, allECS, allRDS, allEKS, allELB)
	regionSummaries := d.buildRegionSummaries(allEC2, allEBS, allECS, allRDS, allEKS, allELB)

	return &types.CostResponse{
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
	}, nil
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

	d.logger.Info("discovered regions", "count", len(regions))
	return regions, nil
}

// DiscoverAccounts returns all accounts from AWS Organizations with the specified assume role
func (d *Discovery) DiscoverAccounts(ctx context.Context, assumeRoleName string) ([]Account, error) {
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

// buildAccountSummaries builds account-level cost summaries
func (d *Discovery) buildAccountSummaries(ec2 []types.EC2Instance, ebs []types.EBSVolume, ecs []types.ECSService, rds []types.RDSInstance, eks []types.EKSCluster, elb []types.LoadBalancer) []types.AccountSummary {
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

	result := make([]types.AccountSummary, 0, len(summaries))
	for _, s := range summaries {
		result = append(result, *s)
	}
	return result
}

// buildRegionSummaries builds region-level cost summaries
func (d *Discovery) buildRegionSummaries(ec2 []types.EC2Instance, ebs []types.EBSVolume, ecs []types.ECSService, rds []types.RDSInstance, eks []types.EKSCluster, elb []types.LoadBalancer) []types.RegionSummary {
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

	result := make([]types.RegionSummary, 0, len(summaries))
	for _, s := range summaries {
		result = append(result, *s)
	}
	return result
}
