package middleware

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/ismailcanuslu/ayws-gateway/config"
)

// jwksCache, realm başına JWKS key set'lerini önbellekler.
type jwksCache struct {
	mu      sync.RWMutex
	store   map[string]*jwksCacheEntry
	ttl     time.Duration
	baseURL string
}

type jwksCacheEntry struct {
	keySet    jose.JSONWebKeySet
	fetchedAt time.Time
}

// claims JWT'den çıkarılan standart + özel claim'leri tutar.
type claims struct {
	jwt.Claims
	Issuer  string `json:"iss"`
	Subject string `json:"sub"`
}

var cache *jwksCache

// InitAuth, auth middleware'ini başlatır (config'den çağrılır).
func InitAuth(cfg *config.KeycloakConfig) {
	cache = &jwksCache{
		store:   make(map[string]*jwksCacheEntry),
		ttl:     time.Duration(cfg.JwksTTL) * time.Second,
		baseURL: cfg.BaseURL,
	}
}

// Auth, JWT doğrulama middleware'i döner.
// Route public: true ise atlanır.
func Auth(publicPrefixes []string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Public route kontrolü
		for _, prefix := range publicPrefixes {
			if strings.HasPrefix(c.Path(), prefix) {
				return c.Next()
			}
		}

		// Authorization header kontrolü
		header := c.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authorization header eksik veya geçersiz",
			})
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")

		// Token'ı parse et (doğrulama hariç) — iss claim'den realm al
		rawToken, err := jwt.ParseSigned(tokenStr, []jose.SignatureAlgorithm{jose.RS256})
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token parse edilemedi",
			})
		}

		var cl claims
		if err := rawToken.UnsafeClaimsWithoutVerification(&cl); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token claim'leri okunamadı",
			})
		}

		// iss: http://localhost:8080/realms/{realm}
		realm, err := extractRealm(cl.Issuer)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token issuer geçersiz",
			})
		}

		// JWKS'ten public key al
		keySet, err := cache.getKeySet(realm)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "JWKS alınamadı: " + err.Error(),
			})
		}

		// Token'ı doğrula
		var verifiedClaims claims
		if err := rawToken.Claims(keySet, &verifiedClaims); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token doğrulanamadı",
			})
		}

		// Expiry kontrolü
		if err := verifiedClaims.ValidateWithLeeway(jwt.Expected{}, 5*time.Second); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token süresi dolmuş",
			})
		}

		// Downstream'e iletilecek bilgileri locals'a yaz
		c.Locals("userId", verifiedClaims.Subject)
		c.Locals("tenantRealm", realm)

		return c.Next()
	}
}

// getKeySet, cache'den JWKS döner; süresi dolmuşsa Keycloak'tan yeniden çeker.
func (c *jwksCache) getKeySet(realm string) (jose.JSONWebKeySet, error) {
	c.mu.RLock()
	entry, ok := c.store[realm]
	c.mu.RUnlock()

	if ok && time.Since(entry.fetchedAt) < c.ttl {
		return entry.keySet, nil
	}

	// Yenile
	url := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs", c.baseURL, realm)
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return jose.JSONWebKeySet{}, fmt.Errorf("JWKS isteği başarısız: %w", err)
	}
	defer resp.Body.Close()

	var keySet jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&keySet); err != nil {
		return jose.JSONWebKeySet{}, fmt.Errorf("JWKS parse hatası: %w", err)
	}

	c.mu.Lock()
	c.store[realm] = &jwksCacheEntry{keySet: keySet, fetchedAt: time.Now()}
	c.mu.Unlock()

	return keySet, nil
}

// extractRealm, "http://localhost:8080/realms/acme-corp" → "acme-corp"
func extractRealm(issuer string) (string, error) {
	idx := strings.Index(issuer, "/realms/")
	if idx == -1 {
		return "", fmt.Errorf("issuer realm içermiyor: %s", issuer)
	}
	realm := issuer[idx+len("/realms/"):]
	realm = strings.TrimSuffix(realm, "/")
	if realm == "" {
		return "", fmt.Errorf("realm boş")
	}
	return realm, nil
}

// publicKeyFromKeySet, imzalama için RSA public key döner (yardımcı — ileride kullanılabilir).
func publicKeyFromKeySet(keySet jose.JSONWebKeySet, kid string) (*rsa.PublicKey, error) {
	keys := keySet.Key(kid)
	if len(keys) == 0 {
		return nil, fmt.Errorf("kid bulunamadı: %s", kid)
	}
	rsaKey, ok := keys[0].Key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("anahtar RSA değil")
	}
	return rsaKey, nil
}
