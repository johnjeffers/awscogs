# awscogs

A simple AWS COGS dashboard.

By "simple", I mean this: It retrieves the resources it can find with the credentials you give it, and shows you how much they are supposed to cost according to pricing data retrieved from the AWS API.

- It does not try to read your CUR.
- It does not know how long you have been running the things that it finds.
- It does not factor in reservations, savings plans, or any other preferred pricing.

It just figures out what you are running, and then tells you what the pricing API says those things are supposed to cost.

In other words, this is a blunt instrument that falls into the "slightly better than nothing" category.

It looks pretty, though.

## Supported Resource Types

These AWS resource types are supported:

- EBS volumes
- EC2 instances
- ECS clusters
- EKS clusters
- Elastic IPs
- Public IPv4 addresses
- Load Balancers
- NAT Gateways
- RDS instances
- AWS Secrets Manager secrets

## Localdev

Currently, the only way to use the app is to clone this repo and run it locally.

### Prerequisites

- Go 1.25
- Node 25
- Valid AWS credentials

### Run in dev mode

Run in dev mode when you're working on the app.

Set `AWS_PROFILE` before starting the app.

```sh
export AWS_PROFILE=profile_name
```

Start the app:

```sh
make install && make dev
```

Running in dev mode exposes the Vite dev server on port 3000.

Open http://localhost:3000

### Build and run a binary

When you're ready to build a binary, run `make build`

Run the binary:

```sh
AWS_PROFILE=profile_name backend/bin/awscogs
```

The built app listens on port 8080.

Open http://localhost:8080

### Environment Variables

| Variable                          | Description                                                    | Default                         |
| --------------------------------- | -------------------------------------------------------------- | ------------------------------- |
| `AWSCOGS_PORT`                    | HTTP server port                                               | `8080`                          |
| `AWSCOGS_LOG_LEVEL`               | Log level (`debug`, `info`, `warn`, `error`)                   | `info`                          |
| `AWSCOGS_DISCOVER_ACCOUNTS`       | Auto-discover accounts from AWS Organizations (`true`/`false`) | `true`                          |
| `AWSCOGS_DISCOVER_REGIONS`        | Auto-discover enabled AWS regions (`true`/`false`)             | `true`                          |
| `AWSCOGS_REGIONS`                 | Comma-separated AWS regions (disables region auto-discovery)   | -                               |
| `AWSCOGS_ASSUME_ROLE_NAME`        | IAM role name to assume into each account                      | `OrganizationAccountAccessRole` |
| `AWSCOGS_PRICING_REFRESH_MINUTES` | AWS pricing cache refresh interval                             | `60`                            |
| `AWSCOGS_PRICING_RATE_LIMIT`      | Max pricing API calls per second                               | `5`                             |
