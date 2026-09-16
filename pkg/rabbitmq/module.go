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
	),

	fx.Invoke(
		RegisterLifecycle,
	),
)
