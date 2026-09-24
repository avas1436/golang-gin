-- services/payment-service/migrations/000002_add_zarinpal_fields.down.sql

-- ==========================================
-- 1. Drop Columns
-- ==========================================

ALTER TABLE payments DROP COLUMN IF EXISTS authority;
ALTER TABLE payments DROP COLUMN IF EXISTS redirect_url;


-- ==========================================
-- 2. Drop Indexes
-- ==========================================

DROP INDEX IF EXISTS idx_payments_authority;


