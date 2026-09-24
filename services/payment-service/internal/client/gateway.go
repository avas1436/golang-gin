// services/payment-service/internal/client/gateway.go

package client

import (
	"context"

	"github.com/google/uuid"
)

// PaymentRequestInput داده‌های مورد نیاز برای شروع یک تراکنش
type PaymentRequestInput struct {
	OrderID     uuid.UUID
	Amount      int64
	Description string
	CallbackURL string
	Email       string
	Mobile      string
}

// PaymentRequestOutput خروجی حاصل از ثبت درخواست در درگاه
type PaymentRequestOutput struct {
	Authority   string // شناسه منحصر به فرد تراکنش در زرین‌پال
	RedirectURL string // لینک کامل هدایت کاربر به درگاه بانک
}

// PaymentVerifyInput داده‌های لازم جهت استعلام و تایید نهایی پرداخت
type PaymentVerifyInput struct {
	Authority string
	Amount    int64
}

// PaymentVerifyOutput نتیجه تاییدیه نهایی از سوی درگاه
type PaymentVerifyOutput struct {
	RefID   string // شماره پیگیری دیجیتال (RRN) صادر شده توسط بانک
	Success bool
	CardPan string // شماره کارت ماسک‌شده پرداخت‌کننده
}

// GatewayClient اینترفیس عمومی متصل‌کننده سرویس به درگاه‌های پرداخت است
type GatewayClient interface {
	RequestPayment(
		ctx context.Context,
		input PaymentRequestInput,
	) (
		*PaymentRequestOutput,
		error,
	)

	VerifyPayment(
		ctx context.Context,
		input PaymentVerifyInput,
	) (
		*PaymentVerifyOutput,
		error,
	)
}
