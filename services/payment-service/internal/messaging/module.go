// services/payment-service/internal/messaging/module.go

package messaging

import (
	"context"

	"pkg/events"
	"pkg/rabbitmq"

	"payment-service/internal/service"

	"go.uber.org/fx"
)

func RegisterConsumers(
	lc fx.Lifecycle,
	consumer *rabbitmq.Consumer,
	orderCreatedHandler *OrderCreatedConsumer,
) {

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go consumer.Consume(
				ctx,
				events.RoutingKeyOrderCreated,
				orderCreatedHandler.Handle,
			)
			return nil
		},
	})
}

var Module = fx.Module(
	"messaging",

	fx.Provide(
		NewOrderCreatedConsumer,
		fx.Annotate(
			NewRabbitMQEventPublisher,
			fx.As(new(service.EventPublisher)),
		),
	),
	fx.Invoke(RegisterConsumers),
)
