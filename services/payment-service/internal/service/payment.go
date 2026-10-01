// services/payment-service/internal/service/payment.go

package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"pkg/auth"
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

	PublishPaymentInitiated(
		ctx context.Context,
		paymentID uuid.UUID,
		orderID uuid.UUID,
		userID uuid.UUID,
		redirectURL string,
		authority string,
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

	if err != nil {
		return err
	}

	// مدیریت حالت بازیابی از کرش (Crash Recovery).
	if alreadyHandled {
		existingPayment, err := s.paymentRepo.GetByOrderID(ctx, orderID)
		if err != nil {
			return err
		}

		// اگر وضعیت پرداخت همچنان Pending باشد، یعنی تماس با
		// درگاه در نوبت قبل انجام نشده بود.
		// در نتیجه پرداختِ موجود را جایگزین کرده و فرآیند درخواست
		// Authority از درگاه را ادامه می‌دهیم.
		if existingPayment.Status == model.PaymentStatusPending {
			payment = existingPayment
			log.Printf(
				"payment-service: recovering pending payment for order %s after previous crash",
				orderID,
			)
		} else {
			// پرداخت قبلاً تعیین تکلیف شده یا Authority
			// دریافت کرده است؛ پس پردازش تکراری انجام نمی‌شود.
			log.Printf(
				"payment-service: order.created event %s already fully processed for order %s, skipping",
				eventID,
				orderID,
			)
			return nil
		}
	}

	// ساختار درخواست لینک پرداخت از زرین پال
	reqInput := client.PaymentRequestInput{
		Amount: payment.Amount,
		Description: fmt.Sprintf(
			"پرداخت سفارش %s", payment.OrderID.String(),
		),
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
		if markErr := payment.MarkFailed(reason); markErr != nil {
			log.Printf(
				"payment-service: failed to mark payment failed locally: %v",
				markErr,
			)
		}

		// اعمال کردن ساختار در دیتابیس
		if updateErr := s.paymentRepo.UpdateFromPending(
			ctx,
			payment,
		); updateErr != nil {

			log.Printf(
				"payment-service: failed to update payment failed status in db for order %s: %v",
				orderID,
				updateErr,
			)
		}

		// انتشار رویداد شکست خوردن پرداخت
		if pubErr := s.publisher.PublishPaymentFailed(
			ctx,
			payment.ID,
			payment.OrderID,
			reason,
		); pubErr != nil {

			log.Printf(
				"payment-service: failed to publish payment.failed for order %s: %v",
				orderID,
				pubErr,
			)
		}

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

	// انتشار رویداد payment.initiated حاوی redirect_url
	// جهت درج در سفارش یا ارسال نوتیفیکیشن
	if pubErr := s.publisher.PublishPaymentInitiated(
		ctx,
		payment.ID,
		payment.OrderID,
		payment.UserID,
		reqOutput.RedirectURL,
		reqOutput.Authority,
	); pubErr != nil {
		log.Printf(
			"payment-service: failed to publish payment.initiated event for order %s: %v",
			orderID,
			pubErr,
		)
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

	// ۲. مدیریت Idempotency:
	// اگر این پرداخت قبلاً با موفقیت تایید شده، بدون خطای
	// اضافی خود رکورد را برمی‌گردانیم
	if payment.Status == model.PaymentStatusCompleted {
		return payment, nil
	}

	// ۳. بررسی وضعیت مجاز:
	// فقط پرداختی که در وضعیت 'awaiting' باشد می‌تواند استعلام و تایید شود
	if payment.Status != model.PaymentStatusAwaitingPayment {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			fmt.Sprintf(
				"payment cannot be verified from status '%s'",
				payment.Status,
			),
		)
	}

	// ۲. بررسی انصراف کاربر یا خطای درگاه قبل از استعلام
	if zarinpalStatus != "OK" {

		reason := "payment canceled by user or rejected by bank"

		if markErr := payment.MarkFailed(reason); markErr != nil {
			return nil, markErr
		}

		if updateErr := s.paymentRepo.Update(
			ctx,
			payment,
		); updateErr != nil {

			return nil, updateErr
		}

		if pubErr := s.publisher.PublishPaymentFailed(
			ctx,
			payment.ID,
			payment.OrderID,
			reason,
		); pubErr != nil {
			log.Printf(
				"payment-service: failed to publish payment.failed event for order %s: %v",
				payment.OrderID,
				pubErr,
			)
		}

		return payment, nil
	}

	// ۳. استعلام تاییدیه پرداخت از API زرین‌پال (Verify)
	verifyInput := client.PaymentVerifyInput{
		Authority: authority,
		Amount:    payment.Amount,
	}

	// درخواست گرفتن تاییدیه پرداخت از زرین پال
	verifyOutput, err := s.gateway.VerifyPayment(ctx, verifyInput)
	if err != nil {

		// تفکیک خطاهای موقتی از خطاهای قطعی رد تراکنش.
		// اگر خطا از جنس KindInternal باشد، ممکن است پول از حساب
		// کاربر کسر شده باشد.
		// در این حالت پرداخت نباید Fail شود و در وضعیت
		// AwaitingPayment باقی می‌ماند تا فرآیند Reconciliation
		// آن را تعیین تکلیف کند.
		if appErrors.GetKind(err) == appErrors.KindInternal {
			log.Printf(
				"payment-service: transient error during zarinpal verify for order %s (payment %s remains in awaiting state): %v",
				payment.OrderID,
				payment.ID,
				err,
			)
			return nil, err
		}

		// تنها در صورت دریافت خطای قطعی کسب‌وکار، پرداخت Failed می‌شود.
		reason := err.Error()

		if markErr := payment.MarkFailed(reason); markErr != nil {
			return nil, markErr
		}

		if updateErr := s.paymentRepo.Update(
			ctx,
			payment,
		); updateErr != nil {

			return nil, updateErr
		}

		if pubErr := s.publisher.PublishPaymentFailed(
			ctx,
			payment.ID,
			payment.OrderID,
			reason,
		); pubErr != nil {

			log.Printf(
				"payment-service: failed to publish payment.failed event for order %s: %v",
				payment.OrderID,
				pubErr,
			)
		}

		return nil, appErrors.Wrap(
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

	// اجازه دسترسی علاوه بر Admin به خود کاربر صاحب پرداخت
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return nil, appErrors.New(
			appErrors.KindPermissionDenied,
			"you cannot view this payment details",
		)
	}

	if claims.Role != roleAdmin && claims.UserID != payment.UserID.String() {
		return nil, appErrors.New(
			appErrors.KindPermissionDenied,
			"you cannot view this payment details",
		)
	}

	return toProtoPayment(payment), nil
}

// TODO: باید یک ورکر برای اجرای این تابع بسازم
// ExpireStalePayments جهت انقضای دوره‌ای پرداخت‌های معلق
// آزادسازی موجودی‌های رزرو شده در سفارشات رهاشده و تغییر وضعیت
// FSM مدل دامنه به PaymentStatusExpired.
func (
	s *PaymentService,
) ExpireStalePayments(
	ctx context.Context,
	timeout time.Duration,
) error {

	// ۱. استعلام پرداخت‌هایی که بیش از timeout مشخص در وضعیت
	// awaiting مانده‌اند
	payments, err := s.paymentRepo.GetStaleAwaitingPayments(
		ctx,
		timeout,
		100,
	)
	if err != nil {
		return err
	}

	for _, payment := range payments {

		// اعمال وضعیت Expired روی مدل دامنه
		if err := payment.MarkExpired(); err != nil {
			log.Printf(
				"payment-service: failed to mark payment %s as expired locally: %v",
				payment.ID,
				err,
			)
			continue
		}

		// ذخیره اتمیک وضعیت Expired در دیتابیس و ثبت رویداد
		// payment.failed در Outbox
		if err := s.paymentRepo.Update(ctx, payment); err != nil {
			return err
		}

		log.Printf(
			"payment-service: successfully expired stale payment %s for order %s",
			payment.ID,
			payment.OrderID,
		)
	}

	return nil
}
