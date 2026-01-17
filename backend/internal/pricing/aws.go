package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	"github.com/aws/aws-sdk-go-v2/service/pricing/types"

	cogtypes "github.com/johnjeffers/infra-utilities/awscogs/backend/internal/types"
)

// AWSProvider implements Provider using the AWS Price List API
type AWSProvider struct {
	client        *pricing.Client
	ec2Cache      map[string]cogtypes.CostValue // key: "region:instanceType"
	ebsCache      map[string]cogtypes.CostValue // key: "region:volumeType"
	ecsCache      map[string]cogtypes.CostValue // key: "region:launchType"
	rdsCache      map[string]cogtypes.CostValue // key: "region:instanceClass:engine:multiAZ"
	cacheMu       sync.RWMutex
	cacheExpiry   time.Time
	cacheDuration time.Duration
}

// NewAWSProvider creates a new AWS pricing provider
func NewAWSProvider(ctx context.Context, cacheDurationMinutes int) (*AWSProvider, error) {
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

	return &AWSProvider{
		client:        client,
		ec2Cache:      make(map[string]cogtypes.CostValue),
		ebsCache:      make(map[string]cogtypes.CostValue),
		ecsCache:      make(map[string]cogtypes.CostValue),
		rdsCache:      make(map[string]cogtypes.CostValue),
		cacheDuration: time.Duration(cacheDurationMinutes) * time.Minute,
	}, nil
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

// GetEC2Price returns the hourly on-demand price for an EC2 instance type
func (p *AWSProvider) GetEC2Price(ctx context.Context, region, instanceType string) (cogtypes.CostValue, error) {
	cacheKey := fmt.Sprintf("%s:%s", region, instanceType)

	// Check cache first
	p.cacheMu.RLock()
	if price, ok := p.ec2Cache[cacheKey]; ok && time.Now().Before(p.cacheExpiry) {
		p.cacheMu.RUnlock()
		return price, nil
	}
	p.cacheMu.RUnlock()

	// Fetch from API
	price, err := p.fetchEC2Price(ctx, region, instanceType)
	if err != nil {
		return 0, err
	}

	// Update cache
	p.cacheMu.Lock()
	p.ec2Cache[cacheKey] = price
	if p.cacheExpiry.IsZero() || time.Now().After(p.cacheExpiry) {
		p.cacheExpiry = time.Now().Add(p.cacheDuration)
	}
	p.cacheMu.Unlock()

	return price, nil
}

// GetEBSPrice returns the hourly price for an EBS volume
func (p *AWSProvider) GetEBSPrice(ctx context.Context, region, volumeType string, sizeGiB, iops, throughput int32) (cogtypes.CostValue, error) {
	// EBS pricing is per GB-month, we convert to hourly
	// Also factor in IOPS and throughput for gp3/io1/io2

	baseCacheKey := fmt.Sprintf("%s:%s", region, volumeType)

	p.cacheMu.RLock()
	basePrice, hasBase := p.ebsCache[baseCacheKey]
	iopsPrice, hasIOPS := p.ebsCache[baseCacheKey+":iops"]
	tpPrice, hasTP := p.ebsCache[baseCacheKey+":throughput"]
	cacheValid := time.Now().Before(p.cacheExpiry)
	p.cacheMu.RUnlock()

	if !hasBase || !cacheValid {
		var err error
		basePrice, iopsPrice, tpPrice, err = p.fetchEBSPrices(ctx, region, volumeType)
		if err != nil {
			return 0, err
		}

		p.cacheMu.Lock()
		p.ebsCache[baseCacheKey] = basePrice
		p.ebsCache[baseCacheKey+":iops"] = iopsPrice
		p.ebsCache[baseCacheKey+":throughput"] = tpPrice
		if p.cacheExpiry.IsZero() || time.Now().After(p.cacheExpiry) {
			p.cacheExpiry = time.Now().Add(p.cacheDuration)
		}
		p.cacheMu.Unlock()
	} else if hasIOPS {
		// Use cached IOPS price
	} else if hasTP {
		// Use cached throughput price
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

	// Check cache first
	p.cacheMu.RLock()
	if price, ok := p.rdsCache[cacheKey]; ok && time.Now().Before(p.cacheExpiry) {
		p.cacheMu.RUnlock()
		return price, nil
	}
	p.cacheMu.RUnlock()

	// Fetch from API
	price, err := p.fetchRDSPrice(ctx, region, instanceClass, engine, multiAZ)
	if err != nil {
		return 0, err
	}

	// Update cache
	p.cacheMu.Lock()
	p.rdsCache[cacheKey] = price
	if p.cacheExpiry.IsZero() || time.Now().After(p.cacheExpiry) {
		p.cacheExpiry = time.Now().Add(p.cacheDuration)
	}
	p.cacheMu.Unlock()

	return price, nil
}

// GetECSPrice returns the hourly price for an ECS Fargate service
// For Fargate, pricing is based on vCPU and memory hours
// Since we don't have task definition details, we use average task estimates
func (p *AWSProvider) GetECSPrice(ctx context.Context, region, launchType string, runningCount int32) (cogtypes.CostValue, error) {
	if runningCount <= 0 {
		return 0, nil
	}

	// EC2 launch type - cost is covered by EC2 instances
	if launchType != "FARGATE" {
		return 0, nil
	}

	cacheKey := fmt.Sprintf("%s:%s", region, launchType)

	// Check cache first
	p.cacheMu.RLock()
	if price, ok := p.ecsCache[cacheKey]; ok && time.Now().Before(p.cacheExpiry) {
		p.cacheMu.RUnlock()
		// Price per task * running count
		return price * cogtypes.CostValue(runningCount), nil
	}
	p.cacheMu.RUnlock()

	// Calculate Fargate price per task
	// Using default estimate of 0.5 vCPU and 1GB memory per task
	// Fargate pricing (Linux, us-east-1):
	// - vCPU: $0.04048 per vCPU per hour
	// - Memory: $0.004445 per GB per hour
	perTaskPrice := getECSFargatePrice(region)

	// Update cache
	p.cacheMu.Lock()
	p.ecsCache[cacheKey] = perTaskPrice
	if p.cacheExpiry.IsZero() || time.Now().After(p.cacheExpiry) {
		p.cacheExpiry = time.Now().Add(p.cacheDuration)
	}
	p.cacheMu.Unlock()

	return perTaskPrice * cogtypes.CostValue(runningCount), nil
}

// getECSFargatePrice returns the estimated hourly cost per Fargate task
// Using default 0.5 vCPU and 1GB memory as a baseline
func getECSFargatePrice(region string) cogtypes.CostValue {
	// Fargate pricing varies by region
	// These are approximate per-task costs for 0.5 vCPU + 1GB memory
	// vCPU: $0.04048/hr, Memory: $0.004445/GB-hr (us-east-1 baseline)
	// Per task (0.5 vCPU + 1GB): 0.5 * 0.04048 + 1 * 0.004445 = $0.02469/hr

	regionPrices := map[string]cogtypes.CostValue{
		"us-east-1":      0.02469,
		"us-east-2":      0.02469,
		"us-west-1":      0.02855,
		"us-west-2":      0.02469,
		"eu-west-1":      0.02697,
		"eu-west-2":      0.02826,
		"eu-west-3":      0.02826,
		"eu-central-1":   0.02826,
		"eu-north-1":     0.02607,
		"ap-southeast-1": 0.02826,
		"ap-southeast-2": 0.02955,
		"ap-northeast-1": 0.02955,
		"ap-northeast-2": 0.02826,
		"ap-south-1":     0.02469,
		"ca-central-1":   0.02697,
		"sa-east-1":      0.03342,
	}

	if price, ok := regionPrices[region]; ok {
		return price
	}
	// Default to us-east-1 pricing
	return 0.02469
}

// RefreshCache forces a refresh of the pricing cache
func (p *AWSProvider) RefreshCache(ctx context.Context) error {
	p.cacheMu.Lock()
	p.ec2Cache = make(map[string]cogtypes.CostValue)
	p.ebsCache = make(map[string]cogtypes.CostValue)
	p.ecsCache = make(map[string]cogtypes.CostValue)
	p.rdsCache = make(map[string]cogtypes.CostValue)
	p.cacheExpiry = time.Time{}
	p.cacheMu.Unlock()
	return nil
}

// fetchEC2Price queries the AWS Price List API for EC2 pricing
func (p *AWSProvider) fetchEC2Price(ctx context.Context, region, instanceType string) (cogtypes.CostValue, error) {
	locationName, ok := regionToLocation[region]
	if !ok {
		return 0, fmt.Errorf("unknown region: %s", region)
	}

	input := &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEC2"),
		Filters: []types.Filter{
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("instanceType"),
				Value: aws.String(instanceType),
			},
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("location"),
				Value: aws.String(locationName),
			},
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("operatingSystem"),
				Value: aws.String("Linux"),
			},
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("tenancy"),
				Value: aws.String("Shared"),
			},
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("preInstalledSw"),
				Value: aws.String("NA"),
			},
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("capacitystatus"),
				Value: aws.String("Used"),
			},
		},
		MaxResults: aws.Int32(1),
	}

	output, err := p.client.GetProducts(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("calling GetProducts for EC2: %w", err)
	}

	if len(output.PriceList) == 0 {
		return 0, fmt.Errorf("no pricing found for EC2 %s in %s", instanceType, region)
	}

	return parsePriceFromProduct(output.PriceList[0])
}

