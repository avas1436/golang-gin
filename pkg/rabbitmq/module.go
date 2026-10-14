// pkg/rabbitmq/module.go

package rabbitmq

import (
	"context"
	"pkg/config"

	"go.uber.org/fx"
)

// یک فابريک که کانفیگ کلی سرویس را می‌گیرد و اتصال
// RabbitMQ را با متد درست برقراری می‌کند
func NewRabbitConnection(cfg config.RabbitMQConfig) (*Connection, error) {
	return Connect(
		cfg.URL(),
		cfg.EnableTLS,
		cfg.TLS,
	)
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
		NewRabbitConnection,
	),

	fx.Invoke(
		RegisterLifecycle,
	),
)
