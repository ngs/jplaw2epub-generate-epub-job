# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files from root
COPY go.mod go.sum ./
RUN go mod download

# Copy all source code
COPY . .

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o generator .

# Final stage
FROM alpine:3.20

# Install required runtime dependencies  
# go-fitz v1.24.15 requires MuPDF 1.24.x
RUN apk --no-cache add ca-certificates libffi mupdf-libs && \
    ln -s /usr/lib/libmupdf.so.* /usr/lib/libmupdf.so && \
    # Set FZ_VERSION based on installed MuPDF version
    echo "export FZ_VERSION=$(ls /usr/lib/libmupdf.so.* | sed 's/.*\.so\.//' | head -1)" >> /etc/profile.d/fz_version.sh

# Set FZ_VERSION environment variable
ENV FZ_VERSION=1.24.2

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/generator .

ENTRYPOINT ["./generator"]

# Default command (can be overridden)
CMD []