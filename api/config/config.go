// api/config/config.go

package config

import (
	commonConfig "pkg/config"
	"pkg/env"
	"time"
)

// Config تمام تنظیمات لازم برای اجرای API Gateway را نگه می‌دارد.
type Config struct {
	HTTPPort        string
	UserServiceAddr string

	Redis  commonConfig.RedisConfig
	Cookie CookieConfig
}

// ساختار کوکی
type CookieConfig struct {
	// Domain دامنه‌ای است که کوکی برایش معتبر است.
	Domain string

	// تعیین میکند که کوکی روی http باشد یا https
	Secure bool

	// مدت زمان اعتبار رفرش توکن روی کوکی
	RefreshMaxAgeSeconds int
}

func Load() (*Config, error) {
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

	cookieSecure, err := env.Bool("COOKIE_SECURE", true)
	if err != nil {
		return nil, err
	}

	refreshMaxAge, err := env.Int(
		"REFRESH_COOKIE_MAX_AGE_SECONDS",
		7*24*60*60,
	)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		HTTPPort: env.String("HTTP_PORT", "8080"),
		UserServiceAddr: env.String(
			"USER_SERVICE_ADDR",
			"localhost:50051",
		),
		Redis: commonConfig.RedisConfig{
			Addr:         env.String("REDIS_ADDR", "localhost:6379"),
			Password:     env.String("REDIS_PASSWORD", ""),
			DB:           redisDB,
			PoolSize:     redisPoolSize,
			MinIdleConns: redisMinIdleConns,
			ConnMaxIdle:  redisConnMaxIdle,
		},
		Cookie: CookieConfig{
			Domain: env.String(
				"COOKIE_DOMAIN",
				"localhost",
			),
			Secure:               cookieSecure,
			RefreshMaxAgeSeconds: refreshMaxAge,
		},
	}

	return cfg, nil
}
