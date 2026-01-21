# Frontend builder
FROM node:20-alpine AS frontend-builder

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

# Build with embedded frontend
RUN CGO_ENABLED=1 GOOS=linux go build -o codesentry ./cmd/server/

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy backend binary (with embedded frontend)
COPY --from=backend-builder /app/codesentry ./

# Create data directory
RUN mkdir -p /app/data

# Environment
ENV SERVER_PORT=8080
ENV SERVER_MODE=release
ENV DB_DRIVER=sqlite
ENV DB_DSN=/app/data/codesentry.db

EXPOSE 8080

CMD ["./codesentry"]
