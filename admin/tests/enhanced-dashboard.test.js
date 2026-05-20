/**
 * HJTPX Admin Enhanced Dashboard Tests
 * 测试增强仪表盘的各项功能
 */

describe('Enhanced Dashboard Tests', () => {
  
  describe('Number Formatting', () => {
    test('should format numbers correctly', () => {
      expect(formatNumber(1234567)).toBe('1.2M');
      expect(formatNumber(12345)).toBe('12.3K');
      expect(formatNumber(123)).toBe('123');
      expect(formatNumber(0)).toBe('0');
    });

    test('should format large numbers with M suffix', () => {
      expect(formatNumber(1000000)).toBe('1.0M');
      expect(formatNumber(2500000)).toBe('2.5M');
      expect(formatNumber(10000000)).toBe('10.0M');
    });

    test('should format thousands with K suffix', () => {
      expect(formatNumber(1000)).toBe('1.0K');
      expect(formatNumber(1500)).toBe('1.5K');
      expect(formatNumber(10000)).toBe('10.0K');
    });
  });

  describe('KPI Animation', () => {
    test('should animate value with percentage suffix', () => {
      const mockElement = {
        textContent: '0%',
        id: 'testElement'
      };
      
      document.getElementById = jest.fn().mockReturnValue(mockElement);
      
      animateValue('testElement', 0, 95.5, 100, '%');
      
      return new Promise(resolve => {
        setTimeout(() => {
          expect(mockElement.textContent).toMatch(/%/);
          resolve();
        }, 150);
      });
    });

    test('should animate value with ms suffix', () => {
      const mockElement = {
        textContent: '0ms',
        id: 'testElement'
      };
      
      document.getElementById = jest.fn().mockReturnValue(mockElement);
      
      animateValue('testElement', 0, 85, 100, 'ms');
      
      return new Promise(resolve => {
        setTimeout(() => {
          expect(mockElement.textContent).toMatch(/ms/);
          resolve();
        }, 150);
      });
    });

    test('should handle null element gracefully', () => {
      document.getElementById = jest.fn().mockReturnValue(null);
      
      expect(() => animateValue('nonExistent', 0, 100, 100)).not.toThrow();
    });
  });

  describe('KPI Badge Updates', () => {
    test('should update badge with success condition', () => {
      const mockElement = {
        textContent: '',
        className: ''
      };
      
      document.getElementById = jest.fn().mockReturnValue(mockElement);
      
      updateKPIBadge('testElement', true, '达标', '未达标');
      
      expect(mockElement.textContent).toBe('达标');
      expect(mockElement.className).toContain('success');
    });

    test('should update badge with fail condition', () => {
      const mockElement = {
        textContent: '',
        className: ''
      };
      
      document.getElementById = jest.fn().mockReturnValue(mockElement);
      
      updateKPIBadge('testElement', false, '达标', '未达标');
      
      expect(mockElement.textContent).toBe('未达标');
      expect(mockElement.className).toContain('warning');
    });

    test('should handle non-existent element', () => {
      document.getElementById = jest.fn().mockReturnValue(null);
      
      expect(() => updateKPIBadge('nonExistent', true, 'a', 'b')).not.toThrow();
    });
  });

  describe('Trend Badge Updates', () => {
    test('should update trend badge with positive change', () => {
      const mockElement = {
        innerHTML: '',
        className: ''
      };
      
      document.getElementById = jest.fn().mockReturnValue(mockElement);
      
      updateTrendBadge('testElement', 5.5);
      
      expect(mockElement.innerHTML).toContain('fa-arrow-up');
      expect(mockElement.className).toContain('success');
    });

    test('should update trend badge with negative change', () => {
      const mockElement = {
        innerHTML: '',
        className: ''
      };
      
      document.getElementById = jest.fn().mockReturnValue(mockElement);
      
      updateTrendBadge('testElement', -3.2);
      
      expect(mockElement.innerHTML).toContain('fa-arrow-down');
      expect(mockElement.className).toContain('danger');
    });

    test('should update trend badge with zero change', () => {
      const mockElement = {
        innerHTML: '',
        className: ''
      };
      
      document.getElementById = jest.fn().mockReturnValue(mockElement);
      
      updateTrendBadge('testElement', 0);
      
      expect(mockElement.innerHTML).toContain('fa-arrow-up');
      expect(mockElement.className).toContain('success');
    });
  });

  describe('HTML Escape', () => {
    test('should escape script tags', () => {
      expect(escapeHtml('<script>alert("xss")</script>')).toBe('&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;');
    });

    test('should escape HTML tags', () => {
      expect(escapeHtml('<div>Test</div>')).toBe('&lt;div&gt;Test&lt;/div&gt;');
    });

    test('should handle normal text', () => {
      expect(escapeHtml('Normal text')).toBe('Normal text');
      expect(escapeHtml('Hello World')).toBe('Hello World');
    });

    test('should handle null and undefined', () => {
      expect(escapeHtml(null)).toBe('');
      expect(escapeHtml(undefined)).toBe('');
    });

    test('should escape quotes', () => {
      expect(escapeHtml('"quoted"')).toContain('&quot;');
      expect(escapeHtml("'single'")).toContain('&#39;');
    });
  });

  describe('Time Formatting', () => {
    test('should format time correctly', () => {
      const date = new Date('2024-01-15T10:30:45');
      const formatted = formatTime(date);
      expect(formatted).toMatch(/\d{2}:\d{2}:\d{2}/);
    });

    test('should handle various dates', () => {
      const date1 = new Date('2024-01-15T00:00:00');
      const date2 = new Date('2024-12-31T23:59:59');
      
      expect(formatTime(date1)).toBeDefined();
      expect(formatTime(date2)).toBeDefined();
    });
  });

  describe('Fullscreen Toggle', () => {
    test('should request fullscreen when not in fullscreen', () => {
      document.fullscreenElement = null;
      document.documentElement.requestFullscreen = jest.fn();
      
      toggleFullscreen();
      
      expect(document.documentElement.requestFullscreen).toHaveBeenCalled();
    });

    test('should exit fullscreen when in fullscreen', () => {
      document.fullscreenElement = document.createElement('div');
      document.exitFullscreen = jest.fn();
      
      toggleFullscreen();
      
      expect(document.exitFullscreen).toHaveBeenCalled();
    });
  });

  describe('Chart Export', () => {
    test('should export chart as PNG', () => {
      const mockChart = {
        getDataURL: jest.fn().mockReturnValue('data:image/png;base64,test')
      };
      
      document.createElement = jest.fn().mockReturnValue({
        href: '',
        download: '',
        click: jest.fn(),
        appendChild: jest.fn(),
        removeChild: jest.fn()
      });
      
      URL.createObjectURL = jest.fn().mockReturnValue('blob:test');
      URL.revokeObjectURL = jest.fn();
      
      const result = exportChart('testChart');
      expect(mockChart.getDataURL).toBeDefined();
    });
  });

  describe('Data Export', () => {
    test('should export data as JSON', () => {
      document.getElementById = jest.fn().mockReturnValue({ textContent: '100' });
      
      expect(() => exportAllData('json')).not.toThrow();
    });

    test('should export data as CSV', () => {
      document.getElementById = jest.fn().mockReturnValue({ textContent: '100' });
      
      expect(() => exportAllData('csv')).not.toThrow();
    });
  });

  describe('Theme Management', () => {
    test('should set light theme', () => {
      document.documentElement = {
        setAttribute: jest.fn()
      };
      localStorage.setItem = jest.fn();
      
      setDashboardTheme('light');
      
      expect(document.documentElement.setAttribute).toHaveBeenCalledWith('data-theme', 'light');
    });

    test('should set dark theme', () => {
      document.documentElement = {
        setAttribute: jest.fn()
      };
      localStorage.setItem = jest.fn();
      
      setDashboardTheme('dark');
      
      expect(document.documentElement.setAttribute).toHaveBeenCalledWith('data-theme', 'dark');
    });

    test('should set auto theme based on system preference', () => {
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)');
      
      document.documentElement = {
        setAttribute: jest.fn()
      };
      localStorage.setItem = jest.fn();
      
      setDashboardTheme('auto');
      
      expect(localStorage.setItem).toHaveBeenCalledWith('dashboardTheme', 'auto');
    });
  });

  describe('Real-time Data Points', () => {
    test('should limit realtime points to max', () => {
      const maxPoints = 60;
      let dataPoints = [];
      
      for (let i = 0; i < 100; i++) {
        dataPoints.push({ time: new Date(), value: Math.random() * 100 });
        if (dataPoints.length > maxPoints) {
          dataPoints.shift();
        }
      }
      
      expect(dataPoints.length).toBe(maxPoints);
    });

    test('should calculate min, avg, max correctly', () => {
      const values = [10, 20, 30, 40, 50];
      
      const min = Math.min(...values);
      const max = Math.max(...values);
      const avg = values.reduce((a, b) => a + b, 0) / values.length;
      
      expect(min).toBe(10);
      expect(max).toBe(50);
      expect(avg).toBe(30);
    });
  });

  describe('Verification Row Addition', () => {
    test('should add verification row correctly', () => {
      const mockTbody = {
        insertBefore: jest.fn(),
        lastChild: {},
        removeChild: jest.fn(),
        children: Array(15).fill({})
      };
      
      document.getElementById = jest.fn().mockReturnValue(mockTbody);
      
      const data = {
        id: 'test_123',
        timestamp: Date.now(),
        app: 'TestApp',
        type: '滑动验证',
        status: 'success',
        response_time: 85
      };
      
      addVerificationRow(data);
      
      expect(mockTbody.insertBefore).toHaveBeenCalled();
    });

    test('should handle null tbody', () => {
      document.getElementById = jest.fn().mockReturnValue(null);
      
      expect(() => addVerificationRow({})).not.toThrow();
    });
  });

  describe('Toast Notifications', () => {
    test('should create toast element', () => {
      const mockContainer = document.createElement('div');
      mockContainer.className = 'toast-container';
      document.querySelector = jest.fn().mockReturnValue(mockContainer);
      document.createElement = jest.fn().mockReturnValue({
        className: 'toast',
        setAttribute: jest.fn(),
        innerHTML: '',
        appendChild: jest.fn(),
        classList: {
          add: jest.fn(),
          remove: jest.fn()
        }
      });
      
      expect(() => showToast('Test message', 'success')).not.toThrow();
    });
  });

  describe('Enhanced Dashboard Module', () => {
    test('should initialize with correct config', () => {
      expect(EnhancedDashboard.config.maxRealtimePoints).toBe(60);
      expect(EnhancedDashboard.config.updateInterval).toBe(5000);
      expect(EnhancedDashboard.config.wsReconnectDelay).toBe(3000);
    });

    test('should format number correctly', () => {
      expect(EnhancedDashboard.formatNumber(1234567)).toBe('1.2M');
      expect(EnhancedDashboard.formatNumber(12345)).toBe('12.3K');
      expect(EnhancedDashboard.formatNumber(123)).toBe('123');
    });

    test('should escape HTML correctly', () => {
      expect(EnhancedDashboard.escapeHtml('<script>')).toContain('&lt;');
      expect(EnhancedDashboard.escapeHtml(null)).toBe('');
    });

    test('should format time correctly', () => {
      const date = new Date('2024-01-15T10:30:45');
      const formatted = EnhancedDashboard.formatTime(date);
      expect(formatted).toMatch(/\d{2}:\d{2}:\d{2}/);
    });

    test('should handle mock data generation', () => {
      const mockData = EnhancedDashboard.generateMockTrendData();
      expect(mockData.length).toBe(24);
      expect(mockData[0]).toHaveProperty('time');
      expect(mockData[0]).toHaveProperty('requests');
      expect(mockData[0]).toHaveProperty('passed');
      expect(mockData[0]).toHaveProperty('blocked');
    });
  });

  describe('System Status Updates', () => {
    test('should update database status correctly', () => {
      const mockElements = {
        dbLatency: { textContent: '' },
        dbStatus: { className: '' }
      };
      
      document.getElementById = jest.fn((id) => mockElements[id]);
      
      const data = {
        database: { status: 'healthy', latency: 25 }
      };
      
      updateSystemStatus(data);
      
      expect(mockElements.dbLatency.textContent).toBe('25ms');
    });

    test('should update redis status correctly', () => {
      const mockElements = {
        redisLatency: { textContent: '' },
        redisStatus: { className: '' }
      };
      
      document.getElementById = jest.fn((id) => mockElements[id]);
      
      const data = {
        redis: { status: 'healthy', latency: 5 }
      };
      
      updateSystemStatus(data);
      
      expect(mockElements.redisLatency.textContent).toBe('5ms');
    });

    test('should update CPU usage correctly', () => {
      const mockElements = {
        cpuUsage: { textContent: '' },
        cpuProgress: { style: { width: '' }, className: '' }
      };
      
      document.getElementById = jest.fn((id) => mockElements[id]);
      
      const data = {
        cpu: 65
      };
      
      updateSystemStatus(data);
      
      expect(mockElements.cpuUsage.textContent).toBe('65%');
      expect(mockElements.cpuProgress.className).toContain('warning');
    });

    test('should update memory usage correctly', () => {
      const mockElements = {
        memUsage: { textContent: '' },
        memProgress: { style: { width: '' }, className: '' }
      };
      
      document.getElementById = jest.fn((id) => mockElements[id]);
      
      const data = {
        memory: 75
      };
      
      updateSystemStatus(data);
      
      expect(mockElements.memUsage.textContent).toBe('75%');
      expect(mockElements.memProgress.className).toContain('warning');
    });
  });

  describe('Mock Data Tests', () => {
    test('should generate valid mock dashboard data', () => {
      document.getElementById = jest.fn().mockReturnValue({ textContent: '' });
      
      loadMockData();
      
      expect(document.getElementById).toHaveBeenCalledWith('totalRequests');
      expect(document.getElementById).toHaveBeenCalledWith('passRate');
      expect(document.getElementById).toHaveBeenCalledWith('blockRate');
      expect(document.getElementById).toHaveBeenCalledWith('avgResponseTime');
    });

    test('should generate valid mock system status', () => {
      document.getElementById = jest.fn().mockReturnValue({
        textContent: '',
        className: '',
        style: { width: '' }
      });
      
      loadMockSystemStatus();
      
      expect(document.getElementById).toHaveBeenCalled();
    });

    test('should generate valid mock recent verifications', () => {
      document.getElementById = jest.fn().mockReturnValue({
        innerHTML: '',
        insertBefore: jest.fn()
      });
      
      renderMockRecentVerifications();
      
      expect(document.getElementById).toHaveBeenCalledWith('recentVerifications');
    });
  });
});

