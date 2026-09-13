// services/order-service/internal/client/module.go

package client

import (
	"pkg/grpcclient"

	commonConfig "order-service/config"

	"go.uber.org/fx"
)

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
