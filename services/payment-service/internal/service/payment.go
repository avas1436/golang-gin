// services/payment-service/internal/service/payment.go

package service

import (
	"context"
	"fmt"
	"log"

	"pkg/auth"
	appErrors "pkg/errors"
	pb "pkg/proto/payment"

	"payment-service/config"
	"payment-service/internal/client"
	"payment-service/internal/model"
	"payment-service/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentService struct {
	pool        *pgxpool.Pool
	paymentRepo repository.PaymentRepository
	outboxRepo  repository.OutboxRepository
	gateway     client.GatewayClient
	publisher   EventPublisher
	cfg         *config.Config
}

// VerifyPayment پس از هدایت کاربر از بانک به HTTP Callback فراخوانی می‌شود.
func (
	s *PaymentService,
) VerifyPayment(
	ctx context.Context,
	authority string,
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

	// ۴. استعلام تاییدیه پرداخت از API زرین‌پال (Verify)
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

		// در صورت شکست درگاه
		if updateErr := s.paymentRepo.UpdateFromAwaiting(
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

	// ۵. ثبت تایید موفق و ذخیره RefID شماره پیگیری بانک
	if err := payment.MarkCompleted(
		"zarinpal",
		verifyOutput.RefID,
	); err != nil {

		return nil, err
	}

	// ذخیره در جدول payment
	if err := s.paymentRepo.UpdateFromAwaiting(
		ctx,
		payment,
	); err != nil {

		return nil, err
	}

	// ۶. انتشار رویداد موفقیت پرداخت روی RabbitMQ برای order-service
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
