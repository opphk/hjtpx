# CaptchaX 常见问题

## 目录

- [部署问题](#部署问题)
- [使用问题](#使用问题)
- [故障排除](#故障排除)
- [性能优化](#性能优化)
- [安全相关](#安全相关)

---

## 部署问题

### Q1: Docker 部署时数据库连接失败？

**问题描述**：容器启动后无法连接到 PostgreSQL。

**可能原因**：
- PostgreSQL 容器未启动
- 网络配置问题
- 密码错误
- 配置文件中主机地址错误

**解决方案**：

1. 检查 PostgreSQL 容器状态：
```bash
docker-compose ps postgres
docker-compose logs postgres
```

2. 确认网络连通性：
```bash
docker exec -it captchax ping postgres
```

3. 检查配置文件中的数据库地址：
```yaml
database:
  host: "postgres"  # Docker Compose 服务名，不是 localhost
  port: 5432
  user: "captcha_admin"
  password: "captcha_pass_2026"
  dbname: "captcha_db"
```

4. 如果使用宿主机数据库，改为：
```yaml
database:
  host: "host.docker.internal"  # macOS/Windows
  # 或 Linux 使用宿主机IP
```

---

### Q2: Redis 连接失败？

**问题描述**：服务启动后无法连接 Redis。

**解决方案**：

1. 检查 Redis 容器状态：
```bash
docker-compose ps redis
docker-compose logs redis
```

2. 测试 Redis 连接：
```bash
docker exec -it captchax redis-cli -h redis ping
```

3. 检查配置：
```yaml
redis:
  host: "redis"  # Docker Compose 服务名
  port: 6379
  password: ""   # 如果有密码需要填写
```

---

### Q3: 端口被占用怎么办？

**问题描述**：启动服务时提示端口已被占用。

**解决方案**：

1. 查找占用端口的进程：
```bash
# Linux
sudo netstat -tlnp | grep 8080
sudo lsof -i :8080

# macOS
lsof -i :8080
```

2. 停止占用进程或修改配置使用其他端口：
```yaml
server:
  port: 8082  # 改用其他端口
```

---

### Q4: 如何迁移数据到新服务器？

**步骤**：

1. 备份原服务器数据：
```bash
# 备份 PostgreSQL
pg_dump -h old-server -U captcha_admin captcha_db > captcha_backup.sql

# 备份 Redis
redis-cli -h old-server CONFIG GET dir  # 查看 RDB 目录
cp /var/lib/redis/dump.rdb /backup/
```

2. 在新服务器安装相同版本
3. 导入数据：
```bash
# 恢复 PostgreSQL
psql -h new-server -U captcha_admin -d captcha_db < captcha_backup.sql

# 恢复 Redis
cp dump.rdb /var/lib/redis/
systemctl restart redis
```

---

## 使用问题

### Q5: 验证码图片显示异常？

**问题描述**：验证码图片无法显示或显示为空白。

**可能原因**：
- Base64 编码问题
- 浏览器不支持
- 图片资源加载失败

**解决方案**：

1. 检查浏览器控制台错误
2. 确认图片格式正确：
```javascript
// 响应示例
{
  "background_b64": "data:image/png;base64,iVBORw0KGgo..."
}
```

3. 检查网络请求是否正常
4. 确认服务器返回的是完整的 Base64 字符串

---

### Q6: 验证总是失败？

**问题描述**：用户完成验证但系统返回失败。

**可能原因**：
- 验证码已过期（默认5分钟）
- 验证次数超限
- 坐标容差设置过小
- IP 被封禁

**解决方案**：

1. 检查验证码有效期配置：
```yaml
captcha:
  expire_minutes: 5    # 增大有效期
  max_attempts: 3      # 增大尝试次数
  tolerance: 5         # 增大容差范围
```

2. 检查黑名单：
```bash
curl http://localhost:8080/admin/api/blacklist?ip=用户IP
```

3. 查看验证日志确认失败原因

---

### Q7: 如何设置只允许某些域名使用？

**解决方案**：

1. 使用白名单功能，在管理后台添加允许的域名
2. 在应用端验证请求来源：
```javascript
const allowedOrigins = ['https://example.com'];
const origin = request.headers.origin;

if (!allowedOrigins.includes(origin)) {
  return res.status(403).json({ error: '域名未授权' });
}
```

3. 配置 CORS 中间件限制来源

---

### Q8: 如何实现无感知验证？

**方案一：预加载验证**

```javascript
// 页面加载时预加载但不显示
const captcha = new CaptchaX({
  appId: 'my-app',
  serverUrl: 'https://captchax.example.com',
  container: '#captcha-container',
  autoRender: false,  // 不自动渲染
  onReady: function() {
    // 缓存验证码数据
    sessionStorage.setItem('captchaReady', 'true');
  }
});

function showAndVerify() {
  if (sessionStorage.getItem('captchaReady')) {
    captcha.render();
  }
}
```

**方案二：验证成功后缓存 Token**

```javascript
// 验证成功后缓存 token，有效期30分钟
const captchaToken = localStorage.getItem('captchaToken');
const tokenTime = localStorage.getItem('captchaTokenTime');

if (captchaToken && tokenTime && (Date.now() - tokenTime < 30*60*1000)) {
  // 使用缓存的 token
  submitForm(captchaToken);
} else {
  // 重新验证
  captcha.verify().then(result => {
    localStorage.setItem('captchaToken', result.token);
    localStorage.setItem('captchaTokenTime', Date.now());
    submitForm(result.token);
  });
}
```

---

## 故障排除

### Q9: 服务启动正常但无法访问？

**排查步骤**：

1. 检查服务状态：
```bash
systemctl status captchax
journalctl -u captchax -n 50
```

2. 检查端口监听：
```bash
netstat -tlnp | grep 8080
```

3. 检查防火墙：
```bash
sudo ufw status
sudo iptables -L -n
```

4. 测试本地访问：
```bash
curl http://127.0.0.1:8080/health
```

---

### Q10: 如何查看详细日志？

**解决方案**：

1. 调整日志级别：
```yaml
log:
  level: "debug"  # debug/info/warn/error
  format: "json"
```

2. Docker 环境查看日志：
```bash
docker-compose logs -f captchax
```

3. 手动部署查看日志：
```bash
journalctl -u captchax -f
```

4. 分析日志关键词：
- `verify_failed`：验证失败
- `rate_limited`：触发限流
- `blacklisted`：命中黑名单

---

### Q11: 数据库迁移失败？

**问题描述**：运行迁移脚本时报错。

**解决方案**：

1. 检查数据库连接：
```bash
psql -h localhost -U captcha_admin -d captcha_db -c "SELECT 1;"
```

2. 检查迁移脚本语法：
```bash
cat migrations/001_initial_schema.sql | head -20
```

3. 如果表已存在，跳过创建：
```sql
-- 确保表不存在时再创建
CREATE TABLE IF NOT EXISTS ...
```

4. 手动执行：
```bash
psql -h localhost -U captcha_admin -d captcha_db -f migrations/001_initial_schema.sql
```

---

### Q12: 内存使用过高？

**可能原因**：
- Redis 连接未释放
- 数据库连接池过大
- 并发请求过多

**解决方案**：

1. 调整连接池大小：
```yaml
database:
  max_open_conns: 10
  max_idle_conns: 3

redis:
  pool_size: 5
```

2. 添加定时任务清理缓存：
```bash
# Redis 内存优化
redis-cli CONFIG SET maxmemory 512mb
redis-cli CONFIG SET maxmemory-policy allkeys-lru
```

3. 限制并发：
```nginx
upstream backend {
    server 127.0.0.1:8080;
    keepalive 32;
}

limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
```

---

## 性能优化

### Q13: 如何提高并发处理能力？

**方案一：水平扩展**

```yaml
# docker-compose.yml
services:
  captchax:
    deploy:
      replicas: 3  # 运行多个实例
```

**方案二：负载均衡**

```nginx
upstream captchax_backend {
    least_conn;  # 最少连接优先
    server 127.0.0.1:8080 weight=2;
    server 127.0.0.1:8082 weight=1;
    keepalive 64;
}
```

**方案三：启用 Redis 缓存**

```yaml
redis:
  pool_size: 50  # 增大连接池
  read_timeout: 3s
  write_timeout: 3s
```

---

### Q14: 如何减少响应时间？

1. **启用 Gzip 压缩**：
```nginx
gzip on;
gzip_types text/plain application/json image/png;
gzip_min_length 1000;
```

2. **使用 CDN 加速静态资源**：
```html
<script src="https://cdn.example.com/captchax.js"></script>
```

3. **减少图片大小**：
```yaml
captcha:
  width: 180  # 适当减小图片尺寸
  height: 60
```

4. **数据库优化**：
```sql
-- 添加索引
CREATE INDEX idx_captcha_logs_ip ON captcha_logs(ip);
CREATE INDEX idx_captcha_logs_created ON captcha_logs(created_at);
```

---

### Q15: 高峰期服务变慢怎么办？

**应急措施**：

1. 临时扩容：
```bash
docker-compose up -d --scale captchax=3
```

2. 启用限流保护：
```yaml
captcha:
  max_attempts: 1  # 严格限制尝试次数
```

3. 启用降级策略：
```yaml
# 返回简单验证码（不生成图片）
captcha:
  fallback_mode: "simple"
```

**长期方案**：
- 使用 Redis Cluster
- 接入 API 网关
- 部署多机房容灾

---

## 安全相关

### Q16: 如何防止暴力破解？

**方案一：IP 限流**

```yaml
captcha:
  max_attempts_per_ip: 5  # 每IP每小时5次
  block_duration_minutes: 60
```

**方案二：验证码挑战**

1. 首次失败后添加验证码
2. 多次失败后增加验证码难度
3. 连续失败锁定账号

**方案三：行为分析**

CaptchaX 内置风险评分引擎，会根据以下维度评分：
- 请求频率
- IP 信誉
- 用户行为轨迹
- 设备指纹

---

### Q17: 如何防止验证码被爬取？

**方案一：隐藏验证端点**

```nginx
location /api/v1/captcha {
    # 只允许特定来源
    valid_referers ~* example.com;
    if ($invalid_referer) {
        return 403;
    }
}
```

**方案二：动态生成验证**

```javascript
// 不暴露固定接口地址
const apiUrl = await fetch('/api/config').then(r => r.json()).then(d => d.captchaUrl);
```

**方案三：绑定 Session**

```go
// 后端验证时检查 Session
session := sessions.Get(c, "captcha-session")
if session["captcha_verified"] != "true" {
    return c.JSON(403, "请先完成验证")
}
```

---

### Q18: 如何审计操作日志？

**日志记录**：

CaptchaX 自动记录以下操作：
- 验证码生成请求
- 验证结果
- 管理后台登录
- 配置变更
- 黑白名单操作

**查看日志**：

```bash
# 查看验证日志
curl http://localhost:8081/admin/api/stats

# 导出详细日志
psql -h localhost -U captcha_admin -d captcha_db -c \
  "COPY (SELECT * FROM captcha_logs WHERE created_at > NOW() - INTERVAL '7 days') TO STDOUT WITH CSV HEADER" > captcha_logs.csv
```

**日志分析**：

```bash
# 分析异常 IP
psql -h localhost -U captcha_admin -d captcha_db -c \
  "SELECT ip, COUNT(*) as cnt FROM captcha_logs WHERE result = false GROUP BY ip ORDER BY cnt DESC LIMIT 10;"
```

---

### Q19: JWT Token 失效了怎么办？

**问题描述**：管理后台 Token 过期需要重新登录。

**解决方案**：

1. 重新登录获取新 Token
2. Token 有效期配置：
```yaml
admin:
  token_ttl_seconds: 86400  # 默认24小时，可增大
```

3. 前端自动刷新 Token：
```javascript
// 检测 401 后自动跳转登录
axios.interceptors.response.use(
  response => response,
  error => {
    if (error.response && error.response.status === 401) {
      window.location.href = '/admin/login';
    }
    return Promise.reject(error);
  }
);
```

---

### Q20: 如何备份和恢复配置？

**备份**：

```bash
# 备份数据库配置
pg_dump -h localhost -U captcha_admin -d captcha_db \
  -t captcha_config -t whitelist -t blacklist > config_backup.sql

# 备份配置文件
cp config/config.yaml config/config.yaml.bak
```

**恢复**：

```bash
# 恢复数据库
psql -h localhost -U captcha_admin -d captcha_db < config_backup.sql

# 恢复配置文件
cp config/config.yaml.bak config/config.yaml
systemctl restart captchax
```

---

## 联系支持

如果以上问题无法解决，请通过以下方式获取帮助：

- 提交 GitHub Issue
- 发送邮件至 support@example.com
- 加入技术交流群

提供以下信息有助于快速定位问题：
1. 部署环境（Docker/手动）
2. 配置文件（脱敏后）
3. 错误日志
4. 复现步骤
