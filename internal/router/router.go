package router

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/ismailcanuslu/ayws-gateway/config"
	"github.com/ismailcanuslu/ayws-gateway/internal/handler"
	"github.com/ismailcanuslu/ayws-gateway/internal/middleware"
	"github.com/ismailcanuslu/ayws-gateway/internal/proxy"
)

// Setup, Fiber uygulamasını yapılandırır ve tüm route'ları kaydeder.
func Setup(cfg *config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		BodyLimit:    cfg.Server.BodyLimit * 1024 * 1024, // MB → byte
		// Streaming response'lar için header'ları erkenden gönder.
		StreamRequestBody:  true,
		DisableKeepalive:   false,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	// ── Global middleware ────────────────────────────────────────────────────
	app.Use(middleware.Recover())
	app.Use(middleware.Logger())
	app.Use(middleware.CORS())
	app.Use(middleware.RateLimit())

	app.Use(middleware.Auth())

	// ── Health (gateway kendi endpoint'i) ────────────────────────────────────
	app.Get("/health", handler.Health)

	// ── Prometheus metrics ────────────────────────────────────────────────────
	app.Get("/metrics", handler.Metrics())

	// ── Reverse Proxy ─────────────────────────────────────────────────────────
	rp := proxy.New(cfg.Routes)
	streamRP := proxy.NewStreamingProxy(cfg.Routes)
	streamingPatterns := cfg.StreamingPaths

	// Dispatcher: streaming path'i net/http reverse proxy'ye, geri kalanını
	// mevcut fasthttp proxy'ye gönderir.
	dispatcher := func(c *fiber.Ctx) error {
		if proxy.IsStreamingPath(c.Path(), streamingPatterns) {
			return streamRP.Handler(c)
		}
		return rp.Handler(c)
	}

	for _, route := range cfg.Routes {
		prefix := route.Prefix
		// Hem /api/auth hem /api/auth/* yakala
		app.All(prefix, dispatcher)
		app.All(withWildcard(prefix), dispatcher)
	}

	return app
}

// withWildcard, "/api/auth" → "/api/auth/*"
func withWildcard(prefix string) string {
	if strings.HasSuffix(prefix, "/") {
		return prefix + "*"
	}
	return prefix + "/*"
}
