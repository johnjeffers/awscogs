package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	"github.com/aws/aws-sdk-go-v2/service/pricing/types"
	"golang.org/x/sync/singleflight"

	cogtypes "github.com/johnjeffers/awscogs/backend/internal/types"
)

// AWSProvider implements Provider using the AWS Price List API
type AWSProvider struct {
	client          *pricing.Client
	ec2Cache        map[string]cogtypes.CostValue // key: "region:instanceType"
	ebsCache        map[string]cogtypes.CostValue // key: "region:volumeType"
	ecsCache        map[string]cogtypes.CostValue // key: "region:launchType"
	rdsCache        map[string]cogtypes.CostValue // key: "region:instanceClass:engine:multiAZ"
	eksCache        map[string]cogtypes.CostValue // key: "region"
	elbCache        map[string]cogtypes.CostValue // key: "region:lbType"
	natCache        map[string]cogtypes.CostValue // key: "region"
	eipCache        map[string]cogtypes.CostValue // key: "region:associated"
	secretCache     map[string]cogtypes.CostValue // key: "region"
	publicIPv4Cache map[string]cogtypes.CostValue // key: "region"
	cacheMu         sync.RWMutex
	cacheExpiry     time.Time
	cacheDuration   time.Duration
	sfGroup         singleflight.Group // Prevents concurrent duplicate pricing API calls
	rateLimitMu     sync.Mutex         // Protects rate limiting
	lastAPICall     time.Time          // Time of last API call
	minCallInterval time.Duration      // Minimum time between API calls
}

// NewAWSProvider creates a new AWS pricing provider
func NewAWSProvider(ctx context.Context, cacheDurationMinutes, rateLimitPerSecond int) (*AWSProvider, error) {
	// AWS Pricing API is only available in us-east-1 and ap-south-1
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	client := pricing.NewFromConfig(cfg)

	// Validate credentials by making a test API call
	if err := validateCredentials(ctx, client); err != nil {
		return nil, err
	}

	// Calculate minimum interval between API calls
	// If rateLimitPerSecond is 0 or negative, no rate limiting
	var minInterval time.Duration
	if rateLimitPerSecond > 0 {
		minInterval = time.Second / time.Duration(rateLimitPerSecond)
	}

	return &AWSProvider{
		client:          client,
		ec2Cache:        make(map[string]cogtypes.CostValue),
		ebsCache:        make(map[string]cogtypes.CostValue),
		ecsCache:        make(map[string]cogtypes.CostValue),
		rdsCache:        make(map[string]cogtypes.CostValue),
		eksCache:        make(map[string]cogtypes.CostValue),
		elbCache:        make(map[string]cogtypes.CostValue),
		natCache:        make(map[string]cogtypes.CostValue),
		eipCache:        make(map[string]cogtypes.CostValue),
		secretCache:     make(map[string]cogtypes.CostValue),
		publicIPv4Cache: make(map[string]cogtypes.CostValue),
		cacheDuration:   time.Duration(cacheDurationMinutes) * time.Minute,
		minCallInterval: minInterval,
	}, nil
}

