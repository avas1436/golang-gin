// services/order-service/internal/messaging/consumer.go

package messaging

import (
	"context"
	"encoding/json"
	"log"

	appErrors "pkg/errors"
	"pkg/events"
	"pkg/rabbitmq"

	"order-service/internal/model"
	"order-service/internal/repository"
)

// PaymentEventConsumer مسئول دریافت و پردازش رویدادهای پرداخت و تکمیل Saga
type PaymentEventConsumer struct {
	consumer  *rabbitmq.Consumer
	orderRepo repository.OrderRepository
	publisher *RabbitMQEventPublisher
}

// StartListening استماع همزمان رویدادهای پرداخت موفق و ناموفق از صف‌های مجزا
func (c *PaymentEventConsumer) StartListening(ctx context.Context) error {
	errCh := make(chan error, 2)

	// ۱. شنود صف پرداخت موفق
	go func() {
		err := c.consumer.Consume(
			ctx,
			events.QueueOrderPaymentCompleted,
			func(ctx context.Context, body []byte) error {
				var event events.PaymentCompleted
				if err := json.Unmarshal(body, &event); err != nil {
					return appErrors.Wrap(
						appErrors.KindInvalidInput,
						err,
						"failed to unmarshal PaymentCompleted event",
					)
				}
				return c.handlePaymentSucceeded(ctx, event)
			},
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
		err := c.consumer.Consume(
			ctx,
			events.QueueOrderPaymentFailed,
			func(ctx context.Context, body []byte) error {
				var event events.PaymentFailed
				if err := json.Unmarshal(body, &event); err != nil {
					return appErrors.Wrap(
						appErrors.KindInvalidInput,
						err,
						"failed to unmarshal PaymentFailed event",
					)
				}
				return c.handlePaymentFailed(ctx, event)
			},
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

func (
	c *PaymentEventConsumer,
) handlePaymentSucceeded(
	ctx context.Context,
	event events.PaymentCompleted,
) error {

	// ۱. تغییر وضعیت سفارش به Confirmed در دیتابیس
	if err := c.orderRepo.UpdateStatus(
		ctx,
		event.OrderID,
		model.OrderStatusConfirmed,
	); err != nil {

		// اگر وضعیت از قبل تغییر کرده است (Redelivery / Retry بعد از Crash)،
		// خطا را نادیده گرفته و برای حفظ یکپارچگی Saga به مرحله بعد می‌رویم.
		if appErrors.GetKind(err) == appErrors.KindAlreadyExists {
			log.Printf(
				"order-service: order %s is already processed (status not pending), continuing to stock confirmation for idempotency recovery",
				event.OrderID,
			)
		} else {
			log.Printf(
				"order-service: failed to update order status to confirmed for order %s: %v",
				event.OrderID,
				err,
			)

			return appErrors.Wrap(
				appErrors.KindInternal,
				err,
				"failed to update order status to confirmed",
			)
		}
	}

	// ۲. دریافت آیتم‌های سفارش جهت قطعی کردن کسر موجودی در Product Service
	order, err := c.orderRepo.GetByID(ctx, event.OrderID)
	if err != nil {
		log.Printf(
			"order-service: failed to fetch order %s: %v",
			event.OrderID,
			err,
		)

		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to fetch order details for stock confirmation",
		)
	}

	for _, item := range order.Items {
		if err := c.publisher.PublishStockConfirmRequested(
			ctx,
			order.ID,
			item.ProductID,
			item.Quantity,
		); err != nil {

			log.Printf(
				"order-service: failed to publish stock confirm for product %s: %v",
				item.ProductID,
				err,
			)

			// در صورت بروز خطا در انتشار رویداد، خطا برمی‌گردانیم
			// تا پیام NACK/Requeue شود
			return appErrors.Wrap(
				appErrors.KindInternal,
				err,
				"failed to publish stock confirm requested event",
			)
		}
	}

	return nil
}

func (
	c *PaymentEventConsumer,
) handlePaymentFailed(
	ctx context.Context,
	event events.PaymentFailed,
) error {

	// ۱. تغییر وضعیت سفارش به Cancelled در دیتابیس
	if err := c.orderRepo.UpdateStatus(
		ctx,
		event.OrderID,
		model.OrderStatusCancelled,
	); err != nil {

		// اگر وضعیت از قبل تغییر کرده است (Redelivery / Retry بعد از Crash)،
		// خطا را نادیده گرفته و برای حفظ یکپارچگی Saga به مرحله بعد می‌رویم.
		if appErrors.GetKind(err) == appErrors.KindAlreadyExists {
			log.Printf(
				"order-service: order %s is already processed (status not pending), continuing to stock release for idempotency recovery",
				event.OrderID,
			)
		} else {
			log.Printf(
				"order-service: failed to update order status to cancelled for order %s: %v",
				event.OrderID,
				err,
			)

			return appErrors.Wrap(
				appErrors.KindInternal,
				err,
				"failed to update order status to cancelled",
			)
		}
	}

	// ۲. دریافت آیتم‌های سفارش جهت آزادسازی موجودی رزرو شده در Product Service
	order, err := c.orderRepo.GetByID(ctx, event.OrderID)
	if err != nil {
		log.Printf(
			"order-service: failed to fetch order %s: %v",
			event.OrderID,
			err,
		)

		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to fetch order details for stock release",
		)
	}

	for _, item := range order.Items {
		if err := c.publisher.PublishStockReleaseRequested(
			ctx,
			item.ProductID,
			item.Quantity,
			"payment_failed",
		); err != nil {

			log.Printf(
				"order-service: failed to publish stock release for product %s: %v",
				item.ProductID,
				err,
			)

			// در صورت بروز خطا در انتشار رویداد، خطا برمی‌گردانیم
			//  تا پیام NACK/Requeue شود
			return appErrors.Wrap(
				appErrors.KindInternal,
				err,
				"failed to publish stock release requested event",
			)

		}
	}

	return nil
}
