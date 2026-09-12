// services/order-service/internal/model/order.go

package model

import (
	appErrors "pkg/errors"
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// محتویات یک سفارش
type Order struct {
	ID          uuid.UUID    `db:"id" json:"id"`
	UserID      uuid.UUID    `db:"user_id" json:"user_id"`
	Status      OrderStatus  `db:"status" json:"status"`
	TotalAmount int64        `db:"total_amount" json:"total_amount"`
	Items       []*OrderItem `db:"-" json:"items,omitempty"`
	CreatedAt   time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time    `db:"updated_at" json:"updated_at"`
}

// OrderItem یک ردیف از سفارش است. ProductName و UnitPrice عمداً
// snapshot هستند (نه رفرنس زنده به Product Service)؛ دلیلش را در
// migration توضیح دادیم: تاریخچه‌ی سفارش نباید با تغییر بعدی قیمت/
// نام محصول عوض شود
type OrderItem struct {
	ID          uuid.UUID `db:"id" json:"id"`
	OrderID     uuid.UUID `db:"order_id" json:"order_id"`
	ProductID   uuid.UUID `db:"product_id" json:"product_id"`
	ProductName string    `db:"product_name" json:"product_name"`
	UnitPrice   int64     `db:"unit_price" json:"unit_price"`
	Quantity    int32     `db:"quantity" json:"quantity"`
	Subtotal    int64     `db:"sub_total" json:"sub_total"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// مجموع تمام آیتم‌های سفارش
func (o *Order) CalculateTotal() int64 {

	var total int64

	for _, item := range o.Items {
		total += item.Subtotal
	}

	return total
}

// پیش از ثبت اولیه‌ی سفارش صدا زده می‌شود
func (o *Order) Validate() error {

	if len(o.Items) == 0 {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"order must contain at least one item",
		)
	}

	for _, item := range o.Items {
		if item.Quantity <= 0 {
			return appErrors.New(
				appErrors.KindInvalidInput,
				"item quantity must be positive",
			)
		}

		if item.UnitPrice < 0 {
			return appErrors.New(
				appErrors.KindInvalidInput,
				"item unit price cannot be negative",
			)
		}
	}

	if o.TotalAmount != o.CalculateTotal() {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"total amount does not match sum of items",
		)
	}

	return nil
}

// CanTransitionTo قوانین گذار وضعیت Saga را اعمال می‌کند: فقط از
// pending می‌شود خارج شد، و pending فقط می‌تواند confirmed یا
// cancelled بشود. این یک لایه‌ی دفاعی جدا از idempotency است
func (o *Order) CanTransitionTo(next OrderStatus) bool {

	if o.Status != OrderStatusPending {
		// وضعیت‌های نهایی دیگر قابل تغییر نیستند
		return false
	}

	return next == OrderStatusConfirmed || next == OrderStatusCancelled
}
