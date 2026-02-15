# Build Stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the binary
# ensure CGO is disabled for static binary with modernc.org/sqlite
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/server

# Runtime Stage
FROM alpine:latest

WORKDIR /app

# Install system dependencies if needed (e.g. for timezones)
RUN apk add --no-cache tzdata

COPY --from=builder /app/server .
COPY config.yaml . 
# We don't copy finance.db because it should be in a volume

EXPOSE 8080

CMD ["./server"]
