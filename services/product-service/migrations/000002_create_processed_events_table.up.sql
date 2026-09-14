-- services/product-service/migrations/000002_create_processed_events_table.up.sql

-- نقش این جدول ذخیره اطلاعات مربوط به تایید هر سفارش برای دیباگ است
CREATE TABLE IF NOT EXISTS processed_events (
    event_id      UUID         PRIMARY KEY,
    event_type    VARCHAR(50)  NOT NULL,
    
    -- شناسه برای وابستگی به موجودیت‌ها
    product_id    UUID,
    order_id      UUID,

    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- ایندکس‌ برای کوئری‌های دیباگ و پاک‌سازی داده‌های قدیمی
CREATE INDEX 
IF NOT EXISTS idx_processed_events_product_id 
ON processed_events(product_id) 
WHERE product_id IS NOT NULL;


CREATE INDEX 
IF NOT EXISTS idx_processed_events_order_id 
ON processed_events(order_id) 
WHERE order_id IS NOT NULL;


CREATE INDEX 
IF NOT EXISTS idx_processed_events_processed_at 
ON processed_events(processed_at);