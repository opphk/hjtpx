# CaptchaX 部署文档

## 环境要求

### 硬件要求

| 资源 | 最低配置 | 推荐配置 |
|------|----------|----------|
| CPU | 1核 | 2核+ |
| 内存 | 1GB | 2GB+ |
| 磁盘 | 10GB | 20GB+ |

### 软件要求

| 软件 | 版本要求 | 说明 |
|------|----------|------|
| Go | 1.21+ | 后端服务运行 |
| Redis | 6.0+ | 验证码缓存 |
| PostgreSQL | 13+ | 数据持久化 |
| Docker | 20.10+ | 容器化部署 |
| Docker Compose | 2.0+ | 服务编排 |

### 网络要求

- 开放端口：8080（API服务）、8081（管理后台）
- 允许外部访问静态资源
- 数据库和 Redis 仅内网访问

---

## Docker 部署（推荐）

### 方式一：使用 Docker Compose

#### 1. 创建 docker-compose.yml

```yaml
version: '3.8'

services:
  captchax:
    image: captchax/server:latest
    container_name: captchax
    ports:
      - "8080:8080"
      - "8081:8081"
    environment:
      - TZ=Asia/Shanghai
    volumes:
      - ./config:/app/config
      - captchax_data:/app/data
    depends_on:
      - redis
      - postgres
    restart: unless-stopped
    networks:
      - captchax-net

  redis:
    image: redis:7-alpine
    container_name: captchax-redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    command: redis-server --appendonly yes
    restart: unless-stopped
    networks:
      - captchax-net

  postgres:
    image: postgres:15-alpine
    container_name: captchax-postgres
    environment:
      - POSTGRES_USER=captcha_admin
      - POSTGRES_PASSWORD=captcha_pass_2026
      - POSTGRES_DB=captcha_db
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    restart: unless-stopped
    networks:
      - captchax-net

volumes:
  captchax_data:
  redis_data:
  postgres_data:

networks:
  captchax-net:
    driver: bridge
```

#### 2. 配置 config/config.yaml

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  host: "postgres"
  port: 5432
  user: "captcha_admin"
  password: "captcha_pass_2026"
  dbname: "captcha_db"
  sslmode: "disable"

redis:
  host: "redis"
  port: 6379
  password: ""
  db: 0

log:
  level: "info"
  format: "json"
  output: "stdout"

captcha:
  expire_minutes: 5
  max_attempts: 3
  width: 200
  height: 80
  slider_size: 50
  tolerance: 5

admin:
  jwt_secret: "change-this-secret-in-production"
  token_ttl_seconds: 86400
  cookie_name: "admin_token"
```

#### 3. 启动服务

```bash
# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f captchax
```

#### 4. 初始化数据库

```bash
# 进入容器执行迁移
docker exec -it captchax psql -h postgres -U captcha_admin -d captcha_db -f /app/migrations/001_initial_schema.sql
```

### 方式二：使用预构建镜像

```bash
# 拉取镜像
docker pull captchax/server:latest

# 创建配置目录
mkdir -p ~/captchax/config

# 创建配置文件
cat > ~/captchax/config/config.yaml << EOF
server:
  host: "0.0.0.0"
  port: 8080

database:
  host: "host.docker.internal"
  port: 5432
  user: "captcha_admin"
  password: "captcha_pass_2026"
  dbname: "captcha_db"
  sslmode: "disable"

redis:
  host: "host.docker.internal"
  port: 6379
  password: ""
  db: 0

captcha:
  expire_minutes: 5
  max_attempts: 3
  width: 200
  height: 80
  slider_size: 50
  tolerance: 5

admin:
  jwt_secret: "change-this-secret-in-production"
  token_ttl_seconds: 86400
  cookie_name: "admin_token"
EOF

# 启动服务
docker run -d \
  --name captchax \
  -p 8080:8080 \
  -p 8081:8081 \
  -v ~/captchax/config:/app/config \
  captchax/server:latest
```

---

## 手动部署

### 1. 安装 Go 环境

```bash
# 下载 Go
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# 配置环境变量
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
source ~/.bashrc

# 验证安装
go version
```

### 2. 安装 PostgreSQL

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y postgresql postgresql-contrib

# 启动服务
sudo systemctl start postgresql
sudo systemctl enable postgresql

# 创建数据库和用户
sudo -u postgres psql << EOF
CREATE USER captcha_admin WITH PASSWORD 'captcha_pass_2026';
CREATE DATABASE captcha_db OWNER captcha_admin;
EOF
```

### 3. 安装 Redis

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y redis-server

# 启动服务
sudo systemctl start redis-server
sudo systemctl enable redis-server

# 验证安装
redis-cli ping
```

### 4. 克隆项目

```bash
git clone https://github.com/your-org/captchax.git
cd captchax
```

### 5. 配置

```bash
# 编辑配置文件
vim config/config.yaml

# 确保配置正确
cat config/config.yaml
```

### 6. 运行数据库迁移

```bash
# 连接数据库
psql -h localhost -U captcha_admin -d captcha_db -f migrations/001_initial_schema.sql
```

### 7. 编译项目

```bash
# 下载依赖
go mod download

# 编译 API 服务
go build -o server ./cmd/server/main.go

