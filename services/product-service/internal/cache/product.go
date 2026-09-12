// services/product-service/internal/cache/product.go

package cache

import (
	"context"
	"strconv"
	"time"

	pkgcache "pkg/cache"

	"product-service/internal/model"
	"product-service/internal/repository"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// نام محیط جداگانه برای سرویس محصولات
const namespace = "product-service"

// cachedProductRepository یک Decorator روی ProductRepository واقعی
// است: قبل از رفتن به دیتابیس، اول کش را چک می‌کند.
// چون همان اینترفیس repository.ProductRepository را پیاده می‌کند،
// لایه‌ی service اصلاً نمی‌فهمد کش وجود دارد یا نه؛ فقط در app.go
// موقع Dependency Injection این Decorator دور رپوزیتوری اصلی
// پیچیده می‌شود
type cachedProductRepository struct {
	// رپوزیتوری پایه سرویس
	repo repository.ProductRepository

	// دیتا تایت های مورد نیاز
	products *pkgcache.Cache[model.Product]
	searches *pkgcache.Cache[[]*model.Product]

	// کلید های namespace ها
	keys KeyConfig

	// انقضای هر قسمت
	productTTL time.Duration
	searchTTL  time.Duration
}

// KeyConfig تفکیک namespace را به بیرون از این فایل expose نمی‌کند؛
// فقط برای خوانایی سازنده استفاده می‌شود
type KeyConfig struct {
	Products pkgcache.KeyBuilder
	Searches pkgcache.KeyBuilder
}

// NewCachedProductRepository یک ProductRepository برمی‌گرداند که
// همان رفتار repo را دارد، به‌علاوه‌ی کش خودکار برای GetByID و
// Search. productTTL/searchTTL عمداً جدا از هم هستند: جزئیات یک
// محصول (نام، توضیحات) کم‌تغییر است پس TTL بلندتر منطقی است، اما
// نتایج جست‌وجو به موجودی لحظه‌ای وابسته‌اند پس باید زودتر منقضی
// شوند
func NewCachedProductRepository(
	repo repository.ProductRepository,
	client *redis.Client,
	productTTL time.Duration,
	searchTTL time.Duration,
) repository.ProductRepository {

	return &cachedProductRepository{
		// رپوزیتوری
		repo: repo,

		// قسمت های مختلف با تایپ های مختلف
		products: pkgcache.New[model.Product](client),
		searches: pkgcache.New[[]*model.Product](client),

		// کلید های هر قسمت
		keys: KeyConfig{
			Products: pkgcache.NewKeyBuilder(namespace + ":product"),
			Searches: pkgcache.NewKeyBuilder(namespace + ":search"),
		},

		// انقضای هر قسمت
		productTTL: productTTL,
		searchTTL:  searchTTL,
	}
}

// ساخت کلید برای هر محصول
func (c *cachedProductRepository) productKey(id uuid.UUID) string {
	return c.keys.Products.Build(id.String())
}

// از namespase جستجو شروع کرده و با قرار دادن تمامی پارامتر های
// دخیل در نتیجه یک جستجو کلید کش برای جستجو میسازد
func (
	c *cachedProductRepository,
) searchKey(
	query,
	category string,
	limit,
	offset int,
) string {

	return c.keys.Searches.Build(
		category,
		query,
		strconv.Itoa(limit),
		strconv.Itoa(offset),
	)
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
	*model.Product, error,
) {

	// کلید در این قسمت ساخته میشود
	// از namespace مربوطه شروع میکند و تا اجزا پیش میرود
	key := c.productKey(id)

	// در این قسمت تنها مدل را میسازیم و به هردلیلی ارور گرفتیم رد میشویم
	// سیستم fail-open است تا از دسترس خارج شدن ردیس تنها در
	// پروفورمنس سیستم اثر بگذارد نه دسترسی
	if cached, ok, err := c.products.Get(ctx, key); err == nil && ok {

		p := cached

		return &p, nil
	}

	// اگر ردیس در دسترس نبود میرویم سراغ رپوزیتوری و دیتا را از دیتابیس
	// استخراج میکنیم
	p, err := c.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// خطای Set هم نادیده گرفته می‌شود؛ نبود کش فقط روی پرفورمنس
	// اثر دارد نه درستی پاسخ
	_ = c.products.Set(ctx, key, *p, c.productTTL)

	return p, nil
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
	_ = c.products.Delete(ctx, c.productKey(p.ID))

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

	key := c.searchKey(query, category, limit, offset)

	if cached, ok, err := c.searches.Get(ctx, key); err == nil && ok {
		return cached, nil
	}

	products, err := c.repo.Search(ctx, query, category, limit, offset)
	if err != nil {
		return nil, err
	}

	_ = c.searches.Set(ctx, key, products, c.searchTTL)

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

	_ = c.products.Delete(ctx, c.productKey(id))

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

	_ = c.products.Delete(ctx, c.productKey(id))

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

	_ = c.products.Delete(ctx, c.productKey(id))

	return nil
}
