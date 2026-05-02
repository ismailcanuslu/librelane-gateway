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
				log.Error().Interface("panic", r).Msg("unhandled panic")
				err = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Beklenmeyen bir hata oluştu",
				})
			}
		}()
		return c.Next()
	}
}
