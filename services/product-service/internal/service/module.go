// services/product-service/internal/service/module.go

package service

import (
	"product-service/internal/repository"

	"go.uber.org/fx"
)

// سازنده یک ساختار سرویس محصول
func NewProductService(
	productRepo repository.ProductRepository,
) *ProductService {

	return &ProductService{
		productRepo: productRepo,
	}
}

var Module = fx.Module(
	"service",
	fx.Provide(
		NewProductService,
	),
)
