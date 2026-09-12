// services/order-service/config/config.go

package config

import (
	commonConfig "pkg/config"
	"pkg/env"
	"time"
)

type Config struct {
	GRPCPort string

	Postgres commonConfig.PostgresConfig
	Redis    commonConfig.RedisConfig
	RabbitMQ commonConfig.RabbitMQConfig
	JWT      JWTConfig

	// باید به سرویس محصولات متصل شویم
	ProductServiceAddr string
}

// برای اعتبار سنجی اتصال به سرویس محصولات
type JWTConfig struct {
	Secret         string
	AccessTokenTTL time.Duration
}

// Load مقادیر را از متغیرهای محیطی می‌خواند.
// در صورت نبود کانفیگ‌های ضروری، fail-fast برنامه درجا بسته می‌شود
func Load() (*Config, error) {

	// لود کردن فایل متغیر های محیطی در محیط پردازش برنامه
	env.Load(".env")

	// fail-fast: بدون پسورد دیتابیس سرویس نباید اصلاً بالا بیاد
	dbPassword, err := env.Require("DB_PASSWORD")
	if err != nil {
		return nil, err
	}

	// fail-fast: بدون این مقدار نمی‌توان توکن‌های کاربران را
	// اعتبارسنجی کرد
	jwtSecret, err := env.Require("JWT_SECRET")
	if err != nil {
		return nil, err
	}

	accessTTL, err := env.Duration("JWT_ACCESS_TTL", 15*time.Minute)
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

	// fail-fast: بدون پسورد RabbitMQ سرویس نباید بالا بیاد؛ چون
	// کل جریان Saga (order.created و گوش‌دادن به payment.*) به این
	// اتصال وابسته است
	rabbitPassword, err := env.Require("RABBITMQ_PASSWORD")
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		// پورت این سرویس باید با سرویس های دیگه متفاوت باشه
		GRPCPort: env.String("GRPC_PORT", "50053"),

		Postgres: commonConfig.PostgresConfig{
			Host:     env.String("DB_HOST", "localhost"),
			Port:     env.String("DB_PORT", "5432"),
			User:     env.String("DB_USER", "order_service"),
			Password: dbPassword,
			DBName:   env.String("DB_NAME", "order_service_db"),
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
			Password: rabbitPassword,
			VHost:    env.String("RABBITMQ_VHOST", ""),
		},

		JWT: JWTConfig{
			Secret:         jwtSecret,
			AccessTokenTTL: accessTTL,
		},

		ProductServiceAddr: env.String(
			"PRODUCT_SERVICE_ADDR",
			"localhost:50052",
		),
	}

	return cfg, nil
}
