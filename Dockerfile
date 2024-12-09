# Use the official Golang image to build the application
FROM golang:1.23 AS builder

# Set the working directory
WORKDIR /app

# Copy the Go modules manifests
COPY go.mod go.sum ./

# Download Go modules
RUN go mod download

# Copy the source code
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Use a minimal image for the final container
FROM alpine:latest

# Set the working directory
WORKDIR /root/

# Copy the compiled Go binary from the builder stage
COPY --from=builder /app/main .

# Expose the port the application runs on
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
