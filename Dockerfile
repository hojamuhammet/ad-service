# Use the latest official Golang image as a build stage
FROM golang:latest AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod tidy

# Copy the source code into the container
COPY . .

# Set environment variables for Linux build and create a statically linked binary
ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

# Build the Go app and output binary to /app/main
RUN go build -o /app/main cmd/main.go

# List files to verify binary exists
RUN ls -l /app

# Start a new stage from scratch
FROM debian:bullseye

# Set the Current Working Directory inside the container
WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/main .
COPY --from=builder /app/config.yaml .

# Ensure the binary has execution permissions
RUN chmod +x ./main

# Verify that the binary is present and has correct permissions
RUN ls -l /root

# Expose port for the HTTP service
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
