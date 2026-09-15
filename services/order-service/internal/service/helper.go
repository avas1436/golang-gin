// services/order-service/internal/service/helper.go

package service

import (
	"context"
	"log"

	"pkg/auth"
	appErrors "pkg/errors"
	pb "pkg/proto/order"

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
}

// requireAuthenticated بررسی می‌کند که اطلاعات احراز هویت
// داخل context وجود داشته باشد.
func requireAuthenticated(ctx context.Context) (*auth.AccessClaims, error) {

	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return nil, appErrors.New(
			appErrors.KindUnauthenticated,
			"authentication required",
		)
	}

	return claims, nil
}

// parseOrderID شناسه سفارش را از string به UUID تبدیل می‌کند.
//
// این helper باعث می‌شود کدهای GetOrder، UpdateOrder و ... مجبور
// نباشند هر بار منطق UUID parsing را تکرار کنند.
func parseOrderID(orderID string) (uuid.UUID, error) {
	id, err := uuid.Parse(orderID)
	if err != nil {
		return uuid.Nil, appErrors.New(
			appErrors.KindInvalidInput,
			"invalid order id",
		)
	}

	return id, nil
}

// buildOrderItems اطلاعات محصول را از Product Service می‌گیرد
// و بر اساس آن OrderItemهای داخلی Order Service را می‌سازد.
//
// این تابع فقط مسئول ساختن itemهای سفارش است.
// هنوز موجودی را رزرو نمی‌کند.
func (
	s *OrderService,
) buildOrderItems(
	ctx context.Context,
	reqItems []*pb.CreateOrderItemRequest,
) (
	[]*model.OrderItem,
	error,
) {

	// ساخت یک آرایه خالی برای آیتم های سفارش
	items := make([]*model.OrderItem, 0, len(reqItems))

	for _, reqItem := range reqItems {

		// بررسی خالی بودن آیتم سفارش
		if reqItem == nil {
			return nil, appErrors.New(
				appErrors.KindInvalidInput,
				"order item is nil",
			)
		}

		// اگر یکی از آیتم ها تعداد کمتر از 1 داشت ارور میدهد
		if reqItem.Quantity <= 0 {
			return nil, appErrors.New(
				appErrors.KindInvalidInput,
				"item quantity must be greater than zero",
			)
		}

		// بررسی اعتبار آیدی محصول
		productID, err := uuid.Parse(reqItem.ProductId)
		if err != nil {
			return nil, appErrors.New(
				appErrors.KindInvalidInput,
				"invalid product id"+reqItem.ProductId,
			)
		}

		// بررسی وجودیت محصول
		product, err := s.productClient.GetProduct(
			ctx,
			reqItem.ProductId,
		)
		if err != nil {
			return nil, err
		}

		// بررسی فعال بودن محصول
		if !product.IsActive {
			return nil, appErrors.New(
				appErrors.KindInvalidInput,
				"product is not active",
			)
		}

		items = append(
			items,
			&model.OrderItem{
				ProductID:   productID,
				ProductName: product.Name,
				Quantity:    reqItem.Quantity,
				UnitPrice:   product.Price,
			},
		)
	}

	return items, nil
}

// reserveItems موجودی تمام itemهای سفارش را رزرو می‌کند.
//
// نکته مهم:
// اگر رزرو یک item موفق شود ولی رزرو item بعدی شکست بخورد،
// itemهای قبلاً رزروشده باید compensation شوند.
//
// بنابراین این تابع successful reservations را نگه می‌دارد
// تا در صورت failure بتوانیم آنها را آزاد کنیم.
func (
	s *OrderService,
) reserveItems(
	ctx context.Context,
	items []*model.OrderItem,
) (
	[]*model.OrderItem,
	error,
) {

	reservedItems := make([]*model.OrderItem, 0, len(items))

	for _, item := range items {
		if err := s.productClient.ReserveStock(
			ctx,
			item.ProductID.String(),
			item.Quantity,
		); err != nil {

			// یکی از Reservationها شکست خورده است.
			// بنابراین تمام Reservationهای موفق قبلی
			// باید compensate شوند.
			s.compensateReservations(
				ctx,
				reservedItems,
				"reservation_failed",
			)

			return nil, err
		}

		reservedItems = append(reservedItems, item)
	}

	return reservedItems, nil
}

// compensateReservations رزرو تمام آیتم‌هایی که تا این لحظه موفق
// شده بودند را آزاد می‌کند. خطای هر ReleaseStock را فقط لاگ می‌کند
// و ادامه می‌دهد؛ اگر همین‌جا هم return early می‌کردیم، ممکن بود
// آیتم‌های بعدی هیچ‌وقت آزاد نشوند و موجودی برای همیشه قفل بماند —
// که دقیقاً همان چیزی است که Compensation باید جلویش را بگیرد
func (
	s *OrderService,
) compensateReservations(
	ctx context.Context,
	items []*model.OrderItem,
	reason string,
) {

	for _, item := range items {

		if err := s.publisher.PublishStockReleaseRequested(
			ctx,
			item.ProductID,
			item.Quantity,
			reason,
		); err != nil {
			log.Printf(
				"order-service: failed to publish stock release for product %s: %v",
				item.ProductID,
				err,
			)
		}
	}
}
