-- تریگر تعیین زمان آپدیت
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS products (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    description     TEXT         NOT NULL DEFAULT '',
    category        VARCHAR(100) NOT NULL,
    price           BIGINT       NOT NULL CHECK (price >= 0),
    total_stock     INTEGER      NOT NULL DEFAULT 0 CHECK (total_stock >= 0),
    reserved_stock  INTEGER      NOT NULL DEFAULT 0 CHECK (reserved_stock >= 0),
    is_active       BOOLEAN      NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT chk_reserved_not_exceed_total CHECK (reserved_stock <= total_stock)
);


-- اندیس برای فیلتر دسته‌بندی و مرتب‌سازی قیمت
CREATE INDEX IF NOT EXISTS idx_products_category_price 
ON products (category, price) 
WHERE is_active;

-- اندیس برای جستجوی متنی در نام و توضیحات
CREATE INDEX IF NOT EXISTS idx_products_name_desc_trgm 
ON products 
USING gin ((name || ' ' || description) gin_trgm_ops);

-- اندیس برای محصولات فعال و موجود در انبار
CREATE INDEX IF NOT EXISTS idx_products_available 
ON products (category, price) 
WHERE is_active AND (total_stock - reserved_stock) > 0;

-- مرتب سازی براساس جدید ترین محصولات
CREATE INDEX IF NOT EXISTS idx_products_category_created 
ON products (category, created_at DESC) 
WHERE is_active;

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    -- تنها در صورتی که حداقل یکی از مقادیر تغییر کرده باشد updated_at بروز شود
    IF NEW IS DISTINCT FROM OLD THEN
        NEW.updated_at = now();
        NEW.created_at = OLD.created_at;    -- تثبیت مقدار created_at
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_products_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();