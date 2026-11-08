module github.com/johnjeffers/awscogs/backend

go 1.27.0

require (
	github.com/aws/aws-sdk-go-v2 v1.47.8
	github.com/aws/aws-sdk-go-v2/config v1.33.16
	github.com/aws/aws-sdk-go-v2/credentials v1.20.16
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.75.6
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.344.2
	github.com/aws/aws-sdk-go-v2/service/ecs v1.104.2
	github.com/aws/aws-sdk-go-v2/service/eks v1.105.5
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.41.9
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.63.9
	github.com/aws/aws-sdk-go-v2/service/iam v1.64.9
	github.com/aws/aws-sdk-go-v2/service/lambda v1.114.3
	github.com/aws/aws-sdk-go-v2/service/organizations v1.62.7
	github.com/aws/aws-sdk-go-v2/service/pricing v1.51.5
	github.com/aws/aws-sdk-go-v2/service/rds v1.132.2
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.50.11
	github.com/aws/aws-sdk-go-v2/service/sts v1.51.9
	github.com/go-chi/chi/v5 v5.5.0
	github.com/go-chi/cors v1.2.2
	golang.org/x/sync v0.24.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.26 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.20.8 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.5.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.24 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.14.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.12.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.39.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.44.4 // indirect
	github.com/aws/smithy-go v1.28.8 // indirect
)
