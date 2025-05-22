FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o app .

# Create a minimal image for running the application
FROM alpine:3.19

WORKDIR /app

# Install PostgreSQL client for health check
RUN apk add --no-cache postgresql-client

# Copy the binary from the builder stage
COPY --from=builder /app/app .

# Copy the wait-for script
COPY wait-for-postgres.sh .
RUN chmod +x wait-for-postgres.sh

# Create log directory
RUN mkdir -p /app/log

# Expose the application port
EXPOSE 3000

# Wait for PostgreSQL to be ready, then start the application
CMD ["./wait-for-postgres.sh", "postgres", "./app"]
