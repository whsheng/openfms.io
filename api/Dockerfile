FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod ./

# Debug: Check go.mod content
RUN echo "=== go.mod content ===" && cat go.mod

# Download dependencies (go.sum will be generated)
RUN go mod tidy || (echo "=== go mod tidy failed ===" && cat go.mod && exit 1)
RUN go mod download || (echo "=== go mod download failed ===" && exit 1)

# Copy source code
COPY . .

# Debug: List files
RUN echo "=== Source files ===" && ls -la && echo "=== cmd/api ===" && ls -la cmd/api/

# Build with verbose output
RUN echo "=== Starting build ===" && \
    CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -o api ./cmd/api || \
    (echo "=== Build failed ===" && ls -la && exit 1)

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/api .

# Expose port
EXPOSE 3000

CMD ["./api"]
