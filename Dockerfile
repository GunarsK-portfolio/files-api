# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN go build -o files-api ./cmd/api

# Production stage
FROM alpine:3.22

RUN apk upgrade --no-cache && apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 app && \
    adduser -D -u 1000 -G app app

WORKDIR /app

COPY --from=builder /app/files-api .

# Change ownership to app user
RUN chown -R app:app /app

USER app

EXPOSE 8085

CMD ["./files-api"]
