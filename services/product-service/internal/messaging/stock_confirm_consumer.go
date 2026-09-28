// services/product-service/internal/messaging/stock_confirm_consumer.go

package messaging

import (
	"context"
	"encoding/json"
	"log"

	"pkg/events"
	"pkg/postgres"

	"product-service/internal/cache"
	"product-service/internal/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const eventTypeStockConfirm = "stock.confirm.requested"

// StockConfirmConsumer پیام‌های stock.confirm.requested را مصرف
// می‌کند. ایدمپوتنسی اینجا از Release هم مهم‌تره چون ConfirmStock
// هم total_stock و هم reserved_stock را واقعاً کم می‌کند
type StockConfirmConsumer struct {
	pool         *pgxpool.Pool
	productStore *cache.ProductCacheStore
}

func NewStockConfirmConsumer(
	pool *pgxpool.Pool,
	productStore *cache.ProductCacheStore,
) *StockConfirmConsumer {

	return &StockConfirmConsumer{
		pool:         pool,
		productStore: productStore,
	}

}

// در این تابع هم دو فرایند در یک تراکنش انجام نمیشوند
func (
	h *StockConfirmConsumer,
) Handle(
	ctx context.Context,
	body []byte,
) error {

	// اعتبار سنجی متن پیام
	var event events.StockConfirmRequested

	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf(
			"product-service: failed to unmarshal stock confirm event: %v",
			err,
		)
		return nil
	}

	err := postgres.WithTx(
		ctx,
		h.pool,
		func(tx pgx.Tx) error {

			eventRepo := repository.NewEventRepository(tx)
			productRepo := repository.NewProductRepository(tx)

			// ذخیره رویداد در جدول دیتابیس
			alreadyProcessed, err := eventRepo.MarkProcessed(
				ctx,
				event.EventID,
				eventTypeStockConfirm,
				event.ProductID,
				event.OrderID,
			)

			// اگر خطا داد
			if err != nil {
				log.Printf(
					"product-service: failed to check idempotency for event %s: %v",
					event.EventID,
					err,
				)
				return err
			}

			// اگر رویداد تکراری بود
			if alreadyProcessed {
				log.Printf(
					"product-service: stock confirm event %s already processed, skipping",
					event.EventID,
				)

				return nil
			}

			// ذخیره در جدول مجصولات
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

	// ابطال کش به‌صورت یکپارچه پس از COMMIT تراکنش
	if err := h.productStore.InvalidateProduct(
		ctx,
		event.ProductID,
	); err != nil {

		log.Printf("product-service: failed to invalidate cache for product %s: %v",
			event.ProductID,
			err,
		)
	}

	return nil
}
