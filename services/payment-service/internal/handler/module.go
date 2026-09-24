// services/payment-service/internal/handler/module.go

package handler

import (
	"payment-service/internal/service"

	"go.uber.org/fx"
)

// NewGRPCServer یک نمونه جدید از سرور gRPC برای هندلر پرداخت می‌سازد
func NewGRPCServer(paymentService *service.PaymentService) *GRPCServer {
	return &GRPCServer{
		paymentService: paymentService,
	}
}

// Module مسئول ارائه GRPCServer لایه handler به گراف تزریق وابستگی Fx است
var Module = fx.Module(
	"handler",
	fx.Provide(
		NewGRPCServer,
	),
)
