# ── Stage 1: Builder ───────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Bağımlılıkları önce indir (layer cache için)
COPY go.mod go.sum ./
RUN go mod download

# Kaynak kodu kopyala
COPY . .

# Binary derle — CGO kapalı, statik binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o bin/gateway ./cmd/gateway

# ── Stage 2: Runtime ───────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Builder'dan binary ve config kopyala
COPY --from=builder /app/bin/gateway .
COPY --from=builder /app/config/gateway.yaml ./config/

# Non-root kullanıcı
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

EXPOSE 8000


ENTRYPOINT ["./gateway"]
