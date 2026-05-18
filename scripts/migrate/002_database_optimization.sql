-- 数据库优化脚本 v2.0
-- 包含：复合索引、部分索引、慢查询分析表、优化建议视图
-- 适用于 PostgreSQL 12+
-- 创建时间: 2026-05-18

BEGIN;

-- ============================================================
-- 1. 复合索引优化
-- ============================================================

-- 1.1 用户表复合索引
CREATE INDEX IF NOT EXISTS idx_users_status_created ON users(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_users_status_email ON users(status, email);
CREATE INDEX IF NOT EXISTS idx_users_username_status ON users(username, status);

-- 1.2 应用表复合索引（高频查询场景）
CREATE INDEX IF NOT EXISTS idx_applications_user_active ON applications(user_id, is_active, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_applications_user_domain ON applications(user_id, domain);
CREATE INDEX IF NOT EXISTS idx_applications_api_key_active ON applications(api_key, is_active);

-- 1.3 验证码验证表复合索引（统计和查询优化）
CREATE INDEX IF NOT EXISTS idx_verification_app_created_status ON verifications(application_id, created_at DESC, status);
CREATE INDEX IF NOT EXISTS idx_verification_user_created ON verifications(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_verification_ip_created ON verifications(ip_address, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_verification_status_risk ON verifications(status, risk_score DESC);
CREATE INDEX IF NOT EXISTS idx_verification_app_status ON verifications(application_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_verification_type_status ON verifications(captcha_type, status);

-- 1.4 验证日志表复合索引
CREATE INDEX IF NOT EXISTS idx_verification_logs_app_created ON verification_logs(application_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_verification_logs_status_created ON verification_logs(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_verification_logs_session_created ON verification_logs(session_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_verification_logs_ip_created ON verification_logs(ip_address, created_at DESC);

-- 1.5 黑名单表复合索引
CREATE INDEX IF NOT EXISTS idx_blacklist_type_status ON blacklist(type, status);
CREATE INDEX IF NOT EXISTS idx_blacklist_status_expires ON blacklist(status, expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_blacklist_target_type ON blacklist(target, type);

-- 1.6 设备指纹表复合索引
CREATE INDEX IF NOT EXISTS idx_device_ip_last_seen ON device_fingerprints(ip_address, last_seen DESC);
CREATE INDEX IF NOT EXISTS idx_device_fingerprint_bot ON device_fingerprints(fingerprint, is_bot);
CREATE INDEX IF NOT EXISTS idx_device_risk_last_seen ON device_fingerprints(risk_level, last_seen DESC);

-- 1.7 API Key 历史表复合索引
CREATE INDEX IF NOT EXISTS idx_api_key_history_app_changed ON api_key_histories(application_id, changed_at DESC);

-- 1.8 管理员登录日志复合索引
CREATE INDEX IF NOT EXISTS idx_admin_login_admin_created ON admin_login_logs(admin_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_login_ip_created ON admin_login_logs(ip_address, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_login_status_created ON admin_login_logs(status, created_at DESC);

-- 1.9 AB测试表复合索引
CREATE INDEX IF NOT EXISTS idx_abtest_app_status ON ab_tests(application_id, status);
CREATE INDEX IF NOT EXISTS idx_abtest_variant_abtest ON ab_test_variants(ab_test_id, id);

-- 1.10 告警记录表复合索引
CREATE INDEX IF NOT EXISTS idx_alert_record_rule_created ON alert_records(rule_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alert_record_status_created ON alert_records(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alert_record_severity_status ON alert_records(severity, status);

-- 1.11 轨迹记录表复合索引
CREATE INDEX IF NOT EXISTS idx_trace_verification_session ON trace_records(verification_id, session_id);
CREATE INDEX IF NOT EXISTS idx_trace_app_created ON trace_records(application_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_trace_ip_created ON trace_records(ip_address, created_at DESC);

-- 1.12 Seamless验证表复合索引
CREATE INDEX IF NOT EXISTS idx_seamless_app_decision ON seamless_verifications(application_id, decision, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_seamless_user_created ON seamless_verifications(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_seamless_fingerprint_decision ON seamless_verifications(device_fingerprint, decision);

-- ============================================================
-- 2. 部分索引（Partial Index）优化
-- ============================================================

-- 2.1 活跃验证码会话（只索引 pending 状态）
CREATE INDEX IF NOT EXISTS idx_captcha_sessions_pending ON captcha_sessions(created_at DESC) WHERE status = 'pending';

-- 2.2 活跃黑名单（只索引启用且未过期的）
CREATE INDEX IF NOT EXISTS idx_blacklist_active_target ON blacklist(target) WHERE is_active = TRUE AND (expires_at IS NULL OR expires_at > NOW());

-- 2.3 风险日志 - 高风险记录
CREATE INDEX IF NOT EXISTS idx_risk_logs_high_risk ON risk_logs(created_at DESC) WHERE risk_level IN ('high', 'critical');

-- 2.4 待处理告警记录
CREATE INDEX IF NOT EXISTS idx_alert_records_pending ON alert_records(created_at DESC) WHERE status = 'triggered';

-- 2.5 已完成的验证记录（归档优化）
CREATE INDEX IF NOT EXISTS idx_verifications_completed ON verifications(application_id, created_at DESC) WHERE status = 'success';

-- 2.6 活跃应用的用户查询
CREATE INDEX IF NOT EXISTS idx_applications_active_user ON applications(user_id) WHERE is_active = TRUE;

-- 2.7 未验证用户
CREATE INDEX IF NOT EXISTS idx_users_unverified ON users(created_at DESC) WHERE is_verified = FALSE;

-- 2.8 最近登录（活跃用户识别）
CREATE INDEX IF NOT EXISTS idx_users_recent_login ON users(last_login_at DESC) WHERE last_login_at IS NOT NULL;

-- 2.9 信任设备（活跃的）
CREATE INDEX IF NOT EXISTS idx_trusted_devices_active ON trusted_devices(user_id, last_used_at DESC) WHERE is_trusted = TRUE;

-- 2.10 启用状态的A/B测试
CREATE INDEX IF NOT EXISTS idx_ab_tests_active ON ab_tests(application_id) WHERE status = 'active';

-- ============================================================
-- 3. 表达式索引（Expression Index）优化
-- ============================================================

-- 3.1 IP地址标准化索引（用于IPv4/IPv6统一查询）
CREATE INDEX IF NOT EXISTS idx_verifications_ip_normalized ON verifications(md5(ip_address::text));
CREATE INDEX IF NOT EXISTS idx_blacklist_ip_normalized ON blacklist(md5(blacklisted_value)) WHERE blacklist_type = 'ip';

-- 3.2 日期截断索引（用于日统计）
CREATE INDEX IF NOT EXISTS idx_verification_logs_created_date ON verification_logs(DATE(created_at));
CREATE INDEX IF NOT EXISTS idx_risk_logs_created_date ON risk_logs(DATE(created_at));

-- 3.3 大小写不敏感的用户名查询
CREATE INDEX IF NOT EXISTS idx_users_username_lower ON users(LOWER(username));

-- ============================================================
-- 4. 覆盖索引（Covering Index）优化
-- ============================================================

-- 4.1 验证查询覆盖索引（常见查询无需回表）
CREATE INDEX IF NOT EXISTS idx_verification_cover_app_status 
    ON verifications(application_id, created_at DESC) 
    INCLUDE (status, captcha_type, risk_score);

-- 4.2 黑名单查询覆盖索引
CREATE INDEX IF NOT EXISTS idx_blacklist_cover_type_status 
    ON blacklist(type, status, target) 
    INCLUDE (reason, severity, expires_at);

-- ============================================================
-- 5. 慢查询分析表
-- ============================================================

CREATE TABLE IF NOT EXISTS slow_query_log (
    id SERIAL PRIMARY KEY,
    query_hash VARCHAR(32) NOT NULL,
    query_text TEXT NOT NULL,
    calls BIGINT NOT NULL DEFAULT 1,
    total_time INTERVAL NOT NULL DEFAULT INTERVAL '0 seconds',
    min_time INTERVAL NOT NULL DEFAULT INTERVAL '0 seconds',
    max_time INTERVAL NOT NULL DEFAULT INTERVAL '0 seconds',
    mean_time INTERVAL NOT NULL DEFAULT INTERVAL '0 seconds',
    stddev_time INTERVAL,
    rows BIGINT NOT NULL DEFAULT 0,
    shared_blks_hit BIGINT NOT NULL DEFAULT 0,
    shared_blks_read BIGINT NOT NULL DEFAULT 0,
    shared_blks_dirtied BIGINT NOT NULL DEFAULT 0,
    shared_blks_written BIGINT NOT NULL DEFAULT 0,
    local_blks_hit BIGINT NOT NULL DEFAULT 0,
    local_blks_read BIGINT NOT NULL DEFAULT 0,
    local_blks_dirtied BIGINT NOT NULL DEFAULT 0,
    local_blks_written BIGINT NOT NULL DEFAULT 0,
    temp_blks_read BIGINT NOT NULL DEFAULT 0,
    temp_blks_written BIGINT NOT NULL DEFAULT 0,
    first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    suggested_action TEXT,
    estimated_savings TEXT
);

CREATE INDEX IF NOT EXISTS idx_slow_query_hash ON slow_query_log(query_hash);
CREATE INDEX IF NOT EXISTS idx_slow_query_last_seen ON slow_query_log(last_seen DESC);
CREATE INDEX IF NOT EXISTS idx_slow_query_mean_time ON slow_query_log(mean_time DESC);

-- ============================================================
-- 6. 索引使用统计表
-- ============================================================

CREATE TABLE IF NOT EXISTS index_usage_stats (
    id SERIAL PRIMARY KEY,
    schemaname VARCHAR(64),
    tablename VARCHAR(64),
    indexname VARCHAR(64),
    idx_scan BIGINT DEFAULT 0,
    idx_tup_read BIGINT DEFAULT 0,
    idx_tup_fetch BIGINT DEFAULT 0,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(schemaname, tablename, indexname)
);

CREATE INDEX IF NOT EXISTS idx_index_usage_scan ON index_usage_stats(idx_scan DESC);

-- ============================================================
-- 7. 优化建议生成函数
-- ============================================================

CREATE OR REPLACE FUNCTION generate_index_recommendations()
RETURNS TABLE (
    table_name TEXT,
    index_name TEXT,
    seq_scan_pct NUMERIC,
    estimated_savings TEXT,
    recommendation TEXT
) AS $$
BEGIN
    RETURN QUERY
    WITH index_stats AS (
        SELECT 
            schemaname,
            relname AS tablename,
            indexrelname AS indexname,
            idx_scan,
            seq_scan,
            pg_size_pretty(pg_relation_size(indexrelid)) AS index_size
        FROM pg_stat_user_indexes
        WHERE schemaname = 'public'
    ),
    unused_indexes AS (
        SELECT 
            schemaname,
            relname AS tablename,
            indexrelname AS indexname,
            pg_size_pretty(pg_relation_size(indexrelid)) AS index_size
        FROM pg_stat_user_indexes
        WHERE idx_scan = 0 
          AND schemaname = 'public'
          AND indexrelname NOT LIKE '%pkey%'
          AND indexrelname NOT LIKE '%unique%'
    )
    SELECT
        ui.tablename::TEXT,
        ui.indexname::TEXT,
        0.0::NUMERIC AS seq_scan_pct,
        ('Size: ' || ui.index_size)::TEXT AS estimated_savings,
        '建议删除：从未使用的索引'::TEXT AS recommendation
    FROM unused_indexes ui;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- 8. 慢查询分析函数
-- ============================================================

CREATE OR REPLACE FUNCTION analyze_slow_queries(threshold_ms INTEGER DEFAULT 100)
RETURNS TABLE (
    query_text TEXT,
    calls BIGINT,
    total_time_ms BIGINT,
    mean_time_ms NUMERIC,
    rows BIGINT,
    suggestion TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        pg_stat_statements.query,
        pg_stat_statements.calls,
        (pg_stat_statements.total_time / 1000)::BIGINT,
        (pg_stat_statements.mean_time)::NUMERIC,
        pg_stat_statements.rows,
        CASE 
            WHEN pg_stat_statements.mean_time > threshold_ms THEN 
                '考虑添加索引或优化查询'
            ELSE
                '性能可接受'
        END AS suggestion
    FROM pg_stat_statements
    WHERE pg_stat_statements.mean_time > threshold_ms
    ORDER BY pg_stat_statements.total_time DESC
    LIMIT 20;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- 9. 统计信息收集触发器
-- ============================================================

CREATE OR REPLACE FUNCTION update_index_stats()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO index_usage_stats 
        (schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch, last_updated)
    VALUES (
        NEW.schemaname,
        NEW.relname,
        NEW.indexrelname,
        NEW.idx_scan,
        NEW.idx_tup_read,
        NEW.idx_tup_fetch,
        CURRENT_TIMESTAMP
    )
    ON CONFLICT (schemaname, tablename, indexname)
    DO UPDATE SET
        idx_scan = NEW.idx_scan,
        idx_tup_read = NEW.idx_tup_read,
        idx_tup_fetch = NEW.idx_tup_fetch,
        last_updated = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- 10. 常用统计查询视图
-- ============================================================

-- 表大小和行数统计视图
CREATE OR REPLACE VIEW v_table_statistics AS
SELECT 
    schemaname,
    relname AS table_name,
    n_tup_ins AS inserts,
    n_tup_upd AS updates,
    n_tup_del AS deletes,
    n_live_tup AS live_rows,
    n_dead_tup AS dead_rows,
    pg_size_pretty(pg_total_relation_size(relid)) AS total_size,
    pg_size_pretty(pg_relation_size(relid)) AS table_size,
    pg_size_pretty(pg_indexes_size(relid)) AS index_size,
    last_vacuum,
    last_autovacuum,
    last_analyze,
    last_autoanalyze
FROM pg_stat_user_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(relid) DESC;

-- 索引效率统计视图
CREATE OR REPLACE VIEW v_index_efficiency AS
SELECT 
    schemaname,
    relname AS table_name,
    indexrelname AS index_name,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch,
    pg_size_pretty(pg_relation_size(indexrelid)) AS index_size,
    CASE 
        WHEN idx_scan = 0 THEN 'unused'
        WHEN idx_tup_fetch > 0 THEN 'efficient'
        ELSE 'inefficient'
    END AS efficiency_status
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
  AND indexrelname NOT LIKE '%pkey%'
ORDER BY idx_scan ASC;

-- 高频查询视图
CREATE OR REPLACE VIEW v_frequent_queries AS
SELECT 
    query,
    calls,
    total_time,
    mean_time,
    rows,
    shared_blks_hit,
    shared_blks_read
FROM pg_stat_statements
ORDER BY calls DESC
LIMIT 50;

-- ============================================================
-- 11. 自动清理函数
-- ============================================================

CREATE OR REPLACE FUNCTION cleanup_old_slow_query_logs(days_to_keep INTEGER DEFAULT 30)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM slow_query_log 
    WHERE last_seen < CURRENT_TIMESTAMP - (days_to_keep || ' days')::INTERVAL;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- 12. 索引重建函数（用于碎片整理）
-- ============================================================

CREATE OR REPLACE FUNCTION reindex_fragmented_indexes(threshold_pct INTEGER DEFAULT 30)
RETURNS TABLE (
    index_name TEXT,
    fragmentation_pct NUMERIC,
    action_taken TEXT
) AS $$
DECLARE
    rec RECORD;
BEGIN
    FOR rec IN
        SELECT 
            indexrelname,
            idx_scan,
            pg_relation_size(indexrelid) AS index_size
        FROM pg_stat_user_indexes
        WHERE schemaname = 'public'
          AND idx_scan > 0
        ORDER BY idx_scan ASC
    LOOP
        IF rec.index_size > 1024 * 1024 THEN
            BEGIN
                EXECUTE 'REINDEX INDEX ' || rec.indexrelname;
                index_name := rec.indexrelname::TEXT;
                fragmentation_pct := 0;
                action_taken := 'Reindexed';
                RETURN NEXT;
            EXCEPTION WHEN OTHERS THEN
                index_name := rec.indexrelname::TEXT;
                fragmentation_pct := 0;
                action_taken := 'Failed: ' || SQLERRM;
                RETURN NEXT;
            END;
        END IF;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

COMMIT;

-- ============================================================
-- 13. 性能验证查询
-- ============================================================

-- 验证索引是否创建成功
SELECT 
    tablename,
    indexname,
    pg_size_pretty(pg_relation_size(indexrelid)) AS index_size
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
  AND indexrelname LIKE 'idx_%'
ORDER BY pg_relation_size(indexrelid) DESC;

-- 分析表统计信息
ANALYZE;

-- 输出创建结果
DO $$
DECLARE
    idx_count INTEGER;
    partial_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO idx_count 
    FROM pg_indexes 
    WHERE schemaname = 'public' 
      AND indexname LIKE 'idx_%';
    
    SELECT COUNT(*) INTO partial_count 
    FROM pg_indexes 
    WHERE schemaname = 'public' 
      AND indexname LIKE 'idx_%'
      AND indexdef LIKE '%WHERE%';
    
    RAISE NOTICE '=== 数据库优化完成 ===';
    RAISE NOTICE '新增索引数量: %', idx_count;
    RAISE NOTICE '新增部分索引数量: %', partial_count;
    RAISE NOTICE '请运行 ANALYZE 收集最新统计信息';
END $$;
