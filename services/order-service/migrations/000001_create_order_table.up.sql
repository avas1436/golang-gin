-- services/order-service/migrations/000001_create_orders_table.up.sql

CREATE TABLE IF NOT EXISTS orders (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),

    -- دیتابیس سرویس کاربران جداست و اینجا تنها آیدی کاربر را داریم
    user_id       UUID         NOT NULL,

    status        VARCHAR(20)  NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending', 'confirmed', 'cancelled')),

    -- مجموع (unit_price * quantity) تمام آیتم‌ها، در لحظه‌ی ثبت
    -- سفارش محاسبه و اینجا snapshot می‌شود
    total_amount  BIGINT       NOT NULL DEFAULT 0 CHECK (total_amount >= 0),

    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- برای کویری سفارش‌های من» (پرتکرارترین کوئری این سرویس)
CREATE INDEX IF NOT EXISTS idx_orders_user_id
ON orders (user_id, created_at DESC);

-- برای داشبورد های ادمین / پیگیری سفارش‌های در حال پردازش
CREATE INDEX IF NOT EXISTS idx_orders_status
ON orders (status, created_at DESC);

-- ایندکس کلی برای مشاهده آخرین سفارش ها
CREATE INDEX idx_orders_created_at ON orders (created_at DESC);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW IS DISTINCT FROM OLD THEN
        NEW.updated_at = now();
        NEW.created_at = OLD.created_at; -- تثبیت مقدار created_at
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();