// services/product-service/internal/model/product.go

package model

import "time"

type Product struct {
	ID            string
	Name          string
	Description   string
	Category      string
	Price         string
	TotalStock    int64
	ReservedStock int64
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
