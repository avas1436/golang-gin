// services/product-service/internal/repository/product.go

package repository

import (
	stdErrors "errors"

	appErrors "pkg/errors"
	"pkg/postgres"

	"context"
	"product-service/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

// ساخت محصول جدید
func (
	r *productRepository,
) Create(
	ctx context.Context,
	p *model.Product,
) error {

	if p == nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"product cannot be nil",
		)
	}

	if err := p.Validate(); err != nil {
		return err
	}

	query := `
		INSERT INTO products (
			name, 
			description, 
			category, 
			price, 
			total_stock
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, reserved_stock, is_active, created_at, updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		p.Name,
		p.Description,
		p.Category,
		p.Price,
		p.TotalStock,
	).Scan(
		&p.ID,
		&p.ReservedStock,
		&p.IsActive,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to create product in database",
		)
	}

	return nil
}

// دریافت اصلاعات کامل محصول
func (
	r *productRepository,
) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (
	*model.Product, error,
) {

	query := `
		SELECT
			id, 
			name, 
			description, 
			category, 
			price,
			total_stock, 
			reserved_stock, 
			is_active,
			created_at, 
			updated_at
		FROM products
		WHERE id = $1
	`

	p := &model.Product{}

	err := r.db.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&p.ID,
		&p.Name,
		&p.Description,
		&p.Category,
		&p.Price,
		&p.TotalStock,
		&p.ReservedStock,
		&p.IsActive,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if err != nil {
		if stdErrors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.New(
				appErrors.KindNotFound,
				"product not found",
			)
		}

		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to get product by id",
		)
	}

	return p, nil
}

// در این تابع فیلد های موجودی قابل تغییر نیست
func (
	r *productRepository,
) Update(
	ctx context.Context,
	p *model.Product,
) error {

	if p == nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"product cannot be nil",
		)
	}

	if err := p.Validate(); err != nil {
		return err
	}

	query := `
		UPDATE products
		SET
			name        = $1,
			description = $2,
			category    = $3,
			price       = $4,
			is_active   = $5,
			updated_at  = NOW()
		WHERE id = $6
		RETURNING updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		p.Name,
		p.Description,
		p.Category,
		p.Price,
		p.IsActive,
		p.ID,
	).Scan(&p.UpdatedAt)

	if err != nil {
		if stdErrors.Is(err, pgx.ErrNoRows) {
			return appErrors.New(
				appErrors.KindNotFound,
				"product not found",
			)
		}

		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to update product",
		)
	}

	return nil
}

// Search از ایندکس GIN روی (name || ' ' || description) با
// pg_trgm استفاده می‌کند تا substring/fuzzy search سریع باشد.
// اگر query خالی باشد، فقط فیلتر category اعمال می‌شود
func (r *productRepository) Search(
	ctx context.Context,
	query string,
	category string,
	limit int,
	offset int,
) (
	[]*model.Product, error,
) {

	if limit <= 0 {
		limit = 20
	}

	sql := `
		SELECT
			id, 
			name, 
			description, 
			category, 
			price,
			total_stock, 
			reserved_stock, 
			is_active,
			created_at, 
			updated_at
		FROM products
		WHERE is_active
		  AND ($1 = '' OR category = $1)
		  AND ($2 = '' OR (name || ' ' || description) % $2)
		ORDER BY
			CASE WHEN $2 = '' THEN created_at END DESC,
			CASE WHEN $2 <> '' THEN similarity(name || ' ' || description, $2) END DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.Query(
		ctx,
		sql,
		category,
		query,
		limit,
		offset,
	)

	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to search products",
		)
	}
	defer rows.Close()

	products := make([]*model.Product, 0)

	for rows.Next() {
		p := &model.Product{}

		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Description,
			&p.Category,
			&p.Price,
			&p.TotalStock,
			&p.ReservedStock,
			&p.IsActive,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {

			return nil, appErrors.Wrap(
				appErrors.KindInternal,
				err,
				"failed to scan product row",
			)

		}

		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"error iterating product rows",
		)
	}

	return products, nil
}

// ReserveStock هسته‌ی همگام‌سازی Saga است: به‌جای SELECT سپس
// UPDATE (که بین آن دو race condition ممکن است رخ دهد)، یک UPDATE
// شرطی تک‌مرحله‌ای اجرا می‌شود. PostgreSQL خودش هنگام UPDATE یک
// قفل ردیفی می‌گیرد، پس نیازی به SELECT ... FOR UPDATE یا ستون
// version نیست. اگر شرط WHERE برقرار نباشد (موجودی ناکافی)،
// RowsAffected برابر صفر می‌شود و ما آن را به خطای دامنه تبدیل می‌کنیم
func (
	r *productRepository,
) ReserveStock(
	ctx context.Context,
	id uuid.UUID,
	quantity int32,
) error {

	if quantity <= 0 {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"quantity must be positive",
		)
	}

	query := `
		UPDATE products
		SET reserved_stock = reserved_stock + $1,
		    updated_at     = NOW()
		WHERE id = $2
		  AND is_active
		  AND (total_stock - reserved_stock) >= $1
	`

	result, err := r.db.Exec(ctx, query, quantity, id)
	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to reserve stock",
		)
	}

	if result.RowsAffected() == 0 {
		// یا محصول اصلاً وجود ندارد، یا غیرفعال است، یا موجودی
		// کافی نیست؛ برای سادگی هر سه را در قالب یک خطای واحد
		// دامنه‌ای گزارش می‌کنیم چون از دید سرویس مصرف کنند
		//  یعنی سرویس سفارشات، واکنش یکسانی دارند: نمی‌شود رزرو کرد
		return appErrors.New(
			appErrors.KindInvalidInput,
			"insufficient stock or product not available",
		)
	}

	return nil
}

// ReleaseStock عملیات جبرانی است. برخلاف ReserveStock، محدودیت
// موجودی معنا ندارد؛ فقط باید مطمئن شویم reserved_stock منفی
// نمی‌شود که همین کار را شرط WHERE و CHECK constraint دیتابیس
// تضمین می‌کنند
func (
	r *productRepository,
) ReleaseStock(
	ctx context.Context,
	id uuid.UUID,
	quantity int32,
) error {

	if quantity <= 0 {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"quantity must be positive",
		)
	}

	query := `
		UPDATE products
		SET reserved_stock = reserved_stock - $1,
		    updated_at     = NOW()
		WHERE id = $2
		  AND reserved_stock >= $1
	`

	result, err := r.db.Exec(ctx, query, quantity, id)
	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to release stock",
		)
	}

	if result.RowsAffected() == 0 {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"cannot release more stock than reserved",
		)
	}

	return nil
}

// ConfirmStock وقتی صدا زده می‌شود که پرداخت قطعی شده؛ رزرو به
// مصرف واقعی تبدیل می‌شود، پس هم total_stock و هم reserved_stock
// به یک اندازه کاهش پیدا می‌کنند
func (
	r *productRepository,
) ConfirmStock(
	ctx context.Context,
	id uuid.UUID,
	quantity int32,
) error {

	if quantity <= 0 {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"quantity must be positive",
		)
	}

	query := `
		UPDATE products
		SET total_stock    = total_stock - $1,
		    reserved_stock = reserved_stock - $1,
		    updated_at     = NOW()
		WHERE id = $2
		  AND reserved_stock >= $1
	`

	result, err := r.db.Exec(ctx, query, quantity, id)
	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to confirm stock",
		)
	}

	if result.RowsAffected() == 0 {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"cannot confirm more stock than reserved",
		)
	}

	return nil
}
