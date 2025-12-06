FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY metachat-api-gateway/go.mod metachat-api-gateway/go.sum* ./
RUN go mod download

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