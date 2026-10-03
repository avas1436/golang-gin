// services/payment-service/internal/repository/tx.go

package repository

import (
	"context"

	"pkg/postgres"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repositories شامل تمام ریپازیتوری‌های فعال در طول یک تراکنش است.
type Repositories struct {
	Payment PaymentRepository
	Event   EventRepository
	Outbox  OutboxRepository
}

// TxManager مدیریت اتمیک تراکنش‌ها را بر عهده دارد.
type TxManager interface {
	ExecInTx(ctx context.Context, fn func(repos Repositories) error) error
}

type txManager struct {
	pool *pgxpool.Pool
}

// ExecInTx یک تراکنش دیتابیس باز کرده و ریپازیتوری‌های ایزوله شده روی آن تراکنش را در اختیار لایه سرویس قرار می‌دهد.
func (
	m *txManager,
) ExecInTx(
	ctx context.Context,
	fn func(repos Repositories) error,
) error {

	return postgres.WithTx(
		ctx,
		m.pool,
		func(tx pgx.Tx) error {
			repos := Repositories{
				Payment: NewPaymentRepository(tx),
				Event:   NewEventRepository(tx),
				Outbox:  NewOutboxRepository(tx),
			}
			return fn(repos)
		},
	)
}
