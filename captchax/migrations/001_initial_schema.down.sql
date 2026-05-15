-- Rollback 001_initial_schema
-- Drop all tables and objects created by the initial schema

BEGIN;

-- Remove migration record
DELETE FROM schema_migrations WHERE version = '001_initial_schema';

-- Drop triggers
DROP TRIGGER IF EXISTS update_captcha_config_updated_at ON captcha_config;

-- Drop functions
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP FUNCTION IF EXISTS cleanup_expired_blacklist();

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS admins CASCADE;
DROP TABLE IF EXISTS blacklist CASCADE;
DROP TABLE IF EXISTS whitelist CASCADE;
DROP TABLE IF EXISTS captcha_config CASCADE;
DROP TABLE IF EXISTS captcha_logs CASCADE;
DROP TABLE IF EXISTS schema_migrations CASCADE;

-- Drop extension if needed (optional, careful!)
-- DROP EXTENSION IF EXISTS "uuid-ossp";

COMMIT;
