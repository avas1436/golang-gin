// services/order-service/internal/messaging/publisher.go

package messaging

import (
	"context"
	"time"

	"pkg/events"
	"pkg/rabbitmq"

	"order-service/internal/model"

	"github.com/google/uuid"
)

// EventPublisher پیاده‌سازی واقعی service.EventPublisher است. جایی
// که تعریف اینترفیس هست (internal/service) این پیاده‌سازی را
// نمی‌شناسد؛ فقط main.go/fx این دو را به هم وصل می‌کند
type EventPublisher struct {
	publisher *rabbitmq.Publisher
}

func NewEventPublisher(publisher *rabbitmq.Publisher) *EventPublisher {
	return &EventPublisher{publisher: publisher}
}

// بعد از کامیت شدن موفق سفارش در سرویس سفارشات این رویداد منتشر میشود
// تا سرویس های محصول و نوتیف و انبار داری و ... از آن استفاده کنند
func (
	p *EventPublisher,
) PublishOrderCreated(
	ctx context.Context,
	order *model.Order,
) error {

	items := make([]events.OrderCreatedItem, 0, len(order.Items))

	for _, item := range order.Items {
		items = append(items, events.OrderCreatedItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
		})
	}

	event := events.OrderCreated{
		EventID:     uuid.New(),
		OrderID:     order.ID,
		UserID:      order.UserID,
		TotalAmount: order.TotalAmount,
		Items:       items,
		CreatedAt:   order.CreatedAt,
	}

	return p.publisher.Publish(ctx, events.RoutingKeyOrderCreated, event)
}

// این تابع برای تک تک آیتم های یک سفارش به صورت جداگانه یک رویداد
// منتشر میکند و رزرو آن محصول در سرویس محصولات کسر خواهد شد
func (p *EventPublisher) PublishStockReleaseRequested(
	ctx context.Context,
	productID uuid.UUID,
	quantity int32,
	reason string,
) error {

	event := events.StockReleaseRequested{
		EventID:     uuid.New(),
		ProductID:   productID,
		Quantity:    quantity,
		Reason:      reason,
		RequestedAt: time.Now(),
	}

	return p.publisher.Publish(
		ctx,
		events.RoutingKeyStockReleaseRequested,
		event,
	)
}

// بعد از تایید پرداخت از سمت سرویس پرداخت ابتدا در سرویس سفارشات
// سفارش مورد نظر تایید شده و بعد این تابع یک رویداد تایید نهای برای سرویس
// های دیگر منتشر خواهد کرد
func (p *EventPublisher) PublishStockConfirmRequested(
	ctx context.Context,
	orderID uuid.UUID,
	productID uuid.UUID,
	quantity int32,
) error {

	event := events.StockConfirmRequested{
		EventID:     uuid.New(),
		OrderID:     orderID,
		ProductID:   productID,
		Quantity:    quantity,
		RequestedAt: time.Now(),
	}

	return p.publisher.Publish(
		ctx, events.RoutingKeyStockConfirmRequested, event,
	)
}
