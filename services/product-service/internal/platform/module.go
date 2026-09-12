// services/product-service/internal/platform/module.go

package platform

import (
	"context"

	"go.uber.org/fx"
)

// ارائه خروجی‌های پلتفرم به Fx
var Module = fx.Module(
	"platform",
	fx.Provide(
		context.Background,
		NewPostgresDB,
		NewRedisClient,
		NewTokenManager,
		NewRateLimiter,
	),
)
