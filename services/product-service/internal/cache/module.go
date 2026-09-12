// services/product-service/internal/cache/module.go

package cache

import (
	redispkg "pkg/redis"
	"product-service/config"
	"product-service/internal/repository"

	"go.uber.org/fx"
)

// این ماژول تنها جایی است که هر دو طرف دکوراتور رپوزیتوری خام و
// رپوزیتوری کش‌شده را به هم وصل می‌کند:
//
//   - پارامتر اول newCachedProductRepositoryFx با تگ
//     name:"rawProductRepository" علامت‌گذاری شده تا دقیقاً همان
//     خروجیِ نام‌دارِ repository.Module را بگیرد

//   - خروجی این تابع بدون تگ (یعنی repository.ProductRepository
//     معمولی) است؛ این یعنی از این نقطه به بعد، هرجای دیگر برنامه
//     (مثل service.Module) که repository.ProductRepository بدون
//     نام بخواهد، همین نسخه‌ی کش‌شده را دریافت می‌کند — بدون این‌که
//     service.go حتی یک خط عوض شود
var Module = fx.Module(
	"cache",
	fx.Provide(
		fx.Annotate(
			newCachedProductRepositoryFx,
			fx.ParamTags(`name:"rawProductRepository"`, ``, ``),
		),
	),
)

// newCachedProductRepositoryFx یک آداپتور مخصوص Fx است. علت وجودش
// این است که NewCachedProductRepository چهار پارامتر ساده می‌گیرد
// (repo, client, productTTL, searchTTL) و دو تای آخر از نوع
// time.Duration هستند — همان مشکل ابهامی که در security.go توضیح
// دادیم. با گرفتن *config.Config (یکتا در کل اپ) و استخراج مقادیر
// از داخل خودِ این آداپتور، تابع اصلی NewCachedProductRepository
// دست‌نخورده و مستقل از Fx باقی می‌ماند (می‌شود در تست هم بدون Fx
// صدایش زد)
func newCachedProductRepositoryFx(
	raw repository.ProductRepository,
	client *redispkg.Client,
	cfg *config.Config,
) repository.ProductRepository {

	return NewCachedProductRepository(
		raw,
		client,
		cfg.Cache.ProductTTL,
		cfg.Cache.SearchTTL,
	)
}
