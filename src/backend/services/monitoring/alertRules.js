const alertRules = {
  rules: [
    {
      id: 'high_error_rate',
      name: 'High Error Rate',
      condition: (stats) => stats.errorRate > 5,
      severity: 'critical',
      description: '错误率超过5%',
      cooldown: 300000, // 5分钟冷却
      enabled: true
    },
    {
      id: 'slow_response',
      name: 'Slow Response Time',
      condition: (stats) => stats.avgResponseTime > 1000,
      severity: 'warning',
      description: '平均响应时间超过1秒',
      cooldown: 180000,
      enabled: true
    },
    {
      id: 'high_cpu',
      name: 'High CPU Usage',
      condition: (stats) => stats.cpuUsage > 80,
      severity: 'warning',
      description: 'CPU使用率超过80%',
      cooldown: 300000,
      enabled: true
    },
    {
      id: 'high_memory',
      name: 'High Memory Usage',
      condition: (stats) => stats.memoryUsage > 85,
      severity: 'critical',
      description: '内存使用率超过85%',
      cooldown: 300000,
      enabled: true
    },
    {
      id: 'database_slow',
      name: 'Slow Database Queries',
      condition: (stats) => stats.dbQueryTime > 500,
      severity: 'warning',
      description: '数据库查询平均时间超过500ms',
      cooldown: 180000,
      enabled: true
    },
    {
      id: 'low_cache_hit',
      name: 'Low Cache Hit Rate',
      condition: (stats) => stats.cacheHitRate < 60,
      severity: 'warning',
      description: '缓存命中率低于60%',
      cooldown: 600000,
      enabled: true
    }
  ],
  
  triggeredAlerts: new Map(),
  
  // 检查所有规则
  checkRules(stats) {
    const alerts = [];
    
    for (const rule of this.rules) {
      if (!rule.enabled) continue;
      
      // 检查冷却时间
      const lastTriggered = this.triggeredAlerts.get(rule.id);
      if (lastTriggered && Date.now() - lastTriggered < rule.cooldown) {
        continue;
      }
      
      // 检查条件
      if (rule.condition(stats)) {
        const alert = {
          id: `${rule.id}_${Date.now()}`,
          ruleId: rule.id,
          name: rule.name,
          severity: rule.severity,
          description: rule.description,
          value: this.getAlertValue(stats, rule.id),
          threshold: this.getThreshold(rule.id),
          triggeredAt: new Date().toISOString(),
          status: 'firing'
        };
        
        alerts.push(alert);
        this.triggeredAlerts.set(rule.id, Date.now());
      }
    }
    
    return alerts;
  },
  
  getAlertValue(stats, ruleId) {
    switch (ruleId) {
      case 'high_error_rate': return stats.errorRate;
      case 'slow_response': return stats.avgResponseTime;
      case 'high_cpu': return stats.cpuUsage;
      case 'high_memory': return stats.memoryUsage;
      case 'database_slow': return stats.dbQueryTime;
      case 'low_cache_hit': return stats.cacheHitRate;
      default: return null;
    }
  },
  
  getThreshold(ruleId) {
    switch (ruleId) {
      case 'high_error_rate': return 5;
      case 'slow_response': return 1000;
      case 'high_cpu': return 80;
      case 'high_memory': return 85;
      case 'database_slow': return 500;
      case 'low_cache_hit': return 60;
      default: return null;
    }
  },
  
  // 添加自定义规则
  addRule(rule) {
    this.rules.push(rule);
    return rule;
  },
  
  // 更新规则
  updateRule(id, updates) {
    const rule = this.rules.find(r => r.id === id);
    if (rule) {
      Object.assign(rule, updates);
    }
    return rule;
  },
  
  // 删除规则
  deleteRule(id) {
    const index = this.rules.findIndex(r => r.id === id);
    if (index !== -1) {
      this.rules.splice(index, 1);
      return true;
    }
    return false;
  },
  
  // 获取所有规则
  getAllRules() {
    return this.rules;
  }
};

module.exports = alertRules;
