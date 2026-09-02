// pkg/postgres/postgres.go

package postgres

import (
	"context"

	appErrors "pkg/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX زیرمجموعه‌ی مشترک متدهایی است که هم *pgxpool.Pool و هم pgx.Tx
// و pgxpool.Conn پیاده‌سازی می‌کنند.
//
// دلیل وجودش: اگر رپوزیتوری ها مستقیماً *pgxpool.Pool را تایپ کنند،
// هیچ‌وقت نمی‌شود چند عملیات از چند رپوزیتوری مختلف را داخل یک
// تراکنش اتمیک انجام داد. با تایپ‌کردن این اینترفیس به‌جای نوع
// مشخص، یک رپوزیتوری می‌تواند هم با پوول معمولی و هم با یک ترنسکشن
// کار کند، بدون این‌که حتی یک خط از کوئری‌هایش عوض شود.
type DBTX interface {
	Exec(
		ctx context.Context,
		sql string,
		arguments ...any,
	) (pgconn.CommandTag, error)

	Query(
		ctx context.Context,
		sql string,
		args ...any,
	) (pgx.Rows, error)

	QueryRow(
		ctx context.Context,
		sql string,
		args ...any,
	) pgx.Row
}

// NewPool یک connection pool به Postgres می‌سازد. عمداً عمومی و
// مستقل از هر سرویس خاص است: فقط یک dsn خام می‌گیرد و یک pool
// برمی‌گرداند. هر سرویس، config مخصوص خودش را نگه می‌دارد و فقط یک
// رشته‌ی dsn ساخته‌شده را به این‌جا می‌دهد.
func NewPool(
	ctx context.Context,
	dsn string,
) (
	*pgxpool.Pool,
	error,
) {

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to create postgres pool",
		)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to ping postgres",
		)
	}

	return pool, nil
}

// WithTx یک تابع دلخواه را داخل یک تراکنش Postgres اجرا می‌کند: اگر
// fn خطا برگرداند، تراکنش rollback می‌شود؛ در غیر این صورت commit
// می‌شود.
//
// این دقیقاً همان چیزی است که برای الگوهایی مثل Transactional Outbox
// لازم است
func WithTx(
	ctx context.Context,
	pool *pgxpool.Pool,
	fn func(tx pgx.Tx) error,
) error {

	tx, err := pool.Begin(ctx)
	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to begin transaction",
		)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// در این قسمت عمدا ارور wrap نمیشه تا خطای دامنه در جای دیگری کنترل شود
	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to commit transaction",
		)
	}

	return nil
}
