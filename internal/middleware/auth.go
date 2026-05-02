package middleware

import "github.com/gofiber/fiber/v2"

// Auth — projede giriş/kayıt yok; JWT veya Keycloak kullanılmıyor. İstekler doğrudan geçer.
func Auth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}
