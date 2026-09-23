// services/payment-service/internal/messaging/order_created_consumer.go

package messaging

import (
	"context"
	"encoding/json"
	"log"

	"pkg/events"
	"pkg/postgres"

	"payment-service/internal/model"
	"payment-service/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const eventTypeOrderCreated = "order.created"

// eventTypeOrderCreated پیام‌های order.created را مصرف
// می‌کند. و یک رکورد پرداخت در حالت pending قرار میدهد
type OrderCreatedConsumer struct {
	pool *pgxpool.Pool
}

func NewOrderCreatedConsumer(
	pool *pgxpool.Pool,
) *OrderCreatedConsumer {

	return &OrderCreatedConsumer{
		pool: pool,
	}
}

// در این تابع هم دو فرایند در یک تراکنش انجام نمیشوند
func (
	h *OrderCreatedConsumer,
) Handle(
	ctx context.Context,
	body []byte,
) error {

	// اعتبار سنجی متن پیام
	var event events.OrderCreated

	// دسریلایز رویداد
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf(
			"payment-service: failed to unmarshal order created event: %v",
			err,
		)
		return nil
	}

	// اعتبار سنجی آیدی رویداد
	if event.EventID == uuid.Nil {
		log.Printf(
			"payment-service: order.created event has empty event_id",
		)

		return nil
	}

	// اعتبار سنجی آیدی سفارش
	if event.OrderID == uuid.Nil {
		log.Printf(
			"payment-service: order.created event %s has empty order_id",
			event.EventID,
		)

		return nil
	}

	return postgres.WithTx(
		ctx,
		h.pool,
		func(tx pgx.Tx) error {

			eventRepo := repository.NewEventRepository(tx)
			paymentRepo := repository.NewPaymentRepository(tx)

			// ساخت یک نسخه اولیه از سفارش
			payment := model.NewPendingPayment(
				event.OrderID,
				event.UserID,
				event.TotalAmount,
			)

			// ذخیره رویداد در جدول دیتابیس
			alreadyProcessed, err := eventRepo.MarkProcessed(
				ctx,
				event.EventID,
				eventTypeOrderCreated,
				&event.OrderID,
				&payment.ID,
			)

			// اگر خطا داد
			if err != nil {
				log.Printf(
					"payment-service: failed to record event %s: %v",
					event.EventID,
					err,
				)

				return err
			}

			// اگر رویداد تکراری بود
			if alreadyProcessed {
				log.Printf(
					"payment-service: event %s already processed, skipping",
					event.EventID,
				)

				return nil
			}

			// ساخت یک رکورد pending پرداخت
			if err := paymentRepo.Create(
				ctx,
				payment,
			); err != nil {

				log.Printf(
					"payment-service: failed to create pending payment for order %s: %v",
					event.OrderID,
					err,
				)

				return err

			}

			return nil
		},
	)
}
