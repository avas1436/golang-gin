// pkg/postgres/module.go

package postgres

import (
	"context"
	"time"

	"pkg/config"

	"go.uber.org/fx"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPoolProvider(
	lc fx.Lifecycle,
	cfg *config.PostgresConfig,
) (
	*pgxpool.Pool,
	error,
) {

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	pool, err := NewPool(ctx, cfg.DSN())
	if err != nil {
		return nil, err
	}

	lc.Append(
		fx.Hook{
			OnStop: func(ctx context.Context) error {
				pool.Close()
				return nil
			},
		},
	)

	return pool, nil
}

// AsDBTX همان Pool را به صورت DBTX در اختیار Fx قرار می‌دهد.
//
// این کار باعث می‌شود:
//
//	*pgxpool.Pool
//
// و:
//
//	postgres.DBTX
//
// هر دو به همان connection pool اشاره کنند.
func AsDBTX(
	pool *pgxpool.Pool,
) DBTX {
	return pool
}

// ماژور خروجی دیتابیس
var Module = fx.Module(
	"postgres",

	fx.Provide(
		NewPoolProvider,

		fx.Annotate(
			AsDBTX,
			fx.As(new(DBTX)),
		),
	),
)
