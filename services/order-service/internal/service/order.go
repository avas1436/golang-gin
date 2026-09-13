// services/order-service/internal/service/order.go

package service

import (
	"context"
	"log"

	"pkg/auth"
	appErrors "pkg/errors"
	pb "pkg/proto/order"

	"order-service/internal/client"
	"order-service/internal/model"
	"order-service/internal/repository"

	"github.com/google/uuid"
)

const roleAdmin = "admin"

// یک اینترفیس برای جدا کردن منطق سرویس از RabbitMQ پس اصلا لایه
// نباید از جزییات این ارتباط مطلع باشد
// ویژگی این سه عملیات اینه که کسی منتظر جواب لحظه ای این ها نیست
// و بهتر است غیر همزمان انجام شوند
type EventPublisher interface {
	PublishOrderCreated(ctx context.Context, order *model.Order) error

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

type OrderService struct {
	orderRepo     repository.OrderRepository
	productClient client.ProductClient
	publisher     EventPublisher
}

func NewOrderService(
	orderRepo repository.OrderRepository,
	productClient client.ProductClient,
	publisher EventPublisher,
) *OrderService {

	return &OrderService{
		orderRepo:     orderRepo,
		productClient: productClient,
		publisher:     publisher,
	}
}

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

func parseOrderID(id string) (uuid.UUID, error) {

	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, appErrors.New(
			appErrors.KindInvalidInput,
			"invalid order id",
		)
	}

	return parsed, nil
}

// CreateOrder هسته‌ی Saga سمت Order Service است. مراحل:
//
//  1. اطلاعات تمامی محصولات را از سرویس محصولات دریافت میکنیم
//  2. اگر رزرو یک آیتم شکست بخورد، تمام آیتم‌های قبلی که تا این
//     لحظه با موفقیت رزرو شده‌اند آزاد میشود
//  3. سفارش ها در تراکنش های متفاوت ثبت میشن ولی در صورت مشکل
//     عملیات جبرانی compensation اتفاق می افته
//  4. اگر ثبت در دیتابیس شکست بخورد، تمام رزروها را آزاد میشود
//  5. رویداد order.created را منتشر می‌کند تا Payment Service
//     شروع به کار کند
func (
	s *OrderService,
) CreateOrder(
	ctx context.Context,
	req *pb.CreateOrderRequest,
) (
	*pb.Order,
	error,
) {

	// بررسی خالی نبودن درخواست
	if req == nil || len(req.Items) == 0 {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"order must contain at least one item",
		)
	}

	// استخراج دیتای احراز هویت از کانتکست
	claims, err := requireAuthenticated(ctx)
	if err != nil {
		return nil, err
	}

	// اعتبار سنجی آیدی
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, appErrors.New(
			appErrors.KindInternal,
			"invalid user id in token claims",
		)
	}

	// ساخت یک آرایه خالی حاوی آیتم های سفارش
	items := make([]*model.OrderItem, 0, len(req.Items))

	for _, reqItem := range req.Items {

		// بررسی خالی بودن آیتم سفارش
		if reqItem == nil {
			return nil, appErrors.New(
				appErrors.KindInvalidInput,
				"order item cannot be nil",
			)
		}

		// اگر یکی از آیتم ها تعداد کمتر از 1 داشت ارور میدهد
		if reqItem.Quantity <= 0 {

			return nil, appErrors.New(
				appErrors.KindInvalidInput,
				"item quantity must be positive",
			)
		}

		// بررسی اعتبار آیدی محصول
		productID, err := uuid.Parse(reqItem.ProductId)
		if err != nil {
			return nil, appErrors.New(
				appErrors.KindInvalidInput,
				"invalid product id: "+reqItem.ProductId,
			)
		}

		// بررسی وجودیت محصول
		product, err := s.productClient.GetProduct(ctx, reqItem.ProductId)
		if err != nil {

			return nil, err
		}

		// بررسی فعال بودن محصول
		if !product.IsActive {

			return nil, appErrors.New(
				appErrors.KindInvalidInput,
				"product is not available: "+product.Name,
			)

		}

		items = append(
			items,
			&model.OrderItem{
				ProductID:   productID,
				ProductName: product.Name,
				UnitPrice:   product.Price,
				Quantity:    reqItem.Quantity,
			},
		)
	}

	reservedItems := make([]*model.OrderItem, 0, len(items))

	for _, item := range items {

		err := s.productClient.ReserveStock(
			ctx,
			item.ProductID.String(),
			item.Quantity,
		)

		if err != nil {
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

		reservedItems = append(
			reservedItems,
			&model.OrderItem{
				ProductID:   item.ProductID,
				ProductName: item.ProductName,
				UnitPrice:   item.UnitPrice,
				Quantity:    item.Quantity,
			},
		)
	}

	order := &model.Order{
		UserID: userID,
		Items:  items,
	}
	order.TotalAmount = order.CalculateTotal()

	// اگر ذخیره کردن در جدول سفارش انجام نشد باید بقیه تغییرات هم به رول بک بشن
	if err := s.orderRepo.Create(ctx, order); err != nil {

		s.compensateReservations(ctx, reservedItems, "order_persist_failed")

		return nil, err
	}

	// انتشار ثبت سفارش برای استفاده در سرویس پرداخت
	if err := s.publisher.PublishOrderCreated(ctx, order); err != nil {

		log.Printf(
			"order-service: failed to publish order.created for order %s: %v",
			order.ID,
			err,
		)

	}

	return toProtoOrder(order), nil
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

// GetOrder فقط برای صاحب سفارش یا ادمین قابل مشاهده است
func (s *OrderService) GetOrder(
	ctx context.Context,
	req *pb.GetOrderRequest,
) (
	*pb.Order, error,
) {

	if req == nil {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"get order request is nil",
		)
	}

	claims, err := requireAuthenticated(ctx)
	if err != nil {
		return nil, err
	}

	id, err := parseOrderID(req.Id)
	if err != nil {
		return nil, err
	}

	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if order.UserID.String() != claims.UserID && claims.Role != roleAdmin {
		return nil, appErrors.New(
			appErrors.KindPermissionDenied,
			"you cannot view this order",
		)
	}

	return toProtoOrder(order), nil
}

// همیشه سفارش‌های همان کاربر احراز هویت‌شده را برمی‌گرداند
func (
	s *OrderService,
) ListMyOrders(
	ctx context.Context,
	req *pb.ListMyOrdersRequest,
) (
	*pb.ListMyOrdersResponse,
	error,
) {

	claims, err := requireAuthenticated(ctx)
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, appErrors.New(
			appErrors.KindInternal,
			"invalid user id in token claims",
		)
	}

	limit := 20
	offset := 0

	if req != nil {
		if req.Limit > 0 {
			limit = int(req.Limit)
		}
		offset = int(req.Offset)
	}

	orders, err := s.orderRepo.ListByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	return &pb.ListMyOrdersResponse{
		Orders: toProtoOrderList(orders),
	}, nil
}
