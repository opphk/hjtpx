const ConnectionLeakDetector = require('../../services/connectionLeakDetector');

describe('ConnectionLeakDetector Service', () => {
  let leakDetector;
  let mockDbPoolManager;

  beforeEach(() => {
    mockDbPoolManager = {
      checkedOutClients: new Map(),
      query: jest.fn(),
      getPoolStats: jest.fn()
    };
    
    leakDetector = new ConnectionLeakDetector(mockDbPoolManager);
  });

  afterEach(() => {
    leakDetector.stop();
    jest.clearAllMocks();
  });

  describe('Initialization', () => {
    test('should initialize with default settings', () => {
      expect(leakDetector.leakThreshold).toBeDefined();
      expect(leakDetector.checkIntervalMs).toBeDefined();
      expect(leakDetector.maxLeakRecords).toBeDefined();
      expect(leakDetector.connectionTracker).toBeDefined();
      expect(leakDetector.leakEvents).toBeDefined();
    });

    test('should initialize statistics', () => {
      expect(leakDetector.statistics.totalCheckedOut).toBe(0);
      expect(leakDetector.statistics.totalReleased).toBe(0);
      expect(leakDetector.statistics.potentialLeaks).toBe(0);
    });
  });

  describe('Connection Tracking', () => {
    test('should track connection', () => {
      const trackingInfo = leakDetector.trackConnection('client-123', {
        query: 'SELECT * FROM users',
        user: 'test_user'
      });

      expect(trackingInfo.clientId).toBe('client-123');
      expect(trackingInfo.query).toBe('SELECT * FROM users');
      expect(trackingInfo.user).toBe('test_user');
      expect(trackingInfo.status).toBe('active');
      expect(leakDetector.connectionTracker.has('client-123')).toBe(true);
    });

    test('should increment total checked out counter', () => {
      leakDetector.trackConnection('client-1');
      leakDetector.trackConnection('client-2');

      expect(leakDetector.statistics.totalCheckedOut).toBe(2);
    });

    test('should emit connectionTracked event', () => {
      const handler = jest.fn();
      leakDetector.on('connectionTracked', handler);

      leakDetector.trackConnection('client-123');

      expect(handler).toHaveBeenCalled();
    });
  });

  describe('Connection Untracking', () => {
    test('should untrack connection', () => {
      leakDetector.trackConnection('client-123');
      const trackingInfo = leakDetector.untrackConnection('client-123');

      expect(trackingInfo).toBeDefined();
      expect(trackingInfo.releasedAt).toBeDefined();
      expect(trackingInfo.status).toBe('released');
      expect(leakDetector.connectionTracker.has('client-123')).toBe(false);
    });

    test('should increment total released counter for normal release', () => {
      leakDetector.leakThreshold = 30000;
      
      leakDetector.trackConnection('client-123');
      leakDetector.untrackConnection('client-123');

      expect(leakDetector.statistics.totalReleased).toBe(1);
    });

    test('should record potential leak for long-held connection', () => {
      leakDetector.leakThreshold = 1000;
      
      const trackingInfo = leakDetector.trackConnection('client-123');
      trackingInfo.checkedOutAt = Date.now() - 5000;
      
      leakDetector.untrackConnection('client-123');

      expect(leakDetector.leakEvents.length).toBeGreaterThanOrEqual(1);
      expect(leakDetector.leakEvents[0].wasLeaked).toBe(true);
    });

    test('should emit connectionUntracked event', () => {
      const handler = jest.fn();
      leakDetector.on('connectionUntracked', handler);

      leakDetector.trackConnection('client-123');
      leakDetector.untrackConnection('client-123');

      expect(handler).toHaveBeenCalled();
    });
  });

  describe('Leak Detection', () => {
    test('should detect leaked connections', () => {
      leakDetector.leakThreshold = 1000;
      leakDetector.connectionTracker.set('client-1', {
        clientId: 'client-1',
        checkedOutAt: Date.now() - 5000,
        stackTrace: 'test',
        query: null,
        user: null
      });

      leakDetector.forceCheck();

      expect(leakDetector.statistics.potentialLeaks).toBe(1);
    });

    test('should emit leaksDetected event', () => {
      const handler = jest.fn();
      leakDetector.on('leaksDetected', handler);

      leakDetector.leakThreshold = 1000;
      leakDetector.connectionTracker.set('client-1', {
        clientId: 'client-1',
        checkedOutAt: Date.now() - 5000,
        stackTrace: 'test',
        query: null,
        user: null
      });

      leakDetector.forceCheck();

      expect(handler).toHaveBeenCalled();
    });

    test('should calculate severity correctly', () => {
      expect(leakDetector._calculateSeverity(60000)).toBe('low');
      expect(leakDetector._calculateSeverity(90000)).toBe('medium');
      expect(leakDetector._calculateSeverity(150000)).toBe('high');
      expect(leakDetector._calculateSeverity(350000)).toBe('critical');
    });
  });

  describe('Auto Cleanup', () => {
    test('should handle auto cleanup configuration', () => {
      const productionDetector = new ConnectionLeakDetector(mockDbPoolManager);
      
      expect(productionDetector.enableAutoCleanup).toBeDefined();
      expect(typeof productionDetector.enableAutoCleanup).toBe('boolean');
    });

    test('should attempt auto cleanup after timeout', () => {
      leakDetector.enableAutoCleanup = true;
      leakDetector.leakThreshold = 1000;
      leakDetector.autoCleanupTimeout = 2000;

      const mockClient = {
        client: {
          release: jest.fn()
        }
      };
      mockDbPoolManager.checkedOutClients.set('client-1', mockClient);

      leakDetector.connectionTracker.set('client-1', {
        clientId: 'client-1',
        checkedOutAt: Date.now() - 5000,
        stackTrace: 'test',
        query: null,
        user: null,
        status: 'active'
      });

      leakDetector.forceCheck();

      expect(leakDetector.statistics.autoCleanups).toBe(1);
    });
  });

  describe('Statistics', () => {
    test('should calculate leak rate', () => {
      leakDetector.statistics.totalCheckedOut = 100;
      leakDetector.leakEvents = [
        { holdDuration: 50000 },
        { holdDuration: 60000 }
      ];

      const stats = leakDetector.getStatistics();

      expect(stats.leakRate).toBe(2);
    });

    test('should calculate average leak duration', () => {
      leakDetector.leakEvents = [
        { holdDuration: 30000 },
        { holdDuration: 60000 }
      ];

      const stats = leakDetector.getStatistics();

      expect(stats.averageLeakDuration).toBe(45000);
    });

    test('should calculate leak trend', () => {
      leakDetector.leakEvents = Array(15).fill({ timestamp: new Date().toISOString() });

      const trend = leakDetector._calculateLeakTrend();

      expect(trend).toBe('increasing');
    });
  });

  describe('Leak Report', () => {
    test('should generate leak report', () => {
      leakDetector.leakEvents = [
        {
          id: 'leak-1',
          clientId: 'client-1',
          holdDuration: 35000,
          severity: 'critical',
          timestamp: new Date().toISOString()
        }
      ];

      const report = leakDetector.getLeakReport();

      expect(report).toHaveProperty('summary');
      expect(report).toHaveProperty('recentLeaks');
      expect(report).toHaveProperty('bySeverity');
      expect(report).toHaveProperty('byDuration');
    });

    test('should group leaks by severity', () => {
      leakDetector.leakEvents = [
        { severity: 'critical', holdDuration: 60000 },
        { severity: 'high', holdDuration: 90000 },
        { severity: 'critical', holdDuration: 120000 }
      ];

      const grouped = leakDetector._groupLeaksBySeverity();

      expect(grouped.critical).toBe(2);
      expect(grouped.high).toBe(1);
    });

    test('should group leaks by duration', () => {
      leakDetector.leakEvents = [
        { holdDuration: 30000 },
        { holdDuration: 120000 },
        { holdDuration: 2000000 }
      ];

      const grouped = leakDetector._groupLeaksByDuration();

      expect(grouped['1-5min']).toBe(1);
      expect(grouped['5-15min']).toBe(1);
      expect(grouped['> 30min']).toBe(1);
    });
  });

  describe('Active Connections', () => {
    test('should get active connections', () => {
      leakDetector.leakThreshold = 30000;
      
      leakDetector.connectionTracker.set('client-1', {
        clientId: 'client-1',
        checkedOutAt: Date.now() - 5000,
        stackTrace: 'test',
        query: 'SELECT 1',
        user: 'test'
      });

      leakDetector.connectionTracker.set('client-2', {
        clientId: 'client-2',
        checkedOutAt: Date.now() - 40000,
        stackTrace: 'test',
        query: 'SELECT 2',
        user: 'test2'
      });

      const active = leakDetector.getActiveConnections();

      expect(active.length).toBe(2);
      expect(active[0].clientId).toBe('client-2');
    });

    test('should mark potential leaks correctly', () => {
      leakDetector.connectionTracker.set('client-1', {
        clientId: 'client-1',
        checkedOutAt: Date.now() - 40000,
        stackTrace: 'test',
        query: null,
        user: null
      });

      const active = leakDetector.getActiveConnections();

      expect(active[0].status).toBe('potential_leak');
    });
  });

  describe('Threshold Configuration', () => {
    test('should update leak threshold', () => {
      const handler = jest.fn();
      leakDetector.on('thresholdChanged', handler);

      leakDetector.setThreshold(60000);

      expect(leakDetector.leakThreshold).toBe(60000);
      expect(handler).toHaveBeenCalled();
    });

    test('should update check interval', () => {
      const handler = jest.fn();
      leakDetector.on('intervalChanged', handler);

      leakDetector.setCheckInterval(5000);

      expect(leakDetector.checkIntervalMs).toBe(5000);
      expect(handler).toHaveBeenCalled();
    });
  });

  describe('False Positive Management', () => {
    test('should mark false positive', () => {
      leakDetector.leakEvents = [
        { clientId: 'client-123', severity: 'medium' }
      ];

      const result = leakDetector.markAsFalsePositive('client-123');

      expect(result).toBe(true);
      expect(leakDetector.leakEvents[0].falsePositive).toBe(true);
      expect(leakDetector.statistics.falsePositives).toBe(1);
    });

    test('should return false for non-existent leak', () => {
      const result = leakDetector.markAsFalsePositive('non-existent');

      expect(result).toBe(false);
    });
  });

  describe('History Management', () => {
    test('should clear leak history', () => {
      leakDetector.leakEvents = [
        { id: 'leak-1' },
        { id: 'leak-2' }
      ];

      leakDetector.clearLeakHistory();

      expect(leakDetector.leakEvents).toEqual([]);
    });

    test('should reset statistics', () => {
      leakDetector.statistics.totalCheckedOut = 100;
      leakDetector.statistics.potentialLeaks = 10;

      leakDetector.resetStatistics();

      expect(leakDetector.statistics.totalCheckedOut).toBe(0);
      expect(leakDetector.statistics.potentialLeaks).toBe(0);
    });
  });

  describe('Data Export', () => {
    test('should export leak data as JSON', () => {
      leakDetector.leakEvents = [
        { id: 'leak-1', holdDuration: 50000 }
      ];

      const exported = leakDetector.exportLeakData('json');

      expect(exported).toContain('leak-1');
      expect(exported).toContain('50000');
    });

    test('should export leak data as CSV', () => {
      leakDetector.leakEvents = [
        {
          id: 'leak-1',
          clientId: 'client-1',
          checkedOutAt: '2024-01-01',
          releasedAt: '2024-01-01',
          holdDuration: 50000,
          severity: 'medium',
          query: 'SELECT 1',
          user: 'test'
        }
      ];

      const exported = leakDetector.exportLeakData('csv');

      expect(exported).toContain('ID');
      expect(exported).toContain('client-1');
      expect(exported).toContain('50000');
    });
  });

  describe('Start and Stop', () => {
    test('should start leak detection', () => {
      const intervalSpy = jest.spyOn(global, 'setInterval');

      leakDetector.start();

      expect(intervalSpy).toHaveBeenCalled();
      expect(leakDetector.isMonitoring).toBe(true);
    });

    test('should stop leak detection', () => {
      leakDetector.start();
      leakDetector.stop();

      expect(leakDetector.isMonitoring).toBe(false);
    });

    test('should prevent double start', () => {
      const intervalSpy = jest.spyOn(global, 'setInterval');

      leakDetector.start();
      const firstInterval = leakDetector.checkInterval;
      leakDetector.start();
      const secondInterval = leakDetector.checkInterval;

      expect(firstInterval).toBe(secondInterval);
    });
  });
});
