-- services/order-service/migrations/000003_create_processed_events_table.up.sql

CREATE TABLE IF NOT EXISTS processed_events (

    -- شناسه یکتای پیام؛ بودنش idempotency را enforce می‌کند
    event_id      UUID         PRIMARY KEY,

    -- صرفاً برای دیباگ
    event_type    VARCHAR(100) NOT NULL,

    -- برای دیباگ: این ایونت مربوط به کدام سفارش/پرداخت بود؟
    order_id      UUID,
    payment_id    UUID,

    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- برای retention/cleanup دوره‌ای
CREATE INDEX IF NOT EXISTS idx_processed_events_processed_at
ON processed_events (processed_at);

-- برای دیباگ «چه event‌هایی روی این سفارش اثر گذاشتند؟»
CREATE INDEX IF NOT EXISTS idx_processed_events_order_id
ON processed_events (order_id)
WHERE order_id IS NOT NULL;