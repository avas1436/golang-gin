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
type EventPublisher interface {
	PublishOrderCreated(ctx context.Context, order *model.Order) error

	PublishStockReleaseRequested(
		ctx context.Context,
		productID uuid.UUID,
		quantity int32,
		reason string,
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
//  1. برای مشخصات محصول یک درخواست همزمان به سرویس محصولات
//     و مشخصات محصول را میگیریم
//  2. اگر رزرو یک آیتم شکست بخورد، تمام آیتم‌های قبلی که تا این
//     لحظه با موفقیت رزرو شده‌اند آزاد میشود
//  3. سفارش در یک تراکنش واحد ثبت میشود
//  4. اگر ثبت در دیتابیس شکست بخورد، تمام رزروها را آزاد میشود
//  5. رویداد order.created را منتشر می‌کند تا Payment Service
//     شروع به کار کند
func (
	s *OrderService,
) CreateOrder(
	ctx context.Context,
	req *pb.CreateOrderRequest,
) (
	*pb.Order, error,
) {

	if req == nil || len(req.Items) == 0 {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"order must contain at least one item",
		)
	}

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

	items := make([]*model.OrderItem, 0, len(req.Items))

	for _, reqItem := range req.Items {

		if reqItem.Quantity <= 0 {

			s.compensateReservations(ctx, items, "invalid_quantity")

			return nil, appErrors.New(
				appErrors.KindInvalidInput,
				"item quantity must be positive",
			)
		}

		product, err := s.productClient.GetProduct(ctx, reqItem.ProductId)
		if err != nil {

			s.compensateReservations(ctx, items, "product_lookup_failed")

			return nil, err
		}

		if !product.IsActive {

			s.compensateReservations(ctx, items, "product_inactive")

			return nil, appErrors.New(
				appErrors.KindInvalidInput,
				"product is not available: "+product.Name,
			)
		}

		if err := s.productClient.ReserveStock(
			ctx,
			reqItem.ProductId,
			reqItem.Quantity,
		); err != nil {

			s.compensateReservations(ctx, items, "reservation_failed")

			return nil, err
		}

		productID, err := uuid.Parse(reqItem.ProductId)
		if err != nil {
			// این حالت عملاً نباید برسد چون GetProduct/ReserveStock
			// قبلش با همین رشته موفق بودند؛ برای اطمینان کامل همین
			// آیتمی که تازه رزرو شد را هم جبران می‌کنیم
			s.publishRelease(
				ctx,
				reqItem.ProductId,
				reqItem.Quantity,
				"internal_error",
			)

			s.compensateReservations(ctx, items, "internal_error")

			return nil, appErrors.New(
				appErrors.KindInternal,
				"invalid product id returned from reservation",
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

	order := &model.Order{
		UserID: userID,
		Items:  items,
	}
	order.TotalAmount = order.CalculateTotal()

	if err := s.orderRepo.Create(ctx, order); err != nil {

		s.compensateReservations(ctx, items, "order_persist_failed")

		return nil, err
	}

	// Dual-Write Problem: سفارش همین الان commit شده. اگر Publish
	// زیر شکست بخورد، دیگر نمی‌شود (و نباید) سفارشی را که مشتری
	// برایش موجودی رزرو کرده rollback کرد؛ برای همین این خطا fatal
	// نیست، فقط لاگ می‌شود. راه‌حل کامل این مشکل الگوی Transactional
	// Outbox است که فعلاً خارج از scope همین قدم است
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
			ctx, item.ProductID, item.Quantity, reason,
		); err != nil {
			log.Printf(
				"order-service: failed to publish stock release for product %s: %v",
				item.ProductID, err,
			)
		}
	}
}

// publishRelease یک تک‌آیتم را به‌صورت event آزاد می‌کند. خطای آن
// فقط لاگ می‌شود: این خودش تابع compensation است، پس اگر دوباره
// خطا بدهد جایی برای «جبرانِ جبران» وجود ندارد — نهایتاً باید یک
// dead-letter queue یا alerting روی همین صف تعریف شود (خارج از
// scope فعلی)
func (
	s *OrderService,
) publishRelease(
	ctx context.Context,
	productID string,
	quantity int32,
	reason string,
) {

	id, err := uuid.Parse(productID)
	if err != nil {
		log.Printf(
			"order-service: cannot publish release for invalid product id %q: %v",
			productID, err,
		)
		return
	}

	if err := s.publisher.PublishStockReleaseRequested(
		ctx,
		id,
		quantity,
		reason,
	); err != nil {
		log.Printf(
			"order-service: failed to publish stock release for product %s: %v",
			id,
			err,
		)
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