# 编译管理后台服务
go build -o admin ./cmd/admin/main.go
```

### 8. 启动服务

```bash
# 启动 API 服务
./server &

# 启动管理后台
./admin &

# 或者使用 systemd 管理服务
sudo tee /etc/systemd/system/captchax.service << EOF
[Unit]
Description=CaptchaX Service
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/captchax
ExecStart=/opt/captchax/server
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable captchax
sudo systemctl start captchax
```

---

## 配置说明

### 服务配置

```yaml
server:
  host: "0.0.0.0"    # 监听地址
  port: 8080          # API 端口
  mode: "release"     # 运行模式：debug/release
```

### 数据库配置

```yaml
database:
  host: "localhost"      # 数据库地址
  port: 5432              # 数据库端口
  user: "captcha_admin"  # 用户名
  password: "xxx"         # 密码
  dbname: "captcha_db"   # 数据库名
  sslmode: "disable"     # SSL 模式
  max_open_conns: 25     # 最大连接数
  max_idle_conns: 5      # 空闲连接数
  conn_max_lifetime: 300 # 连接生命周期（秒）
```

### Redis 配置

```yaml
redis:
  host: "localhost"    # Redis 地址
  port: 6379           # 端口
  password: ""         # 密码（空表示无密码）
  db: 0                # 数据库编号
  pool_size: 10        # 连接池大小
```

### 验证码配置

```yaml
captcha:
  expire_minutes: 5    # 验证码有效期（分钟）
  max_attempts: 3      # 最大验证次数
  width: 200          # 图片宽度
  height: 80          # 图片高度
  slider_size: 50     # 滑块大小
  tolerance: 5        # 容差范围（像素）
```

### 管理后台配置

```yaml
admin:
  jwt_secret: "your-secret-key"    # JWT 密钥（生产环境必须修改）
  token_ttl_seconds: 86400          # Token 有效期（秒）
  cookie_name: "admin_token"        # Cookie 名称
```

---

## 反向代理配置

### Nginx 配置

```nginx
upstream captchax_backend {
    server 127.0.0.1:8080;
    keepalive 64;
}

server {
    listen 80;
    server_name captchax.example.com;

    # SSL 配置（生产环境必须启用）
    # ssl_certificate /etc/nginx/ssl/cert.pem;
    # ssl_certificate_key /etc/nginx/ssl/key.pem;

    client_max_body_size 10M;

    location / {
        proxy_pass http://captchax_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # 超时配置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    location /static/ {
        proxy_pass http://captchax_backend;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}
```

### 启用 HTTPS

```nginx
server {
    listen 443 ssl http2;
    server_name captchax.example.com;

    ssl_certificate /etc/nginx/ssl/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # 其他配置同上...
}
```

---

## 运维手册

### 日常维护

```bash
# 查看服务状态
systemctl status captchax

# 查看服务日志
journalctl -u captchax -f

# 重启服务
systemctl restart captchax

# 更新服务
docker pull captchax/server:latest
docker-compose down
docker-compose up -d
```

### 备份与恢复

#### 数据库备份

```bash
# 备份
pg_dump -h localhost -U captcha_admin captcha_db > backup_$(date +%Y%m%d).sql

# 恢复
psql -h localhost -U captcha_admin captcha_db < backup_20260514.sql
```

#### Redis 备份

```bash
# Redis 会自动持久化，备份 rdb 文件
cp /var/lib/redis/dump.rdb /backup/redis_$(date +%Y%m%d).rdb
```

### 日志管理

```bash
# 配置 logrotate
cat > /etc/logrotate.d/captchax << EOF
/var/log/captchax/*.log {
    daily
    rotate 14
    compress
    delaycompress
    missingok
    notifempty
    create 0640 www-data www-data
}
EOF
```

### 性能优化

```yaml
# config.yaml 优化配置
database:
  max_open_conns: 50
  max_idle_conns: 10

redis:
  pool_size: 20

captcha:
  expire_minutes: 5
  # 增加缓存提升性能
```

### 监控配置

```bash
# 添加健康检查端点监控
curl http://localhost:8080/health
```

### 故障排查

| 问题 | 可能原因 | 解决方案 |
|------|----------|----------|
| 服务无法启动 | 端口被占用 | 检查端口占用 `netstat -tlnp` |
| 数据库连接失败 | 配置错误/服务未启动 | 检查配置和数据库服务 |
| Redis 连接失败 | 配置错误 | 检查 Redis 配置 |
| 验证码生成失败 | 内存不足 | 增加服务器内存 |
| 图片显示异常 | 编码问题 | 检查 Base64 编码 |

---

## 安全加固

### 生产环境检查清单

1. 修改默认密码和密钥
2. 启用 HTTPS
3. 配置防火墙规则
4. 限制数据库访问
5. 启用日志审计
6. 定期备份数据
7. 更新安全补丁

### 防火墙配置

```bash
# 只开放必要端口
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp   # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw enable
```

### 数据库安全

```sql
-- 限制用户权限
GRANT CONNECT ON DATABASE captcha_db TO captcha_admin;
GRANT USAGE ON SCHEMA public TO captcha_admin;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO captcha_admin;
```
