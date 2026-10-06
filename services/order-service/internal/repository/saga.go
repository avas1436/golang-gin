// services/order-service/internal/repository/saga.go

package repository

import (
	"context"

	"github.com/google/uuid"

	appErrors "pkg/errors"
	"pkg/postgres"

	"order-service/internal/model"
)

// SagaRepository یک abstraction روی تراکنش‌های مربوط به Saga است.
type SagaRepository interface {
	// ExecuteInInbox بررسی تکراری نبودن رویداد در Inbox و اجرای
	// اتمیک منطق سفارش را در یک تراکنش تضمین می‌کند.
	ExecuteInInbox(
		ctx context.Context,
		eventID uuid.UUID,
		eventType string,
		orderID *uuid.UUID,
		fn func(
			ctx context.Context,
			repos Repositories,
		) (
			*model.Order,
			error,
		),
	) (
		bool,
		*model.Order,
		error,
	)

	// CreateOrderAtomic یک تراکنش اتمیک باز کرده و ایجاد سفارش
	// و آیتم‌هایش را مدیریت می‌کند. حالا می‌تواند رویداد Outbox
	// را هم در همان تراکنش درج کند
	CreateOrderAtomic(
		ctx context.Context,
		order *model.Order,
		outboxEvent *model.OutboxEvent,
	) error
}

type sagaRepository struct {
	db postgres.DBTX
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
		repos Repositories,
	) (
		*model.Order,
		error,
	),
) (
	bool,
	*model.Order,
	error,
) {

	if eventID == uuid.Nil {
		return false, nil, appErrors.New(
			appErrors.KindInvalidInput,
			"event id cannot be empty",
		)
	}

	if fn == nil {
		return false, nil, appErrors.New(
			appErrors.KindInvalidInput,
			"callback cannot be nil",
		)
	}

	eventRepo := NewEventRepository(r.db)
	orderRepo := NewOrderRepository(r.db)
	outboxRepo := NewOutboxRepository(r.db)

	// ۱. ثبت در جدول processed_events جهت کنترل Idempotency
	isProcessed, err := eventRepo.MarkProcessed(
		ctx,
		eventID,
		eventType,
		orderID,
		nil,
	)
	if err != nil {
		return false, nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"order-service: failed to mark event in inbox",
		)
	}

	if isProcessed {
		return true, nil, nil
	}

	// ۲. ساخت Repositories با DBTX یکسان و فراخوانی callback
	repos := Repositories{
		Order:  orderRepo,
		Event:  eventRepo,
		Outbox: outboxRepo,
		Saga:   nil,
	}

	order, err := fn(ctx, repos)
	if err != nil {
		return false, nil, err
	}

	return false, order, nil

}

// CreateOrderAtomic تراکنش را باز می‌کند، سفارش را insert می‌کند
// و سپس callback را صدا می‌زند تا OutboxEvent را در همان tx درج کند
func (
	r *sagaRepository,
) CreateOrderAtomic(
	ctx context.Context,
	order *model.Order,
	outboxEvent *model.OutboxEvent,
) error {

	if order == nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"order cannot be nil",
		)
	}

	orderRepo := NewOrderRepository(r.db)

	if err := orderRepo.Create(ctx, order); err != nil {
		return err
	}

	if outboxEvent == nil {
		return nil
	}

	outboxRepo := NewOutboxRepository(r.db)

	if err := outboxRepo.Create(ctx, outboxEvent); err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"order-service: failed to create outbox event",
		)
	}

	return nil
}
