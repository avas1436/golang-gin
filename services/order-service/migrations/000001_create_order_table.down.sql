-- services/order-service/migrations/000001_create_orders_table.down.sql

DROP TRIGGER IF EXISTS update_orders_updated_at ON orders;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS orders;