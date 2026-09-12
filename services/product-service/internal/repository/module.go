// services/product-service/internal/repository/module.go

package repository

import "go.uber.org/fx"

// نکته‌ی مهم: خروجی این Provider را با نام "rawProductRepository"
// علامت‌گذاری می‌کنیم، نه به‌صورت type معمولی.
//
// چرا؟ چون internal/cache هم یک ProductRepository دیگر (نسخه‌ی
// کش‌شده) تولید می‌کند. اگر هر دو بدون نام، از نوع یکسان
// repository.ProductRepository به container اضافه بشن، Fx موقع
// resolve کردن این اینترفیس نمی‌داند کدام را بدهد و با خطای
// "ambiguous" از کار می‌افتد.
//
// با این نام‌گذاری: این Provider فقط زمانی قابل‌دریافت است که
// صراحتاً با تگ name:"rawProductRepository" خواسته شود (دقیقاً
// همان کاری که cache.Module انجام می‌دهد). خروجیِ بدون نام و نهاییِ
// ProductRepository که service.Module مصرف می‌کند، از cache.Module
// می‌آید
var Module = fx.Module(
	"repository",
	fx.Provide(
		fx.Annotate(
			NewProductRepository,
			fx.ResultTags(`name:"rawProductRepository"`),
		),
	),
)
