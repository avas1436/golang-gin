// services/payment-service/internal/repository/module.go

package repository

import "go.uber.org/fx"

var Module = fx.Module(
	"repository",
	fx.Provide(
		NewPaymentRepository,
		NewEventRepository,
	),
)
