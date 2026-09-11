// services/product-service/internal/platform/module.go

package platform

import (
	"go.uber.org/fx"
)

// ارائه خروجی‌های پلتفرم به Fx
var Module = fx.Module(
	"platform",
	fx.Provide(
		NewPostgresDB,
		NewRedisClient,
		NewTokenManager,
		NewRateLimiter,
	),
)
