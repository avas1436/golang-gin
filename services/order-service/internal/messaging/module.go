// services/order-service/internal/messaging/module.go

package messaging

import (
	"context"
	"log"
	"order-service/internal/repository"
	"order-service/internal/service"

	"pkg/events"

	"go.uber.org/fx"

	"pkg/rabbitmq"
)

func NewPaymentEventConsumer(
	conn *rabbitmq.Connection,
	orderRepo repository.OrderRepository,
	publisher *RabbitMQEventPublisher,
) (
	*PaymentEventConsumer,
	error,
) {

	// ۱. کانال اختصاصی برای صف پرداخت موفق
	chCompleted, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	consumerCompleted, err := rabbitmq.NewConsumer(chCompleted)
	if err != nil {
		_ = chCompleted.Close()
		return nil, err
	}

	if err := consumerCompleted.BindQueue(
		events.QueueOrderPaymentCompleted,
		events.ExchangeOrderEvents,
		events.RoutingKeyPaymentCompleted,
	); err != nil {
		_ = chCompleted.Close()
		return nil, err
	}

	// ۲. کانال اختصاصی برای صف پرداخت ناموفق
	chFailed, err := conn.Channel()
	if err != nil {
		_ = chCompleted.Close()
		return nil, err
	}

	consumerFailed, err := rabbitmq.NewConsumer(chFailed)
	if err != nil {
		_ = chCompleted.Close()
		_ = chFailed.Close()
		return nil, err
	}

	if err := consumerFailed.BindQueue(
		events.QueueOrderPaymentFailed,
		events.ExchangeOrderEvents,
		events.RoutingKeyPaymentFailed,
	); err != nil {
		_ = chCompleted.Close()
		_ = chFailed.Close()
		return nil, err
	}

	return &PaymentEventConsumer{
		consumerCompleted: consumerCompleted,
		consumerFailed:    consumerFailed,
		orderRepo:         orderRepo,
		publisher:         publisher,
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
		events.ExchangeOrderEvents,
	)
	if err != nil {
		_ = ch.Close()
		return nil, err
	}

	return publisher, nil
}

// RegisterMessagingLifecycle مدیریت شروع Consumer و Shutdown شدن Publisher
func RegisterMessagingLifecycle(
	lc fx.Lifecycle,
	publisher *RabbitMQEventPublisher,
	consumer *PaymentEventConsumer,
) {
	lc.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				go func() {
					if err := consumer.StartListening(context.Background()); err != nil {
						log.Printf(
							"order-service: payment consumer stopped with error: %v",
							err,
						)
					}
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
		NewRabbitMQEventPublisher,
		NewPaymentEventConsumer,

		// نگاشت RabbitMQEventPublisher به اینترفیس service.EventPublisher
		func(p *RabbitMQEventPublisher) service.EventPublisher {
			return p
		},
	),

	fx.Invoke(
		RegisterMessagingLifecycle,
	),
)
