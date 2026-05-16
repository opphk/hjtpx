# 部署指南

## 环境要求

### 硬件要求

| 配置 | 最低配置 | 推荐配置 | 生产环境配置 |
|------|---------|---------|-------------|
| CPU | 1核 | 2核 | 4核+ |
| 内存 | 1GB | 2GB | 4GB+ |
| 磁盘 | 10GB | 20GB | 50GB+ SSD |
| 网络 | 1Mbps | 5Mbps | 10Mbps+ |

### 软件要求

- Docker 20.10+
- Docker Compose 1.29+
- PostgreSQL 16+ (如果单独部署)
- Redis 7+ (如果单独部署)

## 快速部署

### 1. 克隆代码

```bash
git clone https://github.com/your-org/hjtpx.git
cd hjtpx
```

### 2. 配置环境变量

```bash
cp .env.example .env

# 编辑 .env 文件，修改以下配置
vim .env
```

必需的配置项：
- `POSTGRES_PASSWORD`: 数据库密码
- `JWT_SECRET`: JWT密钥（至少32字符）

### 3. 启动服务

使用 Docker Compose 启动所有服务：

```bash
# 启动所有服务（包括 Nginx）
docker-compose up -d

# 或仅启动核心服务
docker-compose up -d --profile core
```

### 4. 验证部署

访问应用验证服务是否正常运行：

```bash
# 检查健康状态
curl http://localhost:8080/health

# 查看日志
docker-compose logs -f app
```

访问以下地址：
- 应用：http://localhost:8080
- 管理后台：http://localhost:8080/admin
- Prometheus：http://localhost:9090
- Grafana：http://localhost:3000

## 生产环境部署

### 1. 服务器准备

```bash
# 更新系统
sudo apt update && sudo apt upgrade -y

# 安装必要工具
sudo apt install -y curl git vim ufw

# 配置防火墙
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

### 2. 安装 Docker

```bash
# 安装 Docker
curl -fsSL https://get.docker.com | sh

# 安装 Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/download/v2.20.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# 添加当前用户到 docker 组
sudo usermod -aG docker $USER
```

### 3. SSL 证书配置

使用 Let's Encrypt 获取免费证书：

```bash
# 安装 Certbot
sudo apt install -y certbot python3-certbot-nginx

# 获取证书（请先确保域名解析正确）
sudo certbot certonly --standalone -d your-domain.com -d admin.your-domain.com

# 复制证书到项目目录
sudo cp /etc/letsencrypt/live/your-domain.com/fullchain.pem ./ssl/server.crt
sudo cp /etc/letsencrypt/live/your-domain.com/privkey.pem ./ssl/server.key
```

### 4. 配置 Nginx

编辑 `nginx.conf` 文件：

```nginx
# HTTPS server
server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /etc/nginx/ssl/server.crt;
    ssl_certificate_key /etc/nginx/ssl/server.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;

    # ... 其他配置
}
```

### 5. 配置反向代理和负载均衡

```yaml
# docker-compose.yml 中添加
services:
  app:
    deploy:
      replicas: 3
    restart: always

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
```

### 6. 性能优化

#### 数据库优化

编辑 PostgreSQL 配置：

```bash
# 编辑 postgresql.conf
sudo vim /etc/postgresql/16/main/postgresql.conf

# 关键配置
max_connections = 200
shared_buffers = 256MB
effective_cache_size = 512MB
maintenance_work_mem = 64MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
max_worker_processes = 8

# 重启服务
sudo systemctl restart postgresql
```

#### Redis 优化

```bash
# 编辑 redis.conf
sudo vim /etc/redis/redis.conf

# 关键配置
maxmemory 512mb
maxmemory-policy allkeys-lru
appendonly yes
appendfsync everysec

# 重启服务
sudo systemctl restart redis-server
```

### 7. 监控配置

#### Prometheus

```bash
# 启动监控服务
docker-compose --profile with-monitoring up -d
```

#### Grafana

默认账号密码：`admin/admin`

首次登录后请修改密码。

### 8. 日志管理

配置 ELK Stack：

```bash
# 启动日志服务
docker-compose --profile with-logging up -d
```

访问 Kibana：http://localhost:5601

## 运维操作

### 部署更新

```bash
cd /opt/hjtpx
sudo ./scripts/deploy.sh
```

### 回滚版本

```bash
# 查看可用备份
sudo ./scripts/rollback.sh --list

