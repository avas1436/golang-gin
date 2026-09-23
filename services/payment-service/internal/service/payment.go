// services/payment-service/internal/service/payment.go

package service

import (
	"context"
	"log"

	appErrors "pkg/errors"
	"pkg/postgres"
	pb "pkg/proto/payment"

	"payment-service/internal/model"
	"payment-service/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventPublisher چیزی است که PaymentService برای انتشار نتیجه‌ی
// نهایی پرداخت نیاز دارد. پیاده‌سازی آن در internal/messaging است
type EventPublisher interface {
	PublishPaymentCompleted(
		ctx context.Context,
		paymentID uuid.UUID,
		orderID uuid.UUID,
		gatewayName string,
		gatewayRefID string,
	) error

	PublishPaymentFailed(
		ctx context.Context,
		paymentID uuid.UUID,
		orderID uuid.UUID,
		reason string,
	) error
}

type PaymentService struct {
	pool        *pgxpool.Pool
	paymentRepo repository.PaymentRepository
	publisher   EventPublisher
}

// HandleOrderCreated زمانی اجرا می‌شود که payment-service
// پیام order.created را دریافت می‌کند.
//
// جریان:
//
//  1. شروع Transaction
//
//  2. ثبت event در processed_events
//
//  3. اگر event قبلاً پردازش شده بود:
//     Transaction تمام می‌شود و ادامه نمی‌دهیم.
//
//  4. ایجاد Payment با وضعیت pending
//
//  5. Commit
//
//  6. تماس با Gateway خارج از DB Transaction
func (
	s *PaymentService,
) HandleOrderCreated(
	ctx context.Context,
	eventID uuid.UUID,
	orderID uuid.UUID,
	userID uuid.UUID,
	amount int64,
) error {

	// ساخت مدل Payment با وضعیت pending
	payment := model.NewPendingPayment(
		orderID,
		userID,
		amount,
	)

	var alreadyHandled bool

	err := postgres.WithTx(
		ctx,
		s.pool,
		func(tx pgx.Tx) error {

			// این Repository مخصوص همین Transaction است.
			//
			// بنابراین تمام Queryهای paymentRepo داخل همین tx
			// اجرا می‌شوند.
			txPaymentRepo := repository.NewPaymentRepository(tx)

			// Event repository نیز روی همان Transaction کار می‌کند.
			eventRepo := repository.NewEventRepository(tx)

			alreadyProcessed, err := eventRepo.MarkProcessed(
				ctx,
				eventID,
				"order.created",
				&orderID,
				nil,
			)
			if err != nil {
				return err
			}

			if alreadyProcessed {
				log.Printf(
					"payment-service: order.created event %s already processed, skipping",
					eventID,
				)
				alreadyHandled = true
				return nil
			}

			return txPaymentRepo.Create(ctx, payment)
		},
	)

	if err != nil || alreadyHandled {
		return err
	}

	s.simulateGatewayAndFinalize(ctx, payment)

	return nil
}

// GetPaymentByOrderID تنها متد gRPC این سرویس است؛ صرفاً برای
// دیباگ/ادمین است و بخشی از جریان اصلی Saga نیست
func (
	s *PaymentService,
) GetPaymentByOrderID(
	ctx context.Context,
	req *pb.GetPaymentByOrderIDRequest,
) (
	*pb.Payment,
	error,
) {

	if req == nil {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"get payment by order id request is nil",
		)
	}

	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"invalid order id",
		)
	}

	payment, err := s.paymentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	return toProtoPayment(payment), nil
}
