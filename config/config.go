package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config tüm gateway konfigürasyonunu tutar.
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Keycloak  KeycloakConfig  `mapstructure:"keycloak"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Routes    []RouteConfig   `mapstructure:"routes"`
}

type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	BodyLimit    int           `mapstructure:"body_limit"`
}

type KeycloakConfig struct {
	BaseURL  string `mapstructure:"base_url"`
	JwksTTL int    `mapstructure:"jwks_ttl"` // saniye
}

type RateLimitConfig struct {
	RequestsPerSecond int `mapstructure:"requests_per_second"`
	Burst             int `mapstructure:"burst"`
	Expiration        int `mapstructure:"expiration"` // saniye
}

type RouteConfig struct {
	Prefix   string `mapstructure:"prefix"`
	Upstream string `mapstructure:"upstream"`
	Public   bool   `mapstructure:"public"`
}

// Load, config/gateway.yaml dosyasını (ve env değişkenlerini) okur.
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("gateway")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")

	// Env override: GATEWAY_SERVER_PORT → server.port
	v.SetEnvPrefix("GATEWAY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
