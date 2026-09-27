// services/user-service/internal/messaging/module.go

package messaging

import (
	"user-service/internal/service"

	"go.uber.org/fx"
)

// Module مربوط به پکیج messaging در User Service است
var Module = fx.Module(
	"messaging",

	fx.Provide(
		fx.Annotate(
			NewRabbitMQEventPublisher,
			fx.As(new(service.EventPublisher)),
		),
	),
)
