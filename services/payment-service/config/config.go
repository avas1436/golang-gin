// services/payment-service/config/config.go

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
	JWT      commonConfig.JWTConfig
	ZarinPal commonConfig.ZarinpalConfig
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

	// fail-fast: بدون این مقدار نمی‌توان توکن‌های ادمین را
	// اعتبارسنجی کرد (متد GetPaymentByOrderID فقط برای ادمین است)
	jwtSecret, err := env.Require("JWT_SECRET")
	if err != nil {
		return nil, err
	}

	accessTTL, err := env.Duration(
		"JWT_ACCESS_TTL",
		15*time.Minute,
	)
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

	// fail-fast: بدون پسورد RabbitMQ سرویس نباید بالا بیاید؛ چون
	// تنها راه ورودی این سرویس مصرف رویداد order.created است
	rabbitPassword, err := env.Require("RABBITMQ_PASSWORD")
	if err != nil {
		return nil, err
	}

	// اطلاعات لازم برای زرین پال
	merchantID, err := env.Require("MERCHANT_ID")
	if err != nil {
		return nil, err
	}

	isSandbox, err := env.Bool("IS_SANDBOX", true)
	if err != nil {
		return nil, err
	}

	paymentCallbackURL, err := env.Require("PAYMENT_CALLBACK_URL")
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		GRPCPort: env.String("GRPC_PORT", "50054"),

		Postgres: commonConfig.PostgresConfig{
			Host:     env.String("DB_HOST", "localhost"),
			Port:     env.String("DB_PORT", "5432"),
			User:     env.String("DB_USER", "payment_service"),
			Password: dbPassword,
			DBName:   env.String("DB_NAME", "payment_service_db"),
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

		JWT: commonConfig.JWTConfig{
			Secret:         jwtSecret,
			AccessTokenTTL: accessTTL,
		},

		ZarinPal: commonConfig.ZarinpalConfig{
			MerchantID:         merchantID,
			IsSandbox:          isSandbox,
			PaymentCallbackURL: paymentCallbackURL,
		},
	}

	return cfg, nil
}
