// // services/product-service/config/module.go

package config

import (
	commonConfig "pkg/config"

	"go.uber.org/fx"
)

func providePostgresConfig(cfg *Config) *commonConfig.PostgresConfig {
	return &cfg.Postgres
}

func provideRedisConfig(cfg *Config) *commonConfig.RedisConfig {
	return &cfg.Redis
}

var Module = fx.Module(
	"config",
	fx.Provide(
		Load,
		providePostgresConfig,
		provideRedisConfig,
	),
)
