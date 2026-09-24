// services/payment-service/internal/repository/module.go

package repository

import (
	"pkg/postgres"

	"go.uber.org/fx"
)

func NewPaymentRepository(db postgres.DBTX) PaymentRepository {
	return &paymentRepository{db: db}
}

var Module = fx.Module(
	"repository",
	fx.Provide(
		NewPaymentRepository,
		NewEventRepository,
	),
)
