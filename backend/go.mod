module github.com/johnjeffers/awscogs/backend

go 1.26

require (
	github.com/aws/aws-sdk-go-v2 v1.42.0
	github.com/aws/aws-sdk-go-v2/config v1.32.25
	github.com/aws/aws-sdk-go-v2/credentials v1.19.24
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.59.0
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.307.1
	github.com/aws/aws-sdk-go-v2/service/ecs v1.85.0
	github.com/aws/aws-sdk-go-v2/service/eks v1.87.0
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.34.6
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.55.4
	github.com/aws/aws-sdk-go-v2/service/iam v1.54.5
	github.com/aws/aws-sdk-go-v2/service/lambda v1.93.0
	github.com/aws/aws-sdk-go-v2/service/organizations v1.51.10
	github.com/aws/aws-sdk-go-v2/service/pricing v1.42.7
	github.com/aws/aws-sdk-go-v2/service/rds v1.119.3
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.42.3
	github.com/aws/aws-sdk-go-v2/service/sts v1.43.3
	github.com/go-chi/chi/v5 v5.3.0
	github.com/go-chi/cors v1.2.2
	golang.org/x/sync v0.21.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.13 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.30 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.29 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.2.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.31.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.36.6 // indirect
	github.com/aws/smithy-go v1.27.2 // indirect
)
