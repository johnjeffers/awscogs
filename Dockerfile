# Build frontend
FROM node:25-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Build backend
FROM golang:1.25-alpine AS backend-builder
WORKDIR /app
COPY backend/ ./backend/
COPY --from=frontend-builder /app/frontend/dist ./backend/internal/api/dist
WORKDIR /app/backend
RUN go build -o /awscogs ./cmd/awscogs

# Runtime image
FROM alpine:3
RUN apk add --no-cache ca-certificates
COPY --from=backend-builder /awscogs /usr/local/bin/awscogs
EXPOSE 8080
ENTRYPOINT ["awscogs"]