// waitForRateLimit waits until enough time has passed since the last API call
// This enforces a maximum of N calls per second by spacing out requests
func (p *AWSProvider) waitForRateLimit(ctx context.Context) error {
	if p.minCallInterval == 0 {
		return nil // No rate limiting configured
	}

	p.rateLimitMu.Lock()
	defer p.rateLimitMu.Unlock()

	// Calculate how long to wait
	elapsed := time.Since(p.lastAPICall)
	if elapsed < p.minCallInterval {
		waitTime := p.minCallInterval - elapsed
		select {
		case <-time.After(waitTime):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	p.lastAPICall = time.Now()
	return nil
}

// validateCredentials checks that AWS credentials are configured and have access to the Pricing API
func validateCredentials(ctx context.Context, client *pricing.Client) error {
	_, err := client.DescribeServices(ctx, &pricing.DescribeServicesInput{
		ServiceCode: aws.String("AmazonEC2"),
		MaxResults:  aws.Int32(1),
	})
	if err != nil {
		return fmt.Errorf("AWS credentials not found or invalid: %w", err)
	}
	return nil
}

// termFilter creates a TermMatch pricing filter
func termFilter(field, value string) types.Filter {
	return types.Filter{
		Type:  types.FilterTypeTermMatch,
		Field: aws.String(field),
		Value: aws.String(value),
	}
}

// getCachedPrice checks the cache for a price, and on miss uses singleflight to
// fetch it exactly once, preventing thundering herd on concurrent requests.
func (p *AWSProvider) getCachedPrice(cache map[string]cogtypes.CostValue, cacheKey, sfKey string, fetch func() (cogtypes.CostValue, error)) (cogtypes.CostValue, error) {
	p.cacheMu.RLock()
	if price, ok := cache[cacheKey]; ok && time.Now().Before(p.cacheExpiry) {
		p.cacheMu.RUnlock()
		return price, nil
	}
	p.cacheMu.RUnlock()

	v, err, _ := p.sfGroup.Do(sfKey, func() (any, error) {
		// Double-check cache after acquiring singleflight
		p.cacheMu.RLock()
		if price, ok := cache[cacheKey]; ok && time.Now().Before(p.cacheExpiry) {
			p.cacheMu.RUnlock()
			return price, nil
		}
		p.cacheMu.RUnlock()

		price, err := fetch()
		if err != nil {
			return cogtypes.CostValue(0), err
		}

		p.cacheMu.Lock()
		cache[cacheKey] = price
		if p.cacheExpiry.IsZero() || time.Now().After(p.cacheExpiry) {
			p.cacheExpiry = time.Now().Add(p.cacheDuration)
		}
		p.cacheMu.Unlock()

		return price, nil
	})
	if err != nil {
		return 0, err
	}

	return v.(cogtypes.CostValue), nil
}

// GetEC2Price returns the hourly on-demand price for an EC2 instance type
func (p *AWSProvider) GetEC2Price(ctx context.Context, region, instanceType string) (cogtypes.CostValue, error) {
	cacheKey := fmt.Sprintf("%s:%s", region, instanceType)
	return p.getCachedPrice(p.ec2Cache, cacheKey, "ec2:"+cacheKey, func() (cogtypes.CostValue, error) {
		return p.fetchEC2Price(ctx, region, instanceType)
	})
}

// GetEBSPrice returns the hourly price for an EBS volume
func (p *AWSProvider) GetEBSPrice(ctx context.Context, region, volumeType string, sizeGiB, iops, throughput int32) (cogtypes.CostValue, error) {
	// EBS pricing is per GB-month, we convert to hourly
	// Also factor in IOPS and throughput for gp3/io1/io2

	baseCacheKey := fmt.Sprintf("%s:%s", region, volumeType)

	p.cacheMu.RLock()
	basePrice, hasBase := p.ebsCache[baseCacheKey]
	iopsPrice := p.ebsCache[baseCacheKey+":iops"]
	tpPrice := p.ebsCache[baseCacheKey+":throughput"]
	cacheValid := time.Now().Before(p.cacheExpiry)
	p.cacheMu.RUnlock()

	if !hasBase || !cacheValid {
		// Use singleflight to prevent concurrent duplicate API calls
		v, err, _ := p.sfGroup.Do("ebs:"+baseCacheKey, func() (any, error) {
			// Double-check cache
			p.cacheMu.RLock()
			bp, ok := p.ebsCache[baseCacheKey]
			ip := p.ebsCache[baseCacheKey+":iops"]
			tp := p.ebsCache[baseCacheKey+":throughput"]
			valid := time.Now().Before(p.cacheExpiry)
			p.cacheMu.RUnlock()

			if ok && valid {
				return [3]cogtypes.CostValue{bp, ip, tp}, nil
			}

			bp, ip, tp, err := p.fetchEBSPrices(ctx, region, volumeType)
			if err != nil {
				return [3]cogtypes.CostValue{}, err
			}

			p.cacheMu.Lock()
			p.ebsCache[baseCacheKey] = bp
			p.ebsCache[baseCacheKey+":iops"] = ip
			p.ebsCache[baseCacheKey+":throughput"] = tp
			if p.cacheExpiry.IsZero() || time.Now().After(p.cacheExpiry) {
				p.cacheExpiry = time.Now().Add(p.cacheDuration)
			}
			p.cacheMu.Unlock()

			return [3]cogtypes.CostValue{bp, ip, tp}, nil
		})
		if err != nil {
			return 0, err
		}
		prices := v.([3]cogtypes.CostValue)
		basePrice = prices[0]
		iopsPrice = prices[1]
		tpPrice = prices[2]
	}

	// Calculate total monthly cost, then convert to hourly
	// Base storage cost (per GB-month)
	monthlyCost := float64(basePrice) * float64(sizeGiB)

	// Add IOPS cost for io1/io2/gp3
	if volumeType == "gp3" && iops > 3000 {
		// gp3 includes 3000 IOPS free
		monthlyCost += float64(iopsPrice) * float64(iops-3000)
	} else if volumeType == "io1" || volumeType == "io2" {
		monthlyCost += float64(iopsPrice) * float64(iops)
	}

	// Add throughput cost for gp3
	if volumeType == "gp3" && throughput > 125 {
		// gp3 includes 125 MiB/s free
		monthlyCost += float64(tpPrice) * float64(throughput-125)
	}

	// Convert monthly to hourly (730 hours per month)
	hourlyCost := monthlyCost / 730.0

	return cogtypes.CostValue(hourlyCost), nil
}

// GetRDSPrice returns the hourly on-demand price for an RDS instance
func (p *AWSProvider) GetRDSPrice(ctx context.Context, region, instanceClass, engine string, multiAZ bool) (cogtypes.CostValue, error) {
	multiAZStr := "false"
	if multiAZ {
		multiAZStr = "true"
	}
	cacheKey := fmt.Sprintf("%s:%s:%s:%s", region, instanceClass, engine, multiAZStr)
	return p.getCachedPrice(p.rdsCache, cacheKey, "rds:"+cacheKey, func() (cogtypes.CostValue, error) {
		return p.fetchRDSPrice(ctx, region, instanceClass, engine, multiAZ)
	})
}

// GetECSPrice returns the hourly price for an ECS Fargate service
// For Fargate, pricing is based on vCPU and memory hours
// Since we don't have task definition details, we estimate with 0.5 vCPU and 1GB memory per task
func (p *AWSProvider) GetECSPrice(ctx context.Context, region, launchType string, runningCount int32) (cogtypes.CostValue, error) {
	if runningCount <= 0 {
		return 0, nil
	}

	// EC2 launch type - cost is covered by EC2 instances
	if launchType != "FARGATE" {
		return 0, nil
	}

	cacheKey := fmt.Sprintf("%s:%s", region, launchType)
	perTaskPrice, err := p.getCachedPrice(p.ecsCache, cacheKey, "ecs:"+cacheKey, func() (cogtypes.CostValue, error) {
		return p.fetchECSFargatePrice(ctx, region)
	})
	if err != nil {
		return 0, err
	}

	return perTaskPrice * cogtypes.CostValue(runningCount), nil
}

// GetEKSPrice returns the hourly price for an EKS cluster control plane
func (p *AWSProvider) GetEKSPrice(ctx context.Context, region string) (cogtypes.CostValue, error) {
	return p.getCachedPrice(p.eksCache, region, "eks:"+region, func() (cogtypes.CostValue, error) {
		return p.fetchEKSPrice(ctx, region)
	})
}

// GetELBPrice returns the hourly price for a load balancer by type
func (p *AWSProvider) GetELBPrice(ctx context.Context, region, lbType string) (cogtypes.CostValue, error) {
	cacheKey := fmt.Sprintf("%s:%s", region, lbType)
	return p.getCachedPrice(p.elbCache, cacheKey, "elb:"+cacheKey, func() (cogtypes.CostValue, error) {
		return p.fetchELBPrice(ctx, region, lbType)
	})
}

// GetNATGatewayPrice returns the hourly price for a NAT Gateway
func (p *AWSProvider) GetNATGatewayPrice(ctx context.Context, region string) (cogtypes.CostValue, error) {
	return p.getCachedPrice(p.natCache, region, "nat:"+region, func() (cogtypes.CostValue, error) {
		return p.fetchNATGatewayPrice(ctx, region)
	})
}

// GetElasticIPPrice returns the hourly price for an Elastic IP
// Associated EIPs attached to running instances are free (billing rule, not API-sourced)
func (p *AWSProvider) GetElasticIPPrice(ctx context.Context, region string, isAssociated bool) (cogtypes.CostValue, error) {
	if isAssociated {
		return 0, nil
	}

	return p.getCachedPrice(p.eipCache, region, "eip:"+region, func() (cogtypes.CostValue, error) {
		return p.fetchElasticIPPrice(ctx, region)
	})
}

// GetSecretPrice returns the hourly price for a Secrets Manager secret
func (p *AWSProvider) GetSecretPrice(ctx context.Context, region string) (cogtypes.CostValue, error) {
	return p.getCachedPrice(p.secretCache, region, "secret:"+region, func() (cogtypes.CostValue, error) {
		return p.fetchSecretPrice(ctx, region)
	})
}

// GetPublicIPv4Price returns the hourly price for a public IPv4 address
func (p *AWSProvider) GetPublicIPv4Price(ctx context.Context, region string) (cogtypes.CostValue, error) {
	return p.getCachedPrice(p.publicIPv4Cache, region, "publicipv4:"+region, func() (cogtypes.CostValue, error) {
		return p.fetchPublicIPv4Price(ctx, region)
	})
}

// RefreshCache forces a refresh of the pricing cache
func (p *AWSProvider) RefreshCache(ctx context.Context) error {
	p.cacheMu.Lock()
	p.ec2Cache = make(map[string]cogtypes.CostValue)
	p.ebsCache = make(map[string]cogtypes.CostValue)
	p.ecsCache = make(map[string]cogtypes.CostValue)
	p.rdsCache = make(map[string]cogtypes.CostValue)
	p.eksCache = make(map[string]cogtypes.CostValue)
	p.elbCache = make(map[string]cogtypes.CostValue)
	p.natCache = make(map[string]cogtypes.CostValue)
	p.eipCache = make(map[string]cogtypes.CostValue)
	p.secretCache = make(map[string]cogtypes.CostValue)
	p.publicIPv4Cache = make(map[string]cogtypes.CostValue)
	p.cacheExpiry = time.Time{}
	p.cacheMu.Unlock()
	return nil
}

// ---- Fetch functions: each queries the AWS Pricing API for a specific resource type ----

// fetchEC2Price queries the AWS Price List API for EC2 pricing
func (p *AWSProvider) fetchEC2Price(ctx context.Context, region, instanceType string) (cogtypes.CostValue, error) {
	locationName, ok := regionToLocation[region]
	if !ok {
		return 0, fmt.Errorf("unknown region: %s", region)
	}

	if err := p.waitForRateLimit(ctx); err != nil {
		return 0, fmt.Errorf("rate limit: %w", err)
	}

	output, err := p.client.GetProducts(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEC2"),
		Filters: []types.Filter{
			termFilter("instanceType", instanceType),
			termFilter("location", locationName),
			termFilter("operatingSystem", "Linux"),
			termFilter("tenancy", "Shared"),
			termFilter("preInstalledSw", "NA"),
			termFilter("capacitystatus", "Used"),
		},
		MaxResults: aws.Int32(1),
	})
	if err != nil {
		return 0, fmt.Errorf("GetProducts for EC2: %w", err)
	}

	if len(output.PriceList) == 0 {
		return 0, fmt.Errorf("no pricing found for EC2 %s in %s", instanceType, region)
	}

	return parsePriceFromProduct(output.PriceList[0])
}

// fetchEBSPrices queries the AWS Price List API for EBS storage, IOPS, and throughput pricing
func (p *AWSProvider) fetchEBSPrices(ctx context.Context, region, volumeType string) (base, iops, throughput cogtypes.CostValue, err error) {
	locationName, ok := regionToLocation[region]
	if !ok {
		return 0, 0, 0, fmt.Errorf("unknown region: %s", region)
	}

	// Fetch base storage price
	if err := p.waitForRateLimit(ctx); err != nil {
		return 0, 0, 0, fmt.Errorf("rate limit: %w", err)
	}

	output, err := p.client.GetProducts(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEC2"),
		Filters: []types.Filter{
			termFilter("productFamily", "Storage"),
			termFilter("location", locationName),
			termFilter("volumeApiName", volumeType),
		},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		return 0, 0, 0, fmt.Errorf("GetProducts for EBS storage: %w", err)
	}

	if len(output.PriceList) == 0 {
		return 0, 0, 0, fmt.Errorf("no pricing found for EBS %s in %s", volumeType, region)
	}

	base, err = parsePriceFromProduct(output.PriceList[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("parsing EBS base price: %w", err)
	}

	// Fetch IOPS pricing for gp3/io1/io2
	if volumeType == "gp3" || volumeType == "io1" || volumeType == "io2" {
		iops, err = p.fetchEBSIOPSPrice(ctx, region, volumeType)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("fetching EBS IOPS price: %w", err)
		}
	}

	// Fetch throughput pricing for gp3
	if volumeType == "gp3" {
		throughput, err = p.fetchEBSThroughputPrice(ctx, region)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("fetching EBS throughput price: %w", err)
		}
	}

	return base, iops, throughput, nil
}

