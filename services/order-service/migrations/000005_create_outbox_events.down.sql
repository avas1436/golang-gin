-- services/order-service/migrations/000005_create_outbox_events.down.sql

-- ==========================================
-- 1. Drop Cleanup Procedure
-- ==========================================

DROP PROCEDURE IF EXISTS purge_old_outbox_events(INT);

-- ==========================================
-- 2. Drop Trigger & Function
-- ==========================================

DROP TRIGGER IF EXISTS trg_outbox_events_updated_at ON outbox_events;
DROP FUNCTION IF EXISTS outbox_events_set_updated_at();

-- ==========================================
-- 3. Drop Indexes
-- ==========================================

DROP INDEX IF EXISTS idx_outbox_events_published_at;
DROP INDEX IF EXISTS idx_outbox_events_aggregate;
DROP INDEX IF EXISTS idx_outbox_events_pending;

-- ==========================================
-- 4. Drop Table
-- ==========================================

DROP TABLE IF EXISTS outbox_events;