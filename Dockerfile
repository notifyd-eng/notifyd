FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -ldflags="-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o notifyd ./cmd/notifyd

FROM alpine:3.19

RUN apk add --no-cache ca-certificates sqlite-libs tzdata && \
    adduser -D -H -s /sbin/nologin notifyd && \
    mkdir -p /var/lib/notifyd /etc/notifyd && \
    chown notifyd:notifyd /var/lib/notifyd

COPY --from=builder /build/notifyd /usr/local/bin/notifyd
COPY config.example.yaml /etc/notifyd/config.yaml

USER notifyd
EXPOSE 8400

HEALTHCHECK --interval=30s --timeout=3s \
    CMD wget -qO- http://localhost:8400/health || exit 1

ENTRYPOINT ["notifyd"]
CMD ["--config", "/etc/notifyd/config.yaml"]
