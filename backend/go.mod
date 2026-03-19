module github.com/johnjeffers/awscogs/backend

go 1.26

require (
	github.com/aws/aws-sdk-go-v2 v1.41.4
	github.com/aws/aws-sdk-go-v2/config v1.32.12
	github.com/aws/aws-sdk-go-v2/credentials v1.19.12
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.55.2
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.295.0
	github.com/aws/aws-sdk-go-v2/service/ecs v1.74.0
	github.com/aws/aws-sdk-go-v2/service/eks v1.81.1
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.33.22
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.54.9
	github.com/aws/aws-sdk-go-v2/service/iam v1.53.6
	github.com/aws/aws-sdk-go-v2/service/organizations v1.50.5
	github.com/aws/aws-sdk-go-v2/service/pricing v1.40.14
	github.com/aws/aws-sdk-go-v2/service/rds v1.116.3
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.41.4
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.9
	github.com/go-chi/chi/v5 v5.2.5
	github.com/go-chi/cors v1.2.2
	golang.org/x/sync v0.20.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.20 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.20 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.20 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.20 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.17 // indirect
	github.com/aws/smithy-go v1.24.2 // indirect
)
