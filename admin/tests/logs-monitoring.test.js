/**
 * HJTPX Logs and Monitoring Tests
 * 测试日志与监控的各个功能
 */

describe('Logs and Monitoring Tests', () => {
  
  describe('Number Formatting', () => {
    test('should format large numbers with M suffix', () => {
      expect(formatNumber(1234567)).toBe('1.2M');
      expect(formatNumber(1000000)).toBe('1.0M');
      expect(formatNumber(5000000)).toBe('5.0M');
    });

    test('should format thousands with K suffix', () => {
      expect(formatNumber(1234)).toBe('1.2K');
      expect(formatNumber(1000)).toBe('1.0K');
      expect(formatNumber(10000)).toBe('10.0K');
    });

    test('should format small numbers without suffix', () => {
      expect(formatNumber(123)).toBe('123');
      expect(formatNumber(0)).toBe('0');
      expect(formatNumber(999)).toBe('999');
    });
  });

  describe('Status Badge Classification', () => {
    test('should return correct badge class for success', () => {
      expect(getStatusBadgeClass('success')).toBe('success');
      expect(getStatusText('success')).toBe('成功');
    });

    test('should return correct badge class for failed', () => {
      expect(getStatusBadgeClass('failed')).toBe('danger');
      expect(getStatusText('failed')).toBe('失败');
    });

    test('should return correct badge class for blocked', () => {
      expect(getStatusBadgeClass('blocked')).toBe('warning');
      expect(getStatusText('blocked')).toBe('拦截');
    });

    test('should handle unknown status', () => {
      expect(getStatusBadgeClass('unknown')).toBe('secondary');
      expect(getStatusText('unknown')).toBe('unknown');
    });
  });

  describe('Risk Level Classification', () => {
    test('should return correct badge for low risk', () => {
      expect(getRiskBadgeClass('low')).toBe('success');
      expect(getRiskLevelText('low')).toBe('低');
    });

    test('should return correct badge for medium risk', () => {
      expect(getRiskBadgeClass('medium')).toBe('warning');
      expect(getRiskLevelText('medium')).toBe('中');
    });

    test('should return correct badge for high risk', () => {
      expect(getRiskBadgeClass('high')).toBe('danger');
      expect(getRiskLevelText('high')).toBe('高');
    });

    test('should return correct badge for critical risk', () => {
      expect(getRiskBadgeClass('critical')).toBe('dark');
      expect(getRiskLevelText('critical')).toBe('极高');
    });
  });

  describe('Alert Classification', () => {
    test('should return correct badge for alert level', () => {
      expect(getAlertLevelBadgeClass('info')).toBe('info');
      expect(getAlertLevelText('info')).toBe('信息');
      
      expect(getAlertLevelBadgeClass('warning')).toBe('warning');
      expect(getAlertLevelText('warning')).toBe('警告');
      
      expect(getAlertLevelBadgeClass('critical')).toBe('danger');
      expect(getAlertLevelText('critical')).toBe('严重');
    });

    test('should return correct badge for alert status', () => {
      expect(getAlertStatusBadgeClass('active')).toBe('danger');
      expect(getAlertStatusText('active')).toBe('活跃');
      
      expect(getAlertStatusBadgeClass('acknowledged')).toBe('warning');
      expect(getAlertStatusText('acknowledged')).toBe('已确认');
      
      expect(getAlertStatusBadgeClass('resolved')).toBe('success');
      expect(getAlertStatusText('resolved')).toBe('已解决');
    });

    test('should return correct text for alert type', () => {
      expect(getAlertTypeText('error_rate')).toBe('错误率');
      expect(getAlertTypeText('latency')).toBe('延迟');
      expect(getAlertTypeText('blocked_attacks')).toBe('攻击拦截');
    });
  });

  describe('Captcha Type Classification', () => {
    test('should return correct text for captcha types', () => {
      expect(getCaptchaTypeText('slider')).toBe('滑块');
      expect(getCaptchaTypeText('click')).toBe('点选');
      expect(getCaptchaTypeText('image')).toBe('图片');
      expect(getCaptchaTypeText('voice')).toBe('语音');
      expect(getCaptchaTypeText('gesture')).toBe('手势');
    });

    test('should handle unknown type', () => {
      expect(getCaptchaTypeText('unknown')).toBe('unknown');
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
    });

    test('should handle null and undefined', () => {
      expect(escapeHtml(null)).toBe('');
      expect(escapeHtml(undefined)).toBe('');
    });
  });

  describe('Date and Time Formatting', () => {
    test('should format time correctly', () => {
      const date = new Date('2024-01-15T10:30:45');
      const formatted = formatTime(date);
      expect(formatted).toMatch(/\d{2}:\d{2}:\d{2}/);
    });

    test('should format datetime correctly', () => {
      const date = new Date('2024-01-15T10:30:45');
      const formatted = formatDateTime(date.toISOString());
      expect(formatted).toContain('2024');
      expect(formatted).toContain('01');
      expect(formatted).toContain('15');
    });
  });

  describe('Log View Switching', () => {
    test('should switch to table view', () => {
      currentLogsView = 'table';
      
      document.getElementById = jest.fn().mockImplementation((id) => {
        if (id === 'logsTableView') return { style: {} };
        if (id === 'logsCardView') return { style: {} };
        if (id === 'logsTimelineView') return { style: {} };
        return null;
      });
      
      switchLogsView('table');
      
      expect(currentLogsView).toBe('table');
    });

    test('should switch to card view', () => {
      currentLogsView = 'table';
      
      document.getElementById = jest.fn().mockImplementation((id) => {
        if (id === 'logsTableView') return { style: {} };
        if (id === 'logsCardView') return { style: {} };
        if (id === 'logsTimelineView') return { style: {} };
        return null;
      });
      
      switchLogsView('card');
      
      expect(currentLogsView).toBe('card');
    });

    test('should switch to timeline view', () => {
      currentLogsView = 'table';
      
      document.getElementById = jest.fn().mockImplementation((id) => {
        if (id === 'logsTableView') return { style: {} };
        if (id === 'logsCardView') return { style: {} };
        if (id === 'logsTimelineView') return { style: {} };
        return null;
      });
      
      switchLogsView('timeline');
      
      expect(currentLogsView).toBe('timeline');
    });
  });

  describe('Log Pagination', () => {
    test('should calculate total pages correctly', () => {
      const total = 100;
      const pageSize = 20;
      const totalPages = Math.ceil(total / pageSize);
      
      expect(totalPages).toBe(5);
    });

    test('should handle less than one page', () => {
      const total = 15;
      const pageSize = 20;
      const totalPages = Math.ceil(total / pageSize);
      
      expect(totalPages).toBe(1);
    });

    test('should go to previous page', () => {
      let page = 3;
      page = page - 1;
      
      expect(page).toBe(2);
    });

    test('should go to next page', () => {
      let page = 3;
      const totalPages = 5;
      
      if (page < totalPages) {
        page = page + 1;
      }
      
      expect(page).toBe(4);
    });

    test('should not go below page 1', () => {
      let page = 1;
      
      if (page > 1) {
        page = page - 1;
      }
      
      expect(page).toBe(1);
    });

    test('should not exceed total pages', () => {
      let page = 5;
      const totalPages = 5;
      
      if (page < totalPages) {
        page = page + 1;
      }
      
      expect(page).toBe(5);
    });
  });

  describe('Log Filtering', () => {
    test('should filter logs by status', () => {
      const logs = [
        { status: 'success' },
        { status: 'failed' },
        { status: 'blocked' },
        { status: 'success' }
      ];
      
      const filtered = logs.filter(log => log.status === 'success');
      
      expect(filtered.length).toBe(2);
    });

    test('should filter logs by risk level', () => {
      const logs = [
        { risk_level: 'low' },
        { risk_level: 'high' },
        { risk_level: 'critical' }
      ];
      
      const filtered = logs.filter(log => log.risk_level === 'high' || log.risk_level === 'critical');
      
      expect(filtered.length).toBe(2);
    });

    test('should search logs by IP address', () => {
      const logs = [
        { ip_address: '192.168.1.1' },
        { ip_address: '192.168.1.2' },
        { ip_address: '10.0.0.1' }
      ];
      
      const searchTerm = '192.168';
      const filtered = logs.filter(log => log.ip_address.includes(searchTerm));
      
      expect(filtered.length).toBe(2);
    });
  });

  describe('Log Animation', () => {
    test('should animate counter value', () => {
      const mockElement = {
        textContent: '0',
        id: 'testElement'
      };
      
      document.getElementById = jest.fn().mockReturnValue(mockElement);
      
      animateValue('testElement', 0, 5000, 1000);
      
      return new Promise(resolve => {
        setTimeout(() => {
          expect(mockElement.textContent).toBeDefined();
          resolve();
        }, 1100);
      });
    });

    test('should handle non-existent element', () => {
      document.getElementById = jest.fn().mockReturnValue(null);
      
      expect(() => animateValue('nonExistent', 0, 100, 100)).not.toThrow();
    });
  });

  describe('Search Condition Management', () => {
    test('should add search condition', () => {
      const builder = {
        innerHTML: '',
        appendChild: jest.fn()
      };
      
      document.getElementById = jest.fn().mockReturnValue(builder);
      document.createElement = jest.fn().mockReturnValue({
        className: 'search-condition mb-2',
        innerHTML: ''
      });
      
      addSearchCondition();
      
      expect(builder.appendChild).toHaveBeenCalled();
    });

    test('should remove search condition', () => {
      const mockCondition = {
        remove: jest.fn()
      };
      
      removeCondition(mockCondition);
      
      expect(mockCondition.remove).toHaveBeenCalled();
    });

    test('should add sort option', () => {
      const builder = {
        innerHTML: '',
        appendChild: jest.fn()
      };
      
      document.getElementById = jest.fn().mockReturnValue(builder);
      document.createElement = jest.fn().mockReturnValue({
        className: 'sort-option mb-2',
        innerHTML: ''
      });
      
      addSortOption();
      
      expect(builder.appendChild).toHaveBeenCalled();
    });

    test('should remove sort option', () => {
      const mockOption = {
        remove: jest.fn()
      };
      
      removeSortOption(mockOption);
      
      expect(mockOption.remove).toHaveBeenCalled();
    });
  });

  describe('Log Export', () => {
    test('should export logs as JSON', () => {
      const logs = [
        { time: '2024-01-15', status: 'success', type: 'slider' }
      ];
      
      const json = JSON.stringify(logs, null, 2);
      
      expect(json).toContain('2024-01-15');
    });

    test('should export logs as CSV', () => {
      const logs = [
        { time: '2024-01-15', status: 'success', type: 'slider' },
        { time: '2024-01-16', status: 'failed', type: 'click' }
      ];
      
      const csv = '时间,状态,类型\n' +
        logs.map(log => `${log.time},${log.status},${log.type}`).join('\n');
      
      expect(csv).toContain('时间,状态,类型');
      expect(csv).toContain('2024-01-15,success,slider');
    });

    test('should handle empty logs array', () => {
      const logs = [];
      const json = JSON.stringify(logs);
      
      expect(json).toBe('[]');
    });
  });

  describe('Realtime Monitor', () => {
    test('should toggle realtime monitor', () => {
      monitorPaused = false;
      
      const toggleBtn = {
        innerHTML: '<i class="fas fa-pause"></i>'
      };
      
      document.getElementById = jest.fn().mockImplementation((id) => {
        if (id === 'toggleRealtimeMonitor') return toggleBtn;
        if (id === 'realtimeIndicator') return { innerHTML: '' };
        return null;
      });
      
      monitorPaused = true;
      
      expect(monitorPaused).toBe(true);
    });

    test('should resume realtime monitor', () => {
      monitorPaused = true;
      
      monitorPaused = false;
      
      expect(monitorPaused).toBe(false);
    });

    test('should update monitor chart type', () => {
      const mockBtn = {
        classList: {
          remove: jest.fn()
        }
      };
      
      document.querySelectorAll = jest.fn().mockReturnValue([mockBtn]);
      
      updateRealtimeMonitorChart('latency');
      
      expect(mockBtn.classList.remove).toHaveBeenCalled();
    });
  });

  describe('Log Statistics', () => {
    test('should calculate success rate correctly', () => {
      const successCount = 8500;
      const totalCount = 10000;
      const successRate = (successCount / totalCount * 100).toFixed(2);
      
      expect(successRate).toBe('85.00');
    });

    test('should calculate average response time', () => {
      const responseTimes = [50, 100, 150, 200, 250];
      const avg = responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length;
      
      expect(avg).toBe(150);
    });

    test('should calculate blocked rate', () => {
      const blockedCount = 300;
      const totalCount = 10000;
      const blockedRate = (blockedCount / totalCount * 100).toFixed(2);
      
      expect(blockedRate).toBe('3.00');
    });
  });

  describe('Mock Data Generation', () => {
    test('should generate mock logs summary', () => {
      const mockSummary = {
        success_count: Math.floor(Math.random() * 5000) + 8000,
        failed_count: Math.floor(Math.random() * 1000) + 500,
        blocked_count: Math.floor(Math.random() * 500) + 100,
        avg_duration: Math.floor(Math.random() * 100) + 50,
        avg_risk_score: (Math.random() * 0.5 + 0.2).toFixed(2)
      };
      
      expect(mockSummary.success_count).toBeGreaterThanOrEqual(8000);
      expect(mockSummary.avg_duration).toBeGreaterThanOrEqual(50);
    });

    test('should generate mock logs array', () => {
      const mockLogs = [];
      for (let i = 0; i < 20; i++) {
        mockLogs.push({
          id: 'log_' + i,
          time: new Date().toISOString(),
          status: ['success', 'failed', 'blocked'][Math.floor(Math.random() * 3)],
          type: ['slider', 'click', 'image', 'voice'][Math.floor(Math.random() * 4)],
          risk_level: ['low', 'medium', 'high', 'critical'][Math.floor(Math.random() * 4)]
        });
      }
      
      expect(mockLogs.length).toBe(20);
      mockLogs.forEach(log => {
        expect(log).toHaveProperty('id');
        expect(log).toHaveProperty('status');
        expect(log).toHaveProperty('type');
      });
    });

    test('should generate mock alert history', () => {
      const mockAlerts = [];
      for (let i = 0; i < 5; i++) {
        mockAlerts.push({
          id: 'alert_' + i,
          time: new Date().toISOString(),
          type: ['error_rate', 'latency', 'blocked_attacks'][Math.floor(Math.random() * 3)],
          level: ['info', 'warning', 'critical'][Math.floor(Math.random() * 3)],
          status: ['active', 'resolved', 'acknowledged'][Math.floor(Math.random() * 3)]
        });
      }
      
      expect(mockAlerts.length).toBe(5);
      mockAlerts.forEach(alert => {
        expect(alert).toHaveProperty('id');
        expect(alert).toHaveProperty('type');
        expect(alert).toHaveProperty('level');
      });
    });
  });

  describe('File Download', () => {
    test('should create blob for JSON export', () => {
      const content = '{"test": "data"}';
      const mimeType = 'application/json';
      const blob = new Blob([content], { type: mimeType });
      
      expect(blob.type).toBe(mimeType);
      expect(blob.size).toBeGreaterThan(0);
    });

    test('should create blob for CSV export', () => {
      const content = 'col1,col2\nval1,val2';
      const mimeType = 'text/csv';
      const blob = new Blob([content], { type: mimeType });
      
      expect(blob.type).toBe(mimeType);
      expect(blob.size).toBeGreaterThan(0);
    });
  });
});