// fetchEBSIOPSPrice queries the Pricing API for EBS provisioned IOPS pricing
func (p *AWSProvider) fetchEBSIOPSPrice(ctx context.Context, region, volumeType string) (cogtypes.CostValue, error) {
	locationName, ok := regionToLocation[region]
	if !ok {
		return 0, fmt.Errorf("unknown region: %s", region)
	}

	if err := p.waitForRateLimit(ctx); err != nil {
		return 0, fmt.Errorf("rate limit: %w", err)
	}

	output, err := p.client.GetProducts(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEC2"),
		Filters: []types.Filter{
			termFilter("productFamily", "System Operation"),
			termFilter("location", locationName),
			termFilter("volumeApiName", volumeType),
			termFilter("group", "EBS IOPS"),
		},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		return 0, fmt.Errorf("GetProducts for EBS IOPS: %w", err)
	}

	if len(output.PriceList) == 0 {
		return 0, fmt.Errorf("no IOPS pricing found for EBS %s in %s", volumeType, region)
	}

	return parsePriceFromProduct(output.PriceList[0])
}

// fetchEBSThroughputPrice queries the Pricing API for gp3 throughput pricing
func (p *AWSProvider) fetchEBSThroughputPrice(ctx context.Context, region string) (cogtypes.CostValue, error) {
	locationName, ok := regionToLocation[region]
	if !ok {
		return 0, fmt.Errorf("unknown region: %s", region)
	}

	if err := p.waitForRateLimit(ctx); err != nil {
		return 0, fmt.Errorf("rate limit: %w", err)
	}

	output, err := p.client.GetProducts(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEC2"),
		Filters: []types.Filter{
			termFilter("productFamily", "Provisioned Throughput"),
			termFilter("location", locationName),
			termFilter("volumeApiName", "gp3"),
		},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		return 0, fmt.Errorf("GetProducts for EBS throughput: %w", err)
	}

	if len(output.PriceList) == 0 {
		return 0, fmt.Errorf("no throughput pricing found for gp3 in %s", region)
	}

	return parsePriceFromProduct(output.PriceList[0])
}

