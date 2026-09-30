// services/order-service/internal/messaging/module.go

package messaging

import (
	"context"
	"log"

	appErrors "pkg/errors"
	"pkg/events"
	"pkg/rabbitmq"

	"order-service/internal/service"

	"go.uber.org/fx"
)

// NewRabbitPublisher یک Publisher اختصاصی برای order-service می‌سازد
func NewRabbitPublisher(
	conn *rabbitmq.Connection,
) (*rabbitmq.Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to open channel for order publisher",
		)
	}

	publisher, err := rabbitmq.NewPublisher(
		ch,
		events.ExchangeOrderEvents,
	)
	if err != nil {
		_ = ch.Close()
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to initialize order event publisher",
		)
	}

	return publisher, nil
}

// RegisterMessagingLifecycle مدیریت شروع Consumer و Shutdown پاکیزه
func RegisterMessagingLifecycle(
	lc fx.Lifecycle,
	publisher *RabbitMQEventPublisher,
	consumer *PaymentEventConsumer,
) {
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(startCtx context.Context) error {
			go func() {
				log.Println("order-service: starting payment event consumer...")
				if err := consumer.StartListening(ctx); err != nil {
					log.Printf(
						"order-service: payment consumer stopped with error: %v",
						err,
					)
				}
			}()
			return nil
		},
		OnStop: func(stopCtx context.Context) error {
			log.Println("order-service: stopping messaging lifecycle...")
			cancel()
			if consumer != nil {
				_ = consumer.Close()
			}
			if publisher != nil {
				_ = publisher.Close()
			}
			return nil
		},
	})
}

var Module = fx.Module(
	"messaging",

	fx.Provide(
		NewRabbitPublisher,
		NewRabbitMQEventPublisher,
		func(p *RabbitMQEventPublisher) service.EventPublisher {
			return p
		},
		NewPaymentEventConsumer,
	),

	fx.Invoke(
		RegisterMessagingLifecycle,
	),
)
