// services/order-service/internal/handler/order.go

package handler

import (
	"context"
	"pkg/grpcerrors"

	"order-service/internal/service"
	pb "pkg/proto/order"
)

// GRPCServer
type GRPCServer struct {
	pb.UnimplementedOrderServiceServer // برای Forward Compatibility
	orderService                       *service.OrderService
}

// یک نمونه جدید از سرور gRPC را می‌سازد
func NewGRPCServer(orderService *service.OrderService) *GRPCServer {

	return &GRPCServer{
		orderService: orderService,
	}

}

// CreateOrder
func (
	s *GRPCServer,
) CreateOrder(
	ctx context.Context,
	req *pb.CreateOrderRequest,
) (
	*pb.Order,
	error,
) {

	resp, err := s.orderService.CreateOrder(ctx, req)
	if err != nil {
		return nil, grpcerrors.FromAppError(err, "order-service")
	}

	return resp, nil
}

// GetOrder
func (
	s *GRPCServer,
) GetOrder(
	ctx context.Context,
	req *pb.GetOrderRequest,
) (
	*pb.Order,
	error,
) {

	resp, err := s.orderService.GetOrder(ctx, req)
	if err != nil {
		return nil, grpcerrors.FromAppError(err, "order-service")
	}

	return resp, nil
}

// ListMyOrders
func (
	s *GRPCServer,
) ListMyOrders(
	ctx context.Context,
	req *pb.ListMyOrdersRequest,
) (
	*pb.ListMyOrdersResponse,
	error,
) {

	resp, err := s.orderService.ListMyOrders(ctx, req)
	if err != nil {
		return nil, grpcerrors.FromAppError(err, "order-service")
	}

	return resp, nil
}
