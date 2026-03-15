FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o crawler-app ./main

FROM alpine:latest

WORKDIR /app

# Add certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/crawler-app .

# Create data directory for persistence
RUN mkdir -p /app/data

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./crawler-app", "--port=8080", "--data-dir=/app/data"]
