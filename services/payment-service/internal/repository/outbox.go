// services/payment-service/internal/repository/outbox.go

package repository

import (
	"context"

	appErrors "pkg/errors"
	"pkg/postgres"

	"payment-service/internal/model"
)

// OutboxRepository مسئول نوشتن رویدادها در جدول outbox_events
// است. این اینترفیس عمداً فقط متد Create را دارد — چون طبق الگوی
// Transactional Outbox، ثبت باید در همان تراکنشی انجام شود که
// تغییر دامنه را انجام می‌دهد، و خواندن/به‌روزرسانی وضعیت توسط
// Poller جداگانه (که هنوز پیاده نشده) انجام می‌شود
type OutboxRepository interface {
	Create(
		ctx context.Context,
		event *model.OutboxEvent,
	) error
}

type outboxRepository struct {
	db postgres.DBTX
}

// Create یک رویداد را در جدول outbox_events ثبت می‌کند.
// فراخوانی این متد باید داخل یک تراکنش دیتابیس انجام شود تا
// atomicity بین تغییر دامنه و ثبت رویداد تضمین شود
func (
	r *outboxRepository,
) Create(
	ctx context.Context,
	event *model.OutboxEvent,
) error {

	if event == nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"outbox event cannot be nil",
		)
	}

	if event.EventType == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"outbox event type is required",
		)
	}

	if event.Exchange == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"outbox event exchange is required",
		)
	}

	if event.RoutingKey == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"outbox event routing key is required",
		)
	}

	if len(event.Payload) == 0 {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"outbox event payload is required",
		)
	}

	if event.AggregateType == "" {
		event.AggregateType = "payment"
	}

	if event.Status == "" {
		event.Status = model.OutboxStatusPending
	}

	query := `
		INSERT INTO outbox_events (
			event_type,
			exchange,
			routing_key,
			payload,
			aggregate_id,
			aggregate_type,
			status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		event.EventType,
		event.Exchange,
		event.RoutingKey,
		event.Payload,
		event.AggregateID,
		event.AggregateType,
		event.Status,
	).Scan(
		&event.ID,
		&event.CreatedAt,
		&event.UpdatedAt,
	)
	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to create outbox event",
		)
	}

	return nil
}
