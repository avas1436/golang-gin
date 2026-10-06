// services/order-service/internal/model/outbox.go

package model

import (
	"time"

	"pkg/events"

	"github.com/google/uuid"
)

// OutboxEventStatus وضعیت پردازش یک رویداد در جدول outbox را
// مشخص می‌کند
type OutboxEventStatus string

const (
	// OutboxStatusPending رویداد درج شده ولی هنوز به RabbitMQ منتشر نشده
	OutboxStatusPending OutboxEventStatus = "pending"

	// OutboxStatusPublished رویداد با موفقیت به RabbitMQ منتشر شد
	OutboxStatusPublished OutboxEventStatus = "published"

	// OutboxStatusFailed انتشار رویداد مکرراً شکست خورده و نیاز
	// به مداخله‌ی دستی یا سیاست retry پیشرفته‌تر دارد
	OutboxStatusFailed OutboxEventStatus = "failed"
)

// OutboxEventType انواع رویدادهایی است که Order Service می‌تواند
// در Outbox ثبت کند. مقادیر از pkg/events گرفته می‌شوند تا از
// هرگونه ناهماهنگی بین routing key واقعی و نوع رویداد جلوگیری شود
type OutboxEventType string

const (
	// OutboxEventTypeOrderCreated رویداد order.created که پس از
	// ثبت موفق سفارش در دیتابیس درج می‌شود
	OutboxEventTypeOrderCreated OutboxEventType = events.RoutingKeyOrderCreated

	// OutboxEventTypeStockReleaseRequested رویداد
	// stock.release.requested برای آزادسازی موجودی رزروشده
	// (چه در جریان compensation و چه در جریان payment.failed)
	OutboxEventTypeStockReleaseRequested OutboxEventType = events.RoutingKeyStockReleaseRequested

	// OutboxEventTypeStockConfirmRequested رویداد
	// stock.confirm.requested برای قطعی کردن کسر موجودی پس از
	// موفقیت پرداخت
	OutboxEventTypeStockConfirmRequested OutboxEventType = events.RoutingKeyStockConfirmRequested
)

// OutboxEvent یک ردیف از جدول outbox_events را نگه می‌دارد.
//
// این جدول برای الگوی Transactional Outbox استفاده می‌شود: ثبت
// رویداد در همان تراکنش دیتابیسی که تغییر اصلی رخ می‌دهد، تضمین
// می‌کند که یا هر دو اتفاق می‌افتند یا هیچ‌کدام. سپس یک Relay
// Worker جداگانه این رویدادها را می‌خواند و به RabbitMQ می‌فرستد
// و status را published می‌کند
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

// NewOutboxEvent یک رویداد Outbox جدید با مقادیر پیش‌فرض
// می‌سازد. این تابع برای کاهش تکرار در لایه‌ی service استفاده
// می‌شود؛ در آنجا فقط eventType، routingKey و payload مشخص
// می‌شوند و بقیه‌ی فیلدها (status = pending، aggregateType = order)
// به‌صورت خودکار پر می‌شوند
func NewOutboxEvent(
	eventType OutboxEventType,
	routingKey string,
	payload []byte,
	aggregateID uuid.UUID,
) *OutboxEvent {
	aggregateIDCopy := aggregateID

	return &OutboxEvent{
		ID:            uuid.New(),
		EventType:     eventType,
		Exchange:      events.ExchangeOrderEvents,
		RoutingKey:    routingKey,
		Payload:       payload,
		AggregateID:   &aggregateIDCopy,
		AggregateType: "order",
		Status:        OutboxStatusPending,
	}
}
