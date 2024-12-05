# Build Stage
FROM golang:1.23-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
RUN go build -o main ./cmd/main.go

# Final Stage
FROM alpine:latest

# Install necessary certificates
RUN apk --no-cache add ca-certificates

# Set the working directory inside the container
WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /app/main .

# Expose ports (optional, for documentation purposes)
# These are the ports your application will use
EXPOSE 8080
EXPOSE 8081
EXPOSE 8082
EXPOSE 5051
EXPOSE 5052
EXPOSE 5053

# Set the entrypoint to the binary
ENTRYPOINT ["./main"]
