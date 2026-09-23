// services/payment-service/internal/handler/payment.go

package handler

import (
	"context"

	"pkg/grpcerrors"
	pb "pkg/proto/payment"

	"payment-service/internal/service"
)

// GRPCServer مسئول هندل کردن RPCهای ورودی gRPC در لایه Handler است
type GRPCServer struct {
	pb.UnimplementedPaymentServiceServer // برای Forward Compatibility
	paymentService                       *service.PaymentService
}

// GetPaymentByOrderID اطلاعات پرداخت را بر اساس شناسه سفارش برمی‌گرداند
func (
	s *GRPCServer,
) GetPaymentByOrderID(
	ctx context.Context,
	req *pb.GetPaymentByOrderIDRequest,
) (
	*pb.Payment,
	error,
) {

	resp, err := s.paymentService.GetPaymentByOrderID(ctx, req)
	if err != nil {
		return nil, grpcerrors.FromAppError(err, "payment-service")
	}

	return resp, nil
}
