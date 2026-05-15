# 数据库查询优化和索引审查文档

## 概述

本文档描述了数据库查询优化策略和索引审查方案，用于提升数据库性能和减少慢查询。

## 文件结构

```
src/backend/
├── services/
│   └── queryOptimizer.js           # 查询优化器
└── tests/database/
    └── queryOptimizer.test.js      # 性能测试

migrations/
├── 010_query_optimization.up.sql    # 优化迁移
└── 010_query_optimization.down.sql  # 回滚迁移
```

## 查询优化策略

### 1. 查询优化器

**文件**: [queryOptimizer.js](file:///workspace/hjtpx/src/backend/services/queryOptimizer.js)

#### 核心功能

- **慢查询检测**: 自动识别执行时间超过阈值的查询
- **查询统计**: 追踪查询次数、失败率、平均延迟
- **索引建议**: 分析查询并提供索引优化建议
- **查询分析**: 使用EXPLAIN分析查询执行计划

#### API

```javascript
class QueryOptimizer {
  execute(query, params)      // 执行查询并统计
  analyzeQuery(query)          // 分析查询计划
  getTableIndexes(tableName)   // 获取表索引
  getMissingIndexes()          // 查找缺失索引
  getIndexUsage()              // 分析索引使用情况
  getSlowestQueries(limit)     // 获取最慢查询
}
```

### 2. 索引推荐系统

**文件**: [queryOptimizer.js](file:///workspace/hjtpx/src/backend/services/queryOptimizer.js#L176)

```javascript
class IndexRecommendation {
  analyze()                    // 分析并生成建议
  getRecommendations()         // 获取优化建议列表
}
```

#### 推荐类型

| 类型 | 优先级 | 说明 |
|------|--------|------|
| missing_index | 高 | 建议添加的索引 |
| unused_index | 中 | 建议删除的无用索引 |
| duplicate_index | 中 | 建议合并的重复索引 |

## 索引优化

### 1. 新增索引

**文件**: [010_query_optimization.up.sql](file:///workspace/hjtpx/migrations/010_query_optimization.up.sql)

```sql
-- 会话表优化
CREATE INDEX idx_sessions_user_id_created 
ON sessions(user_id, created_at DESC);

CREATE INDEX idx_sessions_expires_is_revoked 
ON sessions(expires_at, is_revoked) 
WHERE is_revoked = false;

-- 用户表优化
CREATE INDEX idx_users_email_role 
ON users(email, role) 
WHERE is_active = true;

CREATE INDEX idx_users_created_role 
ON users(created_at DESC, role);

-- 验证码日志表优化
CREATE INDEX idx_captcha_logs_user_id_created 
ON captcha_logs(user_id, created_at DESC);

CREATE INDEX idx_captcha_logs_type_created 
ON captcha_logs(captcha_type, created_at DESC);
```

### 2. 优化函数

#### 获取慢查询

```sql
SELECT * FROM get_slow_queries(100);  -- 100ms阈值
```

#### 获取表统计

```sql
SELECT * FROM get_table_stats();
```

#### 查找未使用索引

```sql
SELECT * FROM get_unused_indexes();
```

#### 查找缺失索引

```sql
SELECT * FROM find_missing_indexes();
```

### 3. 索引策略

#### 复合索引设计原则

1. **最左前缀原则**: 将区分度高的列放在前面
2. **覆盖索引**: 包含查询所需的所有列
3. **范围查询**: 范围列放在索引最后

```sql
-- ✅ 推荐: 查询优化
CREATE INDEX idx_users_email_active 
ON users(email, is_active);

-- 查询: SELECT * FROM users WHERE email = ? AND is_active = true

-- ❌ 不推荐: 无法利用索引
CREATE INDEX idx_users_active_email 
ON users(is_active, email);

-- 查询: SELECT * FROM users WHERE email = ? AND is_active = true
```

#### 部分索引

```sql
-- 只索引活跃用户
CREATE INDEX idx_users_email_active 
ON users(email) 
WHERE is_active = true;

-- 只索引未过期的会话
CREATE INDEX idx_sessions_active 
ON sessions(user_id) 
WHERE is_revoked = false AND expires_at > NOW();
```

## 查询优化技巧

### 1. 避免全表扫描

```sql
-- ❌ 全表扫描
SELECT * FROM users WHERE name LIKE '%john%';

-- ✅ 使用索引
SELECT * FROM users WHERE email = 'john@example.com';

-- ✅ 全文搜索
SELECT * FROM users WHERE MATCH(name) AGAINST('john');
```

### 2. 合理使用JOIN

```sql
-- ❌ N+1问题
SELECT * FROM posts;
-- 然后循环查询每篇文章的作者

-- ✅ 使用JOIN
SELECT p.*, u.name as author_name 
FROM posts p
LEFT JOIN users u ON p.author_id = u.id;

-- ✅ 使用子查询优化
SELECT p.*, 
  (SELECT name FROM users WHERE id = p.author_id) as author_name
FROM posts p;
```

### 3. 分页优化

```sql
-- ❌ 低效分页
SELECT * FROM logs ORDER BY id LIMIT 1000000, 10;

-- ✅ 使用游标分页
SELECT * FROM logs 
WHERE id > 1000000 
ORDER BY id 
LIMIT 10;

-- ✅ 使用WHERE条件分页
SELECT * FROM logs 
WHERE created_at > '2024-01-01' 
ORDER BY created_at 
LIMIT 10;
```

### 4. 批量操作

```sql
-- ❌ 循环插入
INSERT INTO logs (data) VALUES ('value1');
INSERT INTO logs (data) VALUES ('value2');
INSERT INTO logs (data) VALUES ('value3');

-- ✅ 批量插入
INSERT INTO logs (data) VALUES 
  ('value1'), ('value2'), ('value3');

-- ✅ 使用COPY命令（大量数据）
COPY logs(data) FROM stdin;
value1
value2
value3
\.
```

## 性能测试

**文件**: [queryOptimizer.test.js](file:///workspace/hjtpx/src/backend/tests/database/queryOptimizer.test.js)

### 测试用例

1. **基础查询测试**
   - 简单SELECT性能
   - WHERE条件查询
   - JOIN操作性能

2. **索引性能测试**
   - 有索引vs无索引对比
   - 复合索引性能
   - 部分索引性能

3. **慢查询检测**
   - 触发慢查询
   - 记录慢查询日志
   - 分析慢查询原因

4. **查询统计**
   - 查询计数
   - 平均延迟
   - 连接池状态

### 测试结果示例

```
=== Testing Index Performance ===
Index Analysis:
[
  {
    "Plan": {
      "Node Type": "Index Scan",
      "Index Name": "idx_users_email",
      "Startup Cost": 0.43,
      "Total Cost": 8.45,
      "Plan Rows": 1,
      "Actual Rows": 1
    }
  }
]

✓ Index performance test completed

=== Testing Query Statistics ===
Query Statistics:
- Total Queries: 103
- Slow Queries: 2
- Failed Queries: 0
- Average Query Time: 5.23ms

Connection Pool Stats:
- Total Connections: 20
- Idle Connections: 18
- Waiting Requests: 0
```

## 监控和分析

### 1. 实时监控

```javascript
const optimizer = new QueryOptimizer();

// 获取实时统计
const stats = optimizer.getStats();
console.log('总查询数:', stats.totalQueries);
console.log('慢查询数:', stats.slowQueries);
console.log('平均延迟:', stats.averageQueryTime);
console.log('失败查询:', stats.failedQueries);
```

### 2. 慢查询日志

```javascript
// 获取慢查询日志
const slowQueries = optimizer.getSlowQueryLog();
slowQueries.forEach(log => {
  console.log(`时间: ${log.timestamp}`);
  console.log(`查询: ${log.query}`);
  console.log(`延迟: ${log.duration}ms`);
});
```

### 3. 索引使用分析

```javascript
// 获取未使用索引
const unusedIndexes = await optimizer.getIndexUsage();
unusedIndexes.forEach(idx => {
  console.log(`表: ${idx.tablename}`);
  console.log(`索引: ${idx.indexname}`);
  console.log(`扫描次数: ${idx.idx_scan}`);
});
```

## 优化建议

### 高优先级

1. **添加缺失索引**
   ```sql
   CREATE INDEX idx_table_column ON table_name(column);
   ```

2. **删除无用索引**
   ```sql
   DROP INDEX IF EXISTS idx_unused_index;
   ```

3. **优化慢查询**
   ```sql
   EXPLAIN ANALYZE your_slow_query;
   ```

### 中优先级

4. **定期VACUUM和ANALYZE**
   ```sql
   VACUUM ANALYZE table_name;
   ```

5. **分区大表**
   ```sql
   CREATE TABLE logs_2024 (
     LIKE logs INCLUDING ALL
   ) PARTITION BY RANGE (created_at);
   ```

### 低优先级

6. **连接池优化**
   - 调整max_connections
   - 配置idle_in_transaction_session_timeout

7. **查询缓存**
   - 启用query_cache（MySQL）
   - 使用Redis缓存热点数据

## 维护计划

### 每日任务
- [ ] 监控慢查询
- [ ] 检查索引使用情况
- [ ] 记录异常查询

### 每周任务
- [ ] 分析查询日志
- [ ] 优化新增慢查询
- [ ] 清理无用索引

### 每月任务
- [ ] 执行VACUUM和ANALYZE
- [ ] 审查表统计信息
- [ ] 更新优化建议

## 常见问题

### Q: 如何判断索引是否有效？
A: 使用EXPLAIN分析查询计划，查看是否使用索引扫描。

### Q: 索引越多越好吗？
A: 不是，索引会占用存储空间并降低写入性能。只创建必要的索引。

### Q: 如何处理瞬时高负载？
A: 使用连接池限流，启用查询超时，配置慢查询日志。

### Q: 何时应该重建索引？
A: 当索引膨胀率超过10%或大量删除数据后。

---

**版本**: 1.0.0  
**创建日期**: 2026-05-15  
**最后更新**: 2026-05-15
