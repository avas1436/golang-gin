// services/order-service/internal/messaging/consumer.go

package messaging

import (
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"

	appErrors "pkg/errors"
	"pkg/events"
	"pkg/rabbitmq"

	"order-service/internal/service"
)

// PaymentEventConsumer مسئول دریافت و پردازش رویدادهای پرداخت و تکمیل Saga
type PaymentEventConsumer struct {
	consumerCompleted *rabbitmq.Consumer
	consumerFailed    *rabbitmq.Consumer
	orderService      *service.OrderService
}

func NewPaymentEventConsumer(
	conn *rabbitmq.Connection,
	orderService *service.OrderService,
) (
	*PaymentEventConsumer,
	error,
) {

	// ۱. کانال اختصاصی برای صف پرداخت موفق
	chCompleted, err := conn.Channel()
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to open channel for payment completed consumer",
		)
	}

	consumerCompleted, err := rabbitmq.NewConsumer(chCompleted)
	if err != nil {
		_ = chCompleted.Close()
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to initialize payment completed consumer",
		)
	}

	if err := consumerCompleted.BindQueue(
		events.QueueOrderPaymentCompleted,
		events.ExchangeOrderEvents,
		events.RoutingKeyPaymentCompleted,
	); err != nil {
		_ = chCompleted.Close()
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to bind queue to payment completed routing key",
		)
	}

	// ۲. کانال اختصاصی برای صف پرداخت ناموفق
	chFailed, err := conn.Channel()
	if err != nil {
		_ = chCompleted.Close()
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to open channel for payment failed consumer",
		)
	}

	consumerFailed, err := rabbitmq.NewConsumer(chFailed)
	if err != nil {
		_ = chCompleted.Close()
		_ = chFailed.Close()
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to initialize payment failed consumer",
		)
	}

	if err := consumerFailed.BindQueue(
		events.QueueOrderPaymentFailed,
		events.ExchangeOrderEvents,
		events.RoutingKeyPaymentFailed,
	); err != nil {
		_ = chCompleted.Close()
		_ = chFailed.Close()
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to bind queue to payment failed routing key",
		)
	}

	return &PaymentEventConsumer{
		consumerCompleted: consumerCompleted,
		consumerFailed:    consumerFailed,
		orderService:      orderService,
	}, nil
}

// StartListening استماع همزمان رویدادهای پرداخت موفق و ناموفق از صف‌های مجزا
func (c *PaymentEventConsumer) StartListening(ctx context.Context) error {
	errCh := make(chan error, 2)

	// ۱. شنود صف پرداخت موفق
	go func() {
		err := c.consumerCompleted.Consume(
			ctx,
			events.QueueOrderPaymentCompleted,
			c.handlePaymentCompleted,
		)
		if err != nil {
			errCh <- appErrors.Wrap(
				appErrors.KindInternal,
				err,
				"error in payment completed consumer loop",
			)
		}
	}()

	// ۲. شنود صف پرداخت ناموفق
	go func() {
		err := c.consumerFailed.Consume(
			ctx,
			events.QueueOrderPaymentFailed,
			c.handlePaymentFailed,
		)
		if err != nil {
			errCh <- appErrors.Wrap(
				appErrors.KindInternal,
				err,
				"error in payment failed consumer loop",
			)
		}
	}()

	select {

	case <-ctx.Done():
		return ctx.Err()

	case err := <-errCh:
		return err

	}
}

func (c *PaymentEventConsumer) handlePaymentCompleted(
	ctx context.Context,
	body []byte,
) error {
	var event events.PaymentCompleted
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf(
			"order-service: failed to unmarshal PaymentCompleted event: %v",
			err,
		)
		// Poison message: برای جلوگیری از مسدود شدن صف (Infinite Requeue) خطا نادیده گرفته شده و ACK می‌شود
		return nil
	}

	if event.OrderID == uuid.Nil {
		log.Printf(
			"order-service: received PaymentCompleted event with empty order_id",
		)
		return nil
	}

	return c.orderService.HandlePaymentSucceeded(ctx, event)
}

func (c *PaymentEventConsumer) handlePaymentFailed(
	ctx context.Context,
	body []byte,
) error {
	var event events.PaymentFailed
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf(
			"order-service: failed to unmarshal PaymentFailed event: %v",
			err,
		)
		// Poison message: برای جلوگیری از مسدود شدن صف (Infinite Requeue) خطا نادیده گرفته شده و ACK می‌شود
		return nil
	}

	if event.OrderID == uuid.Nil {
		log.Printf(
			"order-service: received PaymentFailed event with empty order_id",
		)
		return nil
	}

	return c.orderService.HandlePaymentFailed(ctx, event)
}

// Close بستن تمام کانال‌های مربوط به مصرف‌کننده‌ها
func (c *PaymentEventConsumer) Close() error {
	if c == nil {
		return nil
	}
	if c.consumerCompleted != nil {
		_ = c.consumerCompleted.Close()
	}
	if c.consumerFailed != nil {
		_ = c.consumerFailed.Close()
	}
	return nil
}
