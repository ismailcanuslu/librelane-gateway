package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

        "github.com/gofiber/fiber/v2/middleware/cors"

	"github.com/ismailcanuslu/ayws-gateway/config"
	"github.com/ismailcanuslu/ayws-gateway/internal/middleware"
	"github.com/ismailcanuslu/ayws-gateway/internal/router"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// ── Logger ───────────────────────────────────────────────────────────────
	zerolog.TimeFieldFormat = time.RFC3339
	if os.Getenv("ENV") != "production" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}

	// ── Config ───────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("config yüklenemedi")
	}

	log.Info().
		Int("port", cfg.Server.Port).
		Int("routes", len(cfg.Routes)).
		Msg("ayws-gateway başlatılıyor")

	// ── Middleware başlangıç ──────────────────────────────────────────────────
	middleware.InitAuth(&cfg.Keycloak)
	middleware.InitRateLimit(&cfg.RateLimit)

	// ── Fiber app ────────────────────────────────────────────────────────────
	app := router.Setup(cfg)
        app.Use(cors.New(cors.Config{
        AllowOrigins:     "https://ayws.anadoluyazilim.com.tr",
        AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
        AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
        AllowCredentials: true,
        }))

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		log.Info().Str("addr", addr).Msg("dinleniyor")
		if err := app.Listen(addr); err != nil {
			log.Fatal().Err(err).Msg("sunucu başlatılamadı")
		}
	}()

	<-quit
	log.Info().Msg("kapatma sinyali alındı, bağlantılar bekleniyor...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = ctx

	if err := app.Shutdown(); err != nil {
		log.Error().Err(err).Msg("graceful shutdown başarısız")
	}
	log.Info().Msg("sunucu kapatıldı")
}
