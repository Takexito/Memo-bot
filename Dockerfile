FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go.mod first
COPY go.mod ./

# Download dependencies and verify them
RUN go mod download && \
    go mod verify

# Get all the direct and indirect dependencies
RUN go mod tidy && \
    go mod vendor

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /app/memo-bot ./cmd/bot

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/memo-bot .
COPY config.production.yaml ./config.yaml

# Run the application
CMD ["./memo-bot"] 