# Build stage
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates (for fetching dependencies and HTTPS)
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o migro .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests and postgresql-client for goose
RUN apk --no-cache add ca-certificates postgresql-client

# Install goose
RUN wget -O /usr/local/bin/goose https://github.com/pressly/goose/releases/latest/download/goose_linux_x86_64 && \
    chmod +x /usr/local/bin/goose

WORKDIR /workspace

# Copy the binary from builder stage
COPY --from=builder /app/migro /usr/local/bin/migro

# Copy example config
COPY migro.example.yaml /workspace/

# Create directory for migrations
RUN mkdir -p /workspace/db/migrations

# Set default command
ENTRYPOINT ["migro"]
CMD ["--help"] 