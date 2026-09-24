// pkg/rabbitmq/module.go

package rabbitmq

import (
	"context"

	"go.uber.org/fx"

	"pkg/config"
)

// NewConnection اتصال اصلی RabbitMQ را ایجاد می‌کند.
func NewConnection(
	cfg *config.RabbitMQConfig,
) (
	*Connection,
	error,
) {

	return Connect(cfg.URL())
}

// NewConsumerProvider یک کانال جدید باز کرده و Consumer را ایجاد می‌کند
func NewConsumerProvider(conn *Connection) (*Consumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	return NewConsumer(ch)
}

// RegisterLifecycle مسئول shutdown کردن Connection است.
func RegisterLifecycle(
	lc fx.Lifecycle,
	conn *Connection,
) {

	lc.Append(
		fx.Hook{

			OnStop: func(ctx context.Context) error {
				return conn.Close()
			},
		},
	)
}

// ماژول
var Module = fx.Module(
	"rabbitmq",

	fx.Provide(
		NewConnection,
		NewConsumerProvider,
	),

	fx.Invoke(
		RegisterLifecycle,
	),
)
