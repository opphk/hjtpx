# CaptchaX 部署文档

## 目录

- [环境要求](#环境要求)
- [Docker 部署](#docker-部署)
- [Kubernetes 部署](#kubernetes-部署)
- [手动部署](#手动部署)
- [配置说明](#配置说明)
- [反向代理配置](#反向代理配置)
- [安全加固](#安全加固)

---

## 环境要求

### 硬件要求

| 资源 | 最低配置 | 推荐配置 | 生产环境配置 |
|------|----------|----------|--------------|
| CPU | 1核 | 2核+ | 4核+ |
| 内存 | 1GB | 2GB+ | 4GB+ |
| 磁盘 | 10GB | 20GB+ | 50GB+ SSD |

### 软件要求

| 软件 | 版本要求 | 说明 |
|------|----------|------|
| Go | 1.21+ | 后端服务运行 |
| Redis | 6.0+ | 验证码缓存（推荐 7.0+） |
| PostgreSQL | 13+ | 数据持久化（推荐 15+） |
| Docker | 20.10+ | 容器化部署 |
| Docker Compose | 2.0+ | 服务编排 |

### 网络要求

| 端口 | 协议 | 说明 |
|------|------|------|
| 8080 | TCP | API 服务端口 |
| 8081 | TCP | 管理后台端口 |
| 6379 | TCP | Redis 端口（仅内网） |
| 5432 | TCP | PostgreSQL 端口（仅内网） |

---

## Docker 部署

### 方式一：使用 Docker Compose（推荐）

#### 1. 创建项目目录

```bash
mkdir -p ~/captchax && cd ~/captchax
```

#### 2. 创建配置文件

```bash
mkdir -p config data backups
chmod 755 config data backups
```

#### 3. 创建 config/config.yaml

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"

database:
  host: "postgres"
  port: 5432
  user: "captcha_admin"
  password: "captcha_pass_2026"
  dbname: "captcha_db"
  sslmode: "disable"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 300

redis:
  host: "redis"
  port: 6379
  password: ""
  db: 0
  pool_size: 10

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

#### 4. 创建 docker-compose.yml

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
      - ./config:/app/config:ro
      - captchax_data:/app/data
    depends_on:
      redis:
        condition: service_healthy
      postgres:
        condition: service_healthy
    restart: unless-stopped
    networks:
      - captchax-net
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  redis:
    image: redis:7-alpine
    container_name: captchax-redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    command: redis-server --appendonly yes --maxmemory 512mb --maxmemory-policy allkeys-lru
    restart: unless-stopped
    networks:
      - captchax-net
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 3

  postgres:
    image: postgres:15-alpine
    container_name: captchax-postgres
    environment:
      - POSTGRES_USER=captcha_admin
      - POSTGRES_PASSWORD=captcha_pass_2026
      - POSTGRES_DB=captcha_db
      - TZ=Asia/Shanghai
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d:ro
    restart: unless-stopped
    networks:
      - captchax-net
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U captcha_admin -d captcha_db"]
      interval: 10s
      timeout: 5s
      retries: 3

volumes:
  captchax_data:
  redis_data:
  postgres_data:

networks:
  captchax-net:
    driver: bridge
```

#### 5. 启动服务

```bash
# 拉取最新镜像
docker-compose pull

# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f captchax
```

#### 6. 初始化数据库

```bash
# 等待 postgres 就绪后执行迁移
sleep 10
docker exec -it captchax psql -h postgres -U captcha_admin -d captcha_db -f /docker-entrypoint-initdb.d/001_initial_schema.sql
```

#### 7. 验证部署

```bash
# 检查 API 服务
curl http://localhost:8080/health

# 检查管理后台
curl http://localhost:8081/health
```

### 方式二：使用预构建镜像

```bash
# 拉取镜像
docker pull captchax/server:latest

# 创建配置目录
mkdir -p ~/captchax/config

# 创建配置文件
cat > ~/captchax/config/config.yaml << 'EOF'
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
  --restart unless-stopped \
  -p 8080:8080 \
  -p 8081:8081 \
  -v ~/captchax/config:/app/config:ro \
  -v captchax_data:/app/data \
  captchax/server:latest
```

### 方式三：自构建镜像

```bash
# 克隆项目
git clone https://github.com/your-org/captchax.git
cd captchax

# 构建镜像
docker build -t captchax/server:latest .

# 或使用多阶段构建（生产环境推荐）
docker build -t captchax/server:latest -f Dockerfile.prod .
```

---

## Kubernetes 部署

### 前置要求

- Kubernetes 1.21+
- kubectl 配置完成
- StorageClass 已配置（用于持久化存储）

### 方式一：使用 Helm（推荐）

#### 1. 添加 Helm 仓库

```bash
helm repo add captchax https://charts.captchax.io
helm repo update
```

#### 2. 配置 values.yaml

```yaml
# values.yaml

image:
  repository: captchax/server
  tag: latest
  pullPolicy: IfNotPresent

replicaCount: 2

service:
  type: ClusterIP
  apiPort: 8080
  adminPort: 8081

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: captchax.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: captchax-tls
      hosts:
        - captchax.example.com

resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80

config:
  server:
    host: "0.0.0.0"
    port: 8080
    mode: "release"

  database:
    host: "postgres-postgresql.default.svc.cluster.local"
    port: 5432
    user: "captcha_admin"
    password: "captcha_pass_2026"
    dbname: "captcha_db"
    sslmode: "disable"
    max_open_conns: 25
    max_idle_conns: 5

  redis:
    host: "redis-master.default.svc.cluster.local"
    port: 6379
    password: ""
    db: 0
    pool_size: 10

  log:
    level: "info"
    format: "json"

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

persistence:
  enabled: true
  storageClass: "standard"
  accessMode: ReadWriteOnce
  size: 10Gi

redis:
  enabled: true
  architecture: standalone
  auth:
    enabled: false
  master:
    persistence:
      enabled: true
      storageClass: "standard"
      size: 5Gi

postgresql:
  enabled: true
  auth:
    username: captcha_admin
    password: captcha_pass_2026
    database: captcha_db
  primary:
    persistence:
      enabled: true
      storageClass: "standard"
      size: 20Gi
```

#### 3. 安装

```bash
# 安装
helm install captchax captchax/captchax -f values.yaml -n captchax --create-namespace

# 升级
helm upgrade captchax captchax/captchax -f values.yaml -n captchax

# 查看状态
kubectl get pods -n captchax

# 查看日志
kubectl logs -n captchax -l app=captchax -f
```

### 方式二：手动部署 YAML

#### 1. 创建 Namespace

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: captchax
  labels:
    app: captchax
```

```bash
kubectl apply -f namespace.yaml
```

#### 2. 创建 ConfigMap

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: captchax-config
  namespace: captchax
data:
  config.yaml: |
    server:
      host: "0.0.0.0"
      port: 8080
      mode: "release"

    database:
      host: "postgres.default.svc.cluster.local"
      port: 5432
      user: "captcha_admin"
      password: "captcha_pass_2026"
      dbname: "captcha_db"
      sslmode: "disable"
      max_open_conns: 25
      max_idle_conns: 5

    redis:
      host: "redis.default.svc.cluster.local"
      port: 6379
      password: ""
      db: 0
      pool_size: 10

    log:
      level: "info"
      format: "json"

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
```

#### 3. 创建 Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: captchax
  namespace: captchax
  labels:
    app: captchax
spec:
  replicas: 2
  selector:
    matchLabels:
      app: captchax
  template:
    metadata:
      labels:
        app: captchax
    spec:
      containers:
        - name: captchax
          image: captchax/server:latest
          imagePullPolicy: Always
          ports:
            - name: api
              containerPort: 8080
              protocol: TCP
            - name: admin
              containerPort: 8081
              protocol: TCP
          env:
            - name: TZ
              value: "Asia/Shanghai"
          volumeMounts:
            - name: config
              mountPath: /app/config
              readOnly: true
            - name: data
              mountPath: /app/data
          resources:
            requests:
              cpu: 500m
              memory: 512Mi
            limits:
              cpu: 1000m
              memory: 1Gi
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 30
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 5
      volumes:
        - name: config
          configMap:
            name: captchax-config
        - name: data
          emptyDir: {}
```

#### 4. 创建 Service

```yaml
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: captchax
  namespace: captchax
spec:
  type: ClusterIP
  ports:
    - name: api
      port: 8080
      targetPort: 8080
      protocol: TCP
    - name: admin
      port: 8081
      targetPort: 8081
      protocol: TCP
  selector:
    app: captchax
```

#### 5. 创建 HPA（水平自动扩缩容）

```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: captchax-hpa
  namespace: captchax
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: captchax
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

#### 6. 创建 Ingress

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: captchax-ingress
  namespace: captchax
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "10m"
    nginx.ingress.kubernetes.io/proxy-connect-timeout: "60"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "60"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "60"
spec:
  ingressClassName: nginx
  rules:
    - host: captchax.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: captchax
                port:
                  number: 8080
  tls:
    - hosts:
        - captchax.example.com
      secretName: captchax-tls
```

#### 7. 部署所有资源

```bash
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f hpa.yaml
kubectl apply -f ingress.yaml

# 查看部署状态
kubectl get all -n captchax

# 查看 Pod 日志
kubectl logs -n captchax -l app=captchax -f
```

### 方式三：使用 PostgreSQL 和 Redis Operator

```yaml
# redis.yaml
apiVersion: redis.redis.opstreelabs.in/v1beta1
kind: RedisCluster
metadata:
  name: captchax-redis
  namespace: captchax
spec:
  clusterSize: 3
  persistence:
    enabled: true
    storageClass: "standard"
    size: 5Gi
```

```yaml
# postgres.yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: captchax-postgres
  namespace: captchax
spec:
  instances: 3
  storage:
    size: 20Gi
    storageClass: "standard"
  resources:
    requests:
      cpu: 500m
      memory: 1Gi
    limits:
      cpu: 2000m
      memory: 2Gi
```

---

## 手动部署

### 1. 安装 Go 环境

```bash
# 下载 Go
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# 配置环境变量
cat >> ~/.bashrc << 'EOF'
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
EOF

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
sudo -u postgres psql << 'EOF'
CREATE USER captcha_admin WITH PASSWORD 'captcha_pass_2026';
CREATE DATABASE captcha_db OWNER captcha_admin;
GRANT ALL PRIVILEGES ON DATABASE captcha_db TO captcha_admin;
EOF

# 验证安装
sudo -u postgres psql -c "SELECT version();"
```

### 3. 安装 Redis

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y redis-server

# 启动服务
sudo systemctl start redis-server
sudo systemctl enable redis-server

# 配置内存限制
sudo sed -i 's/^# maxmemory <bytes>/maxmemory 512mb/' /etc/redis/redis.conf
sudo sed -i 's/^# maxmemory-policy.*/maxmemory-policy allkeys-lru/' /etc/redis/redis.conf

# 重启服务
sudo systemctl restart redis-server

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

# 执行后续优化迁移
for f in migrations/0*.sql; do
    psql -h localhost -U captcha_admin -d captcha_db -f "$f"
done
```

### 7. 编译项目

```bash
# 下载依赖
go mod download

# 编译 API 服务
CGO_ENABLED=0 go build -ldflags="-s -w" -o server ./cmd/server/main.go

# 编译管理后台服务
CGO_ENABLED=0 go build -ldflags="-s -w" -o admin ./cmd/admin/main.go

# 验证编译结果
ls -la server admin
```

### 8. 启动服务

#### 开发环境

```bash
# 启动 API 服务
./server

# 新终端窗口 - 启动管理后台
./admin
```

#### 生产环境（systemd）

```bash
# 创建服务用户
sudo useradd -r -s /sbin/nologin captchax

# 创建目录
sudo mkdir -p /opt/captchax/{config,data,logs}
sudo chown -R captchax:captchax /opt/captchax

# 复制文件
sudo cp server /opt/captchax/
sudo cp admin /opt/captchax/
sudo cp -r config/* /opt/captchax/config/

# 创建 systemd 服务文件
sudo tee /etc/systemd/system/captchax.service << 'EOF'
[Unit]
Description=CaptchaX Service
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=captchax
Group=captchax
WorkingDirectory=/opt/captchax
ExecStart=/opt/captchax/server
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=captchax

# 环境变量
Environment="TZ=Asia/Shanghai"

[Install]
WantedBy=multi-user.target
EOF

# 重新加载 systemd
sudo systemctl daemon-reload

# 启用并启动服务
sudo systemctl enable captchax
sudo systemctl start captchax

# 查看服务状态
sudo systemctl status captchax
```

---

## 配置说明

### 服务配置

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| server.host | 0.0.0.0 | 监听地址 |
| server.port | 8080 | API 端口 |
| server.mode | release | 运行模式 (debug/release) |

### 数据库配置

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| database.host | localhost | 数据库地址 |
| database.port | 5432 | 数据库端口 |
| database.user | - | 用户名 |
| database.password | - | 密码 |
| database.dbname | - | 数据库名 |
| database.sslmode | disable | SSL 模式 |
| database.max_open_conns | 25 | 最大连接数 |
| database.max_idle_conns | 5 | 空闲连接数 |
| database.conn_max_lifetime | 300 | 连接生命周期（秒） |

### Redis 配置

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| redis.host | localhost | Redis 地址 |
| redis.port | 6379 | 端口 |
| redis.password | - | 密码 |
| redis.db | 0 | 数据库编号 |
| redis.pool_size | 10 | 连接池大小 |

### 验证码配置

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| captcha.expire_minutes | 5 | 验证码有效期（分钟） |
| captcha.max_attempts | 3 | 最大验证次数 |
| captcha.width | 200 | 图片宽度 |
| captcha.height | 80 | 图片高度 |
| captcha.slider_size | 50 | 滑块大小 |
| captcha.tolerance | 5 | 容差范围（像素） |

### 管理后台配置

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| admin.jwt_secret | - | JWT 密钥（必须修改） |
| admin.token_ttl_seconds | 86400 | Token 有效期（秒） |
| admin.cookie_name | admin_token | Cookie 名称 |

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

    client_max_body_size 10M;

    # Gzip 压缩
    gzip on;
    gzip_types text/plain application/json image/png;
    gzip_min_length 1000;

    location / {
        proxy_pass http://captchax_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    location /static/ {
        proxy_pass http://captchax_backend;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }

    # WebSocket 支持（如需要）
    location /ws {
        proxy_pass http://captchax_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### 启用 HTTPS

```nginx
server {
    listen 443 ssl http2;
    server_name captchax.example.com;

    # SSL 证书
    ssl_certificate /etc/nginx/ssl/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/privkey.pem;

    # SSL 配置
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # 安全头
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # 其他配置同上...
}

# HTTP 重定向到 HTTPS
server {
    listen 80;
    server_name captchax.example.com;
    return 301 https://$server_name$request_uri;
}
```

---

## 安全加固

### 生产环境检查清单

| 检查项 | 说明 | 状态 |
|--------|------|------|
| 修改默认密码 | 修改 admin 和数据库密码 | ☐ |
| JWT 密钥 | 使用随机字符串替换默认密钥 | ☐ |
| HTTPS | 生产环境必须启用 HTTPS | ☐ |
| 防火墙 | 仅开放必要端口 | ☐ |
| 数据库权限 | 限制数据库用户权限 | ☐ |
| 日志审计 | 启用详细日志记录 | ☐ |
| 定期备份 | 配置自动备份策略 | ☐ |
| 安全更新 | 定期更新系统和依赖 | ☐ |

### 防火墙配置

```bash
# 只开放必要端口
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw --force enable
```

### 数据库安全

```sql
-- 创建受限用户
CREATE USER captcha_app WITH PASSWORD 'strong_password';
GRANT CONNECT ON DATABASE captcha_db TO captcha_app;
GRANT USAGE ON SCHEMA public TO captcha_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO captcha_app;

-- 限制连接来源
ALTER USER captcha_app CONNECTION LIMIT 10;
```

### 备份策略

```bash
# cron 备份脚本
cat > /etc/cron.daily/captchax-backup << 'EOF'
#!/bin/bash
DATE=$(date +%Y%m%d)

# 备份数据库
pg_dump -h localhost -U captcha_admin captcha_db | gzip > /var/backups/captcha_db_$DATE.sql.gz

# 备份 Redis
redis-cli SAVE
cp /var/lib/redis/dump.rdb /var/backups/redis_$DATE.rdb

# 保留最近 30 天备份
find /var/backups -name "captcha_*" -mtime +30 -delete
EOF

chmod +x /etc/cron.daily/captchax-backup
```

### 监控配置

```yaml
# Prometheus 配置
- job_name: 'captchax'
  static_configs:
    - targets: ['localhost:8080']
  metrics_path: '/metrics'
```

```bash
# 检查健康状态
curl http://localhost:8080/health

# 检查指标
curl http://localhost:8080/metrics
```

---

## 故障排查

| 问题 | 可能原因 | 解决方案 |
|------|----------|----------|
| 服务无法启动 | 端口被占用 | `netstat -tlnp \| grep 8080` |
| 数据库连接失败 | 配置错误/服务未启动 | 检查配置和数据库服务 |
| Redis 连接失败 | 配置错误 | 检查 Redis 配置 |
| 验证码生成失败 | 内存不足 | 增加服务器内存 |
| 图片显示异常 | 编码问题 | 检查 Base64 编码 |
| K8s Pod 无法启动 | 镜像拉取失败 | 检查镜像地址和凭证 |
| K8s Pod 无法挂载卷 | StorageClass 不存在 | 检查存储配置 |
