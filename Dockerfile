FROM golang:1.21-alpine

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go.mod first
COPY go.mod .

# Initialize modules and download dependencies
RUN go mod download && \
    go mod tidy

# Copy the rest of the source code
COPY . .

# Verify and ensure all dependencies are downloaded
RUN go mod verify && \
    go mod download all

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o memo-bot ./cmd/bot

# Run the application
CMD ["./memo-bot"] 