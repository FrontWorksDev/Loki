# syntax=docker/dockerfile:1.7

# ===== Build stage =====
FROM golang:1.25.6-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux \
    go build -trimpath -ldflags="-s -w" \
      -o /out/api ./cmd/api

# ===== Runtime stage =====
FROM gcr.io/distroless/base-debian12:nonroot

WORKDIR /app

COPY --from=builder /out/api /app/api
COPY --chown=nonroot:nonroot configs/default.yaml /app/configs/default.yaml

ENV LOKI_API_PORT=8080
EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/api"]
