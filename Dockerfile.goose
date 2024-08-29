# Use the official Golang image as the base image
FROM golang:latest

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files first to take advantage of Docker's caching
COPY go.mod go.sum ./

# Download the dependencies
RUN go mod tidy

# Install goose CLI
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Copy the rest of the application code
COPY . .

# Set the entrypoint to the goose command
ENTRYPOINT ["goose"]
