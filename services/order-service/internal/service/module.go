// services/order-service/internal/service/module.go

package service

import (
	"go.uber.org/fx"

	"order-service/internal/client"
	"order-service/internal/repository"
)

// NewOrderService تمام dependencyهای مورد نیاز OrderService را
// از Fx دریافت می‌کند و یک OrderService می‌سازد.
//
// خود OrderService مسئول ساختن dependencyهایش نیست.
// این کار توسط Fx و Moduleهای مربوط به هر package انجام می‌شود.
func NewOrderService(
	orderRepo repository.OrderRepository,
	productClient client.ProductClient,
	publisher EventPublisher,
) *OrderService {

	return &OrderService{
		orderRepo:     orderRepo,
		productClient: productClient,
		publisher:     publisher,
	}
}

// Module مربوط به Service Layer سفارشات است.
//
// این Module فقط dependencyهای مربوط به Service Layer را
// به Fx معرفی می‌کند.
//
// Dependencyهای مورد نیاز:
//
//	OrderRepository  ← repository.Module
//	ProductClient    ← client.Module
//	EventPublisher   ← messaging.Module
//
// و خروجی:
//
//	*OrderService
var Module = fx.Module(
	"service",

	fx.Provide(
		NewOrderService,
	),
)
