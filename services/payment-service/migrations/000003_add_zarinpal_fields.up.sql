-- services/payment-service/migrations/000002_add_zarinpal_fields.up.sql

ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS authority VARCHAR(255),
    ADD COLUMN IF NOT EXISTS redirect_url TEXT;

-- ایجاد ایندکس روی authority برای جستجوی سریع در زمان Callback بانک
CREATE INDEX IF NOT EXISTS idx_payments_authority ON payments(authority);