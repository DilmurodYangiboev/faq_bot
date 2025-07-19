# Build stage
FROM golang:1.24-alpine AS builder

# Set necessary environment variables
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Move to working directory /build
WORKDIR /build

# Copy and download dependency using go mod
COPY go.mod go.sum ./
RUN go mod download

# Copy the code into the container
COPY . .

# Build the application
RUN go build -a -installsuffix cgo -o faq_bot .

# Production stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create app directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /build/faq_bot .

# Create non-root user
RUN adduser -D -s /bin/sh appuser
USER appuser

# Expose port (optional, as Telegram bots don't need exposed ports)
# EXPOSE 8080

# Command to run
CMD ["./faq_bot"]