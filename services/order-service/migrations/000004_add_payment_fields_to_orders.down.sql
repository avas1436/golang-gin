ALTER TABLE orders
DROP COLUMN IF EXISTS payment_url,
DROP COLUMN IF EXISTS payment_authority;