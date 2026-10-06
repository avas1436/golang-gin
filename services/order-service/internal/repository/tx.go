// services/order-service/internal/repository/tx.go

package repository

import (
	"context"

	appErrors "pkg/errors"
	"pkg/postgres"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repositories شامل تمام ریپازیتوری‌های فعال در طول یک تراکنش است.
//
// این struct ابزاری است برای پاس دادن مجموعه‌ی repositoryهای
// متصل به یک تراکنش به callbackهایی که داخل ExecInTx اجرا می‌شوند.
// با این الگو، دیگر لازم نیست هر callback خودش repositoryها را
// دستی بسازد
type Repositories struct {
	Order  OrderRepository
	Event  EventRepository
	Outbox OutboxRepository
	Saga   SagaRepository
}

// TxManager مدیریت اتمیک تراکنش‌ها را بر عهده دارد.
//
// این اینترفیس ابسترکشن نازکی روی postgres.WithTx است و به لایه‌ی
// service اجازه می‌دهد بدون وابستگی مستقیم به pgxpool.Pool،
// مجموعه‌ای از عملیات را در یک تراکنش اجرا کند
type TxManager interface {
	ExecInTx(
		ctx context.Context,
		fn func(repos Repositories) error,
	) error
}

type txManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) TxManager {
	return &txManager{pool: pool}
}

// ExecInTx یک تراکنش دیتابیس باز می‌کند و repositoryهای ایزوله
// شده روی آن تراکنش را در اختیار callback قرار می‌دهد.
//
// اگر callback خطا برگرداند، تراکنش rollback می‌شود؛ در غیر این
// صورت commit
func (
	m *txManager,
) ExecInTx(
	ctx context.Context,
	fn func(repos Repositories) error,
) error {

	if fn == nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"transaction callback cannot be nil",
		)
	}

	return postgres.WithTx(
		ctx,
		m.pool,
		func(tx pgx.Tx) error {
			repos := Repositories{
				Order:  NewOrderRepository(tx),
				Event:  NewEventRepository(tx),
				Outbox: NewOutboxRepository(tx),
				Saga:   NewSagaRepository(tx),
			}
			return fn(repos)
		},
	)
}
