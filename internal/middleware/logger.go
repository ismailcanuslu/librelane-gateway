package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger, her HTTP isteğini JSON formatında loglar.
func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Sonraki handler'ı çalıştır
		err := c.Next()

		duration := time.Since(start)
		status := c.Response().StatusCode()

		var event *zerolog.Event
		if err != nil || status >= 500 {
			event = log.Error().Err(err)
		} else if status >= 400 {
			event = log.Warn()
		} else {
			event = log.Info()
		}

		event.
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", status).
			Dur("latency", duration).
			Str("ip", c.IP()).
			Str("user_agent", c.Get("User-Agent")).
			Msg("request")

		return err
	}
}
