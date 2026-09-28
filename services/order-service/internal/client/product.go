// services/order-service/internal/client/product.go

package client

import (
	"context"
	"time"

	"pkg/grpcerrors"
	pb "pkg/proto/product"
)

const defaultRequestTimeout = 3 * time.Second

// یک اینترفیس برای ارتباطات سینک با سرویس محصولات
type ProductClient interface {
	GetProduct(ctx context.Context, productID string) (*pb.Product, error)

	GetProductsByIDs(
		ctx context.Context,
		productIDs []string,
	) (
		[]*pb.Product,
		error,
	)

	ReserveStock(ctx context.Context, productID string, quantity int32) error
}

type productClient struct {
	client pb.ProductServiceClient
}

// ارسال درخواست به سرویس محصولات برای دریافت اطلاعات یک محصول
func (c *productClient) GetProduct(
	ctx context.Context,
	productID string,
) (
	*pb.Product, error,
) {

	// ایجاد یک Timeout دفاعی در صورت عدم وجود Deadline در Context ورودی
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	product, err := c.client.GetProduct(
		ctx,
		&pb.GetProductRequest{
			Id: productID,
		})
	if err != nil {
		return nil, grpcerrors.ToAppError(err)
	}

	return product, nil
}

// GetProductsByIDs دریافت دسته‌جمعی اطلاعات چند محصول در یک درخواست
func (c *productClient) GetProductsByIDs(
	ctx context.Context,
	productIDs []string,
) (
	[]*pb.Product,
	error,
) {

	// اعتبار سنجی ورودی
	if len(productIDs) == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	resp, err := c.client.GetProductsByIDs(
		ctx,
		&pb.GetProductsByIDsRequest{
			Ids: productIDs,
		},
	)
	if err != nil {
		return nil, grpcerrors.ToAppError(err)
	}

	return resp.GetProducts(), nil
}

// ارسال درخواست افزودن یک رزرو به سرویس محصولات
func (c *productClient) ReserveStock(
	ctx context.Context,
	productID string,
	quantity int32,
) error {

	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	_, err := c.client.ReserveStock(
		ctx,
		&pb.ReserveStockRequest{
			ProductId: productID,
			Quantity:  quantity,
		})

	if err != nil {
		return grpcerrors.ToAppError(err)
	}

	return nil
}
