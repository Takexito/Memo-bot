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

# Install dependencies for wait-for script
RUN apk add --no-cache postgresql-client

WORKDIR /app

# Copy binary and config
COPY --from=builder /app/memo-bot .
COPY --from=builder /app/config.production.yaml config.yaml

# Create an entrypoint script
RUN echo '#!/bin/sh\n\
while ! pg_isready -h $PGHOST -p $PGPORT -U $PGUSER; do\n\
  echo "Waiting for database...";\n\
  sleep 2;\n\
done;\n\
\n\
./memo-bot' > /app/entrypoint.sh && chmod +x /app/entrypoint.sh

# Set the entrypoint
ENTRYPOINT ["/app/entrypoint.sh"] 