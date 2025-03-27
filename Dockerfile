# The ldapenforcer Dockerfile

# Build stage
FROM golang:1.22-alpine AS builder
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o ldapenforcer ./cmd/ldapenforcer

# Final stage
FROM alpine:3.19
RUN true \
    && addgroup -g 1000 ldapenforcer \
    && adduser -D -u 1000 -G ldapenforcer ldapenforcer \
    && mkdir -p /etc/ldapenforcer \
    && chown ldapenforcer:ldapenforcer /etc/ldapenforcer \
    && true
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/ldapenforcer /usr/local/bin/ldapenforcer

WORKDIR /etc/ldapenforcer
USER ldapenforcer

ENTRYPOINT ["/usr/local/bin/ldapenforcer"]
CMD ["--help"]
