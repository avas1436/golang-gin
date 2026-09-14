// pkg/events/stock.go

package events

// در این فایل زبان مشترک ارتباط بین سرویس های سبد خرید نوشته شده است

import (
	"time"

	"github.com/google/uuid"
)

// ربیت ام کیو بر اساس این کلید تصمیم میگیرد پیام را به کدام صف
// یا مصرف کننده ارسال کند
const (
	RoutingKeyStockReleaseRequested = "stock.release.requested"
	RoutingKeyStockConfirmRequested = "stock.confirm.requested"
	RoutingKeyOrderCreated          = "order.created"
)

// محتوی درخواست غیر همزمانی است که ربیت ام کیو منتقل میکند
// برای آزاد کردن تعداد رزرو یا همان عملیات جبرانی
type StockReleaseRequested struct {
	EventID     uuid.UUID `json:"event_id"`
	ProductID   uuid.UUID `json:"product_id"`
	Quantity    int32     `json:"quantity"`
	Reason      string    `json:"reason"`
	RequestedAt time.Time `json:"requested_at"`
}

// برای قطعی کردن کاهش موجودی محصول
type StockConfirmRequested struct {
	EventID     uuid.UUID `json:"event_id"`
	OrderID     uuid.UUID `json:"order_id"`
	ProductID   uuid.UUID `json:"product_id"`
	Quantity    int32     `json:"quantity"`
	RequestedAt time.Time `json:"requested_at"`
}

// اعلام ساخته شدن یک سفارش جدید
// در سرویس پرداخت و انبار داری و ارسال نوتیف میتواند استفاده شود
type OrderCreatedItem struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int32     `json:"quantity"`
	UnitPrice int64     `json:"unit_price"`
}

type OrderCreated struct {
	EventID     uuid.UUID          `json:"event_id"`
	OrderID     uuid.UUID          `json:"order_id"`
	UserID      uuid.UUID          `json:"user_id"`
	TotalAmount int64              `json:"total_amount"`
	Items       []OrderCreatedItem `json:"items"`
	CreatedAt   time.Time          `json:"created_at"`
}
