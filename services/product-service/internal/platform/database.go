// services/product-service/internal/platform/database.go

package platform

import (
	"context"
	"pkg/postgres"
	redispkg "pkg/redis"
	"product-service/config"

	"go.uber.org/fx"
)

// اتصال دیتابیس را می‌سازد و به اینترفیس DBTX متصل می‌کند
func NewPostgresDB(
	ctx context.Context,
	lc fx.Lifecycle,
	cfg *config.Config,
) (
	postgres.DBTX,
	error,
) {

	pool, err := postgres.NewPool(ctx, cfg.Postgres.DSN())
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			pool.Close()
			return nil
		},
	})

	return pool, nil
}

// اتصال کلینت ردیس را می‌سازد.
func NewRedisClient(
	ctx context.Context,
	lc fx.Lifecycle,
	cfg *config.Config,
) (
	*redispkg.Client,
	error,
) {

	redisClient, err := redispkg.NewClient(
		ctx,
		redispkg.Config{
			Addr:         cfg.Redis.Addr,
			Password:     cfg.Redis.Password,
			DB:           cfg.Redis.DB,
			PoolSize:     cfg.Redis.PoolSize,
			MinIdleConns: cfg.Redis.MinIdleConns,
			ConnMaxIdle:  cfg.Redis.ConnMaxIdle,
		},
	)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return redisClient.Close()
		},
	})

	return redisClient, nil
}
