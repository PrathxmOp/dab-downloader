# Use a multi-stage build for a smaller final image

# Stage 1: Build the Go application
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum and download dependencies
COPY go.mod .
COPY go.sum .
RUN go mod tidy
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the application
# CGO_ENABLED=0 is important for static linking, making the binary self-contained
# -ldflags="-s -w" reduces the binary size by stripping debug information
RUN CGO_ENABLED=0 go build -o dab-downloader -ldflags="-s -w" .

# Stage 2: Create the final lean image
FROM alpine:latest

# Install ffmpeg for audio conversion
RUN apk add --no-cache ffmpeg

# Set working directory
WORKDIR /app

# Copy the built executable from the builder stage
COPY --from=builder /app/dab-downloader .

# Copy version.json from the builder stage
COPY --from=builder /app/version/version.json version/

# Copy example-config.json to be used as a template
COPY config/example-config.json config/

# Expose a volume for persistent data (config and downloads)
VOLUME /app/config /app/music

# Set the entrypoint to run the application
ENTRYPOINT ["./dab-downloader"]
