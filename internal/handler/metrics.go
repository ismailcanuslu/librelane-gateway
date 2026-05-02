package handler

import (
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Prometheus metrics
var (
	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ayws_gateway_requests_total",
		Help: "Toplam işlenen istek sayısı",
	}, []string{"method", "path", "status"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ayws_gateway_request_duration_seconds",
		Help:    "İstek işleme süresi (saniye)",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	activeConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ayws_gateway_active_connections",
		Help: "Anlık aktif bağlantı sayısı",
	})
)

// Metrics, Prometheus /metrics endpoint'ini döner.
func Metrics() fiber.Handler {
	return adaptor.HTTPHandler(promhttp.Handler())
}

// RequestsTotal, istek sayacını döner (logger middleware'i kullanır).
func RequestsTotal() *prometheus.CounterVec { return requestsTotal }

// RequestDuration, süre histogramını döner.
func RequestDuration() *prometheus.HistogramVec { return requestDuration }

// ActiveConnections, aktif bağlantı gauge'unu döner.
func ActiveConnections() prometheus.Gauge { return activeConnections }
