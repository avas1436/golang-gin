// services/user-service/internal/messaging/module.go

package messaging

import (
	commonConfig "pkg/config"
	"user-service/config"
	"user-service/internal/service"

	"go.uber.org/fx"
)

// Module مربوط به پکیج messaging در User Service است
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
		fx.Annotate(
			NewRabbitMQEventPublisher,
			fx.As(new(service.EventPublisher)),
		),
	),
)
