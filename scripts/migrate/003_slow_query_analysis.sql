-- 慢查询分析与优化建议脚本
-- 用于识别和分析 PostgreSQL 慢查询
-- 创建时间: 2026-05-18

-- ============================================================
-- 1. 检查是否启用 pg_stat_statements
-- ============================================================

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_statements') THEN
        RAISE NOTICE 'pg_stat_statements 扩展未启用。建议执行: CREATE EXTENSION pg_stat_statements;';
    ELSE
        RAISE NOTICE 'pg_stat_statements 已启用';
    END IF;
END $$;

-- ============================================================
-- 2. 获取最慢的查询（按平均执行时间排序）
-- ============================================================

-- 最慢的20个查询（按平均时间）
SELECT 
    ROUND(mean_time::NUMERIC / 1000, 2) AS mean_time_ms,
    ROUND(min_time::NUMERIC / 1000, 2) AS min_time_ms,
    ROUND(max_time::NUMERIC / 1000, 2) AS max_time_ms,
    ROUND(total_time::NUMERIC / 1000, 2) AS total_time_ms,
    calls,
    rows,
    ROUND((rows::NUMERIC / NULLIF(calls, 0)), 2) AS avg_rows,
    query AS query_text
FROM pg_stat_statements
WHERE query LIKE '%verifications%' 
   OR query LIKE '%applications%'
   OR query LIKE '%users%'
   OR query LIKE '%blacklist%'
ORDER BY mean_time DESC
LIMIT 20;

-- ============================================================
-- 3. 获取调用频率最高的查询
-- ============================================================

SELECT 
    calls,
    ROUND(mean_time::NUMERIC / 1000, 2) AS mean_time_ms,
    ROUND(total_time::NUMERIC / 1000, 2) AS total_time_ms,
    rows,
    query AS query_text
FROM pg_stat_statements
WHERE calls > 100
ORDER BY calls DESC
LIMIT 30;

-- ============================================================
-- 4. 获取消耗时间最多的查询
-- ============================================================

SELECT 
    ROUND(total_time::NUMERIC / 1000, 2) AS total_time_ms,
    calls,
    ROUND(mean_time::NUMERIC / 1000, 2) AS mean_time_ms,
    rows,
    query AS query_text
FROM pg_stat_statements
ORDER BY total_time DESC
LIMIT 20;

-- ============================================================
-- 5. 获取需要大量 IO 的查询
-- ============================================================

SELECT 
    query,
    shared_blks_hit,
    shared_blks_read,
    ROUND(shared_blks_read::NUMERIC / NULLIF(shared_blks_hit + shared_blks_read, 0) * 100, 2) AS read_ratio_pct,
    temp_blks_read,
    temp_blks_written
FROM pg_stat_statements
WHERE (shared_blks_read > 1000 OR temp_blks_read > 100)
  AND shared_blks_hit + shared_blks_read > 0
ORDER BY shared_blks_read DESC
LIMIT 20;

-- ============================================================
-- 6. 表扫描分析（识别缺少索引的表）
-- ============================================================

SELECT 
    relname AS table_name,
    seq_scan,
    seq_tup_read,
    idx_scan,
    idx_tup_fetch,
    ROUND(idx_tup_fetch::NUMERIC / NULLIF(seq_tup_read + idx_tup_fetch, 0) * 100, 2) AS index_usage_pct,
    n_tup_ins,
    n_tup_upd,
    n_tup_del,
    n_live_tup,
    n_dead_tup,
    last_vacuum,
    last_autovacuum,
    last_analyze
FROM pg_stat_user_tables
WHERE schemaname = 'public'
ORDER BY seq_scan DESC
LIMIT 30;

-- ============================================================
-- 7. 索引使用情况分析
-- ============================================================

SELECT 
    schemaname,
    relname AS table_name,
    indexrelname AS index_name,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch,
    pg_size_pretty(pg_relation_size(indexrelid)) AS index_size,
    CASE 
        WHEN idx_scan = 0 THEN '未使用-建议删除'
        WHEN idx_tup_fetch::NUMERIC / NULLIF(idx_scan, 0) < 0.5 THEN '低效-考虑优化'
        ELSE '正常'
    END AS status
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
  AND indexrelname NOT LIKE '%pkey%'
