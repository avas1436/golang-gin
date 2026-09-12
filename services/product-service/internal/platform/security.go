// services/product-service/internal/platform/security.go

package platform

import (
	"pkg/auth"
	"pkg/ratelimit"
	redispkg "pkg/redis"
	"product-service/config"
)

// NewTokenManager برای AuthInterceptor لازم است تا امضای access
// token کاربران را مستقل اعتبارسنجی کند.
func NewTokenManager(cfg *config.Config) auth.TokenManager {
	return auth.NewTokenManager(cfg.JWT.Secret, cfg.JWT.AccessTokenTTL)
}

// NewRateLimiter برای RateLimitInterceptor لازم است
func NewRateLimiter(client *redispkg.Client) ratelimit.Limiter {
	return ratelimit.New(client)
}
