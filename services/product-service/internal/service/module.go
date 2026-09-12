// services/product-service/internal/service/module.go

package service

import "go.uber.org/fx"

var Module = fx.Module(
	"service",
	fx.Provide(
		NewProductService,
	),
)