ORDER BY idx_scan ASC
LIMIT 50;

-- ============================================================
-- 8. 未使用的索引（建议删除以减少写入开销）
-- ============================================================

SELECT 
    schemaname,
    relname AS table_name,
    indexrelname AS index_name,
    pg_size_pretty(pg_relation_size(indexrelid)) AS index_size,
    idx_scan,
    pg_relation_size(indexrelid) AS size_bytes
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
  AND indexrelname NOT LIKE '%pkey%'
  AND indexrelname NOT LIKE '%_key%'
  AND idx_scan = 0
ORDER BY pg_relation_size(indexrelid) DESC;

-- ============================================================
-- 9. 大表排名（用于优先优化）
-- ============================================================

SELECT 
    schemaname,
    relname AS table_name,
    pg_total_relation_size(relid) AS total_size_bytes,
    pg_size_pretty(pg_total_relation_size(relid)) AS total_size,
    pg_relation_size(relid) AS table_size_bytes,
    pg_size_pretty(pg_relation_size(relid)) AS table_size,
    pg_indexes_size(relid) AS indexes_size_bytes,
    pg_size_pretty(pg_indexes_size(relid)) AS indexes_size,
    n_live_tup,
    n_dead_tup,
    n_tup_ins,
    n_tup_upd,
    n_tup_del
FROM pg_stat_user_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(relid) DESC
LIMIT 20;

-- ============================================================
-- 10. 缓存命中率分析
-- ============================================================

SELECT 
    relname AS table_name,
    heap_blks_read,
    heap_blks_hit,
    ROUND(heap_blks_hit::NUMERIC / NULLIF(heap_blks_hit + heap_blks_read, 0) * 100, 2) AS cache_hit_ratio,
    idx_blks_read,
    idx_blks_hit,
    ROUND(idx_blks_hit::NUMERIC / NULLIF(idx_blks_hit + idx_blks_read, 0) * 100, 2) AS idx_cache_hit_ratio,
    toast_blks_read,
    toast_blks_hit
FROM pg_statio_user_tables
WHERE schemaname = 'public'
  AND heap_blks_hit + heap_blks_read > 0
ORDER BY heap_blks_read DESC
LIMIT 20;

-- ============================================================
-- 11. 等待事件分析（识别锁和阻塞）
-- ============================================================

SELECT 
    wait_event_type,
    wait_event,
    COUNT(*) AS count,
    pg_size_pretty(SUM(pg_blocking_pids_count(pid))) AS blocking_info
FROM pg_stat_activity
WHERE state != 'idle'
GROUP BY wait_event_type, wait_event
ORDER BY count DESC;

-- ============================================================
-- 12. 长时间运行的事务
-- ============================================================

SELECT 
    pid,
    usename,
    application_name,
    state,
    query,
    state_change,
    ROUND(EXTRACT(EPOCH FROM (NOW() - state_change))) AS duration_seconds,
    wait_event_type,
    wait_event
FROM pg_stat_activity
WHERE state != 'idle'
  AND state_change < NOW() - INTERVAL '5 minutes'
ORDER BY state_change ASC;

-- ============================================================
-- 13. 连接数分析
-- ============================================================

SELECT 
    state,
    COUNT(*) AS count,
    ARRAY_AGG(DISTINCT usename) AS users,
    ARRAY_AGG(DISTINCT application_name) AS applications
FROM pg_stat_activity
WHERE datname = current_database()
GROUP BY state
ORDER BY count DESC;

-- ============================================================
-- 14. 生成优化建议
-- ============================================================

DO $$
DECLARE
    v_seq_scan_record RECORD;
    v_unused_index_record RECORD;
    v_cache_miss_record RECORD;
    slow_query_count INTEGER := 0;
    unused_index_count INTEGER := 0;
    cache_miss_count INTEGER := 0;
