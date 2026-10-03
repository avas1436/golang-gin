// services/payment-service/internal/service/saga.go

package service

import (
	"context"
	"fmt"
	"log"
	"payment-service/internal/client"
	"payment-service/internal/model"
	"payment-service/internal/repository"
	"pkg/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

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
