// pkg/events/payment.go

package events

// این فایل زبان مشترک ارتباط Payment Service با بقیه‌ی سرویس‌ها
// (در حال حاضر فقط Order Service) پس از نتیجه‌ی پرداخت است

import (
	"time"

	"github.com/google/uuid"
)

// ربیت‌ام‌کیو بر اساس این کلیدها تصمیم می‌گیرد پیام به کدام صف برود
const (
	RoutingKeyPaymentCompleted = "payment.completed"
	RoutingKeyPaymentFailed    = "payment.failed"
)

// PaymentCompleted بعد از شبیه‌سازی موفق درگاه بانکی توسط Payment
// Service منتشر می‌شود. Order Service با گوش‌دادن به این رویداد،
// وضعیت سفارش را به confirmed تغییر می‌دهد و ConfirmStock را برای
// Product Service صدا می‌زند (از طریق stock.confirm.requested)
type PaymentCompleted struct {
	EventID   uuid.UUID `json:"event_id"`
	OrderID   uuid.UUID `json:"order_id"`
	PaymentID uuid.UUID `json:"payment_id"`

	// اطلاعات درگاه برای ثبت/دیباگ سمت Order Service (اختیاری)
	GatewayName  string `json:"gateway_name"`
	GatewayRefID string `json:"gateway_ref_id"`

	CompletedAt time.Time `json:"completed_at"`
}

// PaymentFailed بعد از شبیه‌سازی ناموفق درگاه بانکی منتشر می‌شود.
// Order Service با گوش‌دادن به این رویداد، وضعیت سفارش را به
// cancelled تغییر داده و موجودی رزروشده را با انتشار
// stock.release.requested آزاد می‌کند (Compensation)
type PaymentFailed struct {
	EventID   uuid.UUID `json:"event_id"`
	OrderID   uuid.UUID `json:"order_id"`
	PaymentID uuid.UUID `json:"payment_id"`

	// دلیل شکست پرداخت، برای لاگ/دیباگ در Order Service
	Reason string `json:"reason"`

	FailedAt time.Time `json:"failed_at"`
}
