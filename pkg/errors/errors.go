// pkg/errors/errors.go

package errors

import (
	"errors"
	"fmt"
)

// Kind دسته‌ی کلی یک خطا را مشخص می‌کند، نه پیام دقیقش را.
//
// این دقیقاً همون چیزیه که لایه‌ی هندلر هر سرویس بهش نیاز داره تا
// یه خطای دامنه‌ای مثلاً کاربر پیدا نشد رو به یک کد استاندارد
// gRPC تبدیل کنه، بدون این‌که لازم باشه تک‌تک
// خطاهای هر سرویس رو از قبل بشناسه.
type Kind int

const (
	KindUnknown Kind = iota
	KindNotFound
	KindAlreadyExists
	KindInvalidInput
	KindUnauthenticated
	KindPermissionDenied
	KindInternal
)

// Error نوع خطای استاندارد این پروژه است: یک Kind قابل‌طبقه‌بندی
// ماشینی، به‌همراه یک پیام قابل‌خواندن برای انسان، و به‌صورت اختیاری
// یک خطای زیرین
// Kind: KindNotFound نوع خطا
// Message: "user not found" پیام خطا
// Underlying Error: pgx.ErrNoRows علت داخلی خطا
type Error struct {
	Kind Kind
	msg  string
	err  error
}

func (e *Error) Error() string {

	if e.err != nil {
		return fmt.Sprintf("%s: %v", e.msg, e.err)
	}

	return e.msg
}

// Unwrap اجازه می‌ده errors.Is / errors.As بتونن از این خطا رد بشن
// و به خطای اصلی زیرینش (مثلاً یه خطای خام pgx) برسن.
func (e *Error) Unwrap() error {
	return e.err
}

// Is باعث می‌شه errors.Is(err, someSentinel) درست کار کنه حتی وقتی
// err با Wrap پیچیده شده باشه؛ دو *Error وقتی برابرن که هم Kind و
// هم پیامشون یکی باشه.
func (e *Error) Is(target error) bool {

	t, ok := target.(*Error)
	if !ok {
		return false
	}

	return e.Kind == t.Kind && e.msg == t.msg
}

// New یک خطای دامنه‌ای جدید (بدون علت زیرین) می‌سازد. برای تعریف
// sentinel error هایی مثل «کاربر پیدا نشد» استفاده می‌شود.
func New(kind Kind, message string) error {
	return &Error{Kind: kind, msg: message}
}

// Wrap یک خطای زیرین (مثلاً خطای خام دیتابیس) را با یک Kind و پیام
// بیشتر می‌پیچد، بدون این‌که خطای اصلی را گم کند.
func Wrap(kind Kind, err error, message string) error {
	return &Error{Kind: kind, msg: message, err: err}
}

// GetKind زنجیره‌ی wrap شده‌ی یک خطا را باز می‌کند تا Kind مربوطه را
// پیدا کند. اگر خطا اصلاً از این پکیج نباشد (مثلاً یک خطای خام و
// دسته‌بندی‌نشده)، KindUnknown برمی‌گردد.
func GetKind(err error) Kind {

	var e *Error

	if errors.As(err, &e) {
		return e.Kind
	}

	return KindUnknown
}

// توضیح خلاصه هر قابلیت:

// New: یک خطای جدید میسازد
// errors.New(
//     KindNotFound,
//     "user not found",
// )

// Wrap: یک خطای موجود را با اطلاعات جدید بپیچ
// errors.Wrap(
//     KindInternal,
//     err,
//     "failed to query database",
// )

// GetKind: بفهم خطا از چه دسته‌ای است
// kind := errors.GetKind(err)

// errors.Is: ببین آیا یک خطای مشخص داخل زنجیره وجود دارد
// errors.Is(err, pgx.ErrNoRows)

// errors.As: آیا داخل زنجیره‌ی این خطا، خطایی از این نوع وجود دارد که بتوانم آن را بگیرم؟
// var e *errors.Error
// if errors.As(err, &e) {
//     fmt.Println(e.Kind)
// }

// طریقه ایمپورت:
// import (
// 	stdErrors "errors" ارور های استاندارد گولنگ
// 	appErrors "pkg/errors" ارور های دستی ما
// )
