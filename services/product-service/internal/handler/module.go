// services/product-service/internal/handler/handler.go

package handler

import (
	"context"
	pb "pkg/proto/product"
	"product-service/internal/service"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"handler",
	fx.Provide(
		NewGRPCHandler,
	),
)

type GRPCHandler struct {
	pb.UnimplementedProductServiceServer
	productService service.ProductService
}

func NewGRPCHandler(svc service.ProductService) pb.ProductServiceServer {

	return &GRPCHandler{
		productService: svc,
	}

}

// در زمان اجرای درخواست، ctx توسط سرور gRPC تولید شده و تا لایه دیتابیس پاس داده می‌شود
func (
	h *GRPCHandler,
) GetProduct(
	ctx context.Context,
	req *pb.GetProductRequest,
) (
	*pb.ProductResponse,
	error,
) {

	return h.productService.GetProduct(ctx, req.GetId())

}
