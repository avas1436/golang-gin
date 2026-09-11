// pkg/cache/key.go

package cache

import "strings"

// این شی یک نام برای کلید هر سرویس یا بخش های مختلف آن میسازد
type KeyBuilder struct {
	namespace string
}

// یک نام برای هر دسته دیتا میسازد. مثال:
// "product-service:search"
// "product-service:product"
func NewKeyBuilder(namespace string) KeyBuilder {

	return KeyBuilder{namespace: namespace}

}

// با namespace شروع میشود و تمامی اجزای کلید را به آن میچسباند
func (k KeyBuilder) Build(parts ...string) string {

	// ساخت یک آرایه با طول پیش فرض صفر ولی به اندازه اجزا ظرفیت دارد
	all := make([]string, 0, len(parts)+1)

	// افزودن namespace در ابتدا
	all = append(all, k.namespace)

	// افزودن اجزا به ترتیب
	all = append(all, parts...)

	// گرفتن خروجی نهایی با قرار دادن : بین اجزای آرایه
	return strings.Join(all, ":")
}

// تنها جز ابتدایی را مثلا namespace را میگیرد و کلید برای حذف دسته ای
// مقادیر میسازد
func (k KeyBuilder) Prefix(parts ...string) string {
	return k.Build(parts...) + ":"
}
