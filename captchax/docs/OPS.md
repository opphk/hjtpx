# CaptchaX 运维手册

## 目录

- [日常运维](#日常运维)
- [监控告警](#监控告警)
- [日志管理](#日志管理)
- [备份恢复](#备份恢复)
- [性能调优](#性能调优)
- [故障排查](#故障排查)
- [安全运维](#安全运维)

---

## 日常运维

### 服务管理

#### Docker 环境

```bash
# 查看服务状态
docker-compose ps

# 查看实时日志
docker-compose logs -f captchax

# 重启服务
docker-compose restart captchax

# 停止服务
docker-compose stop captchax

# 启动服务
docker-compose start captchax

# 重新构建并启动
docker-compose up -d --build
```

#### systemd 环境

```bash
# 查看服务状态
sudo systemctl status captchax

# 查看实时日志
sudo journalctl -u captchax -f

# 重启服务
sudo systemctl restart captchax

# 停止服务
sudo systemctl stop captchax

# 重新加载配置
sudo systemctl reload captchax
```

### 健康检查

```bash
# API 服务健康检查
curl http://localhost:8080/health

# 管理后台健康检查
curl http://localhost:8081/health

# 数据库连接检查
psql -h localhost -U captcha_admin -d captcha_db -c "SELECT 1;"

# Redis 连接检查
redis-cli ping
```

### 资源监控

```bash
# Docker 资源使用
docker stats captchax

# 磁盘使用
df -h

# 内存使用
free -h

# CPU 使用
top -bn1 | grep "Cpu(s)"
```

---

## 监控告警

### Prometheus 监控

#### 安装 Prometheus

```bash
# docker-compose 添加
cat >> docker-compose.yml << 'EOF'
  prometheus:
    image: prom/prometheus:latest
    container_name: captchax-prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
    restart: unless-stopped
    networks:
      - captchax-net
EOF
```

#### prometheus.yml 配置

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

alerting:
  alertmanagers:
    - static_configs:
        - targets: []

rule_files: []

scrape_configs:
  - job_name: 'captchax'
    static_configs:
      - targets: ['captchax:8080']
    metrics_path: '/metrics'
    scrape_interval: 10s
```

### Grafana 仪表盘

#### 安装 Grafana

```yaml
grafana:
  image: grafana/grafana:latest
  container_name: captchax-grafana
  ports:
    - "3000:3000"
  volumes:
    - grafana_data:/var/lib/grafana
    - ./grafana/provisioning:/etc/grafana/provisioning:ro
  environment:
    - GF_SECURITY_ADMIN_USER=admin
    - GF_SECURITY_ADMIN_PASSWORD=admin123
  restart: unless-stopped
  networks:
    - captchax-net
```

#### 常用监控指标

| 指标名称 | 说明 | 告警阈值 |
|----------|------|----------|
| captchax_requests_total | 请求总数 | - |
| captchax_requests_duration_seconds | 请求延迟 | p99 > 1s |
| captchax_verification_total | 验证总数 | - |
| captchax_verification_success_rate | 验证成功率 | < 95% |
| captchax_cache_hit_rate | 缓存命中率 | < 80% |
| captchax_active_sessions | 活跃会话数 | > 10000 |
| captchax_error_rate | 错误率 | > 1% |

### 告警规则

```yaml
# alertmanager.yml
groups:
  - name: captchax
    rules:
      - alert: CaptchaXDown
        expr: up{job="captchax"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "CaptchaX 服务不可用"
          description: "CaptchaX 服务已停止超过 1 分钟"

      - alert: HighErrorRate
        expr: rate(captchax_errors_total[5m]) > 0.01
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "错误率过高"
          description: "CaptchaX 错误率超过 1%"

      - alert: HighLatency
        expr: histogram_quantile(0.99, rate(captchax_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "请求延迟过高"
          description: "99% 分位延迟超过 1 秒"

      - alert: LowSuccessRate
        expr: rate(captchax_verification_success_total[5m]) / rate(captchax_verification_total[5m]) < 0.95
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "验证成功率过低"
          description: "验证成功率低于 95%"

      - alert: RedisConnectionError
        expr: captchax_redis_errors_total > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Redis 连接错误"
          description: "Redis 连接出现错误"

      - alert: DatabaseConnectionError
        expr: captchax_db_errors_total > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "数据库连接错误"
          description: "数据库连接出现错误"
```

---

## 日志管理

### 日志配置

```yaml
log:
  level: "info"           # debug/info/warn/error
  format: "json"          # json/text
  output: "stdout"       # stdout/file
  file_path: "/var/log/captchax/app.log"
  max_size: 100          # MB
  max_backups: 30        # 保留份数
  max_age: 30            # 天
  compress: true         # 压缩
```

### 日志级别说明

| 级别 | 说明 | 使用场景 |
|------|------|----------|
| debug | 调试信息 | 开发环境、问题排查 |
| info | 一般信息 | 正常运行日志 |
| warn | 警告信息 | 需要关注但不阻断 |
| error | 错误信息 | 需要处理的问题 |

### 日志分析

#### 查看错误日志

```bash
# Docker 环境
docker-compose logs captchax | grep ERROR

# systemd 环境
sudo journalctl -u captchax -p err

# 查看最近 100 条错误
docker-compose logs --tail=100 captchax 2>&1 | grep -i error
```

#### 日志关键词

| 关键词 | 含义 |
|--------|------|
| verify_failed | 验证失败 |
| rate_limited | 触发限流 |
| blacklisted | 命中黑名单 |
| ip_blocked | IP 被封禁 |
| token_expired | Token 过期 |
| db_error | 数据库错误 |
| redis_error | Redis 错误 |

#### 日志统计

```bash
# 统计每分钟请求量
docker-compose logs --since 1h captchax 2>&1 | grep "verification" | wc -l

# 统计错误分布
docker-compose logs --since 1h captchax 2>&1 | grep -oE '\[ERROR\] [a-z_]+' | sort | uniq -c

# 分析异常 IP
docker-compose logs --since 24h captchax 2>&1 | grep "blacklisted" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+' | sort | uniq -c | sort -rn | head -10
```

### 日志收集 (ELK)

#### Filebeat 配置

```yaml
filebeat.inputs:
  - type: container
    paths:
      - /var/lib/docker/containers/captchax/*.log
    processors:
      - add_docker_metadata:
          host: "unix:///var/run/docker.sock"
    fields:
      service: captchax
    fields_under_root: true

output.elasticsearch:
  hosts: ["elasticsearch:9200"]

setup.kibana:
  host: "kibana:5601"
```

#### Elasticsearch 索引模板

```bash
curl -X PUT "localhost:9200/_index_template/captchax" -H 'Content-Type: application/json' -d'
{
  "index_patterns": ["captchax-*"],
  "template": {
    "settings": {
      "number_of_shards": 1,
      "number_of_replicas": 1
    },
    "mappings": {
      "properties": {
        "@timestamp": { "type": "date" },
        "level": { "type": "keyword" },
        "message": { "type": "text" },
        "service": { "type": "keyword" },
        "trace_id": { "type": "keyword" },
        "user_ip": { "type": "ip" }
      }
    }
  }
}'
```

---

## 备份恢复

### 自动备份

#### Docker 环境备份脚本

```bash
#!/bin/bash
# backup.sh

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/opt/backups"
RETENTION_DAYS=30

mkdir -p ${BACKUP_DIR}

# 备份 PostgreSQL
echo "开始备份 PostgreSQL..."
docker exec captchax-postgres pg_dump -U captcha_admin captcha_db | gzip > ${BACKUP_DIR}/captcha_db_${DATE}.sql.gz

# 备份 Redis
echo "开始备份 Redis..."
docker exec captchax-redis redis-cli SAVE
docker cp captchax-redis:/data/dump.rdb ${BACKUP_DIR}/redis_${DATE}.rdb

# 备份配置文件
echo "备份配置文件..."
cp -r ~/captchax/config ${BACKUP_DIR}/config_${DATE}

# 备份完成日志
echo "[$(date)] 备份完成: captcha_db_${DATE}.sql.gz" >> ${BACKUP_DIR}/backup.log

# 清理过期备份
echo "清理过期备份..."
find ${BACKUP_DIR} -type f -mtime +${RETENTION_DAYS} -delete

echo "备份完成!"
```

#### 添加定时任务

```bash
# 编辑 crontab
crontab -e

# 添加备份任务（每天凌晨 3 点执行）
0 3 * * * /opt/scripts/backup.sh >> /var/log/backup.log 2>&1
```

### 手动备份

```bash
# 备份数据库
docker exec captchax-postgres pg_dump -U captcha_admin captcha_db > captcha_backup.sql

# 备份 Redis
docker exec captchax-redis redis-cli BGSAVE
docker cp captchax-redis:/data/dump.rdb ./redis_backup.rdb

# 备份配置
tar -czf config_backup.tar.gz ~/captchax/config
```

### 数据恢复

#### 恢复 PostgreSQL

```bash
# 停止服务
docker-compose stop captchax

# 恢复数据库
gunzip < /opt/backups/captcha_db_20260514_030000.sql.gz | docker exec -i captchax-postgres psql -U captcha_admin captcha_db

# 或者
cat /opt/backups/captcha_db_20260514_030000.sql | docker exec -i captchax-postgres psql -U captcha_admin captcha_db

# 启动服务
docker-compose start captchax
```

#### 恢复 Redis

```bash
# 停止 Redis
docker-compose stop redis

# 恢复数据
docker cp /opt/backups/redis_20260514_030000.rdb captchax-redis:/data/dump.rdb

# 启动 Redis
docker-compose start redis
```

#### 恢复配置

```bash
# 恢复配置文件
tar -xzf config_backup.tar.gz -C ~/captchax/

# 重启服务
docker-compose restart captchax
```

### 定时清理

```bash
#!/bin/bash
# cleanup.sh

# 清理 Docker 日志
truncate -s 0 /var/lib/docker/containers/*/*-json.log

# 清理过期备份
find /opt/backups -type f -mtime +90 -delete

# 清理 Redis 历史数据
redis-cli << EOF
  KEYS "captcha:history:*" | xargs -r UNLINK
  KEYS "captcha:log:*" | xargs -r UNLINK
EOF

# 清理数据库历史记录
docker exec captchax-postgres psql -U captcha_admin captcha_db -c "DELETE FROM captcha_logs WHERE created_at < NOW() - INTERVAL '90 days';"
```

---

## 性能调优

### 数据库优化

#### PostgreSQL 配置

```ini
# postgresql.conf

# 连接数
max_connections = 100

# 内存
shared_buffers = 256MB
effective_cache_size = 1GB
work_mem = 16MB
maintenance_work_mem = 128MB

# 写入优化
wal_buffers = 16MB
checkpoint_completion_target = 0.9
max_wal_size = 1GB

# 并行
max_worker_processes = 4
max_parallel_workers_per_gather = 2
max_parallel_workers = 4

# 日志
log_statement = 'none'
log_duration = off
log_lock_waits = on
```

#### 索引优化

```sql
-- 验证码日志索引
CREATE INDEX CONCURRENTLY idx_captcha_logs_ip ON captcha_logs(ip);
CREATE INDEX CONCURRENTLY idx_captcha_logs_created ON captcha_logs(created_at);
CREATE INDEX CONCURRENTLY idx_captcha_logs_app_id ON captcha_logs(app_id);
CREATE INDEX CONCURRENTLY idx_captcha_logs_result ON captcha_logs(result);

-- 复合索引
CREATE INDEX CONCURRENTLY idx_captcha_logs_ip_created ON captcha_logs(ip, created_at);

-- 黑名单索引
CREATE INDEX CONCURRENTLY idx_blacklist_ip ON blacklist(ip);
CREATE INDEX CONCURRENTLY idx_blacklist_expires ON blacklist(expires_at) WHERE expires_at IS NOT NULL;

-- 白名单索引
CREATE INDEX CONCURRENTLY idx_whitelist_ip ON whitelist(ip);
```

#### 定期维护

```sql
--  vacuum 清理
VACUUM (VERBOSE, ANALYZE) captcha_logs;
VACUUM (VERBOSE, ANALYZE) captcha_config;

-- 重建索引
REINDEX TABLE captcha_logs;
REINDEX TABLE blacklist;
REINDEX TABLE whitelist;
```

### Redis 优化

```bash
# redis.conf 优化配置

# 内存配置
maxmemory 512mb
maxmemory-policy allkeys-lru

# 持久化
save 900 1
save 300 10
save 60 10000

# 连接
timeout 300
tcp-keepalive 60

# 慢查询日志
slowlog-log-slower-than 10000
slowlog-max-len 128
```

### 服务优化

```yaml
# config.yaml 优化配置

database:
  max_open_conns: 50
  max_idle_conns: 10
  conn_max_lifetime: 300

redis:
  pool_size: 20
  read_timeout: 3s
  write_timeout: 3s

captcha:
  expire_minutes: 5
  max_attempts: 3
```

### 缓存优化

```bash
# 预热缓存
curl -X POST "http://localhost:8080/admin/api/cache/preheat"

# 查看缓存统计
curl "http://localhost:8080/admin/api/cache/stats"

# 清理过期缓存
curl -X POST "http://localhost:8080/admin/api/cache/cleanup"
```

---

## 故障排查

### 服务无响应

#### 排查步骤

```bash
# 1. 检查服务状态
docker-compose ps
sudo systemctl status captchax

# 2. 检查资源使用
docker stats
free -h
df -h

# 3. 检查端口监听
netstat -tlnp | grep 8080

# 4. 检查进程
ps aux | grep captchax

# 5. 查看错误日志
docker-compose logs --tail=100 captchax
sudo journalctl -u captchax -n 100 --no-pager

# 6. 测试本地连接
curl http://127.0.0.1:8080/health
```

### 数据库连接问题

```bash
# 1. 检查数据库服务
docker-compose ps postgres
sudo systemctl status postgresql

# 2. 测试数据库连接
docker exec -it captchax psql -h postgres -U captcha_admin -d captcha_db -c "SELECT 1;"

# 3. 检查连接数
docker exec -it captchax-postgres psql -U captcha_admin -d captcha_db -c "SELECT count(*) FROM pg_stat_activity;"

# 4. 查看连接配置
grep max_connections config.yaml

# 5. 重启数据库
docker-compose restart postgres
```

### Redis 连接问题

```bash
# 1. 检查 Redis 服务
docker-compose ps redis
sudo systemctl status redis-server

# 2. 测试 Redis 连接
docker exec -it captchax redis-cli -h redis ping
redis-cli ping

# 3. 检查 Redis 内存
redis-cli INFO memory | grep used_memory_human

# 4. 检查 Redis 配置
redis-cli CONFIG GET maxmemory

# 5. 重启 Redis
docker-compose restart redis
```

### 验证码生成失败

```bash
# 1. 检查服务日志
docker-compose logs --tail=50 captchax | grep -i captcha

# 2. 检查图像处理库
docker exec -it captchax identify --version

# 3. 测试手动生成
curl -X POST "http://localhost:8080/api/v1/captcha/slider" \
  -H "Content-Type: application/json" \
  -d '{"app_id": "test"}'

# 4. 检查临时目录空间
df -h /tmp

# 5. 检查内存
free -h
```

### 内存泄漏排查

```bash
# 1. 查看内存使用趋势
docker stats --no-stream

# 2. 查看 Go 堆内存
curl http://localhost:8080/debug/pprof/heap

# 3. 查看 Goroutine 数量
curl http://localhost:8080/debug/pprof/goroutine?debug=1

# 4. 触发 GC
curl -X POST http://localhost:8080/debug/gc

# 5. 重启服务
docker-compose restart captchax
```

### 网络问题

```bash
# 1. 检查网络连通性
ping -c 3 redis
ping -c 3 postgres

# 2. 检查 DNS
nslookup redis
nslookup postgres

# 3. 检查防火墙
sudo iptables -L -n
sudo ufw status

# 4. 检查 Docker 网络
docker network ls
docker network inspect captchax-net

# 5. 测试端口连通性
telnet redis 6379
telnet postgres 5432
```

---

## 安全运维

### 安全检查清单

| 检查项 | 频率 | 执行人 |
|--------|------|--------|
| 检查日志异常 | 每日 | 运维 |
| 更新安全补丁 | 每周 | 运维 |
| 审查访问日志 | 每周 | 安全 |
| 备份验证测试 | 每月 | 运维 |
| 密码更新 | 每季度 | 管理员 |
| 安全审计 | 每季度 | 安全 |

### 密钥管理

```bash
# 生成新 JWT 密钥
openssl rand -base64 32

# 更新配置文件
vim config/config.yaml
# 修改 admin.jwt_secret

# 重启服务
docker-compose restart captchax
```

### 权限检查

```sql
-- 检查用户权限
SELECT usename, connlimit FROM pg_stat_activity;
SELECT * FROM information_schema.role_table_grants WHERE grantee = 'captcha_admin';

-- 检查数据库对象
SELECT * FROM information_schema.tables WHERE table_schema = 'public';
```

### 入侵检测

```bash
# 检查异常登录
grep "login failed" /var/log/captchax/*.log

# 检查异常 IP
netstat -an | grep ESTABLISHED | awk '{print $5}' | cut -d: -f1 | sort | uniq -c | sort -rn

# 检查异常请求
grep -i "union\|select\|drop\|delete" /var/log/captchax/*.log

# 检查文件完整性
sha256sum /opt/captchax/server
```

### 应急响应

```bash
# 1. 封禁攻击 IP
iptables -I INPUT -s <IP> -j DROP

# 2. 添加到黑名单
curl -X POST "http://localhost:8081/admin/api/blacklist" \
  -H "Authorization: Bearer <token>" \
  -d '{"ip": "<IP>", "reason": "attack", "expires_at": null}'

# 3. 启用紧急限流
# 编辑 config.yaml
captcha:
  max_attempts_per_ip: 1
  block_duration_minutes: 1440

# 4. 查看攻击详情
docker-compose logs --since 1h captchax | grep <IP>

# 5. 恢复服务
docker-compose restart captchax
```

### 合规检查

```sql
-- 检查敏感数据访问
SELECT * FROM captcha_logs WHERE created_at > NOW() - INTERVAL '24 hours';

-- 检查管理员操作
SELECT * FROM admin_logs WHERE action IN ('login', 'config_change') AND created_at > NOW() - INTERVAL '7 days';

-- 导出审计报告
COPY (SELECT * FROM captcha_logs WHERE created_at > NOW() - INTERVAL '30 days') TO '/tmp/audit_report.csv' WITH CSV HEADER;
```

---

## 联系方式

| 角色 | 职责 | 联系方式 |
|------|------|----------|
| 运维负责人 | 日常运维 | ops@example.com |
| 安全负责人 | 安全事件 | security@example.com |
| 技术支持 | 技术问题 | support@example.com |
| 值班电话 | 紧急事件 | 400-xxx-xxxx |
