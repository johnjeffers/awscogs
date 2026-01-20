# awscogs

A simple AWS COGS dashboard.

By "simple", I mean this: It retrieves the resources it can find with the credentials you give it, and shows you how much they are supposed to cost according to pricing data retrieved from the AWS API.

* It does not try to read your CUR.
* It does not know how long you have been running the things that it finds.
* It does not factor in reservations, savings plans, or any other preferred pricing.

It just figures out what you are running, and then tells you what the pricing API says those things are supposed to cost.

In other words, this is a blunt instrument that falls into the "slightly better than nothing" category.

It looks pretty, though.

## Supported Resource Types

These AWS resource types are supported:

* EBS volumes
* EC2 instances
* ECS clusters
* EKS clusters
* Elastic IPs
* Public IPv4 addresses
* Load Balancers
* NAT Gateways
* RDS instances
* AWS Secrets Manager secrets

## Localdev

Currently, the only way to use the app is to clone this repo and run it locally.

### Prerequisites

- Go 1.25
- Node 25
- Valid AWS credentials

### Running the App

Set `AWS_PROFILE` before starting the app.

```sh
export AWS_PROFILE=profile_name
```

Start the app:

```sh
make install && make dev
```

Open http://localhost:3000
