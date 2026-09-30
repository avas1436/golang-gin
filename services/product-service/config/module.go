// // services/product-service/config/module.go

package config

import (
	commonConfig "pkg/config"
	"time"

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

func ProvideProductTTL(cfg *Config) time.Duration {
	return cfg.Cache.ProductTTL
}

func ProvideSearchTTL(cfg *Config) time.Duration {
	return cfg.Cache.SearchTTL
}

var Module = fx.Module(
	"config",
	fx.Provide(
		Load,
		ProvidePostgresConfig,
		ProvideRedisConfig,
		ProvideRabbitMQConfig,
		ProvideJWTConfig,

		// ثبت ProductTTL با نام اختصاصی در Fx Container
		fx.Annotate(
			ProvideProductTTL,
			fx.ResultTags(`name:"productTTL"`),
		),

		// ثبت SearchTTL با نام اختصاصی در Fx Container
		fx.Annotate(
			ProvideSearchTTL,
			fx.ResultTags(`name:"searchTTL"`),
		),
	),
)
