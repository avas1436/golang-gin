// services/user-service/config/config.go

package config

import (
	commonConfig "pkg/config"
	"pkg/env"
	"time"
)

// این فایل مسیول نگهداری تنظیمات اجرای این سرویس است
// هر زمینه به ساختار های کوچک تر تقسیم شده تا بتوان هر بخش را جدا پاس داد
type Config struct {
	GRPCPort string

	Postgres     commonConfig.PostgresConfig
	Redis        commonConfig.RedisConfig
	RabbitMQ     commonConfig.RabbitMQConfig
	JWT          commonConfig.JWTConfig
	RefreshToken commonConfig.RefreshTokenConfig
}

// Load مقادیر را از متغیرهای محیطی می‌خواند.
// در صورت نبود کانفیگیوریشن های fail-fast برنامه درجا بسته میشود
func Load() (*Config, error) {

	// لود کردن فایل متغیر های محیطی در محیط پردازش برنامه
	env.Load(".env")

	// متغیرهای اجباری (Fail-Fast)
	dbPassword, err := env.Require("DB_PASSWORD")
	if err != nil {
		return nil, err
	}

	jwtSecret, err := env.Require("JWT_SECRET")
	if err != nil {
		return nil, err
	}

	rabbitmqPassword, err := env.Require("RABBITMQ_PASSWORD")
	if err != nil {
		return nil, err
	}

	// متغیرهای دارای مقدار پیش‌فرض
	accessTTL, err := env.Duration("JWT_ACCESS_TTL", 15*time.Minute)
	if err != nil {
		return nil, err
	}

	refreshTTL, err := env.Duration("JWT_REFRESH_TTL", 7*24*time.Hour)
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
		5*time.Minute,
	)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		GRPCPort: env.String("GRPC_PORT", "50051"),

		Postgres: commonConfig.PostgresConfig{
			Host:     env.String("DB_HOST", "localhost"),
			Port:     env.String("DB_PORT", "5432"),
			User:     env.String("DB_USER", "user_service"),
			Password: dbPassword,
			DBName:   env.String("DB_NAME", "user_service_db"),
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

		RabbitMQ: commonConfig.RabbitMQConfig{
			Host:     env.String("RABBITMQ_HOST", "localhost"),
			Port:     env.String("RABBITMQ_PORT", "5672"),
			User:     env.String("RABBITMQ_USER", "guest"),
			Password: rabbitmqPassword,
			VHost:    env.String("RABBITMQ_VHOST", "/"),
		},

		JWT: commonConfig.JWTConfig{
			Secret:         jwtSecret,
			AccessTokenTTL: accessTTL,
		},

		RefreshToken: commonConfig.RefreshTokenConfig{
			TTL: refreshTTL,
		},
	}

	return cfg, nil
}
