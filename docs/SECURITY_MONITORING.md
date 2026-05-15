# 安全监控统一文档

## 概述

安全监控系统已统一，提供威胁检测、告警管理和安全报告功能。

## 统一安全监控 (UnifiedSecurityMonitor)

### 主要功能

1. **威胁检测**
   - SQL注入检测
   - XSS攻击检测
   - 路径遍历检测
   - 命令注入检测
   - 暴力攻击检测
   - 可疑用户代理检测

2. **告警管理**
   - 阈值触发告警
   - 多渠道通知 (Email, Slack, Webhook)
   - 告警确认
   - 告警历史

3. **统计分析**
   - 实时统计
   - 风险评分
   - 安全报告生成
   - 历史数据分析

### API

```javascript
const { UnifiedSecurityMonitor } = require('./services/unifiedSecurityMonitor');

const monitor = new UnifiedSecurityMonitor({
  enabled: true,
  maxAlerts: 1000,
  alertMethods: ['email', 'slack'],
  alertRecipients: ['admin@example.com']
});

// 威胁检测
const threat = monitor.detectThreat({
  body: { query: "'; DROP TABLE users; --" },
  query: {},
  headers: { 'user-agent': 'Mozilla/5.0' },
  ip: '192.168.1.1',
  path: '/api/users'
});

// 记录可疑活动
monitor.recordFailedLogin('user@example.com', '192.168.1.1', 'Mozilla/5.0');
monitor.recordFailedAuth('token', '192.168.1.1');
monitor.recordRateLimitExceeded('192.168.1.1', '/api/users');
monitor.recordUnauthorizedAccess('user123', '/admin');

// 阈值告警检查
await monitor.checkAndAlert({
  type: 'failedLogin',
  ip: '192.168.1.1',
  severity: 'medium'
});

// 获取告警
const alerts = monitor.getAlerts({ severity: 'critical', limit: 10 });

// 获取威胁
const threats = monitor.getThreats({ severity: 'high', limit: 10 });

// 获取统计
const stats = monitor.getStatistics();

// 生成报告
const report = monitor.generateSecurityReport({
  startDate: new Date('2024-01-01'),
  endDate: new Date(),
  includeDetails: true
});

// 导出报告
const exportData = monitor.exportReport();

// 数据清理
monitor.clearOldData(168); // 保留168小时

// 告警处理
monitor.onAlert((alert) => {
  console.log('Alert:', alert);
});
monitor.addAlertHandler(async (alert) => {
  await sendToPagerDuty(alert);
});

// 确认告警
monitor.acknowledgeAlert('ALERT-001');

// 获取监控状态
const status = monitor.getStats();

// 重置
monitor.reset();

// 关闭
monitor.close();
```

## 前端安全服务

### 功能

```javascript
import securityService from './utils/security';

// CSRF令牌
const csrfToken = securityService.generateCSRFToken();
const isValid = securityService.validateCSRFToken(token);

// 输入清理
const sanitized = securityService.sanitizeInput('<script>alert("xss")</script>');
// 输出: "&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;"

// XSS检测
const hasXSS = securityService.detectXSS('<script>alert("xss")</script>');
// 输出: true

// 对象清理
const sanitizedObj = securityService.sanitizeObject({
  username: '<b>admin</b>',
  password: 'secret'
});

// 密码验证
const passwordResult = securityService.validatePassword('StrongP@ss123');
// { isValid: true, errors: [] }

// 邮箱验证
const isValidEmail = securityService.validateEmail('user@example.com');

// 安全头
const headers = securityService.getSecurityHeaders();

// 数据加密
const encrypted = securityService.encryptData(data, key);
const decrypted = securityService.decryptData(encrypted, key);

// 安全随机数
const random = securityService.generateSecureRandom(32);

// 密码哈希
const hash = securityService.hashPassword(password);
```

## 告警阈值配置

环境变量配置：

```bash
ALERT_FAILED_LOGIN=5
ALERT_SUSPICIOUS_IP=10
ALERT_XSS_ATTEMPT=3
ALERT_SQL_INJECTION=1
ALERT_BRUTE_FORCE=10
ALERT_RATE_LIMIT=50
```

## 测试覆盖

- 威胁检测测试
- 告警管理测试
- 统计分析测试
- 数据清理测试
- 报告生成测试

## 最佳实践

1. 定期检查安全报告
2. 设置合理的告警阈值
3. 启用多渠道通知
4. 定期清理旧数据
5. 监控异常模式
6. 实施纵深防御
