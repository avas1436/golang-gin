// services/payment-service/internal/service/module.go

package service

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"payment-service/internal/repository"
)

// NewPaymentService تمام dependencyهای مورد نیاز PaymentService را
// از Fx دریافت می‌کند و یک PaymentService می‌سازد.
//
// خود PaymentService مسئول ساختن dependencyهایش نیست.
// این کار توسط Fx و Moduleهای مربوط به هر package انجام می‌شود.
func NewPaymentService(
	pool *pgxpool.Pool,
	paymentRepo repository.PaymentRepository,
	publisher EventPublisher,
) *PaymentService {

	return &PaymentService{
		pool:        pool,
		paymentRepo: paymentRepo,
		publisher:   publisher,
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
