package middleware

import "github.com/gofiber/fiber/v2"

// CORS, geliştirme ve prod için uygun CORS header'larını ekler.
// Prod'da AllowOrigins'i spesifik domain'lerle değiştirin.
func CORS() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Request-Id")
		c.Set("Access-Control-Max-Age", "86400")

		// Preflight
		if c.Method() == fiber.MethodOptions {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.Next()
	}
}
