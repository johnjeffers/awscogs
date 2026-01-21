.PHONY: dev backend frontend build clean install vet update docker-build

# Version info (auto-detected from git tags, can be overridden)
VERSION ?= $(shell git describe --tags --match 'v*' --abbrev=0 2>/dev/null || echo "0.0.0-dev")
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Linker flags for version injection
LDFLAGS = -X github.com/johnjeffers/infra-utilities/awscogs/backend/internal/version.Version=$(VERSION) \
          -X github.com/johnjeffers/infra-utilities/awscogs/backend/internal/version.GitCommit=$(GIT_COMMIT) \
          -X github.com/johnjeffers/infra-utilities/awscogs/backend/internal/version.BuildTime=$(BUILD_TIME)

# Run both backend and frontend for local development
dev:
	@trap 'kill 0' EXIT; \
	$(MAKE) backend & \
	$(MAKE) frontend & \
	wait

# Run the Go backend
backend:
	cd backend && go run ./cmd/awscogs

# Run the React frontend dev server
frontend:
	cd frontend && npm run dev

# Install dependencies
install:
	cd frontend && npm install
	cd backend && go mod download

# Update dependencies to latest versions
update:
	cd backend && go get -u ./... && go mod tidy
	cd frontend && npx npm-check-updates -u && npm install

# Build for production (single binary with embedded frontend)
build:
	cd frontend && npm run build
	rm -rf backend/internal/api/dist
	cp -r frontend/dist backend/internal/api/dist
	cd backend && go build -ldflags='$(LDFLAGS)' -o bin/awscogs ./cmd/awscogs

# Run go vet and staticcheck on backend
vet:
	cd backend && go vet ./...
	cd backend && staticcheck ./...

# Clean build artifacts
clean:
	rm -rf backend/bin
	rm -rf frontend/dist
	rm -rf backend/internal/api/dist
	mkdir -p backend/internal/api/dist
	touch backend/internal/api/dist/.gitkeep

# Build Docker image
docker-build:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t awscogs:$(VERSION) \
		-t awscogs:latest \
		.
