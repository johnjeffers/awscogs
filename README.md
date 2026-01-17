# awscogs

A simple AWS COGS dashboard.

⚠️ This is a WORK IN PROGRESS. At the moment, it only gets data for a handful of AWS resource types.

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
