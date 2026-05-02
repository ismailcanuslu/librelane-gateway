package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Logger her HTTP isteğini tek satırda özetler (başarılı / 4xx / 5xx / proxy hataları).
func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		rid := c.Get("X-Request-Id")
		if rid == "" {
			rid = newRequestID()
		}
		c.Locals("request_id", rid)
		c.Set("X-Request-Id", rid)

		err := c.Next()

		duration := time.Since(start)
		status := c.Response().StatusCode()
		uri := string(c.Request().RequestURI())
		path := c.Path()

		var result string
		switch {
		case status >= 500:
			result = "server_error"
		case status >= 400:
			result = "client_error"
		default:
			result = "ok"
		}

		bytesOut := c.Response().Header.ContentLength()
		if bytesOut < 0 {
			bytesOut = len(c.Response().Body())
		}

		event := log.Info()
		if err != nil {
			event = log.Error().Err(err)
			result = "handler_error"
		} else if status >= 500 {
			event = log.Error()
		} else if status >= 400 {
			event = log.Warn()
		}

		event.
			Str("component", "gateway").
			Str("event", "access").
			Str("request_id", rid).
			Str("result", result).
			Str("method", c.Method()).
			Str("path", path).
			Str("uri", uri).
			Int("status", status).
			Int64("duration_ms", duration.Milliseconds()).
			Str("ip", c.IP()).
			Str("user_agent", c.Get("User-Agent")).
			Str("referer", c.Get("Referer"))

		if cl := c.Request().Header.ContentLength(); cl > 0 {
			event = event.Int("request_content_length", cl)
		}

		if bytesOut > 0 {
			event = event.Int("response_bytes", bytesOut)
		}

		if v := c.Locals("proxy_upstream"); v != nil {
			event = event.Str("upstream", fmt.Sprint(v))
		}
		if v := c.Locals("proxy_target"); v != nil {
			event = event.Str("upstream_target", fmt.Sprint(v))
		}
		if v := c.Locals("gateway_error"); v != nil {
			event = event.Str("error_detail", fmt.Sprint(v))
		}

		event.Msg("http_request")

		return err
	}
}

// LogStartup, config yüklendikten sonra route özetini yazar (config paketinden çağrılabilir).
func LogStartup(port int, routePrefixes []string) {
	log.Info().
		Str("component", "gateway").
		Str("event", "startup").
		Int("port", port).
		Strs("routes", routePrefixes).
		Msg("gateway routes")
}
