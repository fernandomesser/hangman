# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o hangman .

# Runtime stage
FROM alpine:latest

# Install dependencies for SQLite
RUN apk add --no-cache libc6-compat

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/hangman .

# Copy static files and templates
COPY static ./static
COPY templates ./templates

# Create directory for SQLite database
RUN mkdir -p /tmp

# Expose port (Vercel will override this)
EXPOSE 8080

# Run the application
CMD ["./hangman"]