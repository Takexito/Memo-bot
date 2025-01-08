FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy only necessary files
COPY go.mod go.sum ./
COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/
COPY config.production.yaml ./

# Download dependencies and build
RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -o memo-bot ./cmd/bot

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy binary and config
COPY --from=builder /app/memo-bot .
COPY --from=builder /app/config.production.yaml config.yaml

# Run the application
CMD ["./memo-bot"] 