module github.com/johnjeffers/awscogs/backend

go 1.26

require (
	github.com/aws/aws-sdk-go-v2 v1.43.3
	github.com/aws/aws-sdk-go-v2/config v1.32.34
	github.com/aws/aws-sdk-go-v2/credentials v1.19.33
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.66.1
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.318.0
	github.com/aws/aws-sdk-go-v2/service/ecs v1.89.2
	github.com/aws/aws-sdk-go-v2/service/eks v1.90.2
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.36.2
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.58.3
	github.com/aws/aws-sdk-go-v2/service/iam v1.57.0
	github.com/aws/aws-sdk-go-v2/service/lambda v1.101.0
	github.com/aws/aws-sdk-go-v2/service/organizations v1.53.3
	github.com/aws/aws-sdk-go-v2/service/pricing v1.44.3
	github.com/aws/aws-sdk-go-v2/service/rds v1.124.0
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.44.2
	github.com/aws/aws-sdk-go-v2/service/sts v1.45.3
	github.com/go-chi/chi/v5 v5.3.1
	github.com/go-chi/cors v1.2.2
	golang.org/x/sync v0.22.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.15 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.35 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.34 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.5.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.33.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.38.3 // indirect
	github.com/aws/smithy-go v1.27.6 // indirect
)
