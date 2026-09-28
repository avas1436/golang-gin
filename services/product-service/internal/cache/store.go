// services/product-service/internal/cache/store.go

// مسیولیت این فایل این است که تمامی اعمال مورد نیاز برای لایه کیشینگ در سوریس
// محصولات توسط یک ماژول یکپارچه انجام بشه و اصلا دلیل وجود آن این است که
// هم consumer های rabbitmq و هم خود رپوزیتوری به دو کاربرد متفاوت از کش نیاز
// دارند و لازم است در هر دو لایه این اعمال هماهنگ باشد

package cache

import (
	"context"
	"strconv"
	"time"

	pkgcache "pkg/cache"
	"product-service/internal/model"

	"github.com/google/uuid"
)

// نام محیط جداگانه برای سرویس محصولات
const namespace = "product-service"

// ProductStore مسئول متمرکز مدیریت کلیدها، ذخیره، دریافت و ابطال کش محصولات است.
// این لایه منبع واحد حقیقت (Single Source of Truth) برای کش محصولات است.
type ProductCacheStore struct {
	products *pkgcache.Cache[model.Product]
	searches *pkgcache.Cache[[]*model.Product]

	productKB pkgcache.KeyBuilder
	searchKB  pkgcache.KeyBuilder
}

// ساخت کلید اختصاصی محصول
func (s *ProductCacheStore) productKey(id uuid.UUID) string {
	return s.productKB.Build(id.String())
}

// ساخت کلید اختصاصی جستجو
// از namespase جستجو شروع کرده و با قرار دادن تمامی پارامتر های
// دخیل در نتیجه یک جستجو کلید کش برای جستجو میسازد
func (s *ProductCacheStore) searchKey(
	query,
	category string,
	limit int,
	offset int,
) string {

	return s.searchKB.Build(
		category,
		query,
		strconv.Itoa(limit),
		strconv.Itoa(offset),
	)
}

// GetProduct دریافت محصول از کش
func (s *ProductCacheStore) GetProduct(
	ctx context.Context,
	id uuid.UUID,
) (
	*model.Product,
	bool,
	error,
) {

	// کلید در این قسمت ساخته میشود
	// از namespace مربوطه شروع میکند و تا اجزا پیش میرود
	key := s.productKey(id)

	p, ok, err := s.products.Get(ctx, key)

	if err != nil || !ok {
		return nil, false, err
	}

	return &p, true, nil
}

// SetProduct ذخیره محصول در کش
func (s *ProductCacheStore) SetProduct(
	ctx context.Context,
	p *model.Product,
	ttl time.Duration,
) error {

	if p == nil {
		return nil
	}

	key := s.productKey(p.ID)

	return s.products.Set(ctx, key, *p, ttl)
}

// InvalidateProduct ابطال کلید کش یک محصول
func (s *ProductCacheStore) InvalidateProduct(
	ctx context.Context,
	id uuid.UUID,
) error {

	key := s.productKey(id)

	return s.products.Delete(ctx, key)
}

// GetSearch دریافت نتیجه جستجو از کش
func (s *ProductCacheStore) GetSearch(
	ctx context.Context,
	query string,
	category string,
	limit int,
	offset int,
) (
	[]*model.Product,
	bool,
	error,
) {

	key := s.searchKey(query, category, limit, offset)

	return s.searches.Get(ctx, key)
}

// SetSearch ذخیره نتیجه جستجو در کش
func (s *ProductCacheStore) SetSearch(
	ctx context.Context,
	query string,
	category string,
	limit int,
	offset int,
	products []*model.Product,
	ttl time.Duration,
) error {

	key := s.searchKey(query, category, limit, offset)

	return s.searches.Set(ctx, key, products, ttl)
}
