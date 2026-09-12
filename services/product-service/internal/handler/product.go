// services/product-service/internal/handler/product_handler.go

package handler

import (
	"context"
	"pkg/grpcerrors"

	pb "pkg/proto/product"
	"product-service/internal/service"
)

// GRPCServer
type GRPCServer struct {
	pb.UnimplementedProductServiceServer // برای Forward Compatibility
	productService                       *service.ProductService
}

// یک نمونه جدید از سرور gRPC را می‌سازد
func NewGRPCServer(productService *service.ProductService) *GRPCServer {

	return &GRPCServer{
		productService: productService,
	}

}

// CreateProduct
func (
	s *GRPCServer,
) CreateProduct(
	ctx context.Context,
	req *pb.CreateProductRequest,
) (
	*pb.Product, error,
) {

	resp, err := s.productService.CreateProduct(ctx, req)
	if err != nil {
		return nil, grpcerrors.FromAppError(err, "product-service")
	}

	return resp, nil
}

// UpdateProduct
func (
	s *GRPCServer,
) UpdateProduct(
	ctx context.Context,
	req *pb.UpdateProductRequest,
) (
	*pb.Product, error,
) {

	resp, err := s.productService.UpdateProduct(ctx, req)
	if err != nil {
		return nil, grpcerrors.FromAppError(err, "product-service")
	}

	return resp, nil
}

// GetProduct
func (
	s *GRPCServer,
) GetProduct(
	ctx context.Context,
	req *pb.GetProductRequest,
) (
	*pb.Product, error,
) {

	resp, err := s.productService.GetProduct(ctx, req)
	if err != nil {
		return nil, grpcerrors.FromAppError(err, "product-service")
	}

	return resp, nil
}

// SearchProducts
func (
	s *GRPCServer,
) SearchProducts(
	ctx context.Context,
	req *pb.SearchProductsRequest,
) (
	*pb.SearchProductsResponse, error,
) {

	resp, err := s.productService.SearchProducts(ctx, req)
	if err != nil {
		return nil, grpcerrors.FromAppError(err, "product-service")
	}

	return resp, nil
}

// ReserveStock
func (
	s *GRPCServer,
) ReserveStock(
	ctx context.Context,
	req *pb.ReserveStockRequest,
) (
	*pb.ReserveStockResponse, error,
) {

	resp, err := s.productService.ReserveStock(ctx, req)
	if err != nil {
		return nil, grpcerrors.FromAppError(err, "product-service")
	}

	return resp, nil
}

// ReleaseStock
func (
	s *GRPCServer,
) ReleaseStock(
	ctx context.Context,
	req *pb.ReleaseStockRequest,
) (
	*pb.ReleaseStockResponse, error,
) {

	resp, err := s.productService.ReleaseStock(ctx, req)
	if err != nil {
		return nil, grpcerrors.FromAppError(err, "product-service")
	}

	return resp, nil
}

// ConfirmStock
func (
	s *GRPCServer,
) ConfirmStock(
	ctx context.Context,
	req *pb.ConfirmStockRequest,
) (
	*pb.ConfirmStockResponse, error,
) {

	resp, err := s.productService.ConfirmStock(ctx, req)
	if err != nil {
		return nil, grpcerrors.FromAppError(err, "product-service")
	}

	return resp, nil
}
