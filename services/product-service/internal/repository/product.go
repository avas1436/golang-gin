// services/product-service/internal/repository/product.go

package repository

import (
	"pkg/postgres"

	"context"
	"product-service/internal/model"

	"github.com/google/uuid"
)

// ProductRepository رابط دسترسی به داده‌ی محصولات را تعریف می‌کند
type ProductRepository interface {
	// ساخت یک محصول جدید
	Create(
		ctx context.Context,
		p *model.Product,
	) error

	// دریافت محصول با شناسه
	GetByID(
		ctx context.Context,
		id uuid.UUID,
	) (
		*model.Product, error,
	)

	// ویرایش محصول
	Update(
		ctx context.Context,
		p *model.Product,
	) error

	// جستجوی محصولات در بین نام و توضیحات به صورت فازی
	Search(
		ctx context.Context,
		query string,
		category string,
		limit,
		offset int,
	) (
		[]*model.Product, error,
	)

	// برای الگوی ساگا موجودی یک محصول را رزرو میکند
	ReserveStock(
		ctx context.Context,
		id uuid.UUID,
		quantity int32,
	) error

	// عملیات خنثی کننده رزرو در الگوی ساگا
	ReleaseStock(
		ctx context.Context,
		id uuid.UUID,
		quantity int32,
	) error

	// تایید نهایی فرایند خرید برای کم کردن کل موجودی
	ConfirmStock(
		ctx context.Context,
		id uuid.UUID,
		quantity int32,
	) error
}

type productRepository struct {
	db postgres.DBTX
}

// Create product repository
func NewProductRepository(db postgres.DBTX) ProductRepository {

	return &productRepository{
		db: db,
	}

}
