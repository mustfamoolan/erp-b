# Build Stage
FROM golang:alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application statically
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main main.go

# Production Stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies (SSL certificates, timezone)
RUN apk add --no-cache ca-certificates tzdata

# Copy binary and required runtime assets
COPY --from=builder /app/main .
COPY --from=builder /app/database/migrations ./database/migrations
COPY --from=builder /app/.env.example .env

# Create required directories
RUN mkdir -p storage/logs uploads

# Expose Fiber port
EXPOSE 8080

CMD ["./main"]
