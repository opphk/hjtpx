-- CaptchaX Database Optimization - Index Enhancement
-- Version: 002_optimize_indexes
-- Description: Add composite indexes for common query patterns
-- Created: 2026-05-14

BEGIN;

-- 1. Composite indexes for captcha_logs table (hot data queries)

-- Composite index for type + created_at range queries (dashboard analytics)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_captcha_logs_type_created
ON captcha_logs (captcha_type, created_at DESC);

-- Composite index for client_id + created_at queries (user history lookup)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_captcha_logs_client_created
ON captcha_logs (client_id, created_at DESC);

-- Composite index for IP + created_at + result (IP-based analytics)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_captcha_logs_ip_created_result
ON captcha_logs (ip, created_at DESC, result);

-- Composite index for risk_score filtering with time bounds
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_captcha_logs_risk_created
ON captcha_logs (risk_score, created_at DESC)
WHERE risk_score >= 50;

-- Composite index for duration analysis with time bounds
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_captcha_logs_duration_created
ON captcha_logs (duration, created_at DESC);

-- 2. Partial indexes for active blacklist/whitelist lookups

-- Partial index for active blacklist entries (most common query)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_blacklist_active
ON blacklist (ip)
WHERE expire_at IS NULL OR expire_at > NOW();

-- Partial index for active whitelist entries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_whitelist_active
ON whitelist (ip, domain);

-- 3. Expression indexes for common patterns

-- Expression index for case-insensitive IP lookups (future-proofing IPv6)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_captcha_logs_ip_text_pattern
ON captcha_logs (text(ip));

-- Index for admin username lookup (authentication)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_admins_username_lower
ON admins (LOWER(username));

-- 4. Covering indexes to avoid table heap access

-- Covering index for List queries (includes frequently selected columns)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_captcha_logs_covering_list
ON captcha_logs (created_at DESC, id)
INCLUDE (captcha_type, client_id, ip, result, risk_score);

-- Covering index for stats aggregation queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_captcha_logs_covering_stats
ON captcha_logs (captcha_type, created_at)
INCLUDE (result, duration, risk_score);

-- 5. Analyze tables after index creation
ANALYZE captcha_logs;
ANALYZE blacklist;
ANALYZE whitelist;
ANALYZE admins;

-- Record migration
INSERT INTO schema_migrations (version) VALUES ('002_optimize_indexes')
ON CONFLICT (version) DO NOTHING;

COMMIT;
