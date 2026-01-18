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

	// GetEKSPrice returns the hourly price for an EKS cluster control plane
	GetEKSPrice(ctx context.Context, region string) (types.CostValue, error)

	// GetELBPrice returns the hourly price for a load balancer by type
	GetELBPrice(ctx context.Context, region, lbType string) (types.CostValue, error)

	// GetNATGatewayPrice returns the hourly price for a NAT Gateway
	GetNATGatewayPrice(ctx context.Context, region string) (types.CostValue, error)

	// GetElasticIPPrice returns the hourly price for an Elastic IP
	// isAssociated indicates if the EIP is attached to a running instance
	GetElasticIPPrice(ctx context.Context, region string, isAssociated bool) (types.CostValue, error)

	// GetSecretPrice returns the hourly price for a Secrets Manager secret
	GetSecretPrice(ctx context.Context, region string) (types.CostValue, error)

	// RefreshCache forces a refresh of the pricing cache
	RefreshCache(ctx context.Context) error
}
