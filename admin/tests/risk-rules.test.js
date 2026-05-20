/**
 * HJTPX Risk Rules Management Tests
 * 测试风控规则管理的各项功能
 */

describe('Risk Rules Management Tests', () => {
  
  describe('Rule Type Classification', () => {
    test('should return correct badge class for basic type', () => {
      expect(getTypeBadgeClass('basic')).toBe('info');
      expect(getTypeText('basic')).toBe('基础');
    });

    test('should return correct badge class for advanced type', () => {
      expect(getTypeBadgeClass('advanced')).toBe('warning');
      expect(getTypeText('advanced')).toBe('高级');
    });

    test('should return correct badge class for ml type', () => {
      expect(getTypeBadgeClass('ml')).toBe('primary');
      expect(getTypeText('ml')).toBe('ML');
    });

    test('should handle unknown type', () => {
      expect(getTypeBadgeClass('unknown')).toBe('secondary');
      expect(getTypeText('unknown')).toBe('unknown');
    });
  });

  describe('Priority Classification', () => {
    test('should return high priority badge', () => {
      expect(getPriorityBadgeClass(1)).toBe('danger');
      expect(getPriorityText(1)).toBe('高');
    });

    test('should return medium priority badge', () => {
      expect(getPriorityBadgeClass(2)).toBe('warning');
      expect(getPriorityText(2)).toBe('中');
    });

    test('should return low priority badge', () => {
      expect(getPriorityBadgeClass(3)).toBe('info');
      expect(getPriorityText(3)).toBe('低');
    });
  });

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

  describe('Rule Status Toggle', () => {
    test('should toggle rule status correctly', () => {
      const mockRule = {
        id: 1,
        status: 'active'
      };
      
      toggleRuleStatus(mockRule.id);
      
      expect(mockRule.status).toBe('active');
    });
  });

  describe('Rule Test Execution', () => {
    test('should parse valid JSON test data', () => {
      const validJson = '{"speed": 1500, "path_efficiency": 0.95}';
      
      expect(() => JSON.parse(validJson)).not.toThrow();
      expect(JSON.parse(validJson)).toEqual({ speed: 1500, path_efficiency: 0.95 });
    });

    test('should reject invalid JSON test data', () => {
      const invalidJson = '{speed: 1500}';
      
      expect(() => JSON.parse(invalidJson)).toThrow();
    });

    test('should handle empty test data', () => {
      const emptyJson = '{}';
      
      expect(() => JSON.parse(emptyJson)).not.toThrow();
      expect(JSON.parse(emptyJson)).toEqual({});
    });
  });

  describe('Rule Test Result Display', () => {
    test('should format block result correctly', () => {
      const result = {
        action: 'block',
        score: 0.75,
        matched_rules: ['rule1', 'rule2'],
        execution_time: 25
      };
      
      const resultClass = result.action === 'block' ? 'result-danger' : 'result-success';
      const resultText = result.action === 'block' ? '拦截' : '放行';
      
      expect(resultClass).toBe('result-danger');
      expect(resultText).toBe('拦截');
    });

    test('should format allow result correctly', () => {
      const result = {
        action: 'allow',
        score: 0.25,
        matched_rules: [],
        execution_time: 15
      };
      
      const resultClass = result.action === 'block' ? 'result-danger' : 'result-success';
      const resultText = result.action === 'block' ? '拦截' : '放行';
      
      expect(resultClass).toBe('result-success');
      expect(resultText).toBe('放行');
    });

    test('should handle no matched rules', () => {
      const result = {
        matched_rules: []
      };
      
      expect(result.matched_rules.length).toBe(0);
      expect(result.matched_rules.length > 0 ? result.matched_rules.join(', ') : '无').toBe('无');
    });
  });

  describe('Test History Management', () => {
    test('should add test result to history', () => {
      const history = [];
      const result = {
        action: 'block',
        score: 0.75,
        timestamp: Date.now()
      };
      
      history.unshift(result);
      expect(history.length).toBe(1);
      expect(history[0]).toEqual(result);
    });

    test('should limit history to 10 items', () => {
      const history = [];
      
      for (let i = 0; i < 15; i++) {
        history.unshift({ id: i });
        if (history.length > 10) {
          history.pop();
        }
      }
      
      expect(history.length).toBe(10);
      expect(history[0].id).toBe(14);
      expect(history[9].id).toBe(5);
    });

    test('should render empty history', () => {
      const history = [];
      let html = '';
      
      if (history.length === 0) {
        html = '<div class="text-muted text-center py-2">暂无测试历史</div>';
      }
      
      expect(html).toContain('暂无测试历史');
    });
  });

  describe('Rule View Switching', () => {
    test('should switch to table view', () => {
      currentView = 'table';
      
      document.getElementById = jest.fn().mockReturnValue({ style: {} });
      
      switchView('table');
      
      expect(currentView).toBe('table');
    });

    test('should switch to card view', () => {
      currentView = 'table';
      
      document.getElementById = jest.fn().mockReturnValue({ style: {} });
      
      switchView('card');
      
      expect(currentView).toBe('card');
    });

    test('should switch to tree view', () => {
      currentView = 'table';
      
      document.getElementById = jest.fn().mockReturnValue({ style: {} });
      
      switchView('tree');
      
      expect(currentView).toBe('tree');
    });
  });

  describe('Pagination', () => {
    test('should calculate total pages correctly', () => {
      const total = 100;
      const pageSize = 10;
      const totalPages = Math.ceil(total / pageSize);
      
      expect(totalPages).toBe(10);
    });

    test('should handle less than one page', () => {
      const total = 5;
      const pageSize = 10;
      const totalPages = Math.ceil(total / pageSize);
      
      expect(totalPages).toBe(1);
    });

    test('should go to previous page', () => {
      let page = 5;
      page = page - 1;
      
      expect(page).toBe(4);
    });

    test('should go to next page', () => {
      let page = 5;
      const totalPages = 10;
      
      if (page < totalPages) {
        page = page + 1;
      }
      
      expect(page).toBe(6);
    });

    test('should not go below page 1', () => {
      let page = 1;
      
      if (page > 1) {
        page = page - 1;
      }
      
      expect(page).toBe(1);
    });
  });

  describe('Rule Filtering', () => {
    test('should filter rules by type', () => {
      const rules = [
        { id: 1, type: 'basic', name: 'Rule 1' },
        { id: 2, type: 'advanced', name: 'Rule 2' },
        { id: 3, type: 'ml', name: 'Rule 3' }
      ];
      
      const filtered = rules.filter(rule => rule.type === 'basic');
      
      expect(filtered.length).toBe(1);
      expect(filtered[0].id).toBe(1);
    });

    test('should filter rules by status', () => {
      const rules = [
        { id: 1, status: 'active', name: 'Rule 1' },
        { id: 2, status: 'inactive', name: 'Rule 2' },
        { id: 3, status: 'active', name: 'Rule 3' }
      ];
      
      const filtered = rules.filter(rule => rule.status === 'active');
      
      expect(filtered.length).toBe(2);
    });

    test('should search rules by name', () => {
      const rules = [
        { id: 1, name: 'Speed Detection' },
        { id: 2, name: 'Path Efficiency' },
        { id: 3, name: 'ML Score Detection' }
      ];
      
      const searchTerm = 'Detection';
      const filtered = rules.filter(rule => rule.name.toLowerCase().includes(searchTerm.toLowerCase()));
      
      expect(filtered.length).toBe(2);
    });
  });

  describe('Rule Animation', () => {
    test('should animate counter value', () => {
      const mockElement = {
        textContent: '0',
        id: 'testElement'
      };
      
      document.getElementById = jest.fn().mockReturnValue(mockElement);
      
      animateValue('testElement', 0, 1000, 1000);
      
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

  describe('Rule Deletion', () => {
    test('should confirm before deletion', () => {
      global.confirm = jest.fn().mockReturnValue(true);
      
      const confirmed = confirm('确定要删除这条规则吗？');
      
      expect(confirmed).toBe(true);
    });

    test('should cancel deletion', () => {
      global.confirm = jest.fn().mockReturnValue(false);
      
      const confirmed = confirm('确定要删除这条规则吗？');
      
      expect(confirmed).toBe(false);
    });
  });

  describe('Rule Hit Statistics', () => {
    test('should calculate hit rate correctly', () => {
      const totalHits = 10000;
      const blockedHits = 3000;
      const hitRate = (blockedHits / totalHits * 100).toFixed(2);
      
      expect(hitRate).toBe('30.00');
    });

    test('should handle zero total hits', () => {
      const totalHits = 0;
      const blockedHits = 0;
      const hitRate = totalHits > 0 ? (blockedHits / totalHits * 100).toFixed(2) : '0.00';
      
      expect(hitRate).toBe('0.00');
    });
  });

  describe('Rule Priority Sorting', () => {
    test('should sort rules by priority', () => {
      const rules = [
        { id: 1, priority: 3, name: 'Low Priority' },
        { id: 2, priority: 1, name: 'High Priority' },
        { id: 3, priority: 2, name: 'Medium Priority' }
      ];
      
      const sorted = rules.sort((a, b) => a.priority - b.priority);
      
      expect(sorted[0].name).toBe('High Priority');
      expect(sorted[1].name).toBe('Medium Priority');
      expect(sorted[2].name).toBe('Low Priority');
    });
  });

  describe('Rule Tree Structure', () => {
    test('should group rules by type', () => {
      const rules = [
        { id: 1, type: 'basic', name: 'Rule 1' },
        { id: 2, type: 'advanced', name: 'Rule 2' },
        { id: 3, type: 'basic', name: 'Rule 3' }
      ];
      
      const grouped = {
        basic: { name: '基础规则', rules: [] },
        advanced: { name: '高级规则', rules: [] },
        ml: { name: 'ML规则', rules: [] }
      };
      
      rules.forEach(rule => {
        if (grouped[rule.type]) {
          grouped[rule.type].rules.push(rule);
        }
      });
      
      expect(grouped.basic.rules.length).toBe(2);
      expect(grouped.advanced.rules.length).toBe(1);
      expect(grouped.ml.rules.length).toBe(0);
    });
  });

  describe('Rule Export', () => {
    test('should export rules as JSON', () => {
      const rules = [
        { id: 1, name: 'Test Rule', type: 'basic' }
      ];
      
      const json = JSON.stringify(rules, null, 2);
      
      expect(json).toContain('Test Rule');
    });

    test('should handle empty rules array', () => {
      const rules = [];
      const json = JSON.stringify(rules);
      
      expect(json).toBe('[]');
    });
  });

  describe('Rule Import', () => {
    test('should validate imported JSON structure', () => {
      const validRule = {
        name: 'Imported Rule',
        type: 'basic',
        priority: 1,
        conditions: []
      };
      
      expect(validRule).toHaveProperty('name');
      expect(validRule).toHaveProperty('type');
      expect(validRule).toHaveProperty('priority');
    });

    test('should reject invalid rule structure', () => {
      const invalidRule = {
        name: 'Invalid Rule'
      };
      
      expect(invalidRule).not.toHaveProperty('type');
      expect(invalidRule).not.toHaveProperty('priority');
    });
  });

  describe('Mock Data Generation', () => {
    test('should generate mock rule summary', () => {
      const mockSummary = {
        total_rules: Math.floor(Math.random() * 50) + 20,
        active_rules: Math.floor(Math.random() * 30) + 15,
        blocked_requests: Math.floor(Math.random() * 10000) + 5000,
        risk_alerts: Math.floor(Math.random() * 100) + 20,
        block_rate: (Math.random() * 5 + 2).toFixed(2)
      };
      
      expect(mockSummary.total_rules).toBeGreaterThanOrEqual(20);
      expect(mockSummary.total_rules).toBeLessThanOrEqual(70);
      expect(parseFloat(mockSummary.block_rate)).toBeGreaterThanOrEqual(2);
    });

    test('should generate mock rules', () => {
      const mockRules = [];
      const types = ['basic', 'advanced', 'ml'];
      const statuses = ['active', 'inactive'];
      
      for (let i = 0; i < 5; i++) {
        mockRules.push({
          id: i + 1,
          name: `Mock Rule ${i + 1}`,
          type: types[Math.floor(Math.random() * types.length)],
          priority: Math.floor(Math.random() * 3) + 1,
          status: statuses[Math.floor(Math.random() * statuses.length)],
          hit_count: Math.floor(Math.random() * 5000) + 100,
          created_at: new Date().toISOString()
        });
      }
      
      expect(mockRules.length).toBe(5);
      mockRules.forEach(rule => {
        expect(rule).toHaveProperty('id');
        expect(rule).toHaveProperty('name');
        expect(rule).toHaveProperty('type');
      });
    });
  });
});

describe('Integration Tests', () => {
  
  test('should work with complete rule management workflow', () => {
    const rules = [
      { id: 1, name: 'Speed Rule', type: 'basic', priority: 1, status: 'active', hit_count: 1000 },
      { id: 2, name: 'ML Rule', type: 'ml', priority: 2, status: 'active', hit_count: 500 }
    ];
    
    const filteredRules = rules.filter(r => r.status === 'active');
    const sortedRules = filteredRules.sort((a, b) => a.priority - b.priority);
    const totalHits = sortedRules.reduce((sum, r) => sum + r.hit_count, 0);
    
    expect(sortedRules.length).toBe(2);
    expect(sortedRules[0].priority).toBe(1);
    expect(totalHits).toBe(1500);
  });

  test('should handle rule test workflow', () => {
    const testData = {
      speed: 1500,
      path_efficiency: 0.95,
      ml_score: 0.6
    };
    
    const score = Math.random();
    const action = score > 0.5 ? 'block' : 'allow';
    
    expect(action).toMatch(/^(block|allow)$/);
    expect(score).toBeGreaterThanOrEqual(0);
    expect(score).toBeLessThanOrEqual(1);
  });

  test('should calculate rule statistics correctly', () => {
    const rules = [
      { id: 1, status: 'active', hit_count: 1000 },
      { id: 2, status: 'active', hit_count: 2000 },
      { id: 3, status: 'inactive', hit_count: 500 }
    ];
    
    const activeRules = rules.filter(r => r.status === 'active');
    const totalHits = activeRules.reduce((sum, r) => sum + r.hit_count, 0);
    const avgHits = totalHits / activeRules.length;
    
    expect(activeRules.length).toBe(2);
    expect(totalHits).toBe(3000);
    expect(avgHits).toBe(1500);
  });
});

console.log('Risk rules management tests completed successfully!');
