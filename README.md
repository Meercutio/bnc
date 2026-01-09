# Bulls & Cows (4-digit simultaneous rounds) — backend skeleton

This is **Iteration 0**: repository skeleton + local infrastructure + CI.

## Stack
- Go
- Postgres
- Redis
- Kafka (KRaft)
- gRPC (internal)
- HTTP (Gateway + health/readiness for every service)

## Quick start
```bash
make up
make logs
make down
```

## Dev commands
```bash
make test
make lint      # requires golangci-lint installed locally
make gen       # protobuf generation via dockerized buf
```

## Services (currently: only health endpoints + empty gRPC)
- gateway: http :8080
- auth:    grpc :50051, http :8001
- matchmaking: grpc :50052, http :8002
- game:    grpc :50053, http :8003
- stats:   grpc :50054, http :8004
