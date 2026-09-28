// services/order-service/internal/messaging/module.go

package messaging

import (
	"context"
	"order-service/internal/repository"
	"order-service/internal/service"
	appErrors "pkg/errors"

	"pkg/events"

	"go.uber.org/fx"

	"pkg/rabbitmq"
)

const (
	paymentEventsQueue = "order.payment_events.queue"
	paymentExchange    = "payment_events"
)

func NewPaymentEventConsumer(
	conn *rabbitmq.Connection,
	orderRepo repository.OrderRepository,
	publisher *RabbitMQEventPublisher,
) (
	*PaymentEventConsumer,
	error,
) {

	ch, err := conn.Channel()
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to open channel for payment consumer",
		)
	}

	consumer, err := rabbitmq.NewConsumer(ch)
	if err != nil {
		_ = ch.Close()
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to initialize rabbitmq consumer",
		)
	}

	// ثبت صف و اتصال آن به Exchange رویدادهای پرداخت (Binding)
	if err := consumer.BindQueue(
		paymentEventsQueue,
		paymentExchange,
		events.RoutingKeyPaymentCompleted,
	); err != nil {

		_ = ch.Close()

		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to bind queue to payment succeeded routing key",
		)
	}

	if err := consumer.BindQueue(
		paymentEventsQueue,
		paymentExchange,
		events.RoutingKeyPaymentFailed,
	); err != nil {

		_ = ch.Close()

		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to bind queue to payment failed routing key",
		)
	}

	return &PaymentEventConsumer{
		consumer:  consumer,
		orderRepo: orderRepo,
		publisher: publisher,
	}, nil
}

// NewRabbitMQEventPublisher یک Event Publisher می‌سازد.
func NewRabbitMQEventPublisher(
	publisher *rabbitmq.Publisher,
) *RabbitMQEventPublisher {

	return &RabbitMQEventPublisher{
		publisher: publisher,
	}
}

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

// RegisterMessagingLifecycle مدیریت کامل Lifecycle مربوط به Publisher و Consumer در زمان Shutdown
func RegisterMessagingLifecycle(
	lc fx.Lifecycle,
	publisher *rabbitmq.Publisher,
	consumer *PaymentEventConsumer,
) {
	lc.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				go func() {
					_ = consumer.StartListening(context.Background())
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				return publisher.Close()
			},
		},
	)
}

var Module = fx.Module(
	"messaging",

	fx.Provide(
		NewRabbitPublisher,
		NewRabbitMQEventPublisher,
		NewPaymentEventConsumer,

		fx.Annotate(
			NewRabbitMQEventPublisher,
			fx.As(new(service.EventPublisher)),
		),
	),

	fx.Invoke(
		RegisterMessagingLifecycle,
	),
)
