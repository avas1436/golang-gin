// services/order-service/internal/worker/relay.go

package worker

import (
	"context"
	"log"
	"time"

	appErrors "pkg/errors"
	"pkg/postgres"

	"order-service/internal/messaging"
	"order-service/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	relayWorkerName    = "outbox-relay-worker"
	relayMaxRetryCount = 5
)

// RelayConfig پارامترهای قابل تنظیم Relay Worker را نگه می‌دارد
type RelayConfig struct {
	// Interval فاصله‌ی زمانی بین هر بار poll کردن outbox
	Interval time.Duration

	// BatchSize حداکثر تعداد رویدادی که در هر tick پردازش می‌شود.
	// FOR UPDATE SKIP LOCKED تضمین می‌کند که چند نمونه‌ی هم‌زمان
	// از ورکر با هم تداخل نخواهند داشت
	BatchSize int
}

// RelayWorker رویدادهای pending جدول outbox_events را
// می‌خواند و به RabbitMQ منتشر می‌کند.
//
// هر tick داخل یک تراکنش اجرا می‌شود تا FOR UPDATE SKIP LOCKED
// فعال بماند. بعد از publish موفق، وضعیت رویداد به published
// تغییر می‌کند. شکست publish منجر به increment retry یا
// mark failed (بعد از رسیدن به سقف) می‌شود.
//
// این ورکر at-least-once delivery را تضمین می‌کند: اگر publish
// موفق باشد اما MarkPublished شکست بخورد، رویداد در tick بعدی
// دوباره ارسال می‌شود. مصرف‌کننده‌ها باید idempotent باشند
// (که با جدول processed_events تضمین شده)
type RelayWorker struct {
	*periodicWorker
}

// NewRelayWorker یک RelayWorker می‌سازد و آن را با periodicWorker
// پایه متصل می‌کند
func NewRelayWorker(
	pool *pgxpool.Pool,
	publisher messaging.RabbitMQEventPublisher,
	cfg RelayConfig,
) *RelayWorker {

	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 50
	}

	interval := cfg.Interval
	if interval <= 0 {
		interval = 2 * time.Second
	}

	r := &RelayWorker{}

	r.periodicWorker = newPeriodicWorker(
		relayWorkerName,
		interval,
		func(ctx context.Context) error {
			return r.processBatch(ctx, pool, publisher, batchSize)
		},
	)

	return r
}

// Start گوروتیین ورکر را اجرا می‌کند
func (w *RelayWorker) Start() { w.start() }

// Stop ورکر را متوقف می‌کند و تا graceful shutdown منتظر می‌ماند
func (w *RelayWorker) Stop(ctx context.Context) error { return w.stop(ctx) }

// processBatch یک batch از رویدادهای pending را داخل یک تراکنش
// واحد می‌خواند، publish می‌کند و وضعیت را به‌روز می‌کند
func (w *RelayWorker) processBatch(
	ctx context.Context,
	pool *pgxpool.Pool,
	publisher messaging.RabbitMQEventPublisher,
	batchSize int,
) error {

	return postgres.WithTx(
		ctx,
		pool,
		func(tx pgx.Tx) error {

			repo := repository.NewOutboxRepository(tx)

			events, err := repo.FetchPendingForRelay(ctx, batchSize)
			if err != nil {
				return appErrors.Wrap(
					appErrors.KindInternal,
					err,
					"relay: failed to fetch pending outbox events",
				)
			}

			if len(events) == 0 {
				return nil
			}

			log.Printf(
				"order-service: relay processing %d pending event(s)",
				len(events),
			)

			for _, event := range events {

				publishErr := publisher.PublishRaw(
					ctx,
					event.RoutingKey,
					event.Payload,
				)

				if publishErr != nil {
					w.handlePublishFailure(
						ctx,
						repo,
						event.ID,
						event.RetryCount,
						publishErr,
					)
					// یک رویداد شکست‌خورده نباید بقیه را متوقف کند
					continue
				}

				if markErr := repo.MarkPublished(ctx, event.ID); markErr != nil {
					// بحرانی نیست — رویداد publish شده ولی status
					// هنوز pending است. در tick بعدی دوباره publish
					// می‌شود (at-least-once)
					log.Printf(
						"order-service: relay failed to mark event %s as published: %v",
						event.ID,
						markErr,
					)
				}
			}

			return nil
		},
	)
}

// handlePublishFailure بر اساس تعداد تلاش‌های قبلی تصمیم می‌گیرد
// رویداد را به failed ببرد یا retry count را زیاد کند
func (w *RelayWorker) handlePublishFailure(
	ctx context.Context,
	repo repository.OutboxRepository,
	eventID uuid.UUID,
	retryCount int,
	publishErr error,
) {
	if retryCount+1 >= relayMaxRetryCount {

		if markErr := repo.MarkFailed(
			ctx,
			eventID,
			publishErr.Error(),
		); markErr != nil {
			log.Printf(
				"order-service: relay failed to mark event %s as failed: %v",
				eventID,
				markErr,
			)
		}

		log.Printf(
			"order-service: relay event %s permanently failed after %d retries: %v",
			eventID,
			relayMaxRetryCount,
			publishErr,
		)

	} else {

		if retryErr := repo.IncrementRetry(
			ctx,
			eventID,
			publishErr.Error(),
		); retryErr != nil {
			log.Printf(
				"order-service: relay failed to increment retry for event %s: %v",
				eventID,
				retryErr,
			)
		}

		log.Printf(
			"order-service: relay event %s publish failed (attempt %d/%d): %v",
			eventID,
			retryCount+1,
			relayMaxRetryCount,
			publishErr,
		)
	}
}
