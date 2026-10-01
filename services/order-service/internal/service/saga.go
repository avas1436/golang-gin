// services/order-service/internal/service/saga.go

package service

import (
	"context"
	"log"

	appErrors "pkg/errors"
	"pkg/events"

	"order-service/internal/model"
)

func (
	c *OrderService,
) HandlePaymentSucceeded(
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
	c *OrderService,
) HandlePaymentFailed(
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

// HandlePaymentInitiated رویداد payment.initiated
// را دریافت کرده و لینک پرداخت را روی سفارش ثبت می‌کند.
func (
	c *OrderService,
) HandlePaymentInitiated(
	ctx context.Context,
	event events.PaymentInitiated,
) error {

	log.Printf(
		"order-service: payment initiated for order %s, redirect_url: %s",
		event.OrderID,
		event.RedirectURL,
	)

	// ۱. به روزرسانی لینک پرداخت و Authority در جدول orders
	if err := c.orderRepo.UpdatePaymentDetails(
		ctx,
		event.OrderID,
		event.RedirectURL,
		event.Authority,
	); err != nil {
		log.Printf(
			"order-service: failed to update payment details for order %s: %v",
			event.OrderID,
			err,
		)

		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to update order payment details",
		)
	}

	return nil
}