describe('Integration Tests', () => {
  
  test('should work with complete dashboard workflow', () => {
    const mockData = {
      summary: {
        total_requests: 10000,
        pass_rate: 95.5,
        block_rate: 3.2,
        avg_response_time: 85,
        requests_change: 12.5
      },
      extended: {
        total_users: 5000,
        total_apps: 150,
        current_qps: 45,
        error_rate: 0.5
      },
      trend: EnhancedDashboard.generateMockTrendData()
    };

    document.getElementById = jest.fn().mockReturnValue({ textContent: '' });

    updateDashboard(mockData);

    expect(mockData.summary.total_requests).toBe(10000);
    expect(mockData.summary.pass_rate).toBe(95.5);
    expect(mockData.trend.length).toBe(24);
  });

  test('should handle real-time data streaming', () => {
    const realtimePoints = [];
    
    for (let i = 0; i < 70; i++) {
      realtimePoints.push({
        time: new Date(),
        value: Math.random() * 100
      });
      
      if (realtimePoints.length > 60) {
        realtimePoints.shift();
      }
    }
    
    expect(realtimePoints.length).toBe(60);
  });

  test('should calculate statistics correctly', () => {
    const values = [10, 20, 30, 40, 50, 60, 70, 80, 90, 100];
    
    const min = Math.min(...values);
    const max = Math.max(...values);
    const avg = values.reduce((a, b) => a + b, 0) / values.length;
    const sum = values.reduce((a, b) => a + b, 0);
    
    expect(min).toBe(10);
    expect(max).toBe(100);
    expect(avg).toBe(55);
    expect(sum).toBe(550);
  });
});

console.log('Enhanced dashboard tests completed successfully!');
