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

// VerifyPayment فراخوانی توسط API Gateway زمانی که کاربر از درگاه زرین‌پال بازمی‌گردد
func (
	s *GRPCServer,
) VerifyPayment(
	ctx context.Context,
	req *pb.VerifyPaymentRequest,
) (
	*pb.VerifyPaymentResponse,
	error,
) {

	payment, err := s.paymentService.VerifyPayment(
		ctx,
		req.Authority,
		req.Status,
	)
	if err != nil {
		return nil, grpcerrors.FromAppError(err, "payment-service")
	}

	refID := ""
	if payment.GatewayRefID != nil {
		refID = *payment.GatewayRefID
	}

	return &pb.VerifyPaymentResponse{
		OrderId: payment.OrderID.String(),
		Status:  string(payment.Status),
		RefId:   refID,
	}, nil
}
