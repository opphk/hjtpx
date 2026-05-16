# 数据库性能优化指南

## 索引优化

### 1. 验证日志表索引

```sql
-- 基础索引
CREATE INDEX idx_verification_logs_session_id ON verification_logs(session_id);
CREATE INDEX idx_verification_logs_application_id ON verification_logs(application_id);
CREATE INDEX idx_verification_logs_created_at ON verification_logs(created_at);
CREATE INDEX idx_verification_logs_status ON verification_logs(status);

-- 复合索引（常用查询模式）
CREATE INDEX idx_verification_logs_composite ON verification_logs(application_id, created_at, status);

-- 用于时间范围统计查询
CREATE INDEX idx_verification_logs_time_range ON verification_logs(created_at DESC);
```

### 2. 验证表索引

```sql
CREATE INDEX idx_verifications_session_id ON verifications(session_id);
CREATE INDEX idx_verifications_application_id ON verifications(application_id);
CREATE INDEX idx_verifications_user_id ON verifications(user_id);
CREATE INDEX idx_verifications_status ON verifications(status);
```

### 3. 静默验证表索引

```sql
CREATE INDEX idx_silent_verification_token ON silent_verifications(token);
CREATE INDEX idx_silent_verification_session_id ON silent_verifications(session_id);
CREATE INDEX idx_silent_verification_user_id ON silent_verifications(user_id);
CREATE INDEX idx_silent_verification_risk_level ON silent_verifications(risk_level);
```

### 4. 应用表索引

```sql
CREATE INDEX idx_apps_user_id ON applications(user_id);
CREATE INDEX idx_apps_api_key ON applications(api_key);
CREATE INDEX idx_apps_is_active ON applications(is_active);
```

### 5. 设备指纹表索引

```sql
CREATE INDEX idx_device_fingerprint_user_id ON device_fingerprints(user_id);
CREATE INDEX idx_device_fingerprint_hash ON device_fingerprints(fingerprint_hash);
CREATE INDEX idx_device_fingerprint_device ON device_fingerprints(device_fingerprint);
```

## 查询优化

### 1. 避免SELECT *

```go
// 不推荐
rows, err := db.Query("SELECT * FROM verification_logs WHERE application_id = ?", appID)

// 推荐
rows, err := db.Query(`
    SELECT id, session_id, captcha_type, status, risk_score, created_at 
    FROM verification_logs 
    WHERE application_id = ?`, appID)
```

### 2. 使用预编译语句

```go
// 预编译语句
stmt, err := db.Prepare("SELECT * FROM verification_logs WHERE session_id = ?")
defer stmt.Close()

// 多次执行
row := stmt.QueryRow("session_123")
```

### 3. 批量操作优化

```go
// 批量插入
const batchSize = 100
for i := 0; i < len(items); i += batchSize {
    batch := items[i:min(i+batchSize, len(items))]
    db.CreateInBatches(batch, 50)
}
```

### 4. 分页查询优化

```go
// 使用游标分页（更高效）
func GetVerificationLogsCursor(appID uint, cursor string, limit int) ([]Log, string, error) {
    var logs []Log
    query := `
        SELECT id, session_id, captcha_type, status, created_at 
        FROM verification_logs 
        WHERE application_id = ? AND id > ?
        ORDER BY id ASC 
        LIMIT ?`
    
    rows, err := db.Query(query, appID, cursor, limit)
    // ...
}
```

### 5. 聚合查询优化

```go
// 使用索引覆盖的聚合查询
func GetStatsByApp(appID uint) (Stats, error) {
    var stats Stats
    query := `
        SELECT 
            COUNT(*) as total,
            SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_count,
            SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count,
            AVG(risk_score) as avg_risk_score
        FROM verification_logs 
        WHERE application_id = ? AND created_at > NOW() - INTERVAL '30 days'`
    
    err := db.QueryRow(query, appID).Scan(&stats.Total, &stats.SuccessCount, &stats.FailedCount, &stats.AvgRiskScore)
    return stats, err
}
```

## 连接池配置

### 1. 推荐配置

```go
// 连接池设置
const (
    MaxOpenConnsDefault    = 100  // 最大打开连接数
    MaxIdleConnsDefault    = 10   // 最大空闲连接数
    ConnMaxLifetimeDefault = 30   // 连接最大生命周期（分钟）
    ConnMaxIdleTimeDefault = 5    // 空闲连接超时（分钟）
)

// 设置
db.SetMaxOpenConns(100)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(30 * time.Minute)
db.SetConnMaxIdleTime(5 * time.Minute)
```

### 2. 生产环境配置建议

