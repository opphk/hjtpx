# 智能仪表盘 v15.0 功能文档

## 概述

智能仪表盘 v15.0 是 HJTPX 管理后台的核心监控模块，提供实时数据可视化、智能告警系统和自定义报表功能。

## 主要功能

### 1. 实时验证流量大屏

- **秒级刷新**：支持每秒更新实时监控数据
- **WebSocket 实时推送**：通过 WebSocket 实现低延迟数据推送
- **历史趋势**：24小时/7天/30天趋势分析

### 2. 多维度数据可视化

使用 ECharts 实现多种高级图表：

- **折线图**：实时流量趋势、验证请求历史
- **饼图**：验证类型分布、地理分布
- **柱状图**：风险等级分布
- **热力图**：访问时段分析
- **仪表盘**：系统资源使用情况

### 3. 智能告警系统

自动检测并告警以下异常：

| 告警类型 | 级别 | 触发条件 |
|---------|------|---------|
| 高拦截率 | 警告 | 拦截率 > 20% |
| 严重拦截率 | 严重 | 拦截率 > 40% |
| 响应时间过长 | 警告 | 响应时间 > 500ms |
| CPU 使用率过高 | 警告 | CPU > 80% |
| CPU 使用率严重过高 | 严重 | CPU > 95% |
| 内存使用率过高 | 警告 | 内存 > 85% |
| 缓存命中率低 | 信息 | 缓存命中率 < 70% |
| 通过率过低 | 严重 | 通过率 < 70% |
| 流量突增 | 严重 | QPS > 平均值 * 3 |
| 流量异常下降 | 警告 | QPS < 平均值 * 0.2 |

### 4. 自定义报表生成器

支持生成多种类型的报表：

- **汇总报表**：核心指标汇总
- **详细报表**：完整数据明细
- **趋势报表**：历史趋势分析
- **风险分析报表**：风险评估报告

### 5. 数据导出功能

支持多种导出格式：

| 格式 | 描述 | MIME 类型 |
|------|------|----------|
| Excel | 多工作表 Excel 文件 | application/vnd.openxmlformats-officedocument.spreadsheetml.sheet |
| CSV | 逗号分隔值文件 | text/csv |
| PDF | 便携式文档格式 | application/pdf |
| JSON | JavaScript 对象表示法 | application/json |

## API 接口

### 基础信息

- **Base URL**: `/api/v1/admin/dashboard/admin`
- **认证**: 需要管理员权限
- **格式**: JSON

### 接口列表

#### 1. 获取仪表盘数据

```
GET /api/v1/admin/dashboard/admin
```

响应示例：

```json
{
  "code": 0,
  "data": {
    "summary": {
      "total_requests": 85000,
      "pass_rate": 92.5,
      "block_rate": 4.3,
      "avg_response_time": 85,
      "active_sessions": 1250
    },
    "extended": {
      "current_qps": 250.5,
      "active_connections": 500,
      "cpu_usage": 35.5,
      "memory_usage": 58.3,
      "cache_hit_rate": 94.7
    }
  }
}
```

#### 2. 获取实时数据

```
GET /api/v1/admin/dashboard/admin/realtime
```

响应示例：

```json
{
  "code": 0,
  "data": {
    "qps": 250.5,
    "active_connections": 500,
    "cpu_usage": 35.5,
    "memory_usage": 58.3,
    "cache_hit_rate": 94.7,
    "timestamp": 1625123456,
    "requests_per_second": [
      {"time": "10:00:00", "value": 250.5}
    ]
  }
}
```

#### 3. 获取趋势数据

```
GET /api/v1/admin/dashboard/admin/trend?period=hour
```

参数：

- `period`: 时间周期 (`hour` | `day` | `week`)

#### 4. 获取告警列表

```
GET /api/v1/admin/dashboard/admin/alerts
```

响应示例：

```json
{
  "code": 0,
  "data": {
    "alerts": [
      {
        "type": "high_block_rate",
        "level": "warning",
        "message": "拦截率异常",
        "timestamp": 1625123456,
        "score": 25.5
      }
    ]
  }
}
```

#### 5. 导出数据

```
GET /api/v1/admin/dashboard/admin/export?format=csv&period=today
```

参数：

- `format`: 导出格式 (`csv` | `excel` | `pdf` | `json`)
- `period`: 时间范围 (`today` | `yesterday` | `week` | `month`)

#### 6. 生成报表

```
POST /api/v1/admin/dashboard/admin/report
```

请求体：

```json
{
  "name": "测试报表",
  "type": "summary",
  "period": "today"
}
```

#### 7. WebSocket 实时推送

```
GET /api/v1/admin/dashboard/admin/ws
```

WebSocket 消息格式：

