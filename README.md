# notifyd

A lightweight, self-hosted notification delivery daemon. Route messages across email, Slack, webhooks, and SMS with retry logic, rate limiting, and delivery guarantees.

## Features

- Multi-channel delivery (email, Slack, webhooks, SMS)
- Exponential backoff retry with configurable limits
- SQLite storage with delivery audit log
- API key authentication
- Docker-ready with health checks
- Structured logging (zerolog)

## Quick Start

```bash
# Build
go build -o notifyd ./cmd/notifyd

# Run with defaults
./notifyd

# Run with config
./notifyd --config config.example.yaml

# Run with Docker
docker build -t notifyd .
docker run -p 8400:8400 -v notifyd-data:/var/lib/notifyd notifyd
```

## API

### Send a notification

```bash
curl -X POST http://localhost:8400/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "webhook",
    "recipient": "https://example.com/hook",
    "subject": "Deploy complete",
    "body": "v1.2.3 deployed to production",
    "priority": 1
  }'
```

### Check status

```bash
curl http://localhost:8400/api/v1/notifications/{id}
```

### List notifications

```bash
curl "http://localhost:8400/api/v1/notifications?status=pending&limit=10"
```

### Stats

```bash
curl http://localhost:8400/api/v1/stats
```

## Configuration

See [config.example.yaml](config.example.yaml) for all options. Environment variables are supported with the `NOTIFYD_` prefix:

```bash
NOTIFYD_SERVER_LISTEN=:9000
NOTIFYD_SERVER_API_KEY=secret
NOTIFYD_STORE_DSN=/tmp/notifyd.db
```

## License

MIT