// fetchEBSPrices queries the AWS Price List API for EBS pricing
func (p *AWSProvider) fetchEBSPrices(ctx context.Context, region, volumeType string) (base, iops, throughput cogtypes.CostValue, err error) {
	locationName, ok := regionToLocation[region]
	if !ok {
		return 0, 0, 0, fmt.Errorf("unknown region: %s", region)
	}

	// Fetch base storage price
	input := &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEC2"),
		Filters: []types.Filter{
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("productFamily"),
				Value: aws.String("Storage"),
			},
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("location"),
				Value: aws.String(locationName),
			},
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("volumeApiName"),
				Value: aws.String(volumeType),
			},
		},
		MaxResults: aws.Int32(10),
	}

	output, err := p.client.GetProducts(ctx, input)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("calling GetProducts for EBS: %w", err)
	}

	// Use fallback prices if API doesn't return results
	if len(output.PriceList) == 0 {
		b, i, t := getEBSFallbackPrices(volumeType)
		return b, i, t, nil
	}

	base, err = parsePriceFromProduct(output.PriceList[0])
	if err != nil {
		b, i, t := getEBSFallbackPrices(volumeType)
		return b, i, t, nil
	}

	// For gp3/io1/io2, also get IOPS pricing
	if volumeType == "gp3" || volumeType == "io1" || volumeType == "io2" {
		iops = getEBSIOPSPrice(volumeType)
	}

	// For gp3, also get throughput pricing
	if volumeType == "gp3" {
		throughput = cogtypes.CostValue(0.040) // $0.040 per MiB/s-month above 125
	}

	return base, iops, throughput, nil
}

