// services/user-service/internal/handler/module.go

package handler

import (
	"user-service/internal/service"

	"go.uber.org/fx"
)

// NewGRPCServer یک نمونه جدید از سرور gRPC برای هندلر کاربر می‌سازد
func NewGRPCServer(userService *service.UserService) *GRPCServer {
	return &GRPCServer{
		userService: userService,
	}
}

// Module مسئول ارائه GRPCServer لایه handler به گراف تزریق وابستگی Fx است
var Module = fx.Module(
	"handler",
	fx.Provide(
		NewGRPCServer,
	),
)
