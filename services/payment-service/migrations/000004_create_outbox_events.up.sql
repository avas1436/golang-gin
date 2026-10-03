-- ==========================================
-- 1. Create Outbox Events Table
-- ==========================================
CREATE TABLE IF NOT EXISTS outbox_events (
    -- شناسه یکتا برای هر رویداد
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),

    -- نوع رویداد (مثلاً payment.failed, payment.expired)
    event_type      VARCHAR(100) NOT NULL,

    -- نام Exchange در RabbitMQ که رویداد باید به آن منتشر شود
    exchange        VARCHAR(100) NOT NULL,

    -- Routing Key برای مسیریابی در Exchange
    routing_key     VARCHAR(100) NOT NULL,

    -- محتوای کامل رویداد به فرمت JSON
    payload         JSONB        NOT NULL,

    -- شناسه‌های مرتبط جهت Traceability
    aggregate_id    UUID,
    aggregate_type  VARCHAR(50)  NOT NULL DEFAULT 'payment',

    -- وضعیت پردازش: pending یعنی هنوز منتشر نشده
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

-- ایندکس اصلی برای Poller: پیدا کردن رویدادهای pending
-- به ترتیب قدیمی‌ترین
CREATE INDEX IF NOT EXISTS idx_outbox_events_pending
ON outbox_events (created_at ASC)
WHERE status = 'pending';

-- برای جستجوی رویدادهای یک aggregate مشخص (دیباگ)
CREATE INDEX IF NOT EXISTS idx_outbox_events_aggregate
ON outbox_events (aggregate_type, aggregate_id)
WHERE aggregate_id IS NOT NULL;

-- برای retention/cleanup دوره‌ای رویدادهای منتشرشده
CREATE INDEX IF NOT EXISTS idx_outbox_events_published_at
ON outbox_events (published_at)
WHERE published_at IS NOT NULL;

-- کوئری ExpireStalePayments که قرار است روی شرط WHERE status = 'PENDING' 
-- AND created_at < cutoff اجرا شود، با سرعت بالا (Index Scan) اجرا میشود
CREATE INDEX IF NOT EXISTS idx_payments_stale_lookup 
ON payments (updated_at ASC)
WHERE status = 'awaiting';

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