// getEBSFallbackPrices returns fallback prices for EBS volumes (us-east-1 prices)
func getEBSFallbackPrices(volumeType string) (base, iops, throughput cogtypes.CostValue) {
	switch volumeType {
	case "gp2":
		return 0.10, 0, 0 // $0.10 per GB-month
	case "gp3":
		return 0.08, 0.005, 0.040 // $0.08/GB, $0.005/IOPS, $0.04/MiB/s
	case "io1":
		return 0.125, 0.065, 0 // $0.125/GB, $0.065/IOPS
	case "io2":
		return 0.125, 0.065, 0 // Same as io1 for basic tier
	case "st1":
		return 0.045, 0, 0 // $0.045 per GB-month
	case "sc1":
		return 0.015, 0, 0 // $0.015 per GB-month
	case "standard":
		return 0.05, 0, 0 // $0.05 per GB-month
	default:
		return 0.10, 0, 0 // Default to gp2 pricing
	}
}

// getEBSIOPSPrice returns the per-IOPS-month price
func getEBSIOPSPrice(volumeType string) cogtypes.CostValue {
	switch volumeType {
	case "gp3":
		return 0.005 // $0.005 per IOPS-month above 3000
	case "io1":
		return 0.065 // $0.065 per IOPS-month
	case "io2":
		return 0.065 // $0.065 per IOPS-month for basic tier
	default:
		return 0
	}
}

