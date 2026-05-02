package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/ismailcanuslu/ayws-gateway/config"
)

// ipEntry, bir IP için istek sayısını ve pencere başlangıcını tutar.
type ipEntry struct {
	count    int
	windowAt time.Time
}

// rateLimiter, sliding window rate limiter.
type rateLimiter struct {
	mu         sync.Mutex
	store      map[string]*ipEntry
	maxReqs    int
	windowSize time.Duration
}

var rl *rateLimiter

// InitRateLimit, rate limiter'ı başlatır.
func InitRateLimit(cfg *config.RateLimitConfig) {
	rl = &rateLimiter{
		store:      make(map[string]*ipEntry),
		maxReqs:    cfg.RequestsPerSecond,
		windowSize: time.Duration(cfg.Expiration) * time.Second,
	}

	// Eski kayıtları periyodik temizle
	go rl.cleanup()
}

// RateLimit, IP başına istek sınırlaması uygular.
func RateLimit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := c.IP()

		rl.mu.Lock()
		entry, ok := rl.store[ip]
		now := time.Now()

		if !ok || now.Sub(entry.windowAt) > rl.windowSize {
			rl.store[ip] = &ipEntry{count: 1, windowAt: now}
			rl.mu.Unlock()
			return c.Next()
		}

		entry.count++
		if entry.count > rl.maxReqs {
			rl.mu.Unlock()
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Çok fazla istek. Lütfen bekleyiniz.",
			})
		}
		rl.mu.Unlock()
		return c.Next()
	}
}

// cleanup, her dakika süresi dolmuş kayıtları temizler.
func (r *rateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		r.mu.Lock()
		now := time.Now()
		for ip, e := range r.store {
			if now.Sub(e.windowAt) > r.windowSize*2 {
				delete(r.store, ip)
			}
		}
		r.mu.Unlock()
	}
}
