-- حذف جدول محصولات (تریگرها و اندیس‌های متصل به جدول خودکار حذف می‌شوند)
DROP TABLE IF EXISTS products CASCADE;

-- حذف تابع تریگر
DROP FUNCTION IF EXISTS update_products_updated_at_column();