BEGIN
    -- 检查顺序扫描过多的表
    FOR v_seq_scan_record IN
        SELECT relname, seq_scan, idx_scan
        FROM pg_stat_user_tables
        WHERE schemaname = 'public'
          AND seq_scan > 1000
          AND (idx_scan = 0 OR idx_scan < seq_scan / 10)
        ORDER BY seq_scan DESC
    LOOP
        RAISE NOTICE '优化建议: 表 % 有过多的顺序扫描 (seq_scan: %, idx_scan: %)', 
            v_seq_scan_record.relname, 
            v_seq_scan_record.seq_scan, 
            v_seq_scan_record.idx_scan;
    END LOOP;

    -- 检查未使用的索引
    FOR v_unused_index_record IN
        SELECT relname, indexrelname, pg_size_pretty(pg_relation_size(indexrelid)) AS size
        FROM pg_stat_user_indexes
        WHERE schemaname = 'public'
          AND indexrelname NOT LIKE '%pkey%'
          AND idx_scan = 0
        ORDER BY pg_relation_size(indexrelid) DESC
    LOOP
        unused_index_count := unused_index_count + 1;
        RAISE NOTICE '优化建议: 索引 %.% 从未使用，可考虑删除以减少写入开销 (大小: %)', 
            v_unused_index_record.relname, 
            v_unused_index_record.indexrelname,
            v_unused_index_record.size;
    END LOOP;

    -- 检查缓存命中率低的表
    FOR v_cache_miss_record IN
        SELECT 
            relname,
            heap_blks_hit,
            heap_blks_read,
            ROUND(heap_blks_hit::NUMERIC / NULLIF(heap_blks_hit + heap_blks_read, 0) * 100, 2) AS hit_ratio
        FROM pg_statio_user_tables
        WHERE schemaname = 'public'
          AND heap_blks_hit + heap_blks_read > 10000
          AND ROUND(heap_blks_hit::NUMERIC / NULLIF(heap_blks_hit + heap_blks_read, 0) * 100, 2) < 90
    LOOP
        cache_miss_count := cache_miss_count + 1;
        RAISE NOTICE '优化建议: 表 % 缓存命中率较低 (%)，建议增加 work_mem 或优化查询', 
            v_cache_miss_record.relname, 
            v_cache_miss_record.hit_ratio;
    END LOOP;

    -- 检查慢查询
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_statements') THEN
        SELECT COUNT(*) INTO slow_query_count
        FROM pg_stat_statements
        WHERE mean_time > 100000;  -- 超过 100ms
        
        RAISE NOTICE 'pg_stat_statements 统计: 超过 100ms 的查询数量: %', slow_query_count;
    ELSE
        RAISE NOTICE 'pg_stat_statements 未启用，无法进行慢查询分析';
    END IF;

    RAISE NOTICE '';
    RAISE NOTICE '=== 优化分析完成 ===';
    RAISE NOTICE '发现未使用索引: % 个', unused_index_count;
    RAISE NOTICE '发现缓存命中率低: % 个', cache_miss_count;
    RAISE NOTICE '慢查询 (>100ms): % 个', slow_query_count;
END $$;

-- ============================================================
-- 15. EXPLAIN ANALYZE 示例查询
-- ============================================================

-- 分析验证码查询性能
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT v.*, u.username, a.name as app_name
FROM verifications v
LEFT JOIN users u ON v.user_id = u.id
LEFT JOIN applications a ON v.application_id = a.id
WHERE v.status = 'success'
  AND v.created_at > NOW() - INTERVAL '7 days'
ORDER BY v.created_at DESC
LIMIT 100;

-- 分析黑名单查询
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT * FROM blacklist
WHERE type = 'ip'
  AND status = 'active'
  AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY created_at DESC
LIMIT 50;

-- 分析用户验证统计
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT 
    u.id, 
    u.username, 
    COUNT(v.id) as verification_count,
    COUNT(CASE WHEN v.status = 'success' THEN 1 END) as success_count
FROM users u
LEFT JOIN verifications v ON u.id = v.user_id
WHERE u.created_at > NOW() - INTERVAL '30 days'
GROUP BY u.id, u.username
ORDER BY verification_count DESC
LIMIT 50;
