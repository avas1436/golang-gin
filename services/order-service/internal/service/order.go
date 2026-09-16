// services/order-service/internal/service/order.go

package service

import (
	"context"
	"log"

	appErrors "pkg/errors"
	pb "pkg/proto/order"

	"order-service/internal/client"
	"order-service/internal/model"
	"order-service/internal/repository"

	"github.com/google/uuid"
)

const roleAdmin = "admin"

type OrderService struct {
	orderRepo     repository.OrderRepository
	productClient client.ProductClient
	publisher     EventPublisher
}

// CreateOrder هسته‌ی Saga سمت Order Service است.
//
// مراحل:
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

	// مشخصات محصولات رو در یک حلقه به صورت همزمان یکی یکی میگیره
	// و بعد و در لیست items برامون قرار میده
	items, err := s.buildOrderItems(ctx, req.Items)
	if err != nil {
		return nil, err
	}

	// به صورت همزمان یکی یکی محصولات رو رزرو میکنه
	reservedItems, err := s.reserveItems(ctx, items)
	if err != nil {
		return nil, err
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

		// 1. لاگ کردن خطا
		log.Printf(
			"order-service: failed to publish order.created for order %s: %v",
			order.ID,
			err,
		)

		// 2. آزاد کردن رزروها در سرویس محصول
		s.compensateReservations(ctx, reservedItems, "publish_failed")

		// 3. در دیتابیس هم سفارش کنسل میشود
		if updateErr := s.orderRepo.UpdateStatus(
			ctx,
			order.ID,
			"failed",
		); updateErr != nil {

			log.Printf(
				"order-service: failed to mark order %s as failed: %v",
				order.ID,
				updateErr,
			)

		}

		return nil, appErrors.New(
			appErrors.KindInternal,
			"failed to start order processing",
		)

	}

	return toProtoOrder(order), nil
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