// fetchRDSPrice queries the AWS Price List API for RDS pricing
func (p *AWSProvider) fetchRDSPrice(ctx context.Context, region, instanceClass, engine string, multiAZ bool) (cogtypes.CostValue, error) {
	locationName, ok := regionToLocation[region]
	if !ok {
		return 0, fmt.Errorf("unknown region: %s", region)
	}

	if err := p.waitForRateLimit(ctx); err != nil {
		return 0, fmt.Errorf("rate limit: %w", err)
	}

	dbEngine := mapRDSEngine(engine)

	deploymentOption := "Single-AZ"
	if multiAZ {
		deploymentOption = "Multi-AZ"
	}

	output, err := p.client.GetProducts(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonRDS"),
		Filters: []types.Filter{
			termFilter("instanceType", instanceClass),
			termFilter("location", locationName),
			termFilter("databaseEngine", dbEngine),
			termFilter("deploymentOption", deploymentOption),
		},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		return 0, fmt.Errorf("GetProducts for RDS: %w", err)
	}

	if len(output.PriceList) == 0 {
		return 0, fmt.Errorf("no pricing found for RDS %s %s in %s", instanceClass, engine, region)
	}

	return parsePriceFromProduct(output.PriceList[0])
}

// fetchECSFargatePrice queries the Pricing API for Fargate vCPU and memory rates,
// then computes an estimated per-task cost using 0.5 vCPU + 1GB memory.
// Verified from AmazonECS bulk pricing:
//   - vCPU: usagetype ends with Fargate-vCPU-Hours:perCPU, cputype=perCPU, tenancy=Shared
//   - Memory: usagetype ends with Fargate-GB-Hours, memorytype=perGB, tenancy=Shared
//   - ARM and Windows variants have different usagetypes (Fargate-ARM-*, Fargate-Windows-*)
func (p *AWSProvider) fetchECSFargatePrice(ctx context.Context, region string) (cogtypes.CostValue, error) {
	locationName, ok := regionToLocation[region]
	if !ok {
		return 0, fmt.Errorf("unknown region: %s", region)
	}

	if err := p.waitForRateLimit(ctx); err != nil {
		return 0, fmt.Errorf("rate limit: %w", err)
	}

	// Fetch all Fargate compute products for this region
	output, err := p.client.GetProducts(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonECS"),
		Filters: []types.Filter{
			termFilter("productFamily", "Compute"),
			termFilter("location", locationName),
			termFilter("tenancy", "Shared"),
		},
		MaxResults: aws.Int32(20),
	})
	if err != nil {
		return 0, fmt.Errorf("GetProducts for ECS Fargate: %w", err)
	}

	// Parse results to find Linux x86 vCPU and memory pricing
	// Exclude ARM (Fargate-ARM-*) and Windows (Fargate-Windows-*) variants
	var vcpuPrice, memPrice cogtypes.CostValue
	for _, pl := range output.PriceList {
		usagetype := getProductAttribute(pl, "usagetype")
		if strings.Contains(usagetype, "ARM") || strings.Contains(usagetype, "Windows") {
			continue
		}

		price, err := parsePriceFromProduct(pl)
		if err != nil {
			continue
		}

		if strings.Contains(usagetype, "Fargate-vCPU-Hours") {
			vcpuPrice = price
		} else if strings.Contains(usagetype, "Fargate-GB-Hours") {
			memPrice = price
		}
	}

	if vcpuPrice == 0 && memPrice == 0 {
		return 0, fmt.Errorf("no Fargate pricing found in %s", region)
	}

	// Estimate per-task cost: 0.5 vCPU + 1GB memory
	perTaskPrice := cogtypes.CostValue(0.5)*vcpuPrice + memPrice
	return perTaskPrice, nil
}

