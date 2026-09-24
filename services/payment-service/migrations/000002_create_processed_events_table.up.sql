-- ==========================================
-- 1. Create Processed Events Table (Inbox Pattern)
-- ==========================================
CREATE TABLE IF NOT EXISTS processed_events (
    -- شناسه یکتا برای هر رویداد
    event_id        UUID         PRIMARY KEY,

    -- نام یا نوع رویداد (مثلاً order.created, order.canceled)
    event_type      VARCHAR(100) NOT NULL,

    -- شناسه‌های مربوطه جهت Traceability و Debugging سریع
    order_id        UUID,
    payment_id      UUID,

    -- منبع صادرکننده رویداد (مثلاً order-service)
    aggregate_type  VARCHAR(50)  NOT NULL DEFAULT 'order',

    -- پیام خطا در صورت ناموفق بودن پردازش اولیه
    error_reason    TEXT,

    processed_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- ==========================================
-- 2. Indexes for Performance & Maintenance
-- ==========================================

-- برای پاک‌سازی دوره‌ای داده‌های قدیمی (Data Retention / Cleanup)
CREATE INDEX IF NOT EXISTS idx_processed_events_processed_at
ON processed_events (processed_at);

-- برای Trace کردن و دیباگ سریع تمام رویدادهای یک سفارش مشخص
CREATE INDEX IF NOT EXISTS idx_processed_events_order_id
ON processed_events (order_id)
WHERE order_id IS NOT NULL;

-- برای Trace کردن تمام رویدادهای مربوط به یک پرداخت مشخص
CREATE INDEX IF NOT EXISTS idx_processed_events_payment_id
ON processed_events (payment_id)
WHERE payment_id IS NOT NULL;

-- ==========================================
-- 3. Automatic Cleanup Function & Procedure
-- ==========================================

-- پروساژور پاک‌سازی رویدادهای قدیمی‌تر از ۳۰ روز (قابل اجرا توسط pg_cron یا CronJob)
CREATE OR REPLACE PROCEDURE purge_old_processed_events(retention_days INT DEFAULT 30)
LANGUAGE plpgsql
AS $$
DECLARE
    deleted_count INT;
BEGIN
    DELETE FROM processed_events
    WHERE processed_at < NOW() - (retention_days || ' days')::INTERVAL;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RAISE NOTICE 'Deleted % old processed events.', deleted_count;
END;
$$;
