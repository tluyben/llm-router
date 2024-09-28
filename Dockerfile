# Start from the official Go image
FROM golang:1.19-alpine AS builder

# Install make and git
RUN apk add --no-cache make git

# Set the working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code and Makefile into the container
COPY . .

# Build the application using make
RUN make build

# Start a new stage from scratch
FROM alpine:latest  

# Install ca-certificates
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/router .

# Expose port 80
EXPOSE 80

# Command to run the executable
CMD ["./router", "--nohosts"]