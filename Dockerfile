# Build stage
FROM golang:1.22-alpine AS builder

# Install git for fetching dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o epub-generator .

# Final stage
FROM gcr.io/distroless/base-debian11

# Copy the binary from builder
COPY --from=builder /app/epub-generator /epub-generator

# Set the entrypoint
ENTRYPOINT ["/epub-generator"]