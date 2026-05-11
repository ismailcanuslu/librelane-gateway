package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/ismailcanuslu/ayws-gateway/config"
	"github.com/rs/zerolog/log"
)

// StreamingProxy, SSE / chunked response gerektiren upstream'ler için
// net/http tabanlı reverse proxy. fasthttp tüm body'yi buffer ettiği için
// SSE bozulur — bu yüzden streaming path'ler bu handler'a yönlendirilir.
type StreamingProxy struct {
	routes []config.RouteConfig
	cache  sync.Map // upstreamURL -> *httputil.ReverseProxy
}

// NewStreamingProxy yeni bir streaming proxy yaratır.
func NewStreamingProxy(routes []config.RouteConfig) *StreamingProxy {
	return &StreamingProxy{routes: routes}
}

// Handler Fiber middleware olarak çalışır.
func (sp *StreamingProxy) Handler(c *fiber.Ctx) error {
	reqPath := c.Path()
	upstream, err := sp.matchUpstream(reqPath)
	if err != nil {
		log.Warn().Str("path", reqPath).Err(err).Msg("streaming proxy: eşleşen route yok")
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "upstream bulunamadı"})
	}

	rp := sp.proxyFor(upstream)
	if rp == nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "geçersiz upstream URL: " + upstream})
	}

	return adaptor.HTTPHandler(rp)(c)
}

// matchUpstream regular proxy ile aynı eşleşme mantığı.
func (sp *StreamingProxy) matchUpstream(p string) (string, error) {
	for _, r := range sp.routes {
		if strings.HasPrefix(p, r.Prefix) {
			return r.Upstream, nil
		}
	}
	return "", &noRouteError{path: p}
}

func (sp *StreamingProxy) proxyFor(upstream string) *httputil.ReverseProxy {
	if cached, ok := sp.cache.Load(upstream); ok {
		return cached.(*httputil.ReverseProxy)
	}
	target, err := url.Parse(upstream)
	if err != nil {
		return nil
	}

	rp := httputil.NewSingleHostReverseProxy(target)

	// SSE için kritik: ResponseWriter'ı her chunk'ta flush et.
	// httputil.ReverseProxy text/event-stream content-type'ını otomatik tanır
	// ama emin olmak için FlushInterval=-1 (her yazımda flush) ayarlıyoruz.
	rp.FlushInterval = -1

	// Buffer kullanımını minimize et — chunked stream için
	rp.BufferPool = nil

	originalDirector := rp.Director
	rp.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("X-Forwarded-Proto", "http")
		req.Host = target.Host
	}

	rp.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		log.Error().Str("upstream", upstream).Str("path", req.URL.Path).Err(err).Msg("streaming proxy: upstream hatası")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"upstream stream hatası: ` + err.Error() + `"}`))
	}

	sp.cache.Store(upstream, rp)
	return rp
}

type noRouteError struct{ path string }

func (e *noRouteError) Error() string { return "eşleşen route yok: " + e.path }

// IsStreamingPath verilen path streaming_paths pattern'lerinden biriyle eşleşir mi?
// Pattern syntax: '*' tek bir path segment'i (örn. /run/*/stream → /run/abc123/stream).
func IsStreamingPath(reqPath string, patterns []string) bool {
	for _, p := range patterns {
		if matchSegmentPattern(p, reqPath) {
			return true
		}
	}
	return false
}

// matchSegmentPattern, '*'i tek segment olarak yorumlar; ileride '**' eklenebilir.
func matchSegmentPattern(pattern, reqPath string) bool {
	pattern = strings.TrimRight(pattern, "/")
	reqPath = strings.TrimRight(reqPath, "/")

	patSegs := strings.Split(strings.TrimPrefix(pattern, "/"), "/")
	pathSegs := strings.Split(strings.TrimPrefix(reqPath, "/"), "/")
	if len(patSegs) != len(pathSegs) {
		return false
	}
	for i := range patSegs {
		if patSegs[i] == "*" {
			continue
		}
		ok, _ := path.Match(patSegs[i], pathSegs[i])
		if !ok {
			return false
		}
	}
	return true
}
