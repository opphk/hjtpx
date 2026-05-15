const alertHistory = {
  history: [],
  maxSize: 10000,
  
  // 添加告警记录
  addAlert(alert) {
    const record = {
      ...alert,
      acknowledged: false,
      acknowledgedBy: null,
      acknowledgedAt: null,
      resolvedAt: null,
      notes: []
    };
    
    this.history.push(record);
    
    // 保持最大容量
    if (this.history.length > this.maxSize) {
      this.history.shift();
    }
    
    return record;
  },
  
  // 获取告警历史
  getHistory(options = {}) {
    let filtered = [...this.history];
    
    if (options.ruleId) {
      filtered = filtered.filter(a => a.ruleId === options.ruleId);
    }
    
    if (options.severity) {
      filtered = filtered.filter(a => a.severity === options.severity);
    }
    
    if (options.status) {
      filtered = filtered.filter(a => a.status === options.status);
    }
    
    if (options.startTime) {
      filtered = filtered.filter(a => 
        new Date(a.triggeredAt).getTime() >= options.startTime
      );
    }
    
    if (options.endTime) {
      filtered = filtered.filter(a => 
        new Date(a.triggeredAt).getTime() <= options.endTime
      );
    }
    
    if (options.limit) {
      filtered = filtered.slice(-options.limit);
    }
    
    return filtered.sort((a, b) => 
      new Date(b.triggeredAt) - new Date(a.triggeredAt)
    );
  },
  
  // 确认告警
  acknowledgeAlert(alertId, userId, note = '') {
    const alert = this.history.find(a => a.id === alertId);
    if (alert) {
      alert.acknowledged = true;
      alert.acknowledgedBy = userId;
      alert.acknowledgedAt = new Date().toISOString();
      if (note) {
        alert.notes.push({ userId, note, timestamp: alert.acknowledgedAt });
      }
      return alert;
    }
    return null;
  },
  
  // 解决告警
  resolveAlert(alertId, userId, note = '') {
    const alert = this.history.find(a => a.id === alertId);
    if (alert) {
      alert.status = 'resolved';
      alert.resolvedAt = new Date().toISOString();
      if (note) {
        alert.notes.push({ userId, note, timestamp: alert.resolvedAt });
      }
      return alert;
    }
    return null;
  },
  
  // 获取统计信息
  getStatistics(timeRange = 86400000) { // 默认24小时
    const startTime = Date.now() - timeRange;
    const alerts = this.getHistory({ startTime });
    
    return {
      total: alerts.length,
      bySeverity: {
        critical: alerts.filter(a => a.severity === 'critical').length,
        warning: alerts.filter(a => a.severity === 'warning').length,
        info: alerts.filter(a => a.severity === 'info').length
      },
      byStatus: {
        firing: alerts.filter(a => a.status === 'firing').length,
        acknowledged: alerts.filter(a => a.status === 'acknowledged').length,
        resolved: alerts.filter(a => a.status === 'resolved').length
      },
      byRule: this.groupByRule(alerts),
      mttd: this.calculateMTTD(alerts), // Mean Time to Detect
      mtta: this.calculateMTTA(alerts), // Mean Time to Acknowledge
      mttr: this.calculateMTTR(alerts)  // Mean Time to Resolve
    };
  },
  
  groupByRule(alerts) {
    const grouped = {};
    alerts.forEach(alert => {
      if (!grouped[alert.ruleId]) {
        grouped[alert.ruleId] = {
          ruleId: alert.ruleId,
          name: alert.name,
          count: 0
        };
      }
      grouped[alert.ruleId].count++;
    });
    return Object.values(grouped);
  },
  
  calculateMTTD(alerts) {
    if (alerts.length === 0) return null;
    return alerts.reduce((sum, alert) => {
      const triggerTime = new Date(alert.triggeredAt).getTime();
      return sum + triggerTime;
    }, 0) / alerts.length;
  },
  
  calculateMTTA(alerts) {
    const acknowledged = alerts.filter(a => a.acknowledgedAt);
    if (acknowledged.length === 0) return null;
    
    return acknowledged.reduce((sum, alert) => {
      const triggerTime = new Date(alert.triggeredAt).getTime();
      const ackTime = new Date(alert.acknowledgedAt).getTime();
      return sum + (ackTime - triggerTime);
    }, 0) / acknowledged.length;
  },
  
  calculateMTTR(alerts) {
    const resolved = alerts.filter(a => a.resolvedAt);
    if (resolved.length === 0) return null;
    
    return resolved.reduce((sum, alert) => {
      const triggerTime = new Date(alert.triggeredAt).getTime();
      const resolveTime = new Date(alert.resolvedAt).getTime();
      return sum + (resolveTime - triggerTime);
    }, 0) / resolved.length;
  }
};

module.exports = alertHistory;
