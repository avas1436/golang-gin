// services/product-service/internal/handler/module.go

package handler

import (
	"product-service/internal/service"

	"go.uber.org/fx"
)

// یک نمونه جدید از سرور gRPC را می‌سازد
func NewGRPCServer(productService *service.ProductService) *GRPCServer {

	return &GRPCServer{
		productService: productService,
	}

}

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
