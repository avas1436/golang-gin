// services/payment-service/internal/client/gateway.go

package client

import (
	"context"
)

// PaymentRequestInput داده‌های لازم برای ایجاد یک درخواست پرداخت است.
type PaymentRequestInput struct {
	Amount      int64
	Description string
	CallbackURL string
	Email       string
	Mobile      string
}

// PaymentRequestOutput نتیجه ایجاد درخواست پرداخت در Gateway است.
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
	Success bool   // آیا پرداخت با موفقیت تأیید شده است؟
	CardPan string // شماره کارت ماسک‌شده پرداخت‌کننده
}

// GatewayClient قرارداد مشترک تمام درگاه‌های پرداخت است.
//
// # Service فقط با این interface کار می‌کند و
//
//	به پیاده‌سازی خاصی مثل ZarinPal وابسته نیست.
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
