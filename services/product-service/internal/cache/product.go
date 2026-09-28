// services/product-service/internal/cache/product.go

package cache

import (
	"context"
	"time"

	"product-service/internal/model"
	"product-service/internal/repository"

	"github.com/google/uuid"
)

// cachedProductRepository یک Decorator روی ProductRepository واقعی
// است: قبل از رفتن به دیتابیس، اول کش را چک می‌کند.
// چون همان اینترفیس repository.ProductRepository را پیاده می‌کند،
// لایه‌ی service اصلاً نمی‌فهمد کش وجود دارد یا نه؛ فقط در app.go
// موقع Dependency Injection این Decorator دور رپوزیتوری اصلی
// پیچیده می‌شود
type cachedProductRepository struct {
	// رپوزیتوری پایه سرویس
	repo repository.ProductRepository

	// استور لازم برای ساخت و اعمال کلید
	productCacheStore *ProductCacheStore

	// انقضای هر قسمت
	productTTL time.Duration
	searchTTL  time.Duration
}

// همان فرایند ساخت محصول جدید در رپوزیتوری که هیچ نیازی به کش ندارد
func (
	c *cachedProductRepository,
) Create(
	ctx context.Context,
	p *model.Product,
) error {

	return c.repo.Create(ctx, p)
}

// همان دریافت مشخصات محصول در رپو فقط اول وجود در کش را چک میکند
func (
	c *cachedProductRepository,
) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (
	*model.Product,
	error,
) {

	// دریافت مقدار کلید
	if p, ok, err := c.productCacheStore.GetProduct(ctx, id); err == nil && ok {
		return p, nil
	}

	// اگر در ردیس نبود دریافت از دیتابیس
	p, err := c.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// قرار دادن مقدار در دیتابیس
	_ = c.productCacheStore.SetProduct(ctx, p, c.productTTL)

	return p, nil

}

// GetByIDs ترکیب کش و دیتابیس برای جلوگیری از مشکل N+1
func (
	c *cachedProductRepository,
) GetByIDs(
	ctx context.Context,
	ids []uuid.UUID,
) (
	[]*model.Product,
	error,
) {

	// بررسی خالی بودن درخواست
	if len(ids) == 0 {
		return []*model.Product{}, nil
	}

	// یک لیست از نتایج دریافتی از ردیس
	result := make([]*model.Product, 0, len(ids))

	// آیدی هر محصولی که در ردیس نباشد در این لیست قرار میگیره
	missingIDs := make([]uuid.UUID, 0)

	for _, id := range ids {

		if p, ok, err := c.productCacheStore.GetProduct(
			ctx, id,
		); err == nil && ok {

			result = append(result, p)

		} else {

			missingIDs = append(missingIDs, id)
		}
	}

	// اگر هیچ آیدی ای miss نشده بود خروجی بده
	if len(missingIDs) == 0 {
		return result, nil
	}

	// دریافت آیدی های میس شده از دیتابیس
	fromDB, err := c.repo.GetByIDs(ctx, missingIDs)
	if err != nil {
		return nil, err
	}

	// قرار دادن نتایح در لیست خروجی
	for _, p := range fromDB {
		result = append(result, p)
		_ = c.productCacheStore.SetProduct(ctx, p, c.productTTL)
	}

	return result, nil
}

// Update دیتابیس را آپدیت می‌کند و بلافاصله کش قدیمی همان محصول را
// پاک می‌کند تا خواننده‌ی بعدی داده‌ی تازه ببیند
func (
	c *cachedProductRepository,
) Update(
	ctx context.Context,
	p *model.Product,
) error {

	// همان فرایند آپدیت محصول بدون هیچ کار اضافه ای
	if err := c.repo.Update(ctx, p); err != nil {
		return err
	}

	// اگر موفقیت آمیز بود پاک کردن مقدار محصول از کش
	_ = c.productCacheStore.InvalidateProduct(ctx, p.ID)

	return nil

}

// حذف نرم محصول
func (
	c *cachedProductRepository,
) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {

	if err := c.repo.Delete(ctx, id); err != nil {
		return err
	}

	_ = c.productCacheStore.InvalidateProduct(ctx, id)
	return nil
}

// Search اول کش را چک می‌کند؛ در صورت miss از دیتابیس می‌خواند و
// نتیجه را با TTL کوتاه‌تر کش می‌کند
func (
	c *cachedProductRepository,
) Search(
	ctx context.Context,
	query string,
	category string,
	limit, offset int,
) (
	[]*model.Product,
	error,
) {

	// ابتدا دریافت از ردیس
	if cached, ok, err := c.productCacheStore.GetSearch(
		ctx,
		query,
		category,
		limit,
		offset,
	); err == nil && ok {

		return cached, nil
	}

	// اگر نبود از دیتابیس
	products, err := c.repo.Search(
		ctx,
		query,
		category,
		limit,
		offset,
	)

	if err != nil {
		return nil, err
	}

	// قرار دادن در ردیس
	_ = c.productCacheStore.SetSearch(
		ctx,
		query,
		category,
		limit,
		offset,
		products,
		c.searchTTL,
	)

	return products, nil
}

// ReserveStock/ReleaseStock/ConfirmStock موجودی را در دیتابیس
// تغییر می‌دهند و سپس کش خودِ همان محصول را پاک می‌کنند، چون صفحه‌ی
// جزئیات محصول باید بلافاصله موجودی درست را نشان بدهد. نتایج
// جست‌وجو دوباره عمداً دست‌نخورده می‌مانند
func (
	c *cachedProductRepository,
) ReserveStock(
	ctx context.Context,
	id uuid.UUID,
	quantity int32,
) error {

	if err := c.repo.ReserveStock(ctx, id, quantity); err != nil {
		return err
	}

	_ = c.productCacheStore.InvalidateProduct(ctx, id)

	return nil
}

func (
	c *cachedProductRepository,
) ReleaseStock(
	ctx context.Context,
	id uuid.UUID,
	quantity int32,
) error {

	if err := c.repo.ReleaseStock(ctx, id, quantity); err != nil {
		return err
	}

	_ = c.productCacheStore.InvalidateProduct(ctx, id)

	return nil
}

func (
	c *cachedProductRepository,
) ConfirmStock(
	ctx context.Context,
	id uuid.UUID,
	quantity int32,
) error {

	if err := c.repo.ConfirmStock(ctx, id, quantity); err != nil {
		return err
	}

	_ = c.productCacheStore.InvalidateProduct(ctx, id)

	return nil
}