# 回滚到指定版本
sudo ./scripts/rollback.sh 20240115_120000
```

### 健康检查

```bash
# 运行所有检查
sudo ./scripts/health_check.sh

# 仅检查应用
sudo ./scripts/health_check.sh --app

# 生成报告
sudo ./scripts/health_check.sh --report
```

### 日志查看

```bash
# 查看应用日志
docker-compose logs -f app

# 查看所有服务日志
docker-compose logs -f

# 查看特定时间段的日志
docker-compose logs --since "2024-01-01T00:00:00" app
```

### 备份

```bash
# 手动备份数据库
docker exec hjtpx-postgres pg_dump -U postgres verification > backup_$(date +%Y%m%d).sql

# 自动备份（已配置在 crontab）
sudo crontab -e
# 添加: 0 2 * * * /opt/hjtpx/scripts/backup.sh
```

## 故障排查

### 服务无法启动

```bash
# 检查 Docker 状态
docker-compose ps

# 查看详细日志
docker-compose logs app --tail=100

# 检查端口占用
sudo netstat -tlnp | grep 8080
```

### 数据库连接失败

```bash
# 检查 PostgreSQL 容器
docker exec hjtpx-postgres psql -U postgres -d verification -c "SELECT 1"

# 检查连接配置
docker-compose config | grep -A5 postgres
```

### 性能问题

```bash
# 查看资源使用
docker stats

# 检查慢查询
docker exec hjtpx-postgres psql -U postgres -d verification -c "SELECT * FROM pg_stat_activity WHERE state = 'active' AND query_start < NOW() - INTERVAL '5 minutes';"
```

## 安全加固

### 1. 修改默认密码

```bash
# 修改数据库密码
docker exec -it hjtpx-postgres psql -U postgres -d postgres -c "ALTER USER postgres WITH PASSWORD 'your_new_password';"

# 更新环境变量
vim .env
```

### 2. 配置防火墙

```bash
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw enable
```

### 3. 启用审计日志

```bash
# 启用 PostgreSQL 审计
docker exec hjtpx-postgres psql -U postgres -d verification -c "CREATE EXTENSION pgaudit;"
```

### 4. 配置 Fail2Ban

```bash
sudo apt install -y fail2ban

sudo cat > /etc/fail2ban/jail.local << EOF
[DEFAULT]
bantime = 3600
findtime = 600
maxretry = 5

[sshd]
enabled = true
port = 22
filter = sshd
logpath = /var/log/auth.log
EOF

sudo systemctl enable fail2ban
sudo systemctl start fail2ban
```

## 备份与恢复

### 备份策略

| 备份类型 | 频率 | 保留时间 | 存储位置 |
|---------|------|---------|---------|
| 全量备份 | 每日 | 30天 | 本地 + 远程 |
| 增量备份 | 每6小时 | 7天 | 本地 |
| 数据库备份 | 每小时 | 7天 | 本地 |
| 配置备份 | 每次部署 | 90天 | Git + 远程 |

### 恢复流程

1. 停止服务
2. 恢复数据库
3. 恢复 Redis 数据（可选）
4. 验证数据完整性
5. 重启服务

```bash
# 恢复数据库
docker exec -i hjtpx-postgres psql -U postgres -d verification < backup.sql

# 验证
docker exec hjtpx-postgres psql -U postgres -d verification -c "SELECT COUNT(*) FROM verification_logs;"
```

## 灾难恢复

### RTO (恢复时间目标)
- 最大可接受停机时间：1小时

### RPO (恢复点目标)
- 最大可接受数据丢失：24小时

### 恢复步骤

1. 评估损失
2. 选择恢复点
3. 启动新服务器
4. 恢复最新备份
5. 验证数据和功能
6. 切换流量

## 联系与支持

- 技术支持邮箱：support@example.com
- 问题反馈：https://github.com/your-org/hjtpx/issues
