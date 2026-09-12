// services/order-service/internal/repository/event.go

package repository

import (
	"context"

	appErrors "pkg/errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventRepository جدول processed_events را مدیریت می‌کند: تنها
// مسیولیتش این است که به مصرف‌کننده‌ی رویداد بگوید
// آیا این event قبلاً پردازش شده یا نه
type EventRepository interface {

	// MarkProcessed سعی می‌کند event را به‌عنوان پردازش‌شده ثبت
	// کند. اگر قبلاً ثبت شده باشد (تکرار پیام از RabbitMQ)،
	// alreadyProcessed برابر true برمی‌گردد و caller باید منطق
	// پردازش را کاملاً رد کند (نه فقط دوباره retry کند)
	MarkProcessed(
		ctx context.Context,
		eventID uuid.UUID,
		eventType string,
		orderID *uuid.UUID,
		paymentID *uuid.UUID,
	) (
		alreadyProcessed bool,
		err error,
	)
}

type eventRepository struct {
	pool *pgxpool.Pool
}

func NewEventRepository(pool *pgxpool.Pool) EventRepository {
	return &eventRepository{pool: pool}
}

// MarkProcessed با یک INSERT ... ON CONFLICT DO NOTHING پیاده‌سازی
// شده، نه با یک SELECT جدا قبل از INSERT. دلیلش همان دلیل
// ReserveStock در Product Service است: بین یک SELECT ("آیا وجود
// دارد؟") و INSERT بعدی، دو مصرف‌کننده‌ی هم‌زمان (که کاملاً ممکن
// است چون RabbitMQ می‌تواند یک پیام را به دو consumer مختلف
// همزمان تحویل بدهد) می‌توانند هر دو فکر کنند "هنوز پردازش نشده"
// و هر دو کارِ اصلی را انجام بدهند. با یک INSERT اتمیک، Primary Key
// خودِ دیتابیس تضمین می‌کند که فقط یکی از آن دو موفق می‌شود
func (
	r *eventRepository,
) MarkProcessed(
	ctx context.Context,
	eventID uuid.UUID,
	eventType string,
	orderID *uuid.UUID,
	paymentID *uuid.UUID,
) (
	bool, error,
) {

	query := `
		INSERT INTO processed_events (
		event_id, 
		event_type, 
		order_id, 
		payment_id
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (event_id) DO NOTHING
	`

	result, err := r.pool.Exec(
		ctx,
		query,
		eventID,
		eventType,
		orderID,
		paymentID,
	)
	if err != nil {
		return false, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to record processed event",
		)
	}

	// اگر هیچ ردیفی درج نشد، یعنی event_id از قبل وجود داشته —
	// این event قبلاً پردازش شده است
	alreadyProcessed := result.RowsAffected() == 0

	return alreadyProcessed, nil
}
