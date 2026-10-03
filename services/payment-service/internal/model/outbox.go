// services/payment-service/internal/model/outbox.go

package model

import (
	"pkg/events"
	"time"

	"github.com/google/uuid"
)

// OutboxEventStatus وضعیت پردازش یک رویداد در جدول outbox را
// مشخص می‌کند
type OutboxEventStatus string

const (
	OutboxStatusPending   OutboxEventStatus = "pending"
	OutboxStatusPublished OutboxEventStatus = "published"
	OutboxStatusFailed    OutboxEventStatus = "failed"
)

// OutboxEventType انواع رویدادهایی است که payment-service
// می‌تواند در Outbox ثبت کند
type OutboxEventType string

const (
	// OutboxEventTypePaymentFailed همان رویداد payment.failed است
	// که در جریان Verify ناموفق یا انقضای پرداخت منتشر می‌شود
	OutboxEventTypePaymentFailed OutboxEventType = events.RoutingKeyPaymentFailed

	// OutboxEventTypePaymentExpired رویداد جدیدی است که هنگام
	// انقضای پرداخت‌های بلاتکلیف منتشر می‌شود و از payment.failed
	// جدا نگه داشته می‌شود تا Consumerها بتوانند بین «رد شدن توسط
	// درگاه» و «منقضی شدن بدون اقدام کاربر» تفاوت قائل شوند
	OutboxEventTypePaymentExpired OutboxEventType = events.RoutingKeyPaymentExpired
)

// OutboxEvent یک ردیف از جدول outbox_events را نگه می‌دارد.
//
// این جدول برای الگوی Transactional Outbox استفاده می‌شود: ثبت
// رویداد در همان تراکنش دیتابیسی که تغییر اصلی رخ می‌دهد، تضمین
// می‌کند که یا هر دو اتفاق می‌افتند یا هیچ‌کدام. سپس یک Poller
// جداگانه این رویدادها را
// می‌خواند و به RabbitMQ می‌فرستد و status را published می‌کند
type OutboxEvent struct {
	ID            uuid.UUID         `db:"id"`
	EventType     OutboxEventType   `db:"event_type"`
	Exchange      string            `db:"exchange"`
	RoutingKey    string            `db:"routing_key"`
	Payload       []byte            `db:"payload"`
	AggregateID   *uuid.UUID        `db:"aggregate_id"`
	AggregateType string            `db:"aggregate_type"`
	Status        OutboxEventStatus `db:"status"`
	RetryCount    int               `db:"retry_count"`
	LastError     *string           `db:"last_error"`
	CreatedAt     time.Time         `db:"created_at"`
	PublishedAt   *time.Time        `db:"published_at"`
	UpdatedAt     time.Time         `db:"updated_at"`
}
