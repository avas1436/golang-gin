// services/product-service/internal/cache/module.go

package cache

import (
	pkgcache "pkg/cache"
	redispkg "pkg/redis"
	"product-service/internal/model"
	"product-service/internal/repository"
	"time"

	"go.uber.org/fx"
)

// این ماژول تنها جایی است که هر دو طرف دکوراتور رپوزیتوری خام و
// رپوزیتوری کش‌شده و همچنین استور را به هم وصل می‌کند:
var Module = fx.Module(
	"cache",
	fx.Provide(
		// ارائه ProductStore به کل سیستم (هم Consumerها و هم Repository)
		NewProductCacheStore,

		// ارائه Decorator رپوزیتوری کش شده
		fx.Annotate(
			NewCachedProductRepository,
			fx.ParamTags(
				`name:"rawProductRepository"`,
				``,
				`name:"productTTL"`,
				`name:"searchTTL"`,
			),
		),
	),
)

func NewProductCacheStore(client *redispkg.Client) *ProductCacheStore {
	return &ProductCacheStore{
		products:  pkgcache.New[model.Product](client),
		searches:  pkgcache.New[[]*model.Product](client),
		productKB: pkgcache.NewKeyBuilder(namespace + ":product"),
		searchKB:  pkgcache.NewKeyBuilder(namespace + ":search"),
	}
}

// NewCachedProductRepository یک ProductRepository برمی‌گرداند که
// همان رفتار repo را دارد، به‌علاوه‌ی کش خودکار برای GetByID و
// Search.
func NewCachedProductRepository(
	repo repository.ProductRepository,
	productCacheStore *ProductCacheStore,
	productTTL time.Duration,
	searchTTL time.Duration,
) repository.ProductRepository {

	return &cachedProductRepository{
		repo:              repo,
		productCacheStore: productCacheStore,
		productTTL:        productTTL,
		searchTTL:         searchTTL,
	}

}
