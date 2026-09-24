// services/payment-service/internal/model/payment.go

package model

import (
	appErrors "pkg/errors"
	"time"

	"github.com/google/uuid"
)

// تعریف وضعیت‌های ممکن برای یک پرداخت
type PaymentStatus string

const (
	PaymentStatusPending         PaymentStatus = "pending"
	PaymentStatusCompleted       PaymentStatus = "completed"
	PaymentStatusFailed          PaymentStatus = "failed"
	PaymentStatusCanceled        PaymentStatus = "canceled"
	PaymentStatusRefunded        PaymentStatus = "refunded"
	PaymentStatusExpired         PaymentStatus = "expired"
	PaymentStatusAwaitingPayment PaymentStatus = "awaiting"
)

// تعریف ارزهای ممکن برای پرداخت
type Currency string

const (
	CurrencyIRR Currency = "IRR"
)

// نتیجه‌ی رکورد پرداخت مرتبط با یک سفارش. این سرویس هیچ API سینکی
// برای فراخوانی از بیرون (کلاینت/Gateway) ندارد؛ تنها راه ورودش
// مصرف رویداد order.created است. تنها متد gRPC آن (GetPaymentByOrderID)
// صرفاً برای دیباگ/ادمین است، نه بخشی از جریان اصلی Saga
type Payment struct {
	ID      uuid.UUID `db:"id" json:"id"`
	OrderID uuid.UUID `db:"order_id" json:"order_id"`

	// دنرمالایز شده از رویداد order.created؛ برای گزارش‌گیری ادمین
	// بدون نیاز به صدا زدن User Service
	UserID uuid.UUID `db:"user_id" json:"user_id"`

	// ارز پرداخت (IRR)
	Currency Currency `db:"currency" json:"currency"`

	// snapshot از total_amount سفارش در لحظه‌ی دریافت رویداد
	Amount int64 `db:"amount" json:"amount"`

	Status PaymentStatus `db:"status" json:"status"`

	// نام پذیرنده پرداخت
	GatewayName *string `db:"gateway_name" json:"gateway_name,omitempty"`

	// شماره پیگیری نهایی بانک (RefID)
	GatewayRefID *string `db:"gateway_ref_id" json:"gateway_ref_id,omitempty"`

	// کد Authority اولیه زرین‌پال
	Authority *string `json:"authority,omitempty"`

	// لینک پرداخت بانک
	RedirectURL *string `json:"redirect_url,omitempty"`

	FailureReason *string `db:"failure_reason" json:"failure_reason,omitempty"`

	// برای نگهداری داده‌های JSONB متغیر از سمت درگاه
	Metadata map[string]any `db:"metadata" json:"metadata"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// NewPendingPayment یک رکورد پرداخت را در وضعیت اولیه (pending)
// می‌سازد؛ درست بعد از دریافت و اعتبارسنجی رویداد order.created و
// پیش از شبیه‌سازی نتیجه‌ی درگاه بانکی صدا زده می‌شود
func NewPendingPayment(
	orderID uuid.UUID,
	userID uuid.UUID,
	amount int64,
) *Payment {

	now := time.Now().UTC()

	return &Payment{
		ID:        uuid.New(),
		OrderID:   orderID,
		UserID:    userID,
		Amount:    amount,
		Currency:  CurrencyIRR,
		Status:    PaymentStatusPending,
		Metadata:  make(map[string]any),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ثبت Authority و RedirectURL و تغییر وضعیت به در انتظار پرداخت
func (
	p *Payment,
) MarkAwaitingPayment(
	gatewayName,
	authority,
	redirectURL string,
) error {

	if p.Status != PaymentStatusPending {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"only pending payments can transition to awaiting payment",
		)
	}

	p.GatewayName = &gatewayName
	p.Authority = &authority
	p.RedirectURL = &redirectURL
	p.Status = PaymentStatusAwaitingPayment
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// پیش از درج اولیه‌ی رکورد پرداخت صدا زده می‌شود
func (p *Payment) Validate() error {

	if p.OrderID == uuid.Nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"payment must reference an order",
		)
	}

	if p.UserID == uuid.Nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"payment must reference a user",
		)
	}

	if p.Amount <= 0 {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"payment amount must be greater than zero",
		)
	}

	if p.Currency == "" {
		p.Currency = CurrencyIRR
	}

	return nil
}

// CanTransitionTo قوانین گذار وضعیت را اعمال می‌کند
func (p *Payment) CanTransitionTo(next PaymentStatus) bool {
	// اگر پرداخت در یکی از وضعیت‌های نهایی باشد، دیگر قابل تغییر نیست
	switch p.Status {

	case PaymentStatusCompleted:
		// پرداخت موفق فقط می‌تواند Refund شود
		return next == PaymentStatusRefunded

	case PaymentStatusFailed,
		PaymentStatusCanceled,
		PaymentStatusExpired,
		PaymentStatusRefunded:

		return false

	case PaymentStatusPending:
		// از pending به هر وضعیتی مجاز است
		return next == PaymentStatusCompleted ||
			next == PaymentStatusFailed ||
			next == PaymentStatusCanceled ||
			next == PaymentStatusExpired

	default:
		return false
	}
}

// MarkCompleted نتیجه‌ی موفق شبیه‌سازی/تایید درگاه را روی مدل اعمال می‌کند
func (p *Payment) MarkCompleted(gatewayName, gatewayRefID string) error {

	// بررسی اینکه اصلا مجاز به تغییر هست یا نه
	if !p.CanTransitionTo(PaymentStatusCompleted) {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"payment cannot transition to completed from "+string(p.Status),
		)
	}

	p.Status = PaymentStatusCompleted
	p.GatewayName = &gatewayName
	p.GatewayRefID = &gatewayRefID
	p.FailureReason = nil
	p.UpdatedAt = time.Now().UTC()

	return nil
}

// MarkFailed نتیجه‌ی ناموفق شبیه‌سازی/تایید درگاه را اعمال می‌کند
func (p *Payment) MarkFailed(reason string) error {

	// بررسی اینکه اصلا مجاز به تغییر هست یا نه
	if !p.CanTransitionTo(PaymentStatusFailed) {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"payment cannot transition to failed from "+string(p.Status),
		)
	}

	p.Status = PaymentStatusFailed
	p.FailureReason = &reason
	p.UpdatedAt = time.Now().UTC()

	return nil
}

// MarkExpired منقضی شدن نشست پرداخت را اعمال می‌کند
func (p *Payment) MarkExpired() error {

	// بررسی اینکه اصلا مجاز به تغییر هست یا نه
	if !p.CanTransitionTo(PaymentStatusExpired) {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"payment cannot transition to expired from "+string(p.Status),
		)
	}

	p.Status = PaymentStatusExpired
	p.UpdatedAt = time.Now().UTC()

	return nil
}
