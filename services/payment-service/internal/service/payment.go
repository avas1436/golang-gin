// services/payment-service/internal/service/payment.go

package service

import (
	"context"
	"fmt"
	"log"

	appErrors "pkg/errors"

	"payment-service/config"
	"payment-service/internal/client"
	"payment-service/internal/model"
	"payment-service/internal/repository"
)

// PaymentService مسئولیت مدیریت جریانات اصلی پرداخت را بر عهده دارد.
type PaymentService struct {
	txManager   repository.TxManager
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

		// اگر به دلیل همزمانی خطا خورد، دوباره دیتابیس را استعلام کن
		if appErrors.GetKind(err) == appErrors.KindAlreadyExists ||
			appErrors.GetKind(err) == appErrors.KindConflict {

			return s.handleConcurrentVerifyFallback(ctx, authority)

		}

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

// handleConcurrentVerifyFallback در صورت بروز Race Condition، آخرین وضعیت رکورد را استعلام کرده و به صورت Idempotent پاسخ می‌دهد.
func (
	s *PaymentService,
) handleConcurrentVerifyFallback(
	ctx context.Context,
	authority string,
) (
	*model.Payment,
	error,
) {

	latestPayment, err := s.paymentRepo.GetByAuthority(ctx, authority)
	if err != nil {
		return nil, err
	}

	// اگر درخواست همزمانِ قبلی وضعیت را به Completed برده باشد،
	// همان رکورد بدون انتشار مجدد Event برگردانده می‌شود.
	if latestPayment.Status == model.PaymentStatusCompleted {
		log.Printf(
			"payment-service: resolved concurrent verify as idempotent success for payment %s",
			latestPayment.ID,
		)
		return latestPayment, nil
	}

	return latestPayment, nil
}
