// services/product-service/internal/messaging/stock_confirm_consumer.go

package messaging

import (
	"context"
	"encoding/json"
	"log"

	"pkg/events"

	"product-service/internal/repository"
)

const eventTypeStockConfirm = "stock.confirm.requested"

// StockConfirmConsumer پیام‌های stock.confirm.requested را مصرف
// می‌کند. ایدمپوتنسی اینجا از Release هم مهم‌تره چون ConfirmStock
// هم total_stock و هم reserved_stock را واقعاً کم می‌کند
type StockConfirmConsumer struct {
	productRepo repository.ProductRepository
	eventRepo   repository.EventRepository
}

func NewStockConfirmConsumer(
	productRepo repository.ProductRepository,
	eventRepo repository.EventRepository,
) *StockConfirmConsumer {

	return &StockConfirmConsumer{
		productRepo: productRepo,
		eventRepo:   eventRepo,
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

	// ذخیره رویداد در جدول دیتابیس
	alreadyProcessed, err := h.eventRepo.MarkProcessed(
		ctx,
		event.EventID,
		eventTypeStockConfirm,
		event.ProductID,
	)
	if err != nil {
		log.Printf(
			"product-service: failed to check idempotency for event %s: %v",
			event.EventID, err,
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

	// ذخیره در جدول محصولات
	if err := h.productRepo.ConfirmStock(
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
}
