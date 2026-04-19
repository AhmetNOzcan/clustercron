# ── Build stage ──
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Cache dependency downloads.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build.
COPY . .
RUN CGO_ENABLED=0 go build -o /clustercron ./cmd/clustercron

# ── Runtime stage ──
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /clustercron /usr/local/bin/clustercron

ENTRYPOINT ["clustercron"]