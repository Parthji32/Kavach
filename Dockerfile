# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /kavach cmd/server/main.go

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /kavach .

# Copy templates and static files
COPY templates/ ./templates/
COPY static/ ./static/
COPY migrations/ ./migrations/

# Create non-root user
RUN adduser -D -u 1000 kavach
USER kavach

EXPOSE 8080

ENV PORT=8080

ENTRYPOINT ["./kavach"]
