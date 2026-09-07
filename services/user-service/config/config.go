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

	Postgres commonConfig.PostgresConfig
	Redis    commonConfig.RedisConfig
	JWT      JWTConfig
}

type JWTConfig struct {
	Secret          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// Load مقادیر را از متغیرهای محیطی می‌خواند.
// در صورت نبود کانفیگیوریشن های fail-fast برنامه درجا بسته میشود
func Load() (*Config, error) {

	// fail-fast
	dbPassword, err := env.Require("DB_PASSWORD")
	if err != nil {
		return nil, err
	}

	// fail-fast
	jwtSecret, err := env.Require("JWT_SECRET")
	if err != nil {
		return nil, err
	}

	// این هم مهمه ولی مقدار پیش فرض داره
	accessTTL, err := env.Duration("JWT_ACCESS_TTL", 15*time.Minute)
	if err != nil {
		return nil, err
	}

	// این هم مهمه ولی مقدار پیش فرض داره
	refreshTTL, err := env.Duration("JWT_REFRESH_TTL", 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	// این هم مهمه ولی مقدار پیش فرض داره
	redisDB, err := env.Int("REDIS_DB", 0)
	if err != nil {
		return nil, err
	}

	// این هم مهمه ولی مقدار پیش فرض داره
	redisPoolSize, err := env.Int("REDIS_POOL_SIZE", 10)
	if err != nil {
		return nil, err
	}

	// این هم مهمه ولی مقدار پیش فرض داره
	redisMinIdleConns, err := env.Int("REDIS_MIN_IDLE_CONNS", 2)
	if err != nil {
		return nil, err
	}

	// این هم مهمه ولی مقدار پیش فرض داره
	redisConnMaxIdle, err := env.Duration(
		"REDIS_CONN_MAX_IDLE",
		5*time.Minute,
	)
	if err != nil {
		return nil, err
	}

	// در اینجا مقادیر وارد کانفیگ میشه
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

		JWT: JWTConfig{
			Secret:          jwtSecret,
			AccessTokenTTL:  accessTTL,
			RefreshTokenTTL: refreshTTL,
		},
	}

	return cfg, nil
}
