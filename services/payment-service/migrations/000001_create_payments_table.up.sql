CREATE TABLE IF NOT EXISTS payments (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),

     -- شناسه سفارش در Order Service
    order_id        UUID         NOT NULL,

    -- شناسه کاربر در User Service
    user_id         UUID         NOT NULL,

    -- قیمت پرداخت
    amount          BIGINT       NOT NULL CHECK (amount >= 0),

    -- ارز پرداخت
    currency        CHAR(3) NOT NULL DEFAULT 'IRR' CHECK (currency ~ '^[A-Z]{3}$'),

    -- وضعیت پرداخت
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (
                        status IN (
                            'pending',
                            'completed',
                            'failed',
                            'canceled',
                            'refunded',
                            'expired'
                        )
                    ),

    gateway_name    VARCHAR(50),
    gateway_ref_id  VARCHAR(100),
    failure_reason  TEXT,

    metadata        JSONB        NOT NULL DEFAULT '{}'::jsonb,

    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- عدم اجازه به داشتن بیش از یک پرداخت موفق یا در جریان برای یک سفارش
-- فقط یک پرداخت pending/completed برای هر سفارش
CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_single_active_order
ON payments (order_id)
WHERE status IN ('pending', 'completed');

-- reference درگاه باید idempotent باشد
CREATE UNIQUE INDEX uq_payments_gateway_ref
ON payments (gateway_name, gateway_ref_id)
WHERE gateway_ref_id IS NOT NULL;

-- تاریخچه پرداخت‌های یک سفارش
CREATE INDEX idx_payments_order_id_created_at
ON payments (order_id, created_at DESC);

-- گزارش‌گیری کاربر
CREATE INDEX IF NOT EXISTS idx_payments_user_id
ON payments (user_id, created_at DESC);

-- گزارش‌گیری status
CREATE INDEX IF NOT EXISTS idx_payments_status
ON payments (status, created_at DESC);

-- آپدیت updated_at قبل از هر تغییر
CREATE OR REPLACE FUNCTION payments_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_payments_updated_at
BEFORE UPDATE ON payments
FOR EACH ROW
EXECUTE FUNCTION payments_set_updated_at();
