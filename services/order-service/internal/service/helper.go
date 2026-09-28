// services/order-service/internal/service/helper.go

package service

import (
	"context"
	"log"
	"time"

	"pkg/auth"
	appErrors "pkg/errors"
	pb "pkg/proto/order"

	"order-service/internal/model"

	"github.com/google/uuid"
)

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
//
// در آخرین آپدیت این تابع به صورت غیر همزمان تمامی درخواست های
// gRPC رو ارسال میکنه تا حداکثر زمان انجام این تابع به اندازه طولانی ترین
// درخواست بشه نه مجموع درخواست ها
func (
	s *OrderService,
) buildOrderItems(
	ctx context.Context,
	reqItems []*pb.CreateOrderItemRequest,
) (
	[]*model.OrderItem,
	error,
) {

	// بررسی خالی بودن آیتم های سفارش
	if len(reqItems) == 0 {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"order must contain at least one item",
		)
	}

	// اعتبار سنجی و ساخت یک لیست از آیدی محصولات
	productIDs := make([]string, 0, len(reqItems))

	for _, reqItem := range reqItems {
		if reqItem == nil {
			return nil, appErrors.New(
				appErrors.KindInvalidInput,
				"order item is nil",
			)
		}

		if reqItem.Quantity <= 0 {
			return nil, appErrors.New(
				appErrors.KindInvalidInput,
				"item quantity must be greater than zero",
			)
		}

		if _, err := uuid.Parse(reqItem.ProductId); err != nil {

			return nil, appErrors.New(appErrors.KindInvalidInput, "invalid product id: "+reqItem.ProductId)

		}

		productIDs = append(productIDs, reqItem.ProductId)
	}

	// دریافت دسته‌جمعی اطلاعات تمام محصولات در یک درخواست gRPC
	products, err := s.productClient.GetProductsByIDs(ctx, productIDs)
	if err != nil {
		return nil, err
	}

	// یک لیست از اطلاعات دریافت شده از سرویس محصولات میسازیم
	productMap := make(map[string]struct {
		Name     string
		Price    int64
		IsActive bool
	}, len(products))

	// در یک حلقه لیست رو پرمیکنیم تا جستجو در آن بر اساس آیدی
	// سریع تر باشد
	for _, p := range products {
		productMap[p.Id] = struct {
			Name     string
			Price    int64
			IsActive bool
		}{
			Name:     p.Name,
			Price:    p.Price,
			IsActive: p.IsActive,
		}
	}

	// ساخت لیست آیتم‌های سفارش دامنه‌ای با قیمت Snapshot شده
	items := make([]*model.OrderItem, 0, len(reqItems))

	// در این حلقه هم داده های سفارش را اعتبار سنجی میکنیم و هم آیتم های
	// سفارش را میسازیم
	for _, reqItem := range reqItems {
		p, exists := productMap[reqItem.ProductId]
		if !exists {
			return nil, appErrors.New(
				appErrors.KindNotFound,
				"product not found: "+reqItem.ProductId,
			)
		}

		if !p.IsActive {
			return nil, appErrors.New(
				appErrors.KindInvalidInput,
				"product is not active: "+p.Name,
			)
		}

		productUUID := uuid.MustParse(reqItem.ProductId)
		unitPrice := p.Price
		subtotal := int64(reqItem.Quantity) * unitPrice

		items = append(items, &model.OrderItem{
			ProductID:   productUUID,
			ProductName: p.Name,
			UnitPrice:   unitPrice,
			Quantity:    reqItem.Quantity,
			Subtotal:    subtotal,
		})
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

	if len(items) == 0 {
		return
	}

	// استفاده از context.WithoutCancel برای تضمین
	// ارسال ایونت حتی در صورت Cancel شدن درخواست اصلی
	compCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		10*time.Second,
	)
	defer cancel()

	for _, item := range items {

		if err := s.publisher.PublishStockReleaseRequested(
			compCtx,
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
