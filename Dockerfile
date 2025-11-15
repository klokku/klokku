# Stage 1: Build the Go application
FROM golang:1.25-alpine AS builder

# Set the working directory in the builder container
WORKDIR /app

# Copy the Go modules manifests and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code and build the application
COPY . .
RUN go build -o klokku .

# Stage 2: Create a minimal runtime image
FROM alpine:latest

LABEL authors="jozala"

# Set the working directory in the runtime container
WORKDIR /app
VOLUME /app/storage

RUN apk add --no-cache tzdata

# Copy the binary from the builder stage
COPY --from=builder /app/klokku .

# Copy SQL migrations
COPY migrations /app/migrations

# Copy the frontend from working dir
COPY frontend /app/frontend

EXPOSE 8181

# Set application default log level to warning
ENV LOG_LEVEL=warn

# Command to run the application
CMD ["./klokku"]
