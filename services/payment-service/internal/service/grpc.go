// services/payment-service/internal/service/grpc.go

package service

import (
	"context"
	"pkg/auth"
	appErrors "pkg/errors"
	pb "pkg/proto/payment"

	"github.com/google/uuid"
)

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

	// اجازه دسترسی علاوه بر Admin به خود کاربر صاحب پرداخت
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return nil, appErrors.New(
			appErrors.KindPermissionDenied,
			"you cannot view this payment details",
		)
	}

	if claims.Role != roleAdmin && claims.UserID != payment.UserID.String() {
		return nil, appErrors.New(
			appErrors.KindPermissionDenied,
			"you cannot view this payment details",
		)
	}

	return toProtoPayment(payment), nil
}
