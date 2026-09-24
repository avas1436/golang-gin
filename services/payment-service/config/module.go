// // services/payment-service/config/module.go

package config

import (
	commonConfig "pkg/config"

	"go.uber.org/fx"
)

func ProvidePostgresConfig(cfg *Config) *commonConfig.PostgresConfig {
	return &cfg.Postgres
}

func ProvideRedisConfig(cfg *Config) *commonConfig.RedisConfig {
	return &cfg.Redis
}

func ProvideJWTConfig(cfg *Config) *commonConfig.JWTConfig {
	return &cfg.JWT
}
func ProvideRabbitMQConfig(cfg *Config) *commonConfig.RabbitMQConfig {
	return &cfg.RabbitMQ
}

var Module = fx.Module(
	"config",

	fx.Provide(
		Load,

		ProvidePostgresConfig,
		ProvideRedisConfig,
		ProvideJWTConfig,
		ProvideRabbitMQConfig,
	),
)