// fetchEKSPrice queries the Pricing API for EKS control plane pricing
// Verified from AmazonEKS bulk pricing:
//   - Standard control plane: operation=CreateOperation, tiertype=HAStandard, locationType=AWS Region
//   - Other products: ExtendedSupport, Outposts, Provisioned, AutoMode, Fargate — must be excluded
func (p *AWSProvider) fetchEKSPrice(ctx context.Context, region string) (cogtypes.CostValue, error) {
	locationName, ok := regionToLocation[region]
	if !ok {
		return 0, fmt.Errorf("unknown region: %s", region)
	}

	if err := p.waitForRateLimit(ctx); err != nil {
		return 0, fmt.Errorf("rate limit: %w", err)
	}

	output, err := p.client.GetProducts(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEKS"),
		Filters: []types.Filter{
			termFilter("productFamily", "Compute"),
			termFilter("location", locationName),
			termFilter("locationType", "AWS Region"),
			termFilter("operation", "CreateOperation"),
		},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		return 0, fmt.Errorf("GetProducts for EKS: %w", err)
	}

	if len(output.PriceList) == 0 {
		return 0, fmt.Errorf("no pricing found for EKS in %s", region)
	}

	return parsePriceFromProduct(output.PriceList[0])
}

// fetchELBPrice queries the Pricing API for load balancer hourly pricing
// Confirmed from AWSELB bulk pricing data:
//   - ALB: productFamily=Load Balancer-Application, usagetype=LoadBalancerUsage, group=ELB:Balancing
//   - NLB: productFamily=Load Balancer-Network, usagetype=LoadBalancerUsage, group=ELB:Balancing
//   - CLB: productFamily=Load Balancer, usagetype=LoadBalancerUsage, group=ELB:Balancing
func (p *AWSProvider) fetchELBPrice(ctx context.Context, region, lbType string) (cogtypes.CostValue, error) {
	locationName, ok := regionToLocation[region]
	if !ok {
		return 0, fmt.Errorf("unknown region: %s", region)
	}

	if err := p.waitForRateLimit(ctx); err != nil {
		return 0, fmt.Errorf("rate limit: %w", err)
	}

	// Map load balancer type to pricing API product family
	var productFamily string
	switch lbType {
	case "application":
		productFamily = "Load Balancer-Application"
	case "network":
		productFamily = "Load Balancer-Network"
	default:
		productFamily = "Load Balancer"
	}

	output, err := p.client.GetProducts(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String("AWSELB"),
		Filters: []types.Filter{
			termFilter("productFamily", productFamily),
			termFilter("location", locationName),
			termFilter("locationType", "AWS Region"),
		},
		MaxResults: aws.Int32(20),
	})
	if err != nil {
		return 0, fmt.Errorf("GetProducts for ELB %s: %w", lbType, err)
	}

	// Find the base hourly product (usagetype ends with "LoadBalancerUsage", not LCU/Reserved/TS)
	for _, pl := range output.PriceList {
		usagetype := getProductAttribute(pl, "usagetype")
		if strings.HasSuffix(usagetype, "LoadBalancerUsage") {
			return parsePriceFromProduct(pl)
		}
	}

	if len(output.PriceList) == 0 {
		return 0, fmt.Errorf("no pricing found for ELB %s in %s", lbType, region)
	}

	// Fallback to first result
	return parsePriceFromProduct(output.PriceList[0])
}

