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

// StartListening استماع رویدادهای نتایج پرداخت از RabbitMQ
func (c *PaymentEventConsumer) StartListening(ctx context.Context) error {

	return c.consumer.Consume(
		ctx,
		paymentEventsQueue,
		func(ctx context.Context, body []byte) error {
			var rawHeader struct {
				Reason string `json:"reason"`
			}

			if err := json.Unmarshal(body, &rawHeader); err != nil {
				log.Printf("order-service: failed to parse payment event header: %v", err)
				return appErrors.Wrap(
					appErrors.KindInvalidInput,
					err,
					"failed to unmarshal payment event header",
				)
			}

			// اگر فیلد Reason وجود داشته باشد، رویداد شکست پرداخت است
			if rawHeader.Reason != "" {
				var failedEvent events.PaymentFailed
				if err := json.Unmarshal(body, &failedEvent); err != nil {
					return appErrors.Wrap(
						appErrors.KindInvalidInput,
						err,
						"failed to unmarshal PaymentFailed event",
					)
				}
				return c.handlePaymentFailed(ctx, failedEvent)
			}

			// در غیر این صورت رویداد پرداخت موفق است
			var succeededEvent events.PaymentCompleted
			if err := json.Unmarshal(body, &succeededEvent); err != nil {
				return appErrors.Wrap(
					appErrors.KindInvalidInput,
					err,
					"failed to unmarshal PaymentSucceeded event",
				)
			}
			return c.handlePaymentSucceeded(ctx, succeededEvent)
		},
	)
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

		}
	}

	return nil
}
