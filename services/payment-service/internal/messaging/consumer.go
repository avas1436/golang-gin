// services/payment-service/internal/messaging/consumer.go

package messaging

import (
	"context"
	"encoding/json"

	appErrors "pkg/errors"
	"pkg/events"
	"pkg/rabbitmq"

	"payment-service/internal/service"

	"github.com/google/uuid"
)

// OrderEventConsumer مسئول decode کردن پیام order.created و
// اعتبارسنجی سطحی آن است؛ منطق دامنه در
type OrderEventConsumer struct {
	consumerCreated *rabbitmq.Consumer
	paymentService  *service.PaymentService
}

func NewOrderEventConsumer(
	conn *rabbitmq.Connection,
	paymentService *service.PaymentService,
) (
	*OrderEventConsumer,
	error,
) {

	// ۱. کانال اختصاصی برای صف پرداخت موفق
	chCreated, err := conn.Channel()
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to open channel for order created consumer",
		)
	}

	consumerCreated, err := rabbitmq.NewConsumer(chCreated)
	if err != nil {
		_ = chCreated.Close()
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to initialize order created consumer",
		)
	}

	if err := consumerCreated.BindQueue(
		events.QueuePaymentOrderCreated,
		events.ExchangeOrderEvents,
		events.RoutingKeyOrderCreated,
	); err != nil {
		_ = chCreated.Close()
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to bind queue to order created routing key",
		)
	}

	return &OrderEventConsumer{
		consumerCreated: consumerCreated,
		paymentService:  paymentService,
	}, nil
}

// StartListening شروع استماع صف و تبدیل بایت‌های ورودی به struct
func (c *OrderEventConsumer) StartListening(ctx context.Context) error {
	return c.consumerCreated.Consume(
		ctx,
		events.QueuePaymentOrderCreated,
		func(ctx context.Context, body []byte) error {
			var event events.OrderCreated
			if err := json.Unmarshal(body, &event); err != nil {
				return appErrors.Wrap(
					appErrors.KindInvalidInput,
					err,
					"payment-service: failed to unmarshal order created event",
				)
			}
			return c.handleOrderCreated(ctx, event)
		},
	)
}

func (c *OrderEventConsumer) handleOrderCreated(
	ctx context.Context,
	event events.OrderCreated,
) error {

	// اعتبارسنجی اولیه شناسه رویداد
	if event.EventID == uuid.Nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"payment-service: order.created event has empty event_id",
		)
	}

	if event.OrderID == uuid.Nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"payment-service: order.created event has empty order_id",
		)
	}

	return c.paymentService.HandleOrderCreated(
		ctx,
		event.EventID,
		event.OrderID,
		event.UserID,
		event.TotalAmount,
	)
}

// Close بستن کانال مصرف‌کننده
func (c *OrderEventConsumer) Close() error {
	if c == nil || c.consumerCreated == nil {
		return nil
	}
	return c.consumerCreated.Close()
}
