-- services/payment-service/migrations/000001_create_payments_table.down.sql

DROP TRIGGER IF EXISTS trg_payments_updated_at ON payments;
DROP FUNCTION IF EXISTS payments_set_updated_at();
DROP TABLE IF EXISTS payments;
