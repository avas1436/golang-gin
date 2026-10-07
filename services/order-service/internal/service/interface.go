// services/order-service/internal/service/interface.go

package service

import (
	"context"
	"order-service/internal/model"

	"github.com/google/uuid"
)

// EventPublisher چیزی است که OrderService برای انتشار Eventها نیاز دارد.
//
// یک اینترفیس برای جدا کردن منطق سرویس از RabbitMQ پس اصلا لایه
// نباید از جزییات این ارتباط مطلع باشد
//
// پیاده‌سازی این interface در internal/messaging قرار دارد.
//
// ویژگی این سه عملیات اینه که کسی منتظر جواب لحظه ای این ها نیست
// و بهتر است غیر همزمان انجام شوند
type EventPublisher interface {
	PublishOrderCreated(
		ctx context.Context,
		order *model.Order,
	) error

	PublishStockReleaseRequested(
		ctx context.Context,
		productID uuid.UUID,
		quantity int32,
		reason string,
	) error

	PublishStockConfirmRequested(
		ctx context.Context,
		orderID uuid.UUID,
		productID uuid.UUID,
		quantity int32,
	) error

	// این متدها فقط payload می‌سازن، publish نمی‌کنن
	BuildOrderCreatedPayload(order *model.Order) ([]byte, error)

	BuildStockConfirmPayload(
		orderID uuid.UUID,
		productID uuid.UUID,
		quantity int32,
	) ([]byte, error)

	BuildStockReleasePayload(
		productID uuid.UUID,
		quantity int32,
		reason string,
	) ([]byte, error)
}
