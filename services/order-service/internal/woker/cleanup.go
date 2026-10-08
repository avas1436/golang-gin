// services/order-service/internal/worker/cleanup.go

package worker

import (
	"context"
	"log"
	"time"

	appErrors "pkg/errors"

	"order-service/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

const cleanupWorkerName = "outbox-cleanup-worker"

// CleanupConfig پارامترهای قابل تنظیم Cleanup Worker را نگه می‌دارد
type CleanupConfig struct {
	// Interval فاصله‌ی زمانی بین هر بار پاکسازی
	Interval time.Duration

	// RetentionDays تعداد روزهایی که رویدادهای published نگه داشته
	// می‌شوند. رویدادهای قدیمی‌تر از این مقدار حذف می‌شوند.
	// مقدار پیش‌فرض ۷ روز است که با migration هماهنگ است
	RetentionDays int
}

// CleanupWorker رویدادهای published قدیمی را از جدول outbox_events
// پاکسازی می‌کند.
//
// این ورکر مستقیماً با OutboxRepository کار می‌کند — نیازی به
// واسطه‌ی لایه‌ی Service ندارد چون هیچ منطق تجاری‌ای اینجا
// وجود ندارد؛ فقط یک عملیات زیرساختی خالص است.
//
// با migration هماهنگ است: همان شرط‌هایی که در
// purge_old_outbox_events procedure تعریف شده را اعمال می‌کند
// (status = 'published' AND published_at < NOW() - interval)
type CleanupWorker struct {
	*periodicWorker
}

// NewCleanupWorker یک CleanupWorker می‌سازد و آن را با periodicWorker
// پایه متصل می‌کند
func NewCleanupWorker(
	pool *pgxpool.Pool,
	cfg CleanupConfig,
) *CleanupWorker {

	retentionDays := cfg.RetentionDays
	if retentionDays <= 0 {
		retentionDays = 7
	}

	interval := cfg.Interval
	if interval <= 0 {
		interval = 1 * time.Hour
	}

	c := &CleanupWorker{}

	c.periodicWorker = newPeriodicWorker(
		cleanupWorkerName,
		interval,
		func(ctx context.Context) error {
			return c.purge(ctx, pool, retentionDays)
		},
	)

	return c
}

// Start گوروتیین ورکر را اجرا می‌کند
func (w *CleanupWorker) Start() { w.start() }

// Stop ورکر را متوقف می‌کند و تا graceful shutdown منتظر می‌ماند
func (w *CleanupWorker) Stop(ctx context.Context) error { return w.stop(ctx) }

// purge رویدادهای published قدیمی‌تر از retentionDays را حذف می‌کند.
// این متد مستقیماً OutboxRepository را می‌سازد — نیازی به TxManager
// نیست چون این یک DELETE ساده بدون وابستگی به تراکنش چند‌مرحله‌ای است
func (w *CleanupWorker) purge(
	ctx context.Context,
	pool *pgxpool.Pool,
	retentionDays int,
) error {

	repo := repository.NewOutboxRepository(pool)

	deleted, err := repo.DeleteOldPublished(ctx, retentionDays)
	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"cleanup: failed to delete old outbox events",
		)
	}

	if deleted > 0 {
		log.Printf(
			"order-service: cleanup deleted %d published outbox event(s) older than %d day(s)",
			deleted,
			retentionDays,
		)
	}

	return nil
}
