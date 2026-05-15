const ConnectionPoolMonitor = require('../../services/connectionPoolMonitor');

describe('ConnectionPoolMonitor Service', () => {
  let monitor;
  let mockDbPoolManager;

  beforeEach(() => {
    mockDbPoolManager = {
      query: jest.fn(),
      getPoolStats: jest.fn(),
      getQueryStats: jest.fn()
    };
    
    monitor = new ConnectionPoolMonitor(mockDbPoolManager);
  });

  afterEach(() => {
    monitor.stop();
    jest.clearAllMocks();
  });

  describe('Initialization', () => {
    test('should initialize with default thresholds', () => {
      expect(monitor.alertThresholds).toBeDefined();
      expect(monitor.alertThresholds.highConnectionUsage).toBeDefined();
      expect(monitor.alertThresholds.criticalConnectionUsage).toBeDefined();
      expect(monitor.metricsHistory).toBeDefined();
      expect(monitor.alerts).toBeDefined();
    });

    test('should set custom thresholds from environment', () => {
      process.env.MONITOR_HIGH_CONNECTION_USAGE = '0.80';
      process.env.MONITOR_HIGH_QUERY_TIME = '2000';
      
      const customMonitor = new ConnectionPoolMonitor(mockDbPoolManager);
      
      expect(customMonitor.alertThresholds.highConnectionUsage).toBe(0.80);
      expect(customMonitor.alertThresholds.highQueryTime).toBe(2000);
    });
  });

  describe('Metrics Collection', () => {
    test('should collect pool stats', () => {
      mockDbPoolManager.getPoolStats.mockReturnValue({
        total: 20,
        idle: 5,
        busy: 15,
        waiting: 2,
        checkedOut: 10,
        capacityUsage: '75.00%',
        config: { min: 5, max: 20 }
      });

      const stats = monitor._collectPoolStats();

      expect(stats.total).toBe(20);
      expect(stats.idle).toBe(5);
      expect(stats.busy).toBe(15);
      expect(stats.capacityUsage).toBe(75);
    });

    test('should collect query stats', () => {
      mockDbPoolManager.getQueryStats.mockReturnValue({
        queries: 1000,
        slowQueries: 10,
        errors: 5,
        avgQueryTime: 50,
        p50QueryTime: 30,
        p95QueryTime: 150,
        p99QueryTime: 300,
        errorRate: '0.50%',
        hitRate: '99.50%'
      });

      const stats = monitor._collectQueryStats();

      expect(stats.totalQueries).toBe(1000);
      expect(stats.slowQueries).toBe(10);
      expect(stats.errors).toBe(5);
    });

    test('should collect system stats', () => {
      const stats = monitor._collectSystemStats();

      expect(stats.cpuUsage).toBeDefined();
      expect(stats.totalMemory).toBeDefined();
      expect(stats.freeMemory).toBeDefined();
      expect(stats.memoryUsagePercent).toBeGreaterThan(0);
    });

    test('should store metrics in history', () => {
      mockDbPoolManager.getPoolStats.mockReturnValue({
        total: 10,
        idle: 5,
        busy: 5,
        waiting: 0,
        capacityUsage: '50.00%',
        config: { min: 2, max: 10 }
      });

      monitor._storeMetrics('pool_stats', { total: 10, idle: 5, busy: 5 });

      expect(monitor.metricsHistory.length).toBe(1);
      expect(monitor.metricsHistory[0].collector).toBe('pool_stats');
    });

    test('should limit history size', () => {
      monitor.maxHistorySize = 5;
      
      for (let i = 0; i < 10; i++) {
        monitor.metricsHistory.push({ id: i });
      }

      expect(monitor.metricsHistory.length).toBe(5);
    });
  });

  describe('Threshold Checking', () => {
    test('should emit alert for high connection usage', () => {
      const handler = jest.fn();
      monitor.on('alert', handler);

      monitor._checkPoolThresholds({
        capacityUsage: 0.90,
        waiting: 2
      });

      expect(handler).toHaveBeenCalled();
      const alert = handler.mock.calls[0][0];
      expect(alert.severity).toBe('warning');
      expect(alert.type).toBe('connection_usage');
    });

    test('should emit critical alert for critical connection usage', () => {
      const handler = jest.fn();
      monitor.on('criticalAlert', handler);

      monitor._checkPoolThresholds({
        capacityUsage: 0.97,
        waiting: 2
      });

      expect(handler).toHaveBeenCalled();
      const alert = handler.mock.calls[0][0];
      expect(alert.severity).toBe('critical');
    });

    test('should emit alert for high query time', () => {
      const handler = jest.fn();
      monitor.on('alert', handler);

      monitor._checkQueryThresholds({
        p95QueryTime: 2000,
        errorRate: 0.01
      });

      expect(handler).toHaveBeenCalled();
    });

    test('should emit alert for high error rate', () => {
      const handler = jest.fn();
      monitor.on('alert', handler);

      monitor._checkQueryThresholds({
        p95QueryTime: 100,
        errorRate: 0.08
      });

      expect(handler).toHaveBeenCalled();
      const alert = handler.mock.calls[0][0];
      expect(alert.type).toBe('error_rate');
    });
  });

  describe('Alert Management', () => {
    test('should store alerts', () => {
      monitor._emitAlert('warning', 'test_alert', {
        message: 'Test alert',
        current: 50,
        threshold: 40
      });

      expect(monitor.alerts.length).toBe(1);
      expect(monitor.alerts[0].severity).toBe('warning');
      expect(monitor.alerts[0].message).toBe('Test alert');
    });

    test('should limit alerts to 100', () => {
      for (let i = 0; i < 110; i++) {
        monitor._emitAlert('warning', 'test_alert', { message: `Alert ${i}` });
      }

      expect(monitor.alerts.length).toBe(100);
    });

    test('should get alerts by severity', () => {
      monitor.alerts = [
        { severity: 'warning', type: 'test' },
        { severity: 'critical', type: 'test' },
        { severity: 'warning', type: 'test' }
      ];

      const criticalAlerts = monitor.getAlerts('critical');
      expect(criticalAlerts.length).toBe(1);
    });

    test('should clear alerts', () => {
      monitor.alerts = [
        { severity: 'warning', type: 'test' },
        { severity: 'critical', type: 'test' }
      ];

      monitor.clearAlerts();

      expect(monitor.alerts.length).toBe(0);
    });
  });

  describe('Health Report Generation', () => {
    test('should generate health report', () => {
      mockDbPoolManager.getPoolStats.mockReturnValue({
        total: 20,
        idle: 5,
        busy: 15,
        waiting: 2,
        capacityUsage: '75.00%',
        config: { min: 5, max: 20 }
      });

      mockDbPoolManager.getQueryStats.mockReturnValue({
        queries: 1000,
        slowQueries: 5,
        errors: 2,
        avgQueryTime: 50,
        p50QueryTime: 30,
        p95QueryTime: 150,
        p99QueryTime: 300,
        errorRate: '0.20%'
      });

      const report = monitor.generateHealthReport();

      expect(report).toHaveProperty('generatedAt');
      expect(report).toHaveProperty('overallHealth');
      expect(report).toHaveProperty('pool');
      expect(report).toHaveProperty('queries');
      expect(report).toHaveProperty('recommendations');
    });

    test('should calculate overall health score', () => {
      const score = monitor._calculateOverallHealth(
        { capacityUsage: 0.70 },
        { errorRate: 0.02, p95QueryTime: 500 }
      );

      expect(score.score).toBeGreaterThan(80);
      expect(score.status).toBe('healthy');
    });

    test('should calculate trend correctly', () => {
      const recentMetrics = Array(20).fill(null).map((_, i) => ({
        collector: 'pool_stats',
        timestamp: new Date(Date.now() - (20 - i) * 60000).toISOString(),
        capacityUsage: 60 + Math.random() * 10
      }));

      const trend = monitor._calculateTrends(recentMetrics);

      expect(trend.connectionUsage).toBeDefined();
    });

    test('should generate recommendations', () => {
      const recommendations = monitor._generateRecommendations(
        { capacityUsage: 0.90, waiting: 10 },
        { p95QueryTime: 2000, slowQueries: 20, errorRate: 0.08 }
      );

      expect(recommendations.length).toBeGreaterThan(0);
      expect(recommendations.some(r => r.type === 'pool_size')).toBe(true);
    });
  });

  describe('Aggregated Metrics', () => {
    test('should calculate aggregated metrics for time range', () => {
      const metrics = [
        {
          collector: 'pool_stats',
          timestamp: new Date().toISOString(),
          capacityUsage: 70,
          busy: 14,
          idle: 6,
          waiting: 0
        },
        {
          collector: 'query_stats',
          timestamp: new Date().toISOString(),
          totalQueries: 100,
          errors: 2,
          avgQueryTime: 50,
          p95QueryTime: 150,
          errorRate: 2
        }
      ];

      monitor.metricsHistory = metrics;

      const aggregated = monitor.getAggregatedMetrics('10m');

      expect(aggregated).toHaveProperty('pool');
      expect(aggregated).toHaveProperty('queries');
      expect(aggregated.timeRange).toBe('10m');
    });

    test('should calculate average correctly', () => {
      const values = [10, 20, 30, 40, 50];
      const avg = monitor._average(values);

      expect(avg).toBe(30);
    });

    test('should handle empty values in average', () => {
      const avg = monitor._average([]);

      expect(avg).toBe(0);
    });
  });

  describe('Metrics Export', () => {
    test('should export metrics as JSON', () => {
      monitor.metricsHistory = [
        { collector: 'pool_stats', total: 10 }
      ];

      const exported = monitor.exportMetrics('json');

      expect(exported).toContain('pool_stats');
      expect(exported).toContain('10');
    });

    test('should export metrics as CSV', () => {
      monitor.metricsHistory = [
        { collector: 'pool_stats', total: 10, idle: 5 }
      ];

      const exported = monitor.exportMetrics('csv');

      expect(exported).toContain('collector');
      expect(exported).toContain('pool_stats');
    });
  });

  describe('Threshold Management', () => {
    test('should update alert threshold', () => {
      const handler = jest.fn();
      monitor.on('thresholdUpdated', handler);

      monitor.setAlertThreshold('highConnectionUsage', 0.80);

      expect(monitor.alertThresholds.highConnectionUsage).toBe(0.80);
      expect(handler).toHaveBeenCalled();
    });

    test('should ignore invalid threshold type', () => {
      const original = monitor.alertThresholds.highConnectionUsage;

      monitor.setAlertThreshold('invalidThreshold', 0.90);

      expect(monitor.alertThresholds.highConnectionUsage).toBe(original);
    });
  });

  describe('Force Collection', () => {
    test('should force metrics collection', () => {
      mockDbPoolManager.getPoolStats.mockReturnValue({
        total: 10,
        idle: 5,
        busy: 5,
        capacityUsage: '50.00%'
      });

      mockDbPoolManager.getQueryStats.mockReturnValue({
        queries: 100,
        errors: 0
      });

      const collected = monitor.forceCollection();

      expect(collected.pool_stats).toBeDefined();
      expect(collected.query_stats).toBeDefined();
      expect(collected.system_stats).toBeDefined();
    });
  });

  describe('Start and Stop', () => {
    test('should start monitoring', () => {
      const intervalSpy = jest.spyOn(global, 'setInterval');

      monitor.start(30000);

      expect(intervalSpy).toHaveBeenCalled();
      expect(monitor.isMonitoring).toBe(true);
    });

    test('should stop monitoring', () => {
      monitor.start();
      monitor.stop();

      expect(monitor.isMonitoring).toBe(false);
    });

    test('should not start if already monitoring', () => {
      const intervalSpy = jest.spyOn(global, 'setInterval');

      monitor.start();
      monitor.start();

      expect(intervalSpy).toHaveBeenCalledTimes(1);
    });
  });

  describe('Reset', () => {
    test('should reset metrics and alerts', () => {
      monitor.metricsHistory = [{ id: 1 }];
      monitor.alerts = [{ id: 1 }];

      monitor.reset();

      expect(monitor.metricsHistory).toEqual([]);
      expect(monitor.alerts).toEqual([]);
    });
  });

  describe('Metrics History', () => {
    test('should get metrics history by collector', () => {
      monitor.metricsHistory = [
        { collector: 'pool_stats', total: 10 },
        { collector: 'query_stats', total: 100 },
        { collector: 'pool_stats', total: 15 }
      ];

      const poolMetrics = monitor.getMetricsHistory('pool_stats');

      expect(poolMetrics.length).toBe(2);
      expect(poolMetrics.every(m => m.collector === 'pool_stats')).toBe(true);
    });

    test('should limit metrics history', () => {
      monitor.metricsHistory = Array(200).fill({ collector: 'pool_stats' });

      const history = monitor.getMetricsHistory('pool_stats', 50);

      expect(history.length).toBe(50);
    });
  });
});
