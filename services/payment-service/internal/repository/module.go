// services/payment-service/internal/repository/module.go

package repository

import (
	"pkg/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

func NewPaymentRepository(db postgres.DBTX) PaymentRepository {
	return &paymentRepository{db: db}
}

func NewOutboxRepository(db postgres.DBTX) OutboxRepository {
	return &outboxRepository{db: db}
}

func NewTxManager(pool *pgxpool.Pool) TxManager {
	return &txManager{pool: pool}
}

var Module = fx.Module(
	"repository",
	fx.Provide(
		NewPaymentRepository,
		NewOutboxRepository,
		NewEventRepository,
		NewTxManager,
	),
)
