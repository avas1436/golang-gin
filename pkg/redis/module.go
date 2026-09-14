// pkg/redis/module.go

package redis

import (
	"context"
	"time"

	"pkg/config"

	"go.uber.org/fx"
)

func NewClientProvider(
	cfg *config.RedisConfig,
) (
	*Client,
	error,
) {

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	return NewClient(
		ctx,
		Config{
			Addr:         cfg.Addr,
			Password:     cfg.Password,
			DB:           cfg.DB,
			PoolSize:     cfg.PoolSize,
			MinIdleConns: cfg.MinIdleConns,
			ConnMaxIdle:  cfg.ConnMaxIdle,
		},
	)
}

func RegisterLifecycle(
	lc fx.Lifecycle,
	client *Client,
) {

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return client.Close()
		},
	},
	)
}

var Module = fx.Module(
	"redis",

	fx.Provide(
		NewClientProvider,
	),

	fx.Invoke(
		RegisterLifecycle,
	),
)