// fetchRDSPrice queries the AWS Price List API for RDS pricing
func (p *AWSProvider) fetchRDSPrice(ctx context.Context, region, instanceClass, engine string, multiAZ bool) (cogtypes.CostValue, error) {
	locationName, ok := regionToLocation[region]
	if !ok {
		return 0, fmt.Errorf("unknown region: %s", region)
	}

	// Map engine names to pricing API database engine names
	dbEngine := mapRDSEngine(engine)

	deploymentOption := "Single-AZ"
	if multiAZ {
		deploymentOption = "Multi-AZ"
	}

	input := &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonRDS"),
		Filters: []types.Filter{
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("instanceType"),
				Value: aws.String(instanceClass),
			},
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("location"),
				Value: aws.String(locationName),
			},
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("databaseEngine"),
				Value: aws.String(dbEngine),
			},
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("deploymentOption"),
				Value: aws.String(deploymentOption),
			},
		},
		MaxResults: aws.Int32(10),
	}

	output, err := p.client.GetProducts(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("calling GetProducts for RDS: %w", err)
	}

	if len(output.PriceList) == 0 {
		// Return fallback price if not found
		return getRDSFallbackPrice(instanceClass, multiAZ), nil
	}

	return parsePriceFromProduct(output.PriceList[0])
}

// mapRDSEngine maps RDS engine names to pricing API database engine names
func mapRDSEngine(engine string) string {
	engineMap := map[string]string{
		"mysql":            "MySQL",
		"postgres":         "PostgreSQL",
		"mariadb":          "MariaDB",
		"oracle-se2":       "Oracle",
		"oracle-ee":        "Oracle",
		"sqlserver-se":     "SQL Server",
		"sqlserver-ee":     "SQL Server",
		"sqlserver-ex":     "SQL Server",
		"sqlserver-web":    "SQL Server",
		"aurora":           "Aurora MySQL",
		"aurora-mysql":     "Aurora MySQL",
		"aurora-postgresql": "Aurora PostgreSQL",
	}

	if mapped, ok := engineMap[engine]; ok {
		return mapped
	}
	return engine
}

// getRDSFallbackPrice returns a fallback price for RDS instances
func getRDSFallbackPrice(instanceClass string, multiAZ bool) cogtypes.CostValue {
	// Very rough estimates based on db.t3.medium in us-east-1
	basePrice := cogtypes.CostValue(0.068)

	// Scale based on instance size (very rough)
	switch {
	case len(instanceClass) > 7 && instanceClass[7:] == "micro":
		basePrice = 0.017
	case len(instanceClass) > 7 && instanceClass[7:] == "small":
		basePrice = 0.034
	case len(instanceClass) > 7 && instanceClass[7:] == "medium":
		basePrice = 0.068
	case len(instanceClass) > 7 && instanceClass[7:] == "large":
		basePrice = 0.136
	case len(instanceClass) > 7 && instanceClass[7:] == "xlarge":
		basePrice = 0.272
	}

	if multiAZ {
		basePrice *= 2
	}

	return basePrice
}

// parsePriceFromProduct extracts the hourly on-demand price from the AWS pricing JSON
func parsePriceFromProduct(priceListJSON string) (cogtypes.CostValue, error) {
	var product map[string]interface{}
	if err := json.Unmarshal([]byte(priceListJSON), &product); err != nil {
		return 0, fmt.Errorf("parsing price list JSON: %w", err)
	}

	terms, ok := product["terms"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("no terms in price list")
	}

	onDemand, ok := terms["OnDemand"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("no OnDemand terms in price list")
	}

	// Get the first (and usually only) offer
	for _, offerVal := range onDemand {
		offer, ok := offerVal.(map[string]interface{})
		if !ok {
			continue
		}

		priceDimensions, ok := offer["priceDimensions"].(map[string]interface{})
		if !ok {
			continue
		}

		// Get the first price dimension
		for _, dimVal := range priceDimensions {
			dim, ok := dimVal.(map[string]interface{})
			if !ok {
				continue
			}

			pricePerUnit, ok := dim["pricePerUnit"].(map[string]interface{})
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
