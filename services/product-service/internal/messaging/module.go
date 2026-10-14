// services/product-service/internal/messaging/module.go

package messaging

import (
	"context"
	"log"

	commonConfig "pkg/config"
	appErrors "pkg/errors"
	"pkg/events"
	"pkg/rabbitmq"

	"product-service/config"
	"product-service/internal/cache"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

func NewOrderEventConsumer(
	conn *rabbitmq.Connection,
	pool *pgxpool.Pool,
	productStore *cache.ProductCacheStore,
) (
	*OrderEventConsumer,
	error,
) {

	// ۱. ایجاد Channel و Consumer اختصاصی برای صف Stock Confirm
	chConfirm, err := conn.Channel()
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to open channel for stock confirm consumer",
		)
	}

	consumerConfirm, err := rabbitmq.NewConsumer(chConfirm)
	if err != nil {
		_ = chConfirm.Close()
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to initialize stock confirm consumer",
		)
	}

	if err := consumerConfirm.BindQueue(
		events.QueueProductStockConfirm,
		events.ExchangeOrderEvents,
		events.RoutingKeyStockConfirmRequested,
	); err != nil {
		_ = chConfirm.Close()
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to bind stock confirm queue",
		)
	}

	// ۲. ایجاد Channel و Consumer اختصاصی برای صف Stock Release
	chRelease, err := conn.Channel()
	if err != nil {
		_ = chConfirm.Close()
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to open channel for stock release consumer",
		)
	}

	consumerRelease, err := rabbitmq.NewConsumer(chRelease)
	if err != nil {
		_ = chConfirm.Close()
		_ = chRelease.Close()
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to initialize stock release consumer",
		)
	}

	if err := consumerRelease.BindQueue(
		events.QueueProductStockRelease,
		events.ExchangeOrderEvents,
		events.RoutingKeyStockReleaseRequested,
	); err != nil {
		_ = chConfirm.Close()
		_ = chRelease.Close()
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to bind stock release queue",
		)
	}

	return &OrderEventConsumer{
		consumerConfirm: consumerConfirm,
		consumerRelease: consumerRelease,
		pool:            pool,
		productStore:    productStore,
	}, nil
}

func RegisterMessagingLifecycle(
	lc fx.Lifecycle,
	consumer *OrderEventConsumer,
) {
	lc.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				go func() {
					if err := consumer.StartListening(context.Background()); err != nil {
						log.Printf(
							"product-service: order event consumer stopped with error: %v",
							err,
						)
					}
				}()
				return nil
			},
		},
	)
}

var Module = fx.Module(
	"messaging",

	// این فابريکِ کوچک، RabbitMQConfig را از کانفیگ کلی سرویس استخراج می‌کند
	// و به پکیج مرکزی rabbitmq تحویل می‌دهد تا وابستگی ایجاد نشود
	fx.Provide(
		func(cfg *config.Config) commonConfig.RabbitMQConfig {
			return cfg.RabbitMQ
		},
	),

	fx.Provide(
		NewOrderEventConsumer,
	),

	fx.Invoke(
		RegisterMessagingLifecycle,
	),
)
