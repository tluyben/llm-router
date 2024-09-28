# Start from the official Go image
FROM golang:1.19-alpine AS builder

# Install make, git, and OpenSSL
RUN apk add --no-cache make git openssl

# Set the working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code and Makefile into the container
COPY . .

# Generate SSL certificate
RUN openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes -subj "/CN=localhost"

# Build the application using make
RUN make build

# Start a new stage from scratch
FROM alpine:latest  

# Install ca-certificates and OpenSSL
RUN apk --no-cache add ca-certificates openssl

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/router .

# Copy SSL certificates
COPY --from=builder /app/server.crt /root/server.crt
COPY --from=builder /app/server.key /root/server.key

# Expose ports 80 and 443
EXPOSE 80 443

# Command to run the executable
CMD ["./router", "--nohosts"]