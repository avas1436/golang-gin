// services/order-service/internal/service/saga.go

package service

import (
	"context"
	"log"

	appErrors "pkg/errors"
	"pkg/events"

	"order-service/internal/model"
	"order-service/internal/repository"
)

func (
	c *OrderService,
) HandlePaymentSucceeded(
	ctx context.Context,
	event events.PaymentCompleted,
) error {

	// استفاده از SagaRepository برای مدیریت اتمیک ثبت رویداد در
	// Inbox و اعمال تغییرات روی سفارش
	alreadyProcessed, order, err := c.sagaRepo.ExecuteInInbox(
		ctx,
		event.EventID,
		events.RoutingKeyPaymentCompleted,
		&event.OrderID,
		func(
			ctx context.Context,
			txRepo repository.OrderRepository,
		) (
			*model.Order,
			error,
		) {

			existingOrder, err := txRepo.GetByID(ctx, event.OrderID)
			if err != nil {
				return nil, err
			}

			// بررسی تضاد وضعیت سفارش - قبلاً کنسل شده اما رویداد
			// پرداخت موفق دیرتر رسیده است
			if existingOrder.Status == model.OrderStatusCancelled || existingOrder.Status == model.OrderStatusFailed {

				log.Printf(
					"CRITICAL WARNING: order-service: order %s is already in state '%s', but received payment.completed (EventID: %s)! Needs refund action.",
					event.OrderID,
					existingOrder.Status,
					event.EventID,
				)

				return nil, nil
			}

			if err := txRepo.UpdateStatus(
				ctx,
				event.OrderID,
				model.OrderStatusConfirmed,
			); err != nil {

				return nil, err
			}

			return existingOrder, nil
		},
	)

	if err != nil {
		return err
	}

	// اگر پیام تکراری بوده یا وضعیت سفارش تغییر نکرده باشد، رویداد قطعی شدن موجودی صادر نمی‌شود
	if alreadyProcessed || order == nil {
		return nil
	}

	// قطعی کردن موجودی در Product Service خارج از تراکنش دیتابیس
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

	alreadyProcessed, order, err := c.sagaRepo.ExecuteInInbox(
		ctx,
		event.EventID,
		events.RoutingKeyPaymentFailed,
		&event.OrderID,
		func(
			ctx context.Context,
			txRepo repository.OrderRepository,
		) (
			*model.Order,
			error,
		) {

			existingOrder, err := txRepo.GetByID(ctx, event.OrderID)
			if err != nil {
				return nil, err
			}

			if err := txRepo.UpdateStatus(
				ctx,
				event.OrderID,
				model.OrderStatusCancelled,
			); err != nil {

				return nil, err
			}

			return existingOrder, nil
		},
	)

	if err != nil {
		return err
	}

	if alreadyProcessed || order == nil {
		return nil
	}

	// آزادسازی موجودی در Product Service
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

	alreadyProcessed, _, err := c.sagaRepo.ExecuteInInbox(
		ctx,
		event.EventID,
		events.RoutingKeyPaymentInitiated,
		&event.OrderID,
		func(
			ctx context.Context,
			txRepo repository.OrderRepository,
		) (
			*model.Order,
			error,
		) {

			log.Printf(
				"order-service: payment initiated for order %s, redirect_url: %s",
				event.OrderID,
				event.RedirectURL,
			)

			if err := txRepo.UpdatePaymentDetails(
				ctx,
				event.OrderID,
				event.RedirectURL,
				event.Authority,
			); err != nil {

				return nil, appErrors.Wrap(
					appErrors.KindInternal,
					err,
					"failed to update order payment details",
				)
			}

			return nil, nil
		},
	)

	if err != nil {
		return err
	}

	if alreadyProcessed {
		log.Printf(
			"order-service: payment initiated event %s already processed, skipping",
			event.EventID,
		)
	}

	return nil
}
