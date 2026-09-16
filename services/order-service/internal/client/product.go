// services/order-service/internal/client/product.go

package client

import (
	"context"

	"pkg/grpcclient"
	"pkg/grpcerrors"
	pb "pkg/proto/product"
)

// یک اینترفیس برای ارتباطات سینک با سرویس محصولات
type ProductClient interface {
	GetProduct(ctx context.Context, productID string) (*pb.Product, error)
	ReserveStock(ctx context.Context, productID string, quantity int32) error
}

type productClient struct {
	client pb.ProductServiceClient
}

// NewProductClient یک اتصال gRPC به Product Service باز می‌کند.
// از grpcclient.ClientManager موجود در pkg استفاده می‌شود تا
// مدیریت اتصال (و بسته‌شدنش هنگام خاموش‌شدن سرویس) یکجا و یکسان
// با بقیه‌ی جاهایی باشد که به سرویس دیگری وصل می‌شوند
func NewProductClient(
	manager *grpcclient.ClientManager,
	addr string,
) (
	ProductClient, error,
) {

	conn, err := manager.Dial(addr)
	if err != nil {
		return nil, err
	}

	return &productClient{
		client: pb.NewProductServiceClient(conn),
	}, nil
}

// ارسال درخواست به سرویس محصولات برای دریافت اطلاعات یک محصول
func (c *productClient) GetProduct(
	ctx context.Context,
	productID string,
) (
	*pb.Product, error,
) {

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

// ارسال درخواست افزودن یک رزرو به سرویس محصولات
func (c *productClient) ReserveStock(
	ctx context.Context,
	productID string,
	quantity int32,
) error {

	_, err := c.client.ReserveStock(
		ctx,
		&pb.ReserveStockRequest{
			ProductId: productID,
			Quantity:  quantity,
		})

	return grpcerrors.ToAppError(err)
}
