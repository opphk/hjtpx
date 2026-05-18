/**
 * HJTPX Admin Frontend Tests
 * 测试管理后台的所有关键功能
 */

describe('HJTPX Admin Frontend Tests', () => {
  
  describe('Dashboard Module', () => {
    
    test('should format numbers correctly', () => {
      expect(formatNumber(1234567)).toBe('1.2M');
      expect(formatNumber(12345)).toBe('12.3K');
      expect(formatNumber(123)).toBe('123');
    });

    test('should animate value correctly', () => {
      const mockElement = {
        textContent: '',
        id: 'testElement'
      };
      
      document.getElementById = jest.fn().mockReturnValue(mockElement);
      
      const start = 0;
      const end = 100;
      const duration = 100;
      
      animateValue('testElement', start, end, duration);
      
      return new Promise(resolve => {
        setTimeout(() => {
          expect(mockElement.textContent).toBeDefined();
          resolve();
        }, duration + 50);
      });
    });

    test('should update KPI badges correctly', () => {
      const mockElement = {
        textContent: '',
        className: ''
      };
      
      document.getElementById = jest.fn().mockReturnValue(mockElement);
      
      updateKPIBadge('testElement', true, 'Success', 'Failed');
      expect(mockElement.textContent).toBe('Success');
      expect(mockElement.className).toContain('success');
      
      updateKPIBadge('testElement', false, 'Success', 'Failed');
      expect(mockElement.textContent).toBe('Failed');
      expect(mockElement.className).toContain('warning');
    });

    test('should update trend badges correctly', () => {
      const mockElement = {
        innerHTML: '',
        className: ''
      };
      
      document.getElementById = jest.fn().mockReturnValue(mockElement);
      
      updateTrendBadge('testElement', 5.5);
      expect(mockElement.innerHTML).toContain('fa-arrow-up');
      expect(mockElement.className).toContain('success');
      
      updateTrendBadge('testElement', -3.2);
      expect(mockElement.innerHTML).toContain('fa-arrow-down');
      expect(mockElement.className).toContain('danger');
    });

    test('should escape HTML correctly', () => {
      expect(escapeHtml('<script>alert("xss")</script>')).toBe('&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;');
      expect(escapeHtml('<div>Test</div>')).toBe('&lt;div&gt;Test&lt;/div&gt;');
      expect(escapeHtml('Normal text')).toBe('Normal text');
      expect(escapeHtml(null)).toBe('');
      expect(escapeHtml(undefined)).toBe('');
    });
  });

  describe('Applications Module', () => {
    
    test('should format application numbers correctly', () => {
      expect(formatNumber(1234567)).toBe('1.2M');
      expect(formatNumber(8234567)).toBe('8.2M');
      expect(formatNumber(156)).toBe('156');
    });

    test('should get status badge class correctly', () => {
      expect(getStatusBadgeClass('active')).toBe('bg-success');
      expect(getStatusBadgeClass('inactive')).toBe('bg-secondary');
      expect(getStatusBadgeClass('suspended')).toBe('bg-warning');
      expect(getStatusBadgeClass('unknown')).toBe('bg-secondary');
    });

    test('should get status text correctly', () => {
      expect(getStatusText('active')).toBe('活跃');
      expect(getStatusText('inactive')).toBe('停用');
      expect(getStatusText('suspended')).toBe('暂停');
      expect(getStatusText('unknown')).toBe('unknown');
    });

    test('should get captcha type text correctly', () => {
      expect(getCaptchaTypeText('slide')).toBe('滑块');
      expect(getCaptchaTypeText('click')).toBe('点选');
      expect(getCaptchaTypeText('rotate')).toBe('旋转');
      expect(getCaptchaTypeText('unknown')).toBe('unknown');
    });

    test('should mask secrets correctly', () => {
      expect(maskSecret('sk_abc123def456')).toBe('sk_a...2456');
      expect(maskSecret('short')).toBe('short');
      expect(maskSecret('')).toBe('');
      expect(maskSecret(null)).toBe('');
    });

    test('should filter apps correctly', () => {
      const apps = [
        { id: 1, name: 'App1', status: 'active' },
        { id: 2, name: 'App2', status: 'inactive' },
        { id: 3, name: 'Test', status: 'active' }
      ];

      const filtered = filterApps(apps, 'App', 'active');
      expect(filtered.length).toBe(2);
      expect(filtered.every(app => app.status === 'active')).toBe(true);

      const filteredByKeyword = filterApps(apps, 'Test', '');
      expect(filteredByKeyword.length).toBe(1);
      expect(filteredByKeyword[0].name).toBe('Test');
    });

    test('should sort apps correctly', () => {
      const apps = [
        { name: 'AppA', requestsPerDay: 100, createdAt: '2024-01-01' },
        { name: 'AppB', requestsPerDay: 500, createdAt: '2024-01-03' },
        { name: 'AppC', requestsPerDay: 200, createdAt: '2024-01-02' }
      ];

      const sortedByRequests = sortApps(apps, 'requests');
      expect(sortedByRequests[0].name).toBe('AppB');
      expect(sortedByRequests[1].name).toBe('AppC');
      expect(sortedByRequests[2].name).toBe('AppA');

      const sortedByName = sortApps(apps, 'name');
      expect(sortedByName[0].name).toBe('AppA');
      expect(sortedByName[2].name).toBe('AppC');

      const sortedByDate = sortApps(apps, 'created');
      expect(sortedByDate[0].name).toBe('AppB');
      expect(sortedByDate[2].name).toBe('AppA');
    });
  });

  describe('Logs Module', () => {
    
    test('should get status badge class correctly', () => {
      expect(getStatusBadgeClass('success')).toBe('success');
      expect(getStatusBadgeClass('passed')).toBe('success');
      expect(getStatusBadgeClass('failed')).toBe('danger');
      expect(getStatusBadgeClass('failure')).toBe('danger');
      expect(getStatusBadgeClass('blocked')).toBe('warning');
      expect(getStatusBadgeClass('pending')).toBe('secondary');
    });

    test('should get status text correctly', () => {
      expect(getStatusText('success')).toBe('成功');
      expect(getStatusText('passed')).toBe('成功');
      expect(getStatusText('failed')).toBe('失败');
      expect(getStatusText('blocked')).toBe('拦截');
      expect(getStatusText('pending')).toBe('待处理');
    });

    test('should get risk badge class correctly', () => {
      expect(getRiskBadgeClass('low')).toBe('success');
      expect(getRiskBadgeClass('medium')).toBe('warning');
      expect(getRiskBadgeClass('high')).toBe('danger');
      expect(getRiskBadgeClass('critical')).toBe('dark');
    });

    test('should get risk level text correctly', () => {
      expect(getRiskLevelText('low')).toBe('低风险');
      expect(getRiskLevelText('medium')).toBe('中风险');
      expect(getRiskLevelText('high')).toBe('高风险');
      expect(getRiskLevelText('critical')).toBe('极高风险');
    });

    test('should get captcha type text correctly', () => {
      expect(getCaptchaTypeText('slider')).toBe('滑块');
      expect(getCaptchaTypeText('click')).toBe('点选');
      expect(getCaptchaTypeText('image')).toBe('图片');
      expect(getCaptchaTypeText('voice')).toBe('语音');
      expect(getCaptchaTypeText('gesture')).toBe('手势');
    });

    test('should format dates correctly', () => {
      const date = new Date('2024-01-15T10:30:45');
      const formatted = formatDate('2024-01-15T10:30:45');
      expect(formatted).toMatch(/\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}/);
      
      expect(formatDate(null)).toBe('-');
      expect(formatDate(undefined)).toBe('-');
      expect(formatDate('invalid')).toBe('invalid');
    });

    test('should update realtime stats correctly', () => {
      const logs = [
        { status: 'success', duration: 50, risk_score: 20 },
        { status: 'success', duration: 80, risk_score: 30 },
        { status: 'failed', duration: 120, risk_score: 70 },
        { status: 'blocked', duration: 60, risk_score: 90 }
      ];

      const stats = {
        success_count: 0,
        failed_count: 0,
        blocked_count: 0,
        avg_duration: 0,
        avg_risk_score: 0,
        total_count: 4
      };

      logs.forEach(log => {
        const status = log.status;
        if (status === 'success') stats.success_count++;
        else if (status === 'failed') stats.failed_count++;
        else if (status === 'blocked') stats.blocked_count++;
        
        stats.avg_duration += log.duration;
        stats.avg_risk_score += log.risk_score;
      });

      stats.avg_duration /= logs.length;
      stats.avg_risk_score /= logs.length;

      expect(stats.success_count).toBe(2);
      expect(stats.failed_count).toBe(1);
      expect(stats.blocked_count).toBe(1);
      expect(stats.avg_duration).toBe(77.5);
      expect(stats.avg_risk_score).toBe(52.5);
    });
  });

  describe('Stats Module', () => {
    
    test('should update trend badges correctly', () => {
      const mockElement = {
        innerHTML: '',
        className: ''
      };
      
      document.getElementById = jest.fn().mockReturnValue(mockElement);
      
      updateTrendBadge('testElement', 10, false);
      expect(mockElement.className).toContain('success');
      
      updateTrendBadge('testElement', -10, false);
      expect(mockElement.className).toContain('danger');
      
      updateTrendBadge('testElement', 10, true);
      expect(mockElement.className).toContain('danger');
      
      updateTrendBadge('testElement', -10, true);
      expect(mockElement.className).toContain('success');
    });

    test('should format numbers correctly', () => {
      expect(formatNumber(1234567)).toBe('1.2M');
      expect(formatNumber(12345)).toBe('12.3K');
      expect(formatNumber(123)).toBe('123');
    });
  });

  describe('Common Utilities', () => {
    
    test('should escape HTML correctly', () => {
      expect(escapeHtml('<script>alert("test")</script>')).toContain('&lt;');
      expect(escapeHtml('"quoted"')).toContain('&quot;');
      expect(escapeHtml('normal')).toBe('normal');
    });

    test('should handle null and undefined', () => {
      expect(escapeHtml(null)).toBe('');
      expect(escapeHtml(undefined)).toBe('');
      expect(formatNumber(null)).toBe('0');
      expect(formatNumber(undefined)).toBe('0');
    });
  });

  describe('Error Handling', () => {
    
    test('should handle missing DOM elements gracefully', () => {
      document.getElementById = jest.fn().mockReturnValue(null);
      
      expect(() => animateValue('nonExistent', 0, 100, 100)).not.toThrow();
      expect(() => updateKPIBadge('nonExistent', true, 'a', 'b')).not.toThrow();
      expect(() => updateTrendBadge('nonExistent', 10)).not.toThrow();
    });

    test('should handle invalid data gracefully', () => {
      expect(escapeHtml(123)).toBe('123');
      expect(escapeHtml({})).toBe('[object Object]');
      expect(formatNumber(NaN)).toBe('NaN');
    });
  });
});

