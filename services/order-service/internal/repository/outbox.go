// services/order-service/internal/repository/outbox.go

package repository

import (
	"context"

	appErrors "pkg/errors"
	"pkg/postgres"

	"order-service/internal/model"

	"github.com/google/uuid"
)

// OutboxRepository مسئول نوشتن و خواندن رویدادهای Outbox است.
type OutboxRepository interface {
	// Create یک رویداد را در جدول outbox_events درج می‌کند.
	// باید داخل یک تراکنش صدا زده شود تا atomicity با تغییر
	// دامنه تضمین شود.
	Create(
		ctx context.Context,
		event *model.OutboxEvent,
	) error

	// FetchPendingForRelay تا batchSize رویداد pending را با
	// SELECT ... FOR UPDATE SKIP LOCKED برمی‌گرداند.
	//
	// FOR UPDATE: ردیف‌ها را قفل می‌کند تا اگر چند نمونه از
	// Relay Worker همزمان اجرا شوند، هر ردیف فقط به یکی از آن‌ها
	// برسد.
	//
	// SKIP LOCKED: ردیف‌هایی که توسط تراکنش دیگری قفل شده‌اند
	// را نادیده می‌گیرد (به‌جای بلاک شدن تا آزاد شدن قفل).
	//
	// این متد باید حتماً داخل یک تراکنش صدا زده شود، چون قفل
	// FOR UPDATE تا پایان تراکنش نگه داشته می‌شود.
	FetchPendingForRelay(
		ctx context.Context,
		batchSize int,
	) ([]*model.OutboxEvent, error)

	// MarkPublished وضعیت رویداد را به published تغییر می‌دهد
	// و published_at را ست می‌کند
	MarkPublished(
		ctx context.Context,
		eventID uuid.UUID,
	) error

	// IncrementRetry تعداد تلاش‌ها را زیاد می‌کند و آخرین خطا
	// را ثبت می‌کند. اگر retryCount به سقف مجاز برسد، Relay
	// Worker خودش MarkFailed را صدا می‌زند
	IncrementRetry(
		ctx context.Context,
		eventID uuid.UUID,
		lastErr string,
	) error

	// MarkFailed رویداد را در وضعیت failed قرار می‌دهد. این
	// زمانی صدا زده می‌شود که تعداد تلاش‌ها از سقف گذشته باشد
	MarkFailed(
		ctx context.Context,
		eventID uuid.UUID,
		reason string,
	) error
}

type outboxRepository struct {
	db postgres.DBTX
}

func NewOutboxRepository(db postgres.DBTX) OutboxRepository {
	return &outboxRepository{db: db}
}

// Create یک رویداد را در جدول outbox_events درج می‌کند
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
		event.AggregateType = "order"
	}

	if event.Status == "" {
		event.Status = model.OutboxStatusPending
	}

	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}

	query := `
		INSERT INTO outbox_events (
			id,
			event_type,
			exchange,
			routing_key,
			payload,
			aggregate_id,
			aggregate_type,
			status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		event.ID,
		event.EventType,
		event.Exchange,
		event.RoutingKey,
		event.Payload,
		event.AggregateID,
		event.AggregateType,
		event.Status,
	).Scan(
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

// FetchPendingForRelay تا batchSize رویداد pending را با قفل
// ردیفی برمی‌گرداند. حتماً باید داخل تراکنش صدا زده شود
func (
	r *outboxRepository,
) FetchPendingForRelay(
	ctx context.Context,
	batchSize int,
) ([]*model.OutboxEvent, error) {

	if batchSize <= 0 {
		batchSize = 50
	}

	query := `
		SELECT
			id,
			event_type,
			exchange,
			routing_key,
			payload,
			aggregate_id,
			aggregate_type,
			status,
			retry_count,
			last_error,
			created_at,
			published_at,
			updated_at
		FROM outbox_events
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`

	rows, err := r.db.Query(ctx, query, batchSize)
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to fetch pending outbox events",
		)
	}
	defer rows.Close()

	events := make([]*model.OutboxEvent, 0, batchSize)

	for rows.Next() {
		event := &model.OutboxEvent{}

		if err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.Exchange,
			&event.RoutingKey,
			&event.Payload,
			&event.AggregateID,
			&event.AggregateType,
			&event.Status,
			&event.RetryCount,
			&event.LastError,
			&event.CreatedAt,
			&event.PublishedAt,
			&event.UpdatedAt,
		); err != nil {
			return nil, appErrors.Wrap(
				appErrors.KindInternal,
				err,
				"failed to scan outbox event row",
			)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"error iterating outbox event rows",
		)
	}

	return events, nil
}

// MarkPublished وضعیت رویداد را به published تغییر می‌دهد
func (
	r *outboxRepository,
) MarkPublished(
	ctx context.Context,
	eventID uuid.UUID,
) error {

	if eventID == uuid.Nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"event id cannot be empty",
		)
	}

	query := `
		UPDATE outbox_events
		SET
			status       = 'published',
			published_at = NOW(),
			updated_at   = NOW()
		WHERE id = $1 AND status = 'pending'
	`

	result, err := r.db.Exec(ctx, query, eventID)
	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to mark outbox event as published",
		)
	}

	if result.RowsAffected() == 0 {
		// یا رویداد وجود ندارد، یا وضعیتش دیگر pending نیست.
		// این خطا در Relay Worker لاگ می‌شود ولی جریان را متوقف
		// نمی‌کند
		return appErrors.New(
			appErrors.KindNotFound,
			"outbox event not found or not in pending state",
		)
	}

	return nil
}

// IncrementRetry تعداد تلاش‌های ناموفق را یک واحد زیاد می‌کند
func (
	r *outboxRepository,
) IncrementRetry(
	ctx context.Context,
	eventID uuid.UUID,
	lastErr string,
) error {

	if eventID == uuid.Nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"event id cannot be empty",
		)
	}

	query := `
		UPDATE outbox_events
		SET
			retry_count = retry_count + 1,
			last_error  = $1,
			updated_at  = NOW()
		WHERE id = $2 AND status = 'pending'
	`

	result, err := r.db.Exec(ctx, query, lastErr, eventID)
	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to increment outbox retry count",
		)
	}

	if result.RowsAffected() == 0 {
		return appErrors.New(
			appErrors.KindNotFound,
			"outbox event not found or not in pending state",
		)
	}

	return nil
}

// MarkFailed رویداد را به وضعیت failed می‌برد
func (
	r *outboxRepository,
) MarkFailed(
	ctx context.Context,
	eventID uuid.UUID,
	reason string,
) error {

	if eventID == uuid.Nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"event id cannot be empty",
		)
	}

	query := `
		UPDATE outbox_events
		SET
			status     = 'failed',
			last_error = $1,
			updated_at = NOW()
		WHERE id = $2 AND status = 'pending'
	`

	result, err := r.db.Exec(ctx, query, reason, eventID)
	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to mark outbox event as failed",
		)
	}

	if result.RowsAffected() == 0 {
		return appErrors.New(
			appErrors.KindNotFound,
			"outbox event not found or not in pending state",
		)
	}

	return nil
}