```yaml
# 数据库配置
max_open_conns: 100
max_idle_conns: 20
conn_max_lifetime: 30m
conn_max_idle_time: 5m

# 连接超时
connect_timeout: 10s
query_timeout: 30s
```

### 3. 监控连接池状态

```go
func MonitorDBPool() {
    stats := db.Stats()
    fmt.Printf("打开连接数: %d\n", stats.OpenConnections)
    fmt.Printf("使用中: %d\n", stats.InUse)
    fmt.Printf("空闲: %d\n", stats.Idle)
    fmt.Printf("等待次数: %d\n", stats.WaitCount)
    fmt.Printf("等待时间: %v\n", stats.WaitDuration)
}
```

## 表分区策略

### 1. 日志表分区（按时间）

```sql
-- 创建分区表
CREATE TABLE verification_logs (
    id BIGSERIAL,
    verification_id BIGINT,
    session_id VARCHAR(100),
    application_id BIGINT NOT NULL,
    captcha_type VARCHAR(50),
    status VARCHAR(50) NOT NULL,
    ip_address VARCHAR(50),
    user_agent VARCHAR(500),
    risk_score DOUBLE PRECISION DEFAULT 0,
    analysis_result TEXT,
    duration BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) PARTITION BY RANGE (created_at);

-- 创建月度分区
CREATE TABLE verification_logs_2024_01 PARTITION OF verification_logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE verification_logs_2024_02 PARTITION OF verification_logs
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- 继续创建其他月份分区...
```

### 2. 自动分区管理

```go
func EnsurePartitions() error {
    now := time.Now()
    for i := 0; i < 3; i++ {
        nextMonth := now.AddDate(0, i, 0)
        partitionName := fmt.Sprintf("verification_logs_%d_%02d", 
            nextMonth.Year(), nextMonth.Month())
        
        query := fmt.Sprintf(`
            CREATE TABLE IF NOT EXISTS %s PARTITION OF verification_logs
            FOR VALUES FROM ('%d-%02d-01') TO ('%d-%02d-01')`,
            partitionName,
            nextMonth.Year(), nextMonth.Month(),
            nextMonth.AddDate(0, 1, 0).Year(), nextMonth.AddDate(0, 1, 0).Month())
        
        if _, err := db.Exec(query); err != nil {
            return err
        }
    }
    return nil
}
```

## 性能监控

### 1. 慢查询日志

```go
// 配置慢查询阈值
db.SetMaxOpenConns(100)
db.SetMaxIdleConns(10)

// 记录慢查询
func LogSlowQuery(query string, duration time.Duration) {
    if duration > 100*time.Millisecond {
        log.Printf("[SLOW] query=%s duration=%v", query, duration)
    }
}
```

### 2. 查询性能追踪

```go
func TrackQuery(name, query string) func() {
    start := time.Now()
    return func() {
        duration := time.Since(start)
        fmt.Printf("Query [%s] took %v\n", name, duration)
        
        if duration > time.Second {
            fmt.Printf("WARNING: Slow query detected!\n%s\n", query)
        }
    }
}

// 使用
defer TrackQuery("GetUserVerifications", query)()
```

### 3. 数据库健康检查

```go
func HealthCheck() error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := db.PingContext(ctx); err != nil {
        return fmt.Errorf("database ping failed: %w", err)
    }
    
    stats := db.Stats()
    if stats.OpenConnections >= stats.MaxOpenConns {
        return fmt.Errorf("database connection pool exhausted")
    }
    
    return nil
}
```

## 备份与恢复

### 1. 自动备份策略

```bash
#!/bin/bash
# 每日备份脚本

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups/postgres"
DB_NAME="verification"

# 创建备份
pg_dump -U postgres -d $DB_NAME -F c -f "$BACKUP_DIR/$DB_NAME-$DATE.dump"

# 删除7天前的备份
find $BACKUP_DIR -name "*.dump" -mtime +7 -delete

# 上传到远程存储
aws s3 cp "$BACKUP_DIR/$DB_NAME-$DATE.dump" s3://backups/$DB_NAME/
```

### 2. 恢复流程

```bash
# 恢复到指定时间点
pg_restore -U postgres -d verification -c --date-time="2024-01-01 12:00:00" backup.dump

# 恢复到新数据库
pg_restore -U postgres -d verification_new backup.dump
```

## 优化检查清单

- [ ] 所有高频查询都有相应索引
- [ ] 使用EXPLAIN分析慢查询
- [ ] 批量操作使用CreateInBatches
- [ ] 连接池配置合理（100/10/30/5）
- [ ] 启用慢查询日志
- [ ] 定期清理过期数据
- [ ] 使用连接池监控
- [ ] 考虑表分区策略
- [ ] 备份策略完善
- [ ] 定期执行ANALYZE更新统计信息
