// services/order-service/internal/repository/module.go

package repository

import (
	"pkg/postgres"

	"go.uber.org/fx"
)

// ساخت یک رپوزیتوری سفارشات
func NewOrderRepository(pool postgres.DBTX) OrderRepository {
	return &orderRepository{pool: pool}
}

// ساخت یک رپوزیتوری ایونت
func NewEventRepository(pool postgres.DBTX) EventRepository {
	return &eventRepository{pool: pool}
}

var Module = fx.Module(
	"repository",
	fx.Provide(
		NewOrderRepository,
		NewEventRepository,
		NewSagaRepository,
	),
)
