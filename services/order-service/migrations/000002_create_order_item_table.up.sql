-- services/order-service/migrations/000002_create_order_items_table.up.sql

CREATE TABLE IF NOT EXISTS order_items (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),

    order_id      UUID        NOT NULL REFERENCES orders (id) ON DELETE CASCADE,

    -- بدون کلید به جدول محصولات چون سرویس محصولات دیتابیس مجزای
    -- خودش را دارد
    product_id    UUID        NOT NULL,

    -- name و price به‌صورت snapshot در لحظه‌ی ثبت سفارش کپی
    -- می‌شوند، نه این‌که هر بار از Product Service خوانده شوند؛
    -- چون اگر بعداً قیمت یا نام محصول در Product Service عوض شود،
    -- نباید تاریخچه‌ی یک سفارشِ قبلاً ثبت‌شده تغییر کند (سفارش باید
    -- همیشه دقیقاً همان چیزی را نشان بدهد که مشتری در لحظه‌ی خرید
    -- دیده و پرداخت کرده)
    product_name  VARCHAR(255) NOT NULL,
    unit_price    BIGINT       NOT NULL CHECK (unit_price >= 0),

    quantity      INTEGER      NOT NULL CHECK (quantity > 0),

    -- مجموع ارزش سفارش
    subtotal      BIGINT       GENERATED ALWAYS AS (unit_price * quantity) STORED,

    -- جلوگیری از آمدن دوباره یک محصول در سفارش
    CONSTRAINT uq_order_product UNIQUE (order_id, product_id)

    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now()

    -- عمداً updated_at ندارد: یک آیتم سفارش پس از ثبت هیچ‌وقت
    -- ویرایش نمی‌شود؛ برای تغییر تعداد باید سفارش لغو و سفارش
    -- جدیدی ثبت شود
);

-- پستگرس به‌صورت خودکار روی ستون‌های کلید خارجی ایندکس نمی‌سازد؛ بدون این
-- ایندکس، خواندن آیتم‌های یک سفارش
-- روی order_items اسکن کامل می‌خورد
CREATE INDEX IF NOT EXISTS idx_order_items_order_id
ON order_items (order_id);

-- «آیا این محصول در سفارش‌های قبلی استفاده شده؟» / گزارش‌گیری
CREATE INDEX IF NOT EXISTS idx_order_items_product_id
ON order_items (product_id);