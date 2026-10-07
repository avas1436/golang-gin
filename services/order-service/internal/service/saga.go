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

	alreadyProcessed, order, err := c.sagaRepo.ExecuteInInbox(
		ctx,
		event.EventID,
		events.RoutingKeyPaymentCompleted,
		&event.OrderID,
		func(
			ctx context.Context,
			repos repository.Repositories,
		) (*model.Order, error) {

			existingOrder, err := repos.Order.GetByID(ctx, event.OrderID)
			if err != nil {
				return nil, err
			}

			if existingOrder.Status == model.OrderStatusCancelled ||
				existingOrder.Status == model.OrderStatusFailed {
				log.Printf(
					"CRITICAL WARNING: order %s is '%s' but got payment.completed (EventID: %s). Needs refund.",
					event.OrderID,
					existingOrder.Status,
					event.EventID,
				)
				return nil, nil
			}

			// UpdateStatus با from+to (optimistic locking)
			if err := repos.Order.UpdateStatus(
				ctx,
				event.OrderID,
				model.OrderStatusPending,
				model.OrderStatusConfirmed,
			); err != nil {
				return nil, err
			}

			// رویداد stock.confirm.requested رو داخل تراکنش به Outbox اضافه کن
			for _, item := range existingOrder.Items {
				payload, err := c.publisher.BuildStockConfirmPayload(
					existingOrder.ID,
					item.ProductID,
					item.Quantity,
				)
				if err != nil {
					return nil, err
				}
				outboxEvent := model.NewOutboxEvent(
					model.OutboxEventTypeStockConfirmRequested,
					events.RoutingKeyStockConfirmRequested,
					payload,
					existingOrder.ID,
				)
				if err := repos.Outbox.Create(ctx, outboxEvent); err != nil {
					return nil, err
				}
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

	return nil // Relay Worker منتشر می‌کند
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
			repos repository.Repositories,
		) (
			*model.Order,
			error,
		) {

			existingOrder, err := repos.Order.GetByID(ctx, event.OrderID)
			if err != nil {
				return nil, err
			}

			if existingOrder.Status != model.OrderStatusPending {
				log.Printf(
					"order %s already in state '%s', skipping cancel",
					event.OrderID,
					existingOrder.Status,
				)
				return nil, nil
			}

			if err := repos.Order.UpdateStatus(
				ctx,
				event.OrderID,
				model.OrderStatusPending,
				model.OrderStatusCancelled,
			); err != nil {

				return nil, err
			}

			// رویداد stock.confirm.requested رو داخل تراکنش به Outbox اضافه کن
			for _, item := range existingOrder.Items {
				payload, err := c.publisher.BuildStockReleasePayload(
					item.ProductID,
					item.Quantity,
					"payment_failed",
				)
				if err != nil {
					return nil, err
				}
				outboxEvent := model.NewOutboxEvent(
					model.OutboxEventTypeStockReleaseRequested,
					events.RoutingKeyStockReleaseRequested,
					payload,
					existingOrder.ID,
				)
				if err := repos.Outbox.Create(ctx, outboxEvent); err != nil {
					return nil, err
				}
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

	return nil // Relay Worker منتشر می‌کند
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
			repos repository.Repositories,
		) (
			*model.Order,
			error,
		) {

			if err := repos.Order.UpdatePaymentDetails(
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
