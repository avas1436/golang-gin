// services/payment-service/internal/service/payment.go

package service

import (
	"context"
	"fmt"
	"log"

	appErrors "pkg/errors"
	"pkg/postgres"
	pb "pkg/proto/payment"

	"payment-service/config"
	"payment-service/internal/client"
	"payment-service/internal/model"
	"payment-service/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventPublisher چیزی است که PaymentService برای انتشار نتیجه‌ی
// نهایی پرداخت نیاز دارد. پیاده‌سازی آن در internal/messaging است
type EventPublisher interface {
	PublishPaymentCompleted(
		ctx context.Context,
		paymentID uuid.UUID,
		orderID uuid.UUID,
		gatewayName string,
		gatewayRefID string,
	) error

	PublishPaymentFailed(
		ctx context.Context,
		paymentID uuid.UUID,
		orderID uuid.UUID,
		reason string,
	) error
}

type PaymentService struct {
	pool        *pgxpool.Pool
	paymentRepo repository.PaymentRepository
	gateway     client.GatewayClient
	publisher   EventPublisher
	cfg         *config.Config
}

// HandleOrderCreated زمانی اجرا می‌شود که payment-service
// پیام order.created را دریافت می‌کند.
//
// جریان:
//
//  1. شروع Transaction
//
//  2. ثبت event در processed_events
//
//  3. اگر event قبلاً پردازش شده بود:
//     Transaction تمام می‌شود و ادامه نمی‌دهیم.
//
//  4. ایجاد Payment با وضعیت pending
//
//  5. Commit
//
//  6. تماس با Gateway خارج از DB Transaction
func (
	s *PaymentService,
) HandleOrderCreated(
	ctx context.Context,
	eventID uuid.UUID,
	orderID uuid.UUID,
	userID uuid.UUID,
	amount int64,
) error {

	// ساخت مدل Payment با وضعیت pending
	payment := model.NewPendingPayment(
		orderID,
		userID,
		amount,
	)

	var alreadyHandled bool

	err := postgres.WithTx(
		ctx,
		s.pool,
		func(tx pgx.Tx) error {

			// این Repository مخصوص همین Transaction است.
			//
			// بنابراین تمام Queryهای paymentRepo داخل همین tx
			// اجرا می‌شوند.
			txPaymentRepo := repository.NewPaymentRepository(tx)

			// Event repository نیز روی همان Transaction کار می‌کند.
			eventRepo := repository.NewEventRepository(tx)

			alreadyProcessed, err := eventRepo.MarkProcessed(
				ctx,
				eventID,
				"order.created",
				&orderID,
				nil,
			)
			if err != nil {
				return err
			}

			if alreadyProcessed {
				log.Printf(
					"payment-service: order.created event %s already processed, skipping",
					eventID,
				)
				alreadyHandled = true
				return nil
			}

			return txPaymentRepo.Create(ctx, payment)
		},
	)

	if err != nil || alreadyHandled {
		return err
	}

	// ساختار درخواست لینک پرداخت از زرین پال
	reqInput := client.PaymentRequestInput{
		Amount:      payment.Amount,
		Description: fmt.Sprintf("پرداخت سفارش %s", payment.OrderID.String()),
		CallbackURL: s.cfg.ZarinPal.PaymentCallbackURL,
	}

	// درخواست لینک پرداخت از درگاه زرین‌پال
	reqOutput, err := s.gateway.RequestPayment(ctx, reqInput)

	// اگر شکست خورد
	if err != nil {

		// ساخت فرمت دلیل شکست درخواست
		reason := fmt.Sprintf(
			"failed to get authority from zarinpal: %v",
			err,
		)

		// تبدیل ساختار پرداخت به شکست خورده
		_ = payment.MarkFailed(reason)

		// اعمال کردن ساختار در دیتابیس
		_ = s.paymentRepo.UpdateFromPending(ctx, payment)

		// انتشار رویداد شکست خوردن پرداخت
		_ = s.publisher.PublishPaymentFailed(
			ctx,
			payment.ID,
			payment.OrderID,
			reason,
		)

		log.Printf(
			"payment-service: failed to initiate zarinpal payment for order %s: %v",
			orderID,
			err,
		)

		return nil
	}

	// ثبت Authority و RedirectURL در دیتابیس و تغییر وضعیت به AWAITING_PAYMENT
	if err := payment.MarkAwaitingPayment(
		"zarinpal",
		reqOutput.Authority,
		reqOutput.RedirectURL,
	); err != nil {

		return err
	}

	// ثبت نهایی وضعیت پرداخت
	if err := s.paymentRepo.UpdateFromPending(ctx, payment); err != nil {

		log.Printf(
			"payment-service: failed to update payment awaiting status for order %s: %v",
			orderID,
			err,
		)

		return err
	}

	log.Printf(
		"payment-service: payment initialized for order %s, authority: %s",
		orderID,
		reqOutput.Authority,
	)

	return nil
}

// VerifyPayment پس از هدایت کاربر از بانک به HTTP Callback فراخوانی می‌شود.
func (
	s *PaymentService,
) VerifyPayment(
	ctx context.Context,
	authority string,
	zarinpalStatus string,
) (
	*model.Payment,
	error,
) {

	// ۱. پیدا کردن رکورد پرداخت بر اساس Authority
	payment, err := s.paymentRepo.GetByAuthority(ctx, authority)
	if err != nil {
		return nil, err
	}

	// Idempotency Check: اگر پرداخت قبلاً تعیین تکلیف شده است
	if !payment.CanTransitionTo(model.PaymentStatusCompleted) {
		return payment, nil
	}

	// ۲. بررسی انصراف کاربر یا خطای درگاه قبل از استعلام
	if zarinpalStatus != "OK" {

		reason := "payment canceled by user or rejected by bank"

		_ = payment.MarkFailed(reason)

		_ = s.paymentRepo.Update(ctx, payment)

		_ = s.publisher.PublishPaymentFailed(
			ctx,
			payment.ID,
			payment.OrderID,
			reason,
		)

		return payment, nil
	}

	// ۳. استعلام تاییدیه پرداخت از API زرین‌پال (Verify)
	verifyInput := client.PaymentVerifyInput{
		Authority: authority,
		Amount:    payment.Amount,
	}

	// درخواست گرفتن تاییدیه پرداخت از زرین پال
	verifyOutput, err := s.gateway.VerifyPayment(ctx, verifyInput)
	if err != nil || !verifyOutput.Success {

		reason := "zarinpal payment verification failed"

		if err != nil {
			reason = err.Error()
		}

		_ = payment.MarkFailed(reason)

		_ = s.paymentRepo.Update(ctx, payment)

		_ = s.publisher.PublishPaymentFailed(
			ctx,
			payment.ID,
			payment.OrderID,
			reason,
		)

		return payment, appErrors.Wrap(
			appErrors.KindInvalidInput,
			err,
			"payment verification failed",
		)
	}

	// ۴. ثبت تایید موفق و ذخیره RefID (شماره پیگیری بانک)
	if err := payment.MarkCompleted(
		"zarinpal",
		verifyOutput.RefID,
	); err != nil {

		return nil, err
	}

	// ذخیره در جدول payment
	if err := s.paymentRepo.Update(ctx, payment); err != nil {
		return nil, err
	}

	// ۵. انتشار رویداد موفقیت پرداخت روی RabbitMQ برای order-service
	if err := s.publisher.PublishPaymentCompleted(
		ctx,
		payment.ID,
		payment.OrderID,
		"zarinpal",
		verifyOutput.RefID,
	); err != nil {

		log.Printf("payment-service: payment verified but failed to publish completed event for order %s: %v",
			payment.OrderID,
			err,
		)

	}

	return payment, nil
}

// GetPaymentByOrderID تنها متد gRPC این سرویس است؛ صرفاً برای
// دیباگ/ادمین است و بخشی از جریان اصلی Saga نیست
func (
	s *PaymentService,
) GetPaymentByOrderID(
	ctx context.Context,
	req *pb.GetPaymentByOrderIDRequest,
) (
	*pb.Payment,
	error,
) {

	if req == nil {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"get payment by order id request is nil",
		)
	}

	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"invalid order id",
		)
	}

	payment, err := s.paymentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	return toProtoPayment(payment), nil
}
