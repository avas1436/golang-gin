// services/order-service/internal/repository/module.go

package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

// ساخت یک رپوزیتوری سفارشات
func NewOrderRepository(pool *pgxpool.Pool) OrderRepository {
	return &orderRepository{pool: pool}
}

// ساخت یک رپوزیتوری ایونت
func NewEventRepository(pool *pgxpool.Pool) EventRepository {
	return &eventRepository{pool: pool}
}

var Module = fx.Module(
	"repository",
	fx.Provide(
		NewOrderRepository,
		NewEventRepository,
	),
)
