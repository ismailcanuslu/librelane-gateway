package proxy

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/ismailcanuslu/ayws-gateway/config"
	"github.com/valyala/fasthttp"
)

// ReverseProxy, gelen isteği upstream servise iletir ve yanıtı geri döner.
type ReverseProxy struct {
	client *fasthttp.Client
	routes []config.RouteConfig
}

// New, yapılandırılmış bir ReverseProxy oluşturur.
func New(routes []config.RouteConfig) *ReverseProxy {
	return &ReverseProxy{
		routes: routes,
		client: &fasthttp.Client{
			// Bağlantı havuzu — yüksek eşzamanlılık için
			MaxConnsPerHost:     512,
			MaxIdleConnDuration: 10 * 1e9, // 10s
			ReadTimeout:         30 * 1e9, // 30s
			WriteTimeout:        30 * 1e9,
		},
	}
}

// Handler, Fiber middleware'i olarak çalışır.
// Gelen isteği eşleşen route'un upstream'ine iletir.
func (rp *ReverseProxy) Handler(c *fiber.Ctx) error {
	path := c.Path()
	upstream, err := rp.matchUpstream(path)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": "upstream bulunamadı",
		})
	}

	// fasthttp request/response kopyala
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	// Orijinal isteği kopyala
	c.Request().CopyTo(req)

	// Hedef URL'yi ayarla
	targetURL := upstream + string(c.Request().RequestURI())
	req.SetRequestURI(targetURL)
	req.Header.SetHostBytes([]byte(extractHost(upstream)))

	// X-Forwarded-For ekle
	req.Header.Set("X-Forwarded-For", c.IP())
	req.Header.Set("X-Forwarded-Proto", "http")

	// Auth middleware'in eklediği header'ları upstream'e ilet
	if userID := c.Locals("userId"); userID != nil {
		req.Header.Set("X-User-Id", fmt.Sprintf("%v", userID))
	}
	if realm := c.Locals("tenantRealm"); realm != nil {
		req.Header.Set("X-Tenant-Realm", fmt.Sprintf("%v", realm))
	}

	// İsteği gönder
	if err := rp.client.Do(req, resp); err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": "upstream'e bağlanılamadı: " + err.Error(),
		})
	}

	// Yanıtı geri kopyala
	resp.Header.VisitAll(func(key, value []byte) {
		c.Set(string(key), string(value))
	})
	c.Status(resp.StatusCode())
	return c.Send(resp.Body())
}

// matchUpstream, path'e göre upstream adresini döner.
func (rp *ReverseProxy) matchUpstream(path string) (string, error) {
	for _, r := range rp.routes {
		if strings.HasPrefix(path, r.Prefix) {
			return r.Upstream, nil
		}
	}
	return "", fmt.Errorf("eşleşen route yok: %s", path)
}

func extractHost(upstream string) string {
	// "http://localhost:5001" → "localhost:5001"
	parts := strings.SplitN(upstream, "//", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return upstream
}