```json
{
  "type": "metrics",
  "timestamp": 1625123456,
  "payload": {
    "qps": 250.5,
    "active_connections": 500,
    "cpu_usage": 35.5,
    "memory_usage": 58.3,
    "cache_hit_rate": 94.7
  }
}
```

```json
{
  "type": "verification",
  "timestamp": 1625123456,
  "payload": {
    "session_id": "abc123",
    "captcha_type": "slider",
    "status": "success",
    "risk_score": 25.5,
    "ip_address": "192.168.1.1",
    "response_time": 85
  }
}
```

```json
{
  "type": "alert",
  "timestamp": 1625123456,
  "payload": {
    "type": "high_block_rate",
    "level": "warning",
    "message": "拦截率异常",
    "score": 25.5
  }
}
```

## 前端页面

访问地址：`/admin/admin-dashboard`

### 功能特性

- **响应式设计**：适配桌面和移动设备
- **实时刷新**：支持每秒、每5秒、每10秒、每30秒刷新
- **全屏模式**：支持浏览器全屏显示
- **数据导出**：一键导出 Excel、CSV、PDF、JSON
- **报表生成**：可视化报表生成器
- **告警通知**：实时告警推送和声音提醒
- **主题切换**：支持浅色/深色主题

### 使用说明

1. **启动实时模式**：点击"实时模式"按钮启用自动刷新
2. **调整刷新频率**：点击刷新频率下拉菜单选择刷新间隔
3. **导出数据**：点击"导出"按钮选择导出格式
4. **生成报表**：点击"生成报表"按钮自定义报表参数
5. **查看告警**：点击"告警"图标查看告警详情
6. **全屏显示**：点击"全屏"按钮进入全屏模式

## 技术架构

### 后端服务

- **服务层**: `AdminDashboardService`
- **数据聚合**: 实时聚合多维度指标数据
- **缓存策略**: 5秒缓存减少数据库压力
- **并发安全**: 使用读写锁保证并发安全

### WebSocket 服务

- **心跳机制**: 30秒发送一次 Ping 消息
- **自动重连**: 断线后自动重连
- **消息队列**: 1000 容量缓冲验证事件

### 前端实现

- **ECharts 5.4.3**: 数据可视化库
- **Socket.IO Client**: WebSocket 客户端
- **SheetJS**: Excel 导出
- **jsPDF**: PDF 生成

## 性能优化

1. **数据缓存**: 5秒缓存减少重复查询
2. **增量更新**: 仅更新变化的指标
3. **WebSocket 推送**: 实时推送减少轮询
4. **防抖节流**: 避免频繁更新图表
5. **懒加载**: 按需加载图表数据

## 安全考虑

1. **认证授权**: 所有接口需要管理员权限
2. **参数校验**: 严格校验输入参数
3. **防 XSS**: 转义所有用户输入
4. **防 CSRF**: 使用 CSRF Token
5. **限流保护**: API 接口限流

## 监控指标

### 核心指标

- 今日验证总量
- 通过率
- 拦截率
- 平均响应时间
- 当前 QPS
- 活跃连接数

### 系统资源

- CPU 使用率
- 内存使用率
- 磁盘使用率
- 网络 I/O

### 缓存指标

- 缓存命中率
- 缓存键数量
- 缓存内存使用

## 扩展功能

### 自定义监控

可以通过扩展 `AlertRule` 结构添加自定义告警规则：

```go
AlertRule{
    Name:      "custom_rule",
    Condition: func(m *DashboardMetrics) bool { return m.CustomValue > threshold },
    Level:     "warning",
    Message:   "自定义告警消息",
    Threshold: threshold,
}
```

### 自定义图表

可以添加新的 ECharts 图表类型：

```javascript
function initCustomChart() {
    const container = document.getElementById('customChart');
    charts.custom = echarts.init(container);
    charts.custom.setOption({/* 配置项 */});
}
```

## 故障排查

### WebSocket 连接失败

1. 检查浏览器控制台错误信息
2. 确认服务器 WebSocket 端点可用
3. 检查网络代理设置

### 数据不更新

1. 刷新页面重新加载
2. 检查浏览器控制台网络请求
3. 确认 API 接口正常响应

### 导出失败

1. 检查浏览器下载设置
2. 确认文件格式支持
3. 检查磁盘空间

## 更新日志

### v15.0 (2024-05-18)

- ✨ 新增智能仪表盘模块
- ✨ 实现秒级实时监控
- ✨ 添加智能告警系统
- ✨ 支持多格式数据导出
- ✨ 实现自定义报表生成器
- ✨ WebSocket 实时推送
- ✨ 多种高级图表可视化
- ✨ 完整的单元测试覆盖

## 相关文档

- [主仪表盘文档](dashboard.html)
- [实时监控文档](monitoring.html)
- [告警系统文档](notifications.html)
- [数据分析文档](stats.html)
