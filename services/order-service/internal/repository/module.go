// services/order-service/internal/repository/module.go

package repository

import (
	"pkg/postgres"

	"go.uber.org/fx"
)

// ساخت یک رپوزیتوری سفارشات
func NewOrderRepository(db postgres.DBTX) OrderRepository {
	return &orderRepository{db: db}
}

// ساخت یک رپوزیتوری ایونت
func NewEventRepository(db postgres.DBTX) EventRepository {
	return &eventRepository{db: db}
}

func NewSagaRepository(db postgres.DBTX) SagaRepository {
	return &sagaRepository{db: db}
}

func NewOutboxRepository(db postgres.DBTX) OutboxRepository {
	return &outboxRepository{db: db}
}

var Module = fx.Module(
	"repository",
	fx.Provide(
		NewOrderRepository,
		NewEventRepository,
		NewOutboxRepository,
		NewSagaRepository,
		NewTxManager,
	),
)
