// services/payment-service/internal/messaging/order_created_consumer.go

package messaging

import (
	"context"
	"encoding/json"
	"log"

	"pkg/events"

	"payment-service/internal/service"

	"github.com/google/uuid"
)

// OrderCreatedConsumer تنها مسئول decode کردن پیام order.created و
// اعتبارسنجی سطحی آن است؛ منطق دامنه در
// service.PaymentService.HandleOrderCreated زندگی می‌کند
type OrderCreatedConsumer struct {
	paymentService *service.PaymentService
}

func NewOrderCreatedConsumer(
	paymentService *service.PaymentService,
) *OrderCreatedConsumer {

	return &OrderCreatedConsumer{
		paymentService: paymentService,
	}
}

// Handle امضای rabbitmq.HandlerFunc را دارد
func (
	h *OrderCreatedConsumer,
) Handle(
	ctx context.Context,
	body []byte,
) error {

	// اعتبار سنجی متن پیام
	var event events.OrderCreated

	// دسریالایز رویداد؛ بدنه‌ی خراب با تکرار مجدد درست نمی‌شود، پس
	// nil برمی‌گردانیم تا پیام ACK شود و صف را مسدود نکند (poison
	// message) — دقیقاً همان تصمیم product-service
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

	return h.paymentService.HandleOrderCreated(
		ctx,
		event.EventID,
		event.OrderID,
		event.UserID,
		event.TotalAmount,
	)
}
