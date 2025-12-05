FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

RUN rm -rf /tmp/metachat-proto && \
    git clone --depth 1 --branch v0.2.1 https://github.com/kegazani/metachat-proto.git /tmp/metachat-proto || \
    git clone --depth 1 https://github.com/kegazani/metachat-proto.git /tmp/metachat-proto

COPY metachat-api-gateway/go.mod metachat-api-gateway/go.sum* ./
RUN rm -f go.sum
RUN go mod edit -require github.com/kegazani/metachat-proto@v0.2.1
RUN go mod edit -replace github.com/kegazani/metachat-proto=/tmp/metachat-proto

COPY metachat-api-gateway/ .

RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-gateway ./cmd/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/api-gateway .

# Copy configuration files
COPY --from=builder /app/config ./config

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./api-gateway"]