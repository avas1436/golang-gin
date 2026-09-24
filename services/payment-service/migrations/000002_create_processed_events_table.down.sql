-- services/payment-service/migrations/000002_create_processed_events_table.down.sql

-- ==========================================
-- 1. Drop Cleanup Procedure
-- ==========================================

DROP PROCEDURE IF EXISTS purge_old_processed_events(INT);


-- ==========================================
-- 2. Drop Indexes
-- ==========================================

DROP INDEX IF EXISTS idx_processed_events_payment_id;

DROP INDEX IF EXISTS idx_processed_events_order_id;

DROP INDEX IF EXISTS idx_processed_events_processed_at;


-- ==========================================
-- 3. Drop Processed Events Table
-- ==========================================

DROP TABLE IF EXISTS processed_events;
