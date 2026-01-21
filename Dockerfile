# Build frontend
FROM node:25-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Build backend
FROM golang:1.25-alpine AS backend-builder
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown
WORKDIR /app
COPY backend/ ./backend/
COPY --from=frontend-builder /app/frontend/dist ./backend/internal/api/dist
WORKDIR /app/backend
RUN go build -ldflags="-X github.com/johnjeffers/awscogs/backend/internal/version.Version=${VERSION} -X github.com/johnjeffers/awscogs/backend/internal/version.GitCommit=${GIT_COMMIT} -X github.com/johnjeffers/awscogs/backend/internal/version.BuildTime=${BUILD_TIME}" -o /awscogs ./cmd/awscogs

# Runtime image
FROM alpine:3
RUN apk add --no-cache ca-certificates && \
    addgroup -g 1000 awscogs && \
    adduser -u 1000 -G awscogs -D awscogs
COPY --from=backend-builder /awscogs /usr/local/bin/awscogs
USER awscogs
EXPOSE 8080
ENTRYPOINT ["awscogs"]
