// services/product-service/config/config.go

package config

import (
	commonConfig "pkg/config"
	"pkg/env"
)

type Config struct {
	GRPCPort string

	Postgres commonConfig.PostgresConfig
	Redis    commonConfig.RedisConfig
}

// Load مقادیر را از متغیرهای محیطی می‌خواند.
// در صورت نبود کانفیگ‌های ضروری، fail-fast برنامه درجا بسته می‌شود
func Load() (*Config, error) {

	// fail-fast: بدون پسورد دیتابیس سرویس نباید اصلاً بالا بیاد
	dbPassword, err := env.Require("DB_PASSWORD")
	if err != nil {
		return nil, err
	}

	redisDB, err := env.Int("REDIS_DB", 0)
	if err != nil {
		return nil, err
	}

	redisPoolSize, err := env.Int("REDIS_POOL_SIZE", 10)
	if err != nil {
		return nil, err
	}

	redisMinIdleConns, err := env.Int("REDIS_MIN_IDLE_CONNS", 2)
	if err != nil {
		return nil, err
	}

	redisConnMaxIdle, err := env.Duration(
		"REDIS_CONN_MAX_IDLE",
		0,
	)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		GRPCPort: env.String("GRPC_PORT", "50052"),

		Postgres: commonConfig.PostgresConfig{
			Host:     env.String("DB_HOST", "localhost"),
			Port:     env.String("DB_PORT", "5432"),
			User:     env.String("DB_USER", "product_service"),
			Password: dbPassword,
			DBName:   env.String("DB_NAME", "product_service_db"),
			SSLMode:  env.String("DB_SSLMODE", "disable"),
		},

		Redis: commonConfig.RedisConfig{
			Addr:         env.String("REDIS_ADDR", "localhost:6379"),
			Password:     env.String("REDIS_PASSWORD", ""),
			DB:           redisDB,
			PoolSize:     redisPoolSize,
			MinIdleConns: redisMinIdleConns,
			ConnMaxIdle:  redisConnMaxIdle,
		},
	}

	return cfg, nil
}
