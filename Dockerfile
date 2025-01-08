FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/
COPY config.production.yaml config.yaml

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o memo-bot ./cmd/bot

# Final stage
FROM alpine:latest

# Install dependencies for wait-for script
RUN apk add --no-cache postgresql-client

WORKDIR /app

# Copy wait-for script
COPY --from=builder /app/memo-bot .
COPY config.production.yaml config.yaml

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