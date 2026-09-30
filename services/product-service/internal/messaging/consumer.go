// services/product-service/internal/messaging/consumer.go

package messaging

import (
	"context"
	"encoding/json"
	"log"
	appErrors "pkg/errors"
	"pkg/postgres"

	"pkg/events"
	"pkg/rabbitmq"

	"product-service/internal/cache"
	"product-service/internal/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrderEventConsumer مسئول دریافت و پردازش رویدادهای پرداخت و تکمیل Saga
type OrderEventConsumer struct {
	consumerConfirm *rabbitmq.Consumer
	consumerRelease *rabbitmq.Consumer
	pool            *pgxpool.Pool
	productStore    *cache.ProductCacheStore
}

// StartListening استماع همزمان رویدادهای آزاد سازی یا نهایی سازی موجودی
// از صف‌های مجزا
func (c *OrderEventConsumer) StartListening(ctx context.Context) error {
	errCh := make(chan error, 2)

	// ۱. شنود صف پرداخت موفق
	go func() {
		err := c.consumerConfirm.Consume(
			ctx,
			events.QueueProductStockConfirm,
			c.handleStockConfirm,
		)

		if err != nil {
			errCh <- appErrors.Wrap(
				appErrors.KindInternal,
				err,
				"error in stock confirm consumer loop",
			)
		}
	}()

	// ۲. شنود صف پرداخت ناموفق
	go func() {
		err := c.consumerRelease.Consume(
			ctx,
			events.QueueOrderPaymentFailed,
			c.handleStockRelease,
		)

		if err != nil {
			errCh <- appErrors.Wrap(
				appErrors.KindInternal,
				err,
				"error in stock release consumer loop",
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

func (c *OrderEventConsumer) handleStockConfirm(
	ctx context.Context,
	body []byte,
) error {
	var event events.StockConfirmRequested
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf(
			"product-service: failed to unmarshal stock confirm event: %v",
			err,
		)
		return nil // Poison message - برای جلوگیری از مسدود شدن صف Ack می‌شود
	}

	err := postgres.WithTx(
		ctx,
		c.pool,
		func(tx pgx.Tx) error {
			eventRepo := repository.NewEventRepository(tx)
			productRepo := repository.NewProductRepository(tx)

			alreadyProcessed, err := eventRepo.MarkProcessed(
				ctx,
				event.EventID,
				events.RoutingKeyStockConfirmRequested,
				event.ProductID,
				event.OrderID,
			)
			if err != nil {
				log.Printf(
					"product-service: failed to check idempotency for event %s: %v",
					event.EventID,
					err,
				)
				return err
			}

			if alreadyProcessed {
				log.Printf(
					"product-service: stock confirm event %s already processed, skipping",
					event.EventID,
				)
				return nil
			}

			if err := productRepo.ConfirmStock(
				ctx,
				event.ProductID,
				event.Quantity,
			); err != nil {
				log.Printf(
					"product-service: failed to confirm stock for product %s (order %s): %v",
					event.ProductID,
					event.OrderID,
					err,
				)
				return err
			}

			return nil
		},
	)

	if err != nil {
		return err
	}

	if err := c.productStore.InvalidateProduct(
		ctx,
		event.ProductID,
	); err != nil {
		log.Printf(
			"product-service: failed to invalidate cache for product %s: %v",
			event.ProductID,
			err,
		)
	}

	return nil
}

func (c *OrderEventConsumer) handleStockRelease(
	ctx context.Context,
	body []byte,
) error {
	var event events.StockReleaseRequested
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf(
			"product-service: failed to unmarshal stock release event: %v",
			err,
		)
		return nil
	}

	err := postgres.WithTx(
		ctx,
		c.pool,
		func(tx pgx.Tx) error {
			eventRepo := repository.NewEventRepository(tx)
			productRepo := repository.NewProductRepository(tx)

			alreadyProcessed, err := eventRepo.MarkProcessed(
				ctx,
				event.EventID,
				events.RoutingKeyStockReleaseRequested,
				event.ProductID,
				event.OrderID,
			)
			if err != nil {
				log.Printf(
					"product-service: failed to check idempotency for event %s: %v",
					event.EventID,
					err,
				)
				return err
			}

			if alreadyProcessed {
				log.Printf(
					"product-service: stock release event %s already processed, skipping",
					event.EventID,
				)
				return nil
			}

			if err := productRepo.ReleaseStock(
				ctx,
				event.ProductID,
				event.Quantity,
			); err != nil {
				log.Printf(
					"product-service: failed to release stock for product %s: %v",
					event.ProductID,
					err,
				)
				return err
			}

			return nil
		},
	)

	if err != nil {
		return err
	}

	if err := c.productStore.InvalidateProduct(
		ctx,
		event.ProductID,
	); err != nil {
		log.Printf(
			"product-service: failed to invalidate cache for product %s: %v",
			event.ProductID,
			err,
		)
	}

	return nil
}
