// services/order-service/internal/messaging/module.go

package messaging

import (
	"context"
	"order-service/internal/service"

	"go.uber.org/fx"

	"pkg/rabbitmq"
)

func NewRabbitPublisher(
	conn *rabbitmq.Connection,
) (
	*rabbitmq.Publisher,
	error,
) {

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	publisher, err := rabbitmq.NewPublisher(
		ch,
		"order.events",
	)
	if err != nil {
		_ = ch.Close()
		return nil, err
	}

	return publisher, nil
}

// RegisterPublisherLifecycle مسئول بستن Publisher هنگام
// shutdown شدن Order Service است.
//
// چون Publisher از یک RabbitMQ Channel استفاده می‌کند،
// باید در زمان shutdown آن را ببندیم.
func RegisterPublisherLifecycle(
	lc fx.Lifecycle,
	publisher *rabbitmq.Publisher,
) {

	lc.Append(
		fx.Hook{
			OnStop: func(ctx context.Context) error {
				return publisher.Close()
			},
		},
	)
}

// Module تمام Dependencyهای مربوط به Messaging در
// Order Service را ثبت می‌کند.
var Module = fx.Module(
	"messaging",

	fx.Provide(
		NewRabbitPublisher,

		fx.Annotate(
			NewRabbitMQEventPublisher,
			fx.As(new(service.EventPublisher)),
		),
	),

	fx.Invoke(
		RegisterPublisherLifecycle,
	),
)
