# ayws-gateway

> **Ay.Commerce — Yüksek Performanslı Go API Gateway**
> Tüm microservice'lere tek giriş noktası. JWT doğrulama, rate limiting, reverse proxy.

## Mimari

```
İnternet
    │
    ▼
ayws-gateway :8000   (Go / Fiber v2 — ~36k RPS)
    │
    ├── POST /api/auth/**      → security-service :5001  [PUBLIC]
    ├── /api/tenants/**        → security-service :5001  [KORUNAN]
    └── GET  /health           → gateway'in kendisi
```

**JWT Akışı:**
```
Client              Gateway                    Upstream
  │── Bearer token ──▶│                           │
  │                   │─ JWKS doğrula (Keycloak) ─▶│
  │                   │─ X-User-Id header ekle ───▶│
  │◀── yanıt ─────────│◀── yanıt ─────────────────│
```

Token `iss` claim'inden realm dinamik okunur → **multi-tenant** destekli.

## Proje Yapısı

```
ayws-gateway/
├── cmd/gateway/main.go          # Giriş noktası — graceful shutdown
├── config/
│   ├── config.go                # Viper config loader (YAML + env)
│   └── gateway.yaml             # Route tablosu, Keycloak, rate limit
├── internal/
│   ├── handler/health.go        # GET /health → {"status":"ok"}
│   ├── middleware/
│   │   ├── auth.go              # Keycloak JWKS ile JWT doğrulama
│   │   ├── cors.go              # CORS headers
│   │   ├── logger.go            # zerolog JSON loglama
│   │   ├── ratelimit.go         # IP başına sliding window
│   │   └── recover.go           # Panic recovery
│   ├── proxy/
│   │   ├── proxy.go             # fasthttp reverse proxy
│   │   └── balancer.go          # Round-robin load balancer
│   └── router/router.go         # Fiber app + middleware zinciri
├── Dockerfile                   # Multi-stage (builder + alpine ~15MB)
└── Makefile
```

## Hızlı Başlangıç

### Gereksinimler
- [Go 1.23+](https://go.dev/dl/)
- Docker & Docker Compose (Keycloak için)
- Çalışan `ayws-security-service`

### 1. Bağımlılıkları İndir

```bash
cd ayws-gateway
go mod tidy
```

### 2. Çalıştır

```bash
make run
# veya
go run cmd/gateway/main.go
```

> Varsayılan port: **8000**

### 3. Docker ile Çalıştır

```bash
make docker-build
make docker-run
```

## Konfigürasyon

`config/gateway.yaml` dosyasını düzenleyin:

```yaml
server:
  port: 8000

keycloak:
  base_url: http://localhost:8080
  jwks_ttl: 300          # JWKS önbellek süresi (saniye)

rate_limit:
  requests_per_second: 100
  expiration: 60

routes:
  - prefix: /api/auth
    upstream: http://localhost:5001
    public: true          # JWT gerekmez
  - prefix: /api/tenants
    upstream: http://localhost:5001
    public: false         # JWT zorunlu
```

**Env override:** `GATEWAY_SERVER_PORT=9000` gibi `GATEWAY_` önekiyle tüm değerler override edilebilir.

## Middleware Zinciri

```
Request
  └─▶ Recover (panic)
        └─▶ Logger (zerolog)
              └─▶ CORS
                    └─▶ RateLimit (IP / sliding window)
                          └─▶ Auth (JWKS / public route atla)
                                └─▶ ReverseProxy → upstream
```

## API

| Endpoint | Auth | Açıklama |
|---|---|---|
| `GET /health` | ✅ Herkese açık | Gateway durumu |
| `POST /api/auth/register` | ✅ Public | Yeni tenant + owner kaydı |
| `ANY /api/tenants/**` | 🔒 JWT gerekli | Tenant yönetimi |

## Yeni Upstream Ekleme

`config/gateway.yaml`'a satır eklemek yeterli:

```yaml
routes:
  - prefix: /api/products
    upstream: http://localhost:5002
    public: false
```

Kod değişikliği gerekmez.

## Geliştirme Komutları

```bash
make build      # Binary derle → bin/gateway
make run        # Geliştirme modunda çalıştır
make tidy       # go mod tidy
make test       # Testleri çalıştır
make lint       # gofmt + go vet
make docker-build
```

## Performans

| Metrik | Değer |
|---|---|
| Framework | Fiber v2 (fasthttp) |
| Proxy | fasthttp.Client (512 conn/host pool) |
| Bağlantı havuzu | ✅ Aktif |
| JWKS önbellek | ✅ TTL tabanlı (realm başına) |
| Rate limiter | IP başına sliding window |
| Docker image | ~15 MB (multi-stage alpine) |
