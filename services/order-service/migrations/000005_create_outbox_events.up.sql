-- services/order-service/migrations/000005_create_outbox_events.up.sql

-- ==========================================
-- 1. Create Outbox Events Table
-- ==========================================
CREATE TABLE IF NOT EXISTS outbox_events (
    -- شناسه یکتا برای هر رویداد
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),

    -- نوع رویداد (مثلاً order.created, stock.release.requested)
    event_type      VARCHAR(100) NOT NULL,

    -- نام Exchange در RabbitMQ که رویداد باید به آن منتشر شود
    exchange        VARCHAR(100) NOT NULL,

    -- Routing Key برای مسیریابی در Exchange
    routing_key     VARCHAR(100) NOT NULL,

    -- محتوای کامل رویداد به فرمت JSON
    payload         JSONB        NOT NULL,

    -- شناسه‌های مرتبط جهت Traceability
    -- aggregate_id معمولاً order_id است
    aggregate_id    UUID,
    aggregate_type  VARCHAR(50)  NOT NULL DEFAULT 'order',

    -- وضعیت پردازش: pending یعنی هنوز به RabbitMQ منتشر نشده
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'published', 'failed')),

    -- تعداد تلاش‌های ناموفق برای انتشار
    retry_count     INT          NOT NULL DEFAULT 0,

    -- آخرین خطای رخ‌داده در زمان انتشار
    last_error      TEXT,

    -- زمان‌بندی
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    published_at    TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- ==========================================
-- 2. Indexes for Performance
-- ==========================================

-- ایندکس اصلی برای Relay Worker: پیدا کردن رویدادهای pending
-- به ترتیب قدیمی‌ترین. از Partial Index استفاده می‌کنیم چون
-- Relay فقط به ردیف‌های pending اهمیت می‌دهد و این ایندکس
-- با بزرگ شدن جدول کوچک می‌ماند
CREATE INDEX IF NOT EXISTS idx_outbox_events_pending
ON outbox_events (created_at ASC)
WHERE status = 'pending';

-- برای جستجوی رویدادهای یک aggregate مشخص
CREATE INDEX IF NOT EXISTS idx_outbox_events_aggregate
ON outbox_events (aggregate_type, aggregate_id)
WHERE aggregate_id IS NOT NULL;

-- برای retention/cleanup دوره‌ای رویدادهای منتشرشده
CREATE INDEX IF NOT EXISTS idx_outbox_events_published_at
ON outbox_events (published_at)
WHERE published_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_outbox_events_pending_claim
ON outbox_events (created_at, id)
WHERE status = 'pending';

-- ==========================================
-- 3. Auto-update updated_at Trigger
-- ==========================================

CREATE OR REPLACE FUNCTION outbox_events_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_outbox_events_updated_at
BEFORE UPDATE ON outbox_events
FOR EACH ROW
EXECUTE FUNCTION outbox_events_set_updated_at();

-- ==========================================
-- 4. Cleanup Procedure (Retention)
-- ==========================================
-- همان الگوی purge_old_processed_events در payment-service.
-- این procedure را می‌توان با pg_cron یا یک CronJob جداگانه
-- به‌صورت دوره‌ای اجرا کرد
CREATE OR REPLACE PROCEDURE purge_old_outbox_events(retention_days INT DEFAULT 7)
LANGUAGE plpgsql
AS $$
DECLARE
    deleted_count INT;
BEGIN
    DELETE FROM outbox_events
    WHERE status IN ('published', 'failed')
      AND published_at < NOW() - (retention_days || ' days')::INTERVAL;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RAISE NOTICE 'Deleted % processed outbox events.', deleted_count;
END;
$$;