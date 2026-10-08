// pkg/config/outbox.go

package config

import "time"

// OutboxConfig تنظیمات مربوط به الگوی Transactional Outbox را
// نگه می‌دارد. هر سرویسی که از این الگو استفاده می‌کند باید
// این struct را در Config اصلی خود embed کند.
//
// مقادیر پیش‌فرض در NewRelayWorker و NewCleanupWorker اعمال
// می‌شوند — صفر بودن یک فیلد یعنی از پیش‌فرض استفاده شود
type OutboxConfig struct {
	// RelayInterval فاصله‌ی زمانی بین هر poll از جدول outbox.
	// پیش‌فرض: ۲ ثانیه
	RelayInterval time.Duration

	// RelayBatchSize حداکثر تعداد رویدادی که در هر tick پردازش
	// می‌شود. پیش‌فرض: ۵۰
	RelayBatchSize int

	// CleanupInterval فاصله‌ی زمانی بین هر بار پاکسازی.
	// پیش‌فرض: ۱ ساعت
	CleanupInterval time.Duration

	// RetentionDays تعداد روزهایی که رویدادهای published نگه
	// داشته می‌شوند. پیش‌فرض: ۷ روز
	RetentionDays int
}