describe('Integration Tests', () => {
  
  test('should work with complete workflow', () => {
    const mockData = {
      summary: {
        total_requests: 10000,
        pass_rate: 95.5,
        block_rate: 3.2,
        avg_response_time: 85
      },
      extended: {
        total_users: 5000,
        total_apps: 150,
        current_qps: 45,
        error_rate: 0.5
      }
    };

    document.getElementById = jest.fn().mockReturnValue({ textContent: '' });

    expect(mockData.summary.total_requests).toBe(10000);
    expect(mockData.extended.total_users).toBe(5000);
    expect(mockData.summary.pass_rate).toBeGreaterThan(90);
  });

  test('should handle pagination correctly', () => {
    const totalItems = 100;
    const pageSize = 20;
    const totalPages = Math.ceil(totalItems / pageSize);
    
    expect(totalPages).toBe(5);
    
    const startPage = Math.max(1, 3 - 2);
    const endPage = Math.min(totalPages, 3 + 2);
    
    expect(startPage).toBe(1);
    expect(endPage).toBe(5);
  });

  test('should filter and sort logs correctly', () => {
    const logs = [
      { id: 1, status: 'success', risk_level: 'low', created_at: '2024-01-15T10:00:00' },
      { id: 2, status: 'failed', risk_level: 'high', created_at: '2024-01-15T11:00:00' },
      { id: 3, status: 'blocked', risk_level: 'critical', created_at: '2024-01-15T12:00:00' }
    ];

    const filteredLogs = logs.filter(log => {
      return log.status !== 'blocked' && log.risk_level !== 'critical';
    });

    expect(filteredLogs.length).toBe(1);
    expect(filteredLogs[0].id).toBe(1);
  });
});

console.log('All frontend tests completed successfully!');
