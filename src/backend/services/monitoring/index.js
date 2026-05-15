const monitoringService = {
  metrics: new Map(),
  
  // 记录指标
  recordMetric(name, value, tags = {}) {
    const timestamp = Date.now();
    const metric = {
      name,
      value,
      tags,
      timestamp
    };
    
    if (!this.metrics.has(name)) {
      this.metrics.set(name, []);
    }
    
    this.metrics.get(name).push(metric);
    
    // 只保留最近1000条
    const metrics = this.metrics.get(name);
    if (metrics.length > 1000) {
      metrics.shift();
    }
    
    return metric;
  },
  
  // 获取指标
  getMetric(name, options = {}) {
    const { startTime, endTime, limit } = options;
    let metrics = this.metrics.get(name) || [];
    
    if (startTime) {
      metrics = metrics.filter(m => m.timestamp >= startTime);
    }
    
    if (endTime) {
      metrics = metrics.filter(m => m.timestamp <= endTime);
    }
    
    if (limit) {
      metrics = metrics.slice(-limit);
    }
    
    return metrics;
  },
  
  // 计算聚合值
  getAggregatedMetric(name, aggregation = 'avg') {
    const metrics = this.metrics.get(name) || [];
    const values = metrics.map(m => m.value);
    
    if (values.length === 0) return null;
    
    switch (aggregation) {
      case 'avg':
        return values.reduce((a, b) => a + b, 0) / values.length;
      case 'min':
        return Math.min(...values);
      case 'max':
        return Math.max(...values);
      case 'sum':
        return values.reduce((a, b) => a + b, 0);
      case 'count':
        return values.length;
      default:
        return values[values.length - 1];
    }
  },
  
  // 获取实时统计数据
  getRealtimeStats() {
    return {
      requests: this.getAggregatedMetric('http.requests', 'count'),
      avgResponseTime: this.getAggregatedMetric('http.response_time', 'avg'),
      errorRate: this.calculateErrorRate(),
      activeUsers: this.getAggregatedMetric('users.active', 'count'),
      cpuUsage: this.getAggregatedMetric('system.cpu', 'avg'),
      memoryUsage: this.getAggregatedMetric('system.memory', 'avg'),
      dbQueryTime: this.getAggregatedMetric('database.query_time', 'avg'),
      cacheHitRate: this.getAggregatedMetric('cache.hit_rate', 'avg'),
      timestamp: Date.now()
    };
  },
  
  // 计算错误率
  calculateErrorRate() {
    const total = this.metrics.get('http.requests')?.length || 0;
    const errors = this.metrics.get('http.errors')?.length || 0;
    return total > 0 ? (errors / total) * 100 : 0;
  },
  
  // 获取时间序列数据
  getTimeSeriesData(name, interval = 60000) {
    const metrics = this.metrics.get(name) || [];
    const now = Date.now();
    const start = now - 3600000; // 最近1小时
    
    const timeBuckets = new Map();
    
    metrics
      .filter(m => m.timestamp >= start)
      .forEach(m => {
        const bucket = Math.floor(m.timestamp / interval) * interval;
        if (!timeBuckets.has(bucket)) {
          timeBuckets.set(bucket, []);
        }
        timeBuckets.get(bucket).push(m.value);
      });
    
    return Array.from(timeBuckets.entries())
      .map(([timestamp, values]) => ({
        timestamp,
        value: values.reduce((a, b) => a + b, 0) / values.length
      }))
      .sort((a, b) => a.timestamp - b.timestamp);
  }
};

module.exports = monitoringService;