// fetchNATGatewayPrice queries the Pricing API for NAT Gateway hourly pricing
// Verified from AmazonEC2 bulk pricing:
//   - Hourly: operation=NatGateway, usagetype=NatGateway-Hours, group=NGW:NatGateway
//   - Data processing (excluded): usagetype=NatGateway-Bytes
//   - Regional variants (excluded): operation=RegionalNatGateway
//   - Provisioned (excluded): usagetype=NatGateway-Prvd-*
func (p *AWSProvider) fetchNATGatewayPrice(ctx context.Context, region string) (cogtypes.CostValue, error) {
	locationName, ok := regionToLocation[region]
	if !ok {
		return 0, fmt.Errorf("unknown region: %s", region)
	}

	if err := p.waitForRateLimit(ctx); err != nil {
		return 0, fmt.Errorf("rate limit: %w", err)
	}

	output, err := p.client.GetProducts(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEC2"),
		Filters: []types.Filter{
			termFilter("productFamily", "NAT Gateway"),
			termFilter("location", locationName),
			termFilter("operation", "NatGateway"),
			termFilter("group", "NGW:NatGateway"),
		},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		return 0, fmt.Errorf("GetProducts for NAT Gateway: %w", err)
	}

	// Find the hourly product (usagetype=NatGateway-Hours, not Bytes or Prvd)
	for _, pl := range output.PriceList {
		usagetype := getProductAttribute(pl, "usagetype")
		if usagetype == "NatGateway-Hours" {
			return parsePriceFromProduct(pl)
		}
	}

	if len(output.PriceList) == 0 {
		return 0, fmt.Errorf("no pricing found for NAT Gateway in %s", region)
	}

	return parsePriceFromProduct(output.PriceList[0])
}

