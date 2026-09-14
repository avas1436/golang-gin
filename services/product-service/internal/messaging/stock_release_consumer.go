// services/product-service/internal/messaging/stock_release_consumer.go

package messaging

import (
	"context"
	"encoding/json"
	"log"

	"pkg/events"

	"product-service/internal/repository"
)

const eventTypeStockRelease = "stock.release.requested"

// StockReleaseConsumer پیام‌های stock.release.requested را مصرف
// می‌کند. حالا قبل از هر کاری با EventRepository چک می‌کند که این
// event قبلاً پردازش نشده باشد
type StockReleaseConsumer struct {
	productRepo repository.ProductRepository
	eventRepo   repository.EventRepository
}

func NewStockReleaseConsumer(
	productRepo repository.ProductRepository,
	eventRepo repository.EventRepository,
) *StockReleaseConsumer {

	return &StockReleaseConsumer{
		productRepo: productRepo,
		eventRepo:   eventRepo,
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

	// تلاش میکند تا رویداد را در دیتابیس ذخیره کند
	alreadyProcessed, err := h.eventRepo.MarkProcessed(
		ctx,
		event.EventID,
		eventTypeStockRelease,
		event.ProductID,
	)
	if err != nil {
		// اینجا نمی‌دانیم event قبلاً پردازش شده یا نه؛ امن‌ترین
		// کار برگرداندن خطاست تا با Nack دوباره تلاش شود
		log.Printf(
			"product-service: failed to check idempotency for event %s: %v",
			event.EventID, err,
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

	// در اینجا تلاش میکند تا در جدول محصول تغییر را ذخیره کند
	if err := h.productRepo.ReleaseStock(
		ctx,
		event.ProductID,
		event.Quantity,
	); err != nil {

		log.Printf(
			"product-service: failed to release stock for product %s (reason: %s): %v",
			event.ProductID, event.Reason, err,
		)

		return err
	}

	return nil
}
