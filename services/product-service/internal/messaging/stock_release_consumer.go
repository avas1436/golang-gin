// services/product-service/internal/messaging/stock_release_consumer.go

package messaging

import (
	"context"
	"encoding/json"
	"log"
	"product-service/internal/cache"
	"product-service/internal/repository"

	"pkg/events"
	"pkg/postgres"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const eventTypeStockRelease = "stock.release.requested"

// StockReleaseConsumer پیام‌های stock.release.requested را مصرف
// می‌کند. حالا قبل از هر کاری با EventRepository چک می‌کند که این
// event قبلاً پردازش نشده باشد
type StockReleaseConsumer struct {
	pool         *pgxpool.Pool
	productStore *cache.ProductCacheStore
}

func NewStockReleaseConsumer(
	pool *pgxpool.Pool,
	productStore *cache.ProductCacheStore,
) *StockReleaseConsumer {

	return &StockReleaseConsumer{
		pool:         pool,
		productStore: productStore,
	}

}

// Handle امضای rabbitmq.HandlerFunc را دارد
// مشکل این تابع در این است که دو فرایند ذخیره در جدول محصول
// و رویداد ها را در یک تراکنش انجام نمیدهد
func (
	h *StockReleaseConsumer,
) Handle(
	ctx context.Context,
	body []byte,
) error {

	var event events.StockReleaseRequested

	// در این قسمت امضای پیام اعتبار سنجی میشه
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf(
			"product-service: failed to unmarshal stock release event: %v",
			err,
		)

		// بدنه‌ی خراب با تکرار مجدد درست نمی‌شود؛ Ack می‌شود تا صف را
		// مسدود نکند (poison message)
		return nil
	}

	err := postgres.WithTx(
		ctx,
		h.pool,
		func(tx pgx.Tx) error {

			// هر دو Repository با همان Transaction ساخته می‌شوند.
			eventRepo := repository.NewEventRepository(tx)
			productRepo := repository.NewProductRepository(tx)

			// -----------------------------
			// 1. Idempotency
			// -----------------------------
			alreadyProcessed, err := eventRepo.MarkProcessed(
				ctx,
				event.EventID,
				eventTypeStockRelease,
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

			// -----------------------------
			// 2. Release Stock
			// -----------------------------
			if err := productRepo.ReleaseStock(
				ctx,
				event.ProductID,
				event.Quantity,
			); err != nil {

				log.Printf("product-service: failed to release stock for product %s: %v", event.ProductID, err)

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