// fetchElasticIPPrice queries the Pricing API for idle Elastic IP hourly pricing
// Verified from AmazonVPC bulk pricing: EIP pricing is under AmazonVPC (not AmazonEC2)
// as public IPv4 addresses. Since Feb 2024, all public IPv4 addresses are charged.
//   - Idle: group=VPCPublicIPv4Address, usagetype ends with PublicIPv4:IdleAddress
//   - In-use: group=VPCPublicIPv4Address, usagetype ends with PublicIPv4:InUseAddress
func (p *AWSProvider) fetchElasticIPPrice(ctx context.Context, region string) (cogtypes.CostValue, error) {
	locationName, ok := regionToLocation[region]
	if !ok {
		return 0, fmt.Errorf("unknown region: %s", region)
	}

	if err := p.waitForRateLimit(ctx); err != nil {
		return 0, fmt.Errorf("rate limit: %w", err)
	}

	output, err := p.client.GetProducts(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonVPC"),
		Filters: []types.Filter{
			termFilter("location", locationName),
			termFilter("group", "VPCPublicIPv4Address"),
		},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		return 0, fmt.Errorf("GetProducts for Elastic IP: %w", err)
	}

	// Find the idle address product
	for _, pl := range output.PriceList {
		usagetype := getProductAttribute(pl, "usagetype")
		if strings.HasSuffix(usagetype, "PublicIPv4:IdleAddress") {
			return parsePriceFromProduct(pl)
		}
	}

	if len(output.PriceList) == 0 {
		return 0, fmt.Errorf("no pricing found for Elastic IP in %s", region)
	}

	return parsePriceFromProduct(output.PriceList[0])
}

// fetchSecretPrice queries the Pricing API for Secrets Manager per-secret pricing
// Returns the hourly cost (monthly cost / 730 hours)
func (p *AWSProvider) fetchSecretPrice(ctx context.Context, region string) (cogtypes.CostValue, error) {
	locationName, ok := regionToLocation[region]
	if !ok {
		return 0, fmt.Errorf("unknown region: %s", region)
	}

	if err := p.waitForRateLimit(ctx); err != nil {
		return 0, fmt.Errorf("rate limit: %w", err)
	}

	output, err := p.client.GetProducts(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String("AWSSecretsManager"),
		Filters: []types.Filter{
			termFilter("productFamily", "Secret"),
			termFilter("location", locationName),
		},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		return 0, fmt.Errorf("GetProducts for Secrets Manager: %w", err)
	}

	if len(output.PriceList) == 0 {
		return 0, fmt.Errorf("no pricing found for Secrets Manager in %s", region)
	}

	monthlyPrice, err := parsePriceFromProduct(output.PriceList[0])
	if err != nil {
		return 0, err
	}

	// Convert monthly to hourly (730 hours per month)
	return monthlyPrice / 730.0, nil
}

// fetchPublicIPv4Price queries the Pricing API for public IPv4 address hourly pricing
// Verified from AmazonVPC bulk pricing:
//   - In-use: group=VPCPublicIPv4Address, usagetype ends with PublicIPv4:InUseAddress
//   - Idle: group=VPCPublicIPv4Address, usagetype ends with PublicIPv4:IdleAddress
func (p *AWSProvider) fetchPublicIPv4Price(ctx context.Context, region string) (cogtypes.CostValue, error) {
	locationName, ok := regionToLocation[region]
	if !ok {
		return 0, fmt.Errorf("unknown region: %s", region)
	}

	if err := p.waitForRateLimit(ctx); err != nil {
		return 0, fmt.Errorf("rate limit: %w", err)
	}

	output, err := p.client.GetProducts(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonVPC"),
		Filters: []types.Filter{
			termFilter("location", locationName),
			termFilter("group", "VPCPublicIPv4Address"),
		},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		return 0, fmt.Errorf("GetProducts for public IPv4: %w", err)
	}

	// Find the in-use address product
	for _, pl := range output.PriceList {
		usagetype := getProductAttribute(pl, "usagetype")
		if strings.HasSuffix(usagetype, "PublicIPv4:InUseAddress") {
			return parsePriceFromProduct(pl)
		}
	}

	if len(output.PriceList) == 0 {
		return 0, fmt.Errorf("no pricing found for public IPv4 in %s", region)
	}

	return parsePriceFromProduct(output.PriceList[0])
}

