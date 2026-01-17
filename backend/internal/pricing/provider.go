package pricing

import (
	"context"

	"github.com/johnjeffers/infra-utilities/awscogs/backend/internal/types"
)

// Provider retrieves pricing information for AWS resources
type Provider interface {
	// GetEC2Price returns the hourly on-demand price for an EC2 instance type in a region
	GetEC2Price(ctx context.Context, region, instanceType string) (types.CostValue, error)

	// GetEBSPrice returns the hourly price for an EBS volume
	GetEBSPrice(ctx context.Context, region, volumeType string, sizeGiB, iops, throughput int32) (types.CostValue, error)

	// GetRDSPrice returns the hourly on-demand price for an RDS instance
	GetRDSPrice(ctx context.Context, region, instanceClass, engine string, multiAZ bool) (types.CostValue, error)

	// GetECSPrice returns the hourly price for an ECS Fargate service
	GetECSPrice(ctx context.Context, region, launchType string, runningCount int32) (types.CostValue, error)

	// RefreshCache forces a refresh of the pricing cache
	RefreshCache(ctx context.Context) error
}
