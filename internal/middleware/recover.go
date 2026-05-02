package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// Recover, panic'leri yakalar ve 500 döner.
func Recover() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error().
					Str("component", "gateway").
					Str("event", "panic").
					Str("path", c.Path()).
					Str("method", c.Method()).
					Interface("panic", r).
					Msg("unhandled panic")
				err = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Beklenmeyen bir hata oluştu",
				})
			}
		}()
		return c.Next()
	}
}