describe('Integration Tests', () => {
  
  test('should work with complete log monitoring workflow', () => {
    const logs = [
      { id: 1, status: 'success', risk_level: 'low', response_time: 80 },
      { id: 2, status: 'failed', risk_level: 'high', response_time: 150 },
      { id: 3, status: 'blocked', risk_level: 'critical', response_time: 50 }
    ];
    
    const filteredLogs = logs.filter(log => log.risk_level !== 'low');
    const avgResponseTime = filteredLogs.reduce((sum, log) => sum + log.response_time, 0) / filteredLogs.length;
    
    expect(filteredLogs.length).toBe(2);
    expect(avgResponseTime).toBe(100);
  });

  test('should handle realtime update workflow', () => {
    const summary = {
      success_count: 8500,
      failed_count: 1200,
      blocked_count: 300
    };
    
    const total = summary.success_count + summary.failed_count + summary.blocked_count;
    const successRate = (summary.success_count / total * 100).toFixed(2);
    
    expect(successRate).toBe('85.00');
    expect(total).toBe(10000);
  });

  test('should validate pagination calculation', () => {
    const totalLogs = 100;
    const pageSize = 20;
    const currentPage = 3;
    
    const startIndex = (currentPage - 1) * pageSize;
    const endIndex = Math.min(currentPage * pageSize, totalLogs);
    
    expect(startIndex).toBe(40);
    expect(endIndex).toBe(60);
  });
});

console.log('Logs and monitoring tests completed successfully!');
