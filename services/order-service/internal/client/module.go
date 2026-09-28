// services/order-service/internal/client/module.go

package client

import (
	"pkg/grpcclient"
	pb "pkg/proto/product"

	commonConfig "order-service/config"

	"go.uber.org/fx"
)

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

var Module = fx.Module(
	"client",
	fx.Provide(
		// تعریف چگونگی ارائه ProductClient
		func(
			manager *grpcclient.ClientManager,
			cfg *commonConfig.Config,
		) (
			ProductClient,
			error,
		) {

			return NewProductClient(manager, cfg.ProductServiceAddr)

		},
	),
)
