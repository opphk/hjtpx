# 灾难恢复计划

## 概述

本文档定义了行为验证系统的灾难恢复策略、流程和职责，以确保在发生灾难时能够快速恢复服务并最小化数据丢失。

## 灾难恢复目标

### 恢复时间目标 (RTO)

- **最大可接受停机时间**：1小时
- **目标恢复时间**：30分钟

### 恢复点目标 (RPO)

- **最大可接受数据丢失**：24小时
- **目标数据丢失**：1小时

### 可用性目标

- **目标可用性**：99.9% (年度停机时间 < 8.76小时)

## 备份策略

### 备份类型

| 备份类型 | 频率 | 保留时间 | 存储位置 | 加密 |
|---------|------|---------|---------|------|
| 全量备份 | 每日 02:00 | 30天 | 本地 + S3 | 是 |
| 增量备份 | 每6小时 | 7天 | 本地 | 是 |
| 数据库备份 | 每小时 | 7天 | 本地 + S3 | 是 |
| Redis RDB | 每15分钟 | 3天 | 本地 | 是 |
| 配置文件备份 | 每次部署 | 90天 | Git | 否 |
| 日志归档 | 每日 | 90天 | S3 | 是 |

### 备份验证

- 每日自动验证备份完整性
- 每周执行恢复测试
- 每月进行完整灾难恢复演练

## 数据保护

### PostgreSQL 数据

#### 全量备份脚本

```bash
#!/bin/bash
# /opt/hjtpx/scripts/backup-postgres.sh

set -e

BACKUP_DIR="/opt/hjtpx/backups/postgres"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/postgres_full_${DATE}.sql.gz"

mkdir -p ${BACKUP_DIR}

docker exec hjtpx-postgres pg_dump -U postgres -d verification \
    | gzip > ${BACKUP_FILE}

# 上传到 S3
aws s3 cp ${BACKUP_FILE} s3://hjtpx-backups/postgres/

# 清理本地旧备份（保留7天）
find ${BACKUP_DIR} -name "postgres_full_*.sql.gz" -mtime +7 -delete

echo "PostgreSQL backup completed: ${BACKUP_FILE}"
```

#### 增量备份脚本

```bash
#!/bin/bash
# /opt/hjtpx/scripts/backup-postgres-incremental.sh

set -e

BACKUP_DIR="/opt/hjtpx/backups/postgres/incremental"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/postgres_incr_${DATE}.sql.gz"

mkdir -p ${BACKUP_DIR}

# 使用 pg_dump 的差异备份功能
docker exec hjtpx-postgres pg_dump -U postgres -d verification \
    --schema-only | gzip > ${BACKUP_DIR}/schema_${DATE}.sql.gz

docker exec hjtpx-postgres pg_dump -U postgres -d verification \
    -t verification_logs -t verifications \
    | gzip > ${BACKUP_FILE}

# 清理旧备份（保留3天）
find ${BACKUP_DIR} -name "postgres_incr_*.sql.gz" -mtime +3 -delete

echo "PostgreSQL incremental backup completed"
```

### Redis 数据

```bash
#!/bin/bash
# /opt/hjtpx/scripts/backup-redis.sh

set -e

BACKUP_DIR="/opt/hjtpx/backups/redis"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/redis_${DATE}.rdb"

mkdir -p ${BACKUP_DIR}

# 触发 Redis BGSAVE
docker exec hjtpx-redis redis-cli BGSAVE

# 等待保存完成
sleep 5

# 复制 RDB 文件
docker cp hjtpx-redis:/data/dump.rdb ${BACKUP_FILE}

# 压缩
gzip ${BACKUP_FILE}

# 上传到 S3
aws s3 cp ${BACKUP_FILE}.gz s3://hjtpx-backups/redis/

# 清理旧备份
find ${BACKUP_DIR} -name "redis_*.rdb.gz" -mtime +3 -delete

echo "Redis backup completed: ${BACKUP_FILE}.gz"
```

### 文件系统备份

```bash
#!/bin/bash
# /opt/hjtpx/scripts/backup-files.sh

set -e

BACKUP_DIR="/opt/hjtpx/backups/files"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p ${BACKUP_DIR}

# 备份配置文件
tar -czf ${BACKUP_DIR}/configs_${DATE}.tar.gz \
    /opt/hjtpx/docker-compose.yml \
    /opt/hjtpx/.env \
    /opt/hjtpx/nginx.conf \
    /opt/hjtpx/scripts/

# 上传到 S3
aws s3 cp ${BACKUP_DIR}/configs_${DATE}.tar.gz \
    s3://hjtpx-backups/configs/

# 清理旧备份（保留90天）
find ${BACKUP_DIR} -name "configs_*.tar.gz" -mtime +90 -delete

echo "Files backup completed"
```

## 恢复流程

### 数据库恢复

#### 恢复到本地

```bash
#!/bin/bash
# /opt/hjtpx/scripts/restore-postgres.sh

BACKUP_FILE=$1

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file>"
    exit 1
fi

echo "Stopping application..."
docker-compose stop app

echo "Restoring database..."
gunzip -c ${BACKUP_FILE} | docker exec -i hjtpx-postgres psql -U postgres -d verification

echo "Starting application..."
docker-compose start app

echo "Database restore completed"
```

