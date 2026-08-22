DROP TABLE IF EXISTS refresh_tokens;

DROP TRIGGER IF EXISTS trg_users_updated_at ON users;

DROP FUNCTION IF EXISTS set_updated_at();