// ---- Helpers ----

// mapRDSEngine maps RDS engine names to pricing API database engine names
func mapRDSEngine(engine string) string {
	engineMap := map[string]string{
		"mysql":             "MySQL",
		"postgres":          "PostgreSQL",
		"mariadb":           "MariaDB",
		"oracle-se2":        "Oracle",
		"oracle-ee":         "Oracle",
		"sqlserver-se":      "SQL Server",
		"sqlserver-ee":      "SQL Server",
		"sqlserver-ex":      "SQL Server",
		"sqlserver-web":     "SQL Server",
		"aurora":            "Aurora MySQL",
		"aurora-mysql":      "Aurora MySQL",
		"aurora-postgresql": "Aurora PostgreSQL",
	}

	if mapped, ok := engineMap[engine]; ok {
		return mapped
	}
	return engine
}

// getProductAttribute extracts a named attribute from the AWS pricing product JSON
func getProductAttribute(priceListJSON, attrName string) string {
	var product map[string]any
	if err := json.Unmarshal([]byte(priceListJSON), &product); err != nil {
		return ""
	}

	prod, ok := product["product"].(map[string]any)
	if !ok {
		return ""
	}

	attrs, ok := prod["attributes"].(map[string]any)
	if !ok {
		return ""
	}

	val, _ := attrs[attrName].(string)
	return val
}

// parsePriceFromProduct extracts the hourly on-demand price from the AWS pricing JSON
func parsePriceFromProduct(priceListJSON string) (cogtypes.CostValue, error) {
	var product map[string]any
	if err := json.Unmarshal([]byte(priceListJSON), &product); err != nil {
		return 0, fmt.Errorf("parsing price list JSON: %w", err)
	}

	terms, ok := product["terms"].(map[string]any)
	if !ok {
		return 0, fmt.Errorf("no terms in price list")
	}

	onDemand, ok := terms["OnDemand"].(map[string]any)
	if !ok {
		return 0, fmt.Errorf("no OnDemand terms in price list")
	}

	// Get the first (and usually only) offer
	for _, offerVal := range onDemand {
		offer, ok := offerVal.(map[string]any)
		if !ok {
			continue
		}

		priceDimensions, ok := offer["priceDimensions"].(map[string]any)
		if !ok {
			continue
		}

		// Get the first price dimension
		for _, dimVal := range priceDimensions {
			dim, ok := dimVal.(map[string]any)
			if !ok {
				continue
			}

			pricePerUnit, ok := dim["pricePerUnit"].(map[string]any)
			if !ok {
				continue
			}

			usdStr, ok := pricePerUnit["USD"].(string)
			if !ok {
				continue
			}

			price, err := strconv.ParseFloat(usdStr, 64)
			if err != nil {
				return 0, fmt.Errorf("parsing USD price: %w", err)
			}

			return cogtypes.CostValue(price), nil
		}
	}

	return 0, fmt.Errorf("could not extract price from product")
}

// regionToLocation maps AWS region codes to pricing API location names
var regionToLocation = map[string]string{
	"us-east-1":      "US East (N. Virginia)",
	"us-east-2":      "US East (Ohio)",
	"us-west-1":      "US West (N. California)",
	"us-west-2":      "US West (Oregon)",
	"af-south-1":     "Africa (Cape Town)",
	"ap-east-1":      "Asia Pacific (Hong Kong)",
	"ap-south-1":     "Asia Pacific (Mumbai)",
	"ap-south-2":     "Asia Pacific (Hyderabad)",
	"ap-southeast-1": "Asia Pacific (Singapore)",
	"ap-southeast-2": "Asia Pacific (Sydney)",
	"ap-southeast-3": "Asia Pacific (Jakarta)",
	"ap-southeast-4": "Asia Pacific (Melbourne)",
	"ap-northeast-1": "Asia Pacific (Tokyo)",
	"ap-northeast-2": "Asia Pacific (Seoul)",
	"ap-northeast-3": "Asia Pacific (Osaka)",
	"ca-central-1":   "Canada (Central)",
	"ca-west-1":      "Canada West (Calgary)",
	"eu-central-1":   "EU (Frankfurt)",
	"eu-central-2":   "EU (Zurich)",
	"eu-west-1":      "EU (Ireland)",
	"eu-west-2":      "EU (London)",
	"eu-west-3":      "EU (Paris)",
	"eu-south-1":     "EU (Milan)",
	"eu-south-2":     "EU (Spain)",
	"eu-north-1":     "EU (Stockholm)",
	"il-central-1":   "Israel (Tel Aviv)",
	"me-south-1":     "Middle East (Bahrain)",
	"me-central-1":   "Middle East (UAE)",
	"sa-east-1":      "South America (Sao Paulo)",
}
