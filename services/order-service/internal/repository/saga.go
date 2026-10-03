// services/order-service/internal/repository/saga.go

package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	appErrors "pkg/errors"
	"pkg/postgres"

	"order-service/internal/model"
)

type SagaRepository interface {
	// ExecuteInInbox بررسی تکراری نبودن رویداد در Inbox و اجرای اتمیک
	// منطق سفارش را در یک تراکنش تضمین می‌کند
	ExecuteInInbox(
		ctx context.Context,
		eventID uuid.UUID,
		eventType string,
		orderID *uuid.UUID,
		fn func(
			ctx context.Context,
			txOrderRepo OrderRepository,
		) (
			*model.Order,
			error,
		),
	) (
		alreadyProcessed bool,
		order *model.Order,
		err error,
	)

	// CreateOrderAtomic یک تراکنش اتمیک باز کرده و ایجاد سفارش
	// و آیتم‌هایش را مدیریت می‌کند
	CreateOrderAtomic(ctx context.Context, order *model.Order) error
}

type sagaRepository struct {
	pool *pgxpool.Pool
}

func NewSagaRepository(pool *pgxpool.Pool) SagaRepository {
	return &sagaRepository{pool: pool}
}

func (
	r *sagaRepository,
) ExecuteInInbox(
	ctx context.Context,
	eventID uuid.UUID,
	eventType string,
	orderID *uuid.UUID,
	fn func(
		ctx context.Context,
		txOrderRepo OrderRepository,
	) (
		*model.Order,
		error,
	),
) (
	bool,
	*model.Order,
	error,
) {

	var resultOrder *model.Order
	var alreadyProcessed bool

	err := postgres.WithTx(
		ctx,
		r.pool,
		func(tx pgx.Tx) error {
			// ساخت نسخه‌های متصل به تراکنش جاری
			eventRepo := &eventRepository{pool: tx}
			txOrderRepo := &orderRepository{pool: tx}

			// ۱. ثبت در جدول processed_events جهت کنترل Idempotency
			isProcessed, err := eventRepo.MarkProcessed(
				ctx,
				eventID,
				eventType,
				orderID,
				nil,
			)
			if err != nil {
				return appErrors.Wrap(
					appErrors.KindInternal,
					err,
					"order-service: failed to mark event in inbox",
				)
			}

			if isProcessed {
				alreadyProcessed = true
				return nil
			}

			// ۲. اجرای متد بزینس روی OrderRepository متصل به همین تراکنش
			order, err := fn(ctx, txOrderRepo)
			if err != nil {
				return err
			}

			resultOrder = order
			return nil
		},
	)

	if err != nil {
		return false, nil, err
	}

	return alreadyProcessed, resultOrder, nil
}

// CreateOrderAtomic مدیریت باز کردن تراکنش را انجام داده و متد Create از orderRepository را صدا می‌زند
func (
	r *sagaRepository,
) CreateOrderAtomic(
	ctx context.Context,
	order *model.Order,
) error {

	return postgres.WithTx(
		ctx,
		r.pool,
		func(tx pgx.Tx) error {
			// ساخت یک نمونه OrderRepository متصل به همین تراکنش جاری
			txOrderRepo := &orderRepository{pool: tx}

			return txOrderRepo.Create(ctx, order)
		},
	)
}
