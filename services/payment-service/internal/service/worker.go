// services/payment-service/internal/service/worker.go

package service

import (
	"context"
	"encoding/json"
	"log"
	"payment-service/internal/model"
	"payment-service/internal/repository"
	appErrors "pkg/errors"
	"pkg/events"
	"time"

	"github.com/google/uuid"
)

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

	// بررسی اینکه هیچ پرداخت خالی وجود دارد
	if len(payments) == 0 {
		return nil
	}

	// ۲. برای هر پرداخت، تغییر وضعیت + ثبت رویداد را در یک
	// تراکنش اتمیک انجام بده
	for _, payment := range payments {

		if err := s.expireSinglePayment(ctx, payment); err != nil {
			// خطای یک پرداخت نباید بقیه را متوقف کند
			log.Printf(
				"payment-service: failed to expire payment %s for order %s: %v",
				payment.ID,
				payment.OrderID,
				err,
			)
			continue
		}

		log.Printf(
			"payment-service: successfully expired payment %s for order %s",
			payment.ID,
			payment.OrderID,
		)
	}

	return nil
}

// expireSinglePayment یک پرداخت را منقضی می‌کند و رویداد مربوطه
// را در همان تراکنش در outbox ثبت می‌کند
func (
	s *PaymentService,
) expireSinglePayment(
	ctx context.Context,
	payment *model.Payment,
) error {

	// ۱. ساخت payload رویداد payment.expired
	// (این رویداد در pkg/events/payment.go اضافه خواهد شد)
	eventPayload := events.PaymentExpired{
		EventID:   uuid.New(),
		OrderID:   payment.OrderID,
		PaymentID: payment.ID,
		ExpiredAt: time.Now().UTC(),
	}

	payloadBytes, err := json.Marshal(eventPayload)
	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to marshal payment.expired event",
		)
	}

	// ۲. ساخت OutboxEvent
	orderID := payment.OrderID
	outboxEvent := &model.OutboxEvent{
		EventType:     model.OutboxEventTypePaymentExpired,
		Exchange:      events.ExchangePaymentEvents,
		RoutingKey:    events.RoutingKeyPaymentExpired,
		Payload:       payloadBytes,
		AggregateID:   &orderID,
		AggregateType: "payment",
		Status:        model.OutboxStatusPending,
	}

	// ۳. اجرای اتمیک در یک تراکنش: تغییر وضعیت + ثبت رویداد
	err = s.txManager.ExecInTx(
		ctx,
		func(repos repository.Repositories) error {

			return repos.Payment.MarkExpiredAtomic(
				ctx,
				payment.ID,
				outboxEvent,
			)
		},
	)
	if err != nil {
		return err
	}

	log.Printf(
		"payment-service: outbox event %s created for expired payment %s",
		outboxEvent.ID,
		payment.ID,
	)

	return nil
}
