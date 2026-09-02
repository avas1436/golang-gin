// pkg/redis/client.go

package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config ساختار پیکربندی اتصال به ردیس را تعریف می‌کند
type Config struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int           // حداکثر تعداد سوکت‌های باز
	MinIdleConns int           // حداقل تعداد کانکشن‌های بیکار
	ConnMaxIdle  time.Duration // حداکثر زمانی که یک کانکشن بیکار می‌تواند باز بماند
}

// NewClient یک کلاینت ردیس با تنظیمات بهینه و تست اتصال اولیه می‌سازد
func NewClient(
	ctx context.Context,
	cfg Config,
) (
	*redis.Client,
	error,
) {

	// تنظیم مقادیر پیش‌فرض منطقی در صورتی که در کانفیگ مقداردهی نشده باشند
	if cfg.PoolSize == 0 {
		cfg.PoolSize = 10
	}
	if cfg.MinIdleConns == 0 {
		cfg.MinIdleConns = 2
	}
	if cfg.ConnMaxIdle == 0 {
		cfg.ConnMaxIdle = 5 * time.Minute
	}

	client := redis.NewClient(
		&redis.Options{
			Addr:            cfg.Addr,
			Password:        cfg.Password,
			DB:              cfg.DB,
			PoolSize:        cfg.PoolSize,
			MinIdleConns:    cfg.MinIdleConns,
			ConnMaxIdleTime: cfg.ConnMaxIdle,
		})

	// الگوی Fail-Fast: بررسی می‌کنیم که ردیس واقعاً در دسترس باشد
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return client, nil
}
