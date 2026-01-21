.PHONY: dev backend frontend build clean install vet update docker-build

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
	cd backend && go build -o bin/awscogs ./cmd/awscogs

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
	docker build -t awscogs .
