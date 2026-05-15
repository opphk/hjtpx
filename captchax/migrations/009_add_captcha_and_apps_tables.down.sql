-- Rollback 009_add_captcha_and_apps_tables
-- Drop captcha and apps tables

BEGIN;

-- Remove migration record
DELETE FROM schema_migrations WHERE version = '009_add_captcha_and_apps_tables';

-- Drop triggers
DROP TRIGGER IF EXISTS update_captcha_updated_at ON captcha;
DROP TRIGGER IF EXISTS update_apps_updated_at ON apps;

-- Drop tables
DROP TABLE IF EXISTS captcha CASCADE;
DROP TABLE IF EXISTS apps CASCADE;

COMMIT;
