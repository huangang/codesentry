# Frontend builder
FROM node:22-alpine AS frontend-builder

WORKDIR /app

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Backend builder
FROM golang:1.24-alpine AS backend-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy backend source code
COPY backend/ ./

# Copy frontend build to static folder (before go build, so go:embed works)
COPY --from=frontend-builder /app/dist ./cmd/server/static/

# Build with embedded frontend (stripped binary for smaller size)
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o codesentry ./cmd/server/

# Final stage
FROM alpine:3.21

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy backend binary (with embedded frontend)
COPY --from=backend-builder /app/codesentry ./

# Create data directory with correct ownership
RUN mkdir -p /app/data && chown -R appuser:appgroup /app/data

# Switch to non-root user
USER appuser

# Environment
ENV SERVER_PORT=8080
ENV SERVER_MODE=release
ENV DB_DRIVER=sqlite
ENV DB_DSN=/app/data/codesentry.db

EXPOSE 8080

CMD ["./codesentry"]
