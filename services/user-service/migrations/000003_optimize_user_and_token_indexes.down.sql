-- ۱. حذف Partial Index ایجاد شده
DROP INDEX IF EXISTS idx_refresh_tokens_active_tokens;

-- ۲. بازگرداندن ایندکس‌های جدول users
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_phone_number ON users (phone_number);