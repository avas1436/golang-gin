// services/product-service/internal/model/product.go

package model

import (
	appErrors "pkg/errors"
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID            uuid.UUID `db:"id" json:"id"`
	Name          string    `db:"name" json:"name"`
	Description   string    `db:"description" json:"description"`
	Category      string    `db:"category" json:"category"`
	Price         int64     `db:"price" json:"price"`
	TotalStock    int32     `db:"total_stock" json:"total_stock"`
	ReservedStock int32     `db:"reserved_stock" json:"reserved_stock"`
	IsActive      bool      `db:"is_active" json:"is_active"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

// AvailableStock محاسبه موجودی قابل فروش در سطح مدل
func (p *Product) AvailableStock() int32 {
	return p.TotalStock - p.ReservedStock
}

// Validate بررسی قوانین دامنه پیش از ذخیره‌سازی
func (p *Product) Validate() error {
	if p.Price < 0 {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"price cannot be negative",
		)
	}
	if p.ReservedStock > p.TotalStock {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"reserved stock cannot exceed total stock",
		)
	}
	return nil
}
