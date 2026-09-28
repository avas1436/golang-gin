-- ۱. حذف ایندکس‌های تکراری 
-- پوستگرس به صورت خودکار برای ستون های یونیک ایندکس میسازد
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_phone_number;

-- ۲. ایجاد Partial Index برای افزایش سرعت اعتبارسنجی توکن‌های فعال 
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_active_tokens 
ON refresh_tokens (token_hash) 
WHERE revoked = false;