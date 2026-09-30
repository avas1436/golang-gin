// pkg/events/topology.go

// --------------------------------------------------------------------------
//  راهنمای توپولوژی :
// --------------------------------------------------------------------------
//  ۱. Exchange (تبادل‌کننده):
//     نقطه ورود پیام‌ها به RabbitMQ است. سرویس ناشر پیام را مستقیماً
//     به صف نمی‌فرستد، بلکه به Exchange ارسال می‌کند.
//     Exchange بر اساس Routing Key،
//     پیام را به یک یا چند صف متصل شده هدایت می‌کند.
//
//  ۲. Routing Key (کلید مسیریابی):
//     برچسب یا آدرسی است که همراه پیام فرستاده می‌شود و مشخص‌کننده
//     «نوع رویداد»
//     رخ‌داده است. Exchange از روی این کلید تصمیم می‌گیرد
//     پیام را به کدام صف بفرستد.
//
//  ۳. Queue (صف):
//     محل ذخیره‌سازی پیام‌هاست. هر میکروسرویس شنونده (Consumer)
//     صف اختصاصی خود را
//     به Exchange وصل (Bind) می‌کند تا پیام‌ها را دریافت و پردازش کند.
// --------------------------------------------------------------------------

package events

// Exchanges
const (
	// ExchangeOrderEvents: نقطه انتشار تمام رویدادهایی که منشا آن‌ها order-service است
	// (مانند ثبت سفارش، درخواست قطعی کردن یا آزادسازی موجودی).
	ExchangeOrderEvents = "order_events"

	// ExchangePaymentEvents: نقطه انتشار تمامی رویدادهای مالی و پرداخت
	// (مانند پرداخت موفق یا ناموفق) توسط payment-service.
	ExchangePaymentEvents = "payment_events"

	// ExchangeUserEvents: نقطه انتشار رویدادهای مربوط به مدیریت کاربران
	// (مانند درخواست کد OTP یا ثبت‌نام).
	ExchangeUserEvents = "user_events"

	// ExchangeProductEvents: نقطه انتشار تغییرات مستقیم در کاتالوگ محصولات
	// توسط product-service.
	ExchangeProductEvents = "product_events"
)

// Routing Keys
// در Publish برای تعیین موضوع پیام و در BindQueue برای فیلتر پیام‌ها.
const (

	// --- Order Events ---

	// RoutingKeyOrderCreated: اعلام ساخت یک سفارش جدید در دیتابیس سفارشات.
	RoutingKeyOrderCreated = "order.created"

	// RoutingKeyStockReleaseRequested: دستور آزادسازی موجودی رزروشده .
	RoutingKeyStockReleaseRequested = "stock.release.requested"

	// RoutingKeyStockConfirmRequested: دستور قطعی کردن کسر موجودی رزروشده .
	RoutingKeyStockConfirmRequested = "stock.confirm.requested"

	// --- Payment Events ---

	// RoutingKeyPaymentCompleted: اعلام موفقیت تراکنش پرداخت در درگاه.
	RoutingKeyPaymentCompleted = "payment.completed"

	// RoutingKeyPaymentFailed: اعلام شکست یا انصراف از تراکنش پرداخت.
	RoutingKeyPaymentFailed = "payment.failed"

	// --- User Events ---

	// RoutingKeyUserOTPRequested: اعلام درخواست ارسال کد تایید ورود/ثبت‌نام.
	RoutingKeyUserOTPRequested = "user.otp.requested"
)

// Queue Names
// در consumer.BindQueue و consumer.Consume جهت دریافت پیام‌ها.
// الگوی نام‌گذاری: {service_name}.{event_type}.queue
const (

	// --- Order Service Queues ---

	// QueueOrderPaymentCompleted: دریافت نتیجه پرداخت موفق جهت تایید سفارش
	QueueOrderPaymentCompleted = "order.payment_completed.queue"

	// QueueOrderPaymentFailed: دریافت نتیجه پرداخت ناموفق جهت لغو سفارش
	QueueOrderPaymentFailed = "order.payment_failed.queue"

	// --- Product Service Queues ---

	// QueueProductStockRelease: دریافت دستور آزادسازی موجودی رزروشده.
	QueueProductStockRelease = "product.stock_release.queue"

	// QueueProductStockConfirm: دریافت دستور قطعی کردن کسر موجودی.
	QueueProductStockConfirm = "product.stock_confirm.queue"

	// QueueProductOrderCreated: دریافت خبر ثبت سفارش جهت رزرو اولیه موجودی.
	QueueProductOrderCreated = "product.order_created.queue"

	// --- Notification Service Queues ---

	// QueueNotificationUserOTP: دریافت رویداد ارسال پیامک به کاربر
	QueueNotificationUserOTP = "notification.user_otp.queue"
)
