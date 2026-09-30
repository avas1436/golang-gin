// services/order-service/internal/messaging/module.go

package messaging

import (
	"context"
	"log"
	"order-service/internal/repository"
	"order-service/internal/service"
	appErrors "pkg/errors"

	"pkg/events"

	"go.uber.org/fx"

	"pkg/rabbitmq"
)

// صف های هر قسمت کاملا از هم جدا هستند
const (
	// صف مصرف کننده رویداد تکمیل سفارش
	paymentCompletedQueue = "order.payment_completed.queue"

	// صف مصرف کننده رویداد شکست سفارش
	paymentFailedQueue = "order.payment_failed.queue"

	// اتصال به محل تبادل پیام سیستم پرداخت
	paymentExchange = "payment_events"
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
		paymentCompletedQueue,
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
		paymentFailedQueue,
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
