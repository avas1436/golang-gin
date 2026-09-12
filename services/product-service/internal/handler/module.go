// services/product-service/internal/handler/module.go

package handler

import "go.uber.org/fx"

// PublicMethods و RateLimitRules نیازی به Provide شدن ندارند: توابع
// خالصی هستند که هیچ dependency ندارند و مستقیماً در internal/server
// صدا زده می‌شوند؛ فقط چیزی که واقعاً یک شیء با dependency است
// (GRPCServer) اینجا Provide می‌شود
var Module = fx.Module(
	"handler",
	fx.Provide(
		NewGRPCServer,
	),
)
