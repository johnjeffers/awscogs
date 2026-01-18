module github.com/johnjeffers/infra-utilities/awscogs/backend

go 1.25

require (
	github.com/aws/aws-sdk-go-v2 v1.41.1
	github.com/aws/aws-sdk-go-v2/config v1.32.7
	github.com/aws/aws-sdk-go-v2/credentials v1.19.7
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.279.2
	github.com/aws/aws-sdk-go-v2/service/ecs v1.71.0
	github.com/aws/aws-sdk-go-v2/service/eks v1.77.0
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.33.19
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.54.6
	github.com/aws/aws-sdk-go-v2/service/iam v1.53.2
	github.com/aws/aws-sdk-go-v2/service/organizations v1.50.1
	github.com/aws/aws-sdk-go-v2/service/pricing v1.40.11
	github.com/aws/aws-sdk-go-v2/service/rds v1.114.0
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.6
	github.com/go-chi/chi/v5 v5.2.4
	github.com/go-chi/cors v1.2.2
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.41.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.13 // indirect
	github.com/aws/smithy-go v1.24.0 // indirect
)