#### 从 S3 恢复

```bash
#!/bin/bash
# /opt/hjtpx/scripts/restore-postgres-s3.sh

BACKUP_DATE=$1

if [ -z "$BACKUP_DATE" ]; then
    echo "Usage: $0 <YYYYMMDD>"
    exit 1
fi

# 下载备份
aws s3 cp s3://hjtpx-backups/postgres/postgres_full_${BACKUP_DATE}*.sql.gz /tmp/

# 停止应用
docker-compose stop app

# 恢复数据库
gunzip -c /tmp/postgres_full_${BACKUP_DATE}*.sql.gz | \
    docker exec -i hjtpx-postgres psql -U postgres -d verification

# 启动应用
docker-compose start app

echo "Database restore from S3 completed"
```

### Redis 恢复

```bash
#!/bin/bash
# /opt/hjtpx/scripts/restore-redis.sh

BACKUP_FILE=$1

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file>"
    exit 1
fi

# 停止 Redis
docker-compose stop redis

# 清理旧数据
docker exec hjtpx-redis rm -f /data/dump.rdb

# 复制新数据
gunzip -c ${BACKUP_FILE} > /tmp/dump.rdb
docker cp /tmp/dump.rdb hjtpx-redis:/data/dump.rdb

# 启动 Redis
docker-compose start redis

echo "Redis restore completed"
```

## 灾难场景

### 场景 1: 单服务器硬件故障

**触发条件**：服务器硬件（CPU、内存、磁盘）故障

**恢复步骤**：
1. 评估硬件状态
2. 启动备用服务器
3. 从最新备份恢复数据和配置
4. 验证服务功能
5. 切换流量

**预计恢复时间**：30分钟 - 1小时

### 场景 2: 数据库故障

**触发条件**：PostgreSQL 数据库损坏或不可恢复

**恢复步骤**：
1. 停止应用服务
2. 清理损坏的数据库
3. 从最新备份恢复数据库
4. 启动数据库服务
5. 启动应用服务
6. 验证数据完整性

**预计恢复时间**：15 - 30分钟

### 场景 3: 数据中心级故障

**触发条件**：整个数据中心不可用

**恢复步骤**：
1. 在备用数据中心启动基础设施
2. 从异地备份恢复所有数据
3. 重新配置网络和 DNS
4. 启动所有服务
5. 验证功能
6. 切换用户流量

**预计恢复时间**：2 - 4小时

### 场景 4: 安全事件（数据泄露）

**触发条件**：发现数据泄露或被攻击

**恢复步骤**：
1. 隔离受影响的系统
2. 评估影响范围
3. 通知相关人员
4. 从干净备份恢复
5. 更新所有凭据和密钥
6. 加强安全措施
7. 恢复服务
8. 事后分析

**预计恢复时间**：4 - 8小时

### 场景 5: 代码/配置错误

**触发条件**：错误的配置或代码部署导致服务不可用

**恢复步骤**：
1. 立即停止错误版本的部署
2. 从版本控制系统恢复上一稳定版本
3. 回滚配置变更
4. 重新部署
5. 验证功能
6. 如果需要，从备份恢复数据

**预计恢复时间**：10 - 30分钟

## 应急联系人

| 角色 | 姓名 | 电话 | 邮箱 |
|------|------|------|------|
| 技术负责人 | | | |
| DBA 负责人 | | | |
| 安全负责人 | | | |
| 运维负责人 | | | |
| 外部支持 | AWS 支持 | | |

## 恢复验证

### 定期演练

- **频率**：每季度至少一次
- **参与人员**：运维、DBA、开发
- **验证内容**：
  - 备份完整性
  - 恢复流程
  - 恢复时间
  - 人员响应能力

### 演练报告

每次演练后需完成以下报告：
1. 演练日期和时间
2. 参与人员
3. 演练场景
4. 发现的问题
5. 改进措施
6. 下次演练计划

## 文档更新

本计划应在下述情况下更新：
- 系统架构变更
- 新备份策略实施
- 人员变动
- 每次演练后
- 实际灾难恢复后

**最后更新日期**：_______________
**下次审查日期**：_______________
**文档版本**：_______________

## 附录

### A. 关键命令参考

```bash
# 检查服务状态
docker-compose ps

# 查看日志
docker-compose logs -f app

# 检查数据库连接
docker exec hjtpx-postgres pg_isready -U postgres

# 检查 Redis 连接
docker exec hjtpx-redis redis-cli ping

# 查看资源使用
docker stats

# 查看磁盘空间
df -h

# 查看内存使用
free -h
```

### B. 监控指标

恢复后应检查以下指标：
- CPU 使用率 < 70%
- 内存使用率 < 80%
- 磁盘使用率 < 70%
- 数据库连接数正常
- Redis 内存使用正常
- API 响应时间 < 1秒
- 错误率 < 1%

### C. 回滚检查清单

- [ ] 旧版本代码已部署
- [ ] 旧版本配置已恢复
- [ ] 数据库已恢复到旧版本
- [ ] 服务已启动
- [ ] 健康检查通过
- [ ] 功能验证通过
- [ ] 用户流量正常
