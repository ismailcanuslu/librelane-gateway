package handler

import "github.com/gofiber/fiber/v2"

const version = "1.0.0"

// Health, gateway'in kendisinin sağlık durumunu döner.
// Kubernetes liveness/readiness probe olarak kullanılabilir.
func Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"version": version,
		"service": "ayws-gateway",
	})
}
