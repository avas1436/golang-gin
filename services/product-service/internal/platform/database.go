// services/product-service/internal/platform/database.go

package platform

import (
	"context"
	"pkg/postgres"
	redispkg "pkg/redis"
	"product-service/config"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"platform",
	fx.Provide(
		NewPostgresDB,
		NewRedisClient,
	),
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

	// فرض بر این است که پکیج pkg/postgres تابعی برای ساخت pool دارد
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

// اتصال کلینت ردیس را می‌سازد
func NewRedisClient(
	ctx context.Context,
	lc fx.Lifecycle,
	cfg *config.Config,
) *redispkg.Client {

	redisClient, _ := redispkg.NewClient(
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

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return redisClient.Close()
		},
	})

	return redisClient
}
