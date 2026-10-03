// services/payment-service/internal/service/module.go

package service

import (
	"go.uber.org/fx"

	"payment-service/config"
	"payment-service/internal/client"
	"payment-service/internal/repository"
)

// NewPaymentService تمام dependencyهای مورد نیاز PaymentService را
// از Fx دریافت می‌کند و یک PaymentService می‌سازد.
//
// خود PaymentService مسئول ساختن dependencyهایش نیست.
// این کار توسط Fx و Moduleهای مربوط به هر package انجام می‌شود.
func NewPaymentService(
	txManager repository.TxManager,
	paymentRepo repository.PaymentRepository,
	outboxRepo repository.OutboxRepository,
	gateway client.GatewayClient,
	publisher EventPublisher,
	cfg *config.Config,
) *PaymentService {

	return &PaymentService{
		txManager:   txManager,
		paymentRepo: paymentRepo,
		outboxRepo:  outboxRepo,
		gateway:     gateway,
		publisher:   publisher,
		cfg:         cfg,
	}
}

// Module مربوط به Service Layer پرداخت است.
//
// این Module فقط dependencyهای مربوط به Service Layer را
// به Fx معرفی می‌کند.
//
// Dependencyهای مورد نیاز:
//
//	PaymentRepository  ← repository.Module
//	EventPublisher   ← messaging.Module
//
// و خروجی:
//
//	*PaymentService
var Module = fx.Module(
	"service",

	fx.Provide(
		NewPaymentService,
	),
)
