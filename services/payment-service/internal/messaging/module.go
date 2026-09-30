// services/payment-service/internal/messaging/module.go

package messaging

import (
	"context"
	"log"

	appErrors "pkg/errors"
	"pkg/events"
	"pkg/rabbitmq"

	"payment-service/internal/service"

	"go.uber.org/fx"
)

// NewRabbitPublisher یک Publisher اختصاصی با Channel مجزا برای سرویس پرداخت می‌سازد
func NewRabbitPublisher(
	conn *rabbitmq.Connection,
) (*rabbitmq.Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to open channel for payment publisher",
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
			"failed to initialize payment event publisher",
		)
	}

	return publisher, nil
}

// RegisterMessagingLifecycle شروع استماع صف و مدیریت Shutdown پاکیزه را بر عهده دارد
func RegisterMessagingLifecycle(
	lc fx.Lifecycle,
	publisher *RabbitMQEventPublisher,
	consumer *OrderEventConsumer,
) {
	// استفاده از Background Context ماندگار به همراه Cancel برای لغو اجرا در زمان Shutdown
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(startCtx context.Context) error {
			go func() {
				log.Println("payment-service: starting order event consumer...")
				if err := consumer.StartListening(ctx); err != nil {
					log.Printf(
						"payment-service: order event consumer stopped with error: %v",
						err,
					)
				}
			}()
			return nil
		},
		OnStop: func(stopCtx context.Context) error {
			log.Println("payment-service: stopping messaging lifecycle...")
			cancel()
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
		fx.Annotate(
			NewRabbitMQEventPublisher,
			fx.As(new(service.EventPublisher)),
		),
		NewOrderEventConsumer,
	),

	fx.Invoke(RegisterMessagingLifecycle),
)
