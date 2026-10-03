-- ==========================================
-- 1. Drop Trigger & Function
-- ==========================================

DROP TRIGGER IF EXISTS trg_outbox_events_updated_at ON outbox_events;
DROP FUNCTION IF EXISTS outbox_events_set_updated_at();

-- ==========================================
-- 2. Drop Indexes
-- ==========================================

DROP INDEX IF EXISTS idx_outbox_events_published_at;
DROP INDEX IF EXISTS idx_outbox_events_aggregate;
DROP INDEX IF EXISTS idx_outbox_events_pending;
DROP INDEX IF EXISTS idx_payments_stale_lookup;

-- ==========================================
-- 3. Drop Table
-- ==========================================

DROP TABLE IF EXISTS outbox_events;