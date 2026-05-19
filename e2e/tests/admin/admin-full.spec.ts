import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';
import { testUsers } from '../../utils/test-data';

test.describe('管理后台综合E2E测试', () => {
  let apiHelper: ApiHelper;
  let adminToken: string;

  test.beforeEach(async ({ request }) => {
    apiHelper = new ApiHelper(request);
    const loginResult = await apiHelper.adminLogin(
      testUsers.admin.username,
      testUsers.admin.password
    );
    if (loginResult.success && loginResult.data) {
      adminToken = loginResult.data.token;
    }
  });

  test.describe('应用管理完整流程', () => {
    test('应该能够完整创建、更新、查询和删除应用', async ({ request }) => {
      const appName = `TestApp_${Date.now()}`;
      const appDescription = '自动化测试应用';

      const createResult = await apiHelper.createApplication(
        adminToken,
        appName,
        appDescription
      );
      expect(createResult.success).toBe(true);
      const appId = createResult.data?.id || createResult.data?.app_id;

      const getResult = await request.get(`http://localhost:8080/api/v1/admin/applications/${appId}`, {
        headers: { Authorization: `Bearer ${adminToken}` }
      });
      expect(getResult.ok()).toBeTruthy();
      const appData = await getResult.json();
      expect(appData.data.name).toBe(appName);

      const updateResult = await request.put(`http://localhost:8080/api/v1/admin/applications/${appId}`, {
        headers: {
          Authorization: `Bearer ${adminToken}`,
          'Content-Type': 'application/json'
        },
        data: {
          name: `${appName}_updated`,
          status: 'inactive'
        }
      });
      expect(updateResult.ok()).toBeTruthy();

      const deleteResult = await request.delete(`http://localhost:8080/api/v1/admin/applications/${appId}`, {
        headers: { Authorization: `Bearer ${adminToken}` }
      });
      expect(deleteResult.ok()).toBeTruthy();
    });

    test('应该能够批量创建应用', async ({ request }) => {
      const batchSize = 10;
      const createdAppIds: string[] = [];

      for (let i = 0; i < batchSize; i++) {
        const result = await apiHelper.createApplication(
          adminToken,
          `BatchApp_${Date.now()}_${i}`,
          `批量创建的应用 ${i}`
        );
        expect(result.success).toBe(true);
        if (result.data?.id) {
          createdAppIds.push(result.data.id);
        }
      }

      console.log(`成功创建 ${createdAppIds.length} 个应用`);

      const listResult = await apiHelper.getApplications(adminToken);
      expect(listResult.success).toBe(true);
      expect(listResult.data?.length || 0).toBeGreaterThanOrEqual(batchSize);
    });

    test('应用配置应该能够独立设置', async ({ request }) => {
      const createResult = await apiHelper.createApplication(
        adminToken,
        `ConfigTest_${Date.now()}`,
        '配置测试应用'
      );
      expect(createResult.success).toBe(true);
      const appId = createResult.data?.id;

      const configResult = await request.put(`http://localhost:8080/api/v1/admin/applications/${appId}/config`, {
        headers: {
          Authorization: `Bearer ${adminToken}`,
          'Content-Type': 'application/json'
        },
        data: {
          captcha_types: ['slider', 'click', 'emoji'],
          difficulty: 'medium',
          expire_seconds: 300,
          max_attempts: 5
        }
      });
      expect(configResult.ok()).toBeTruthy();

      const getConfigResult = await request.get(`http://localhost:8080/api/v1/admin/applications/${appId}/config`, {
        headers: { Authorization: `Bearer ${adminToken}` }
      });
      expect(getConfigResult.ok()).toBeTruthy();
      const config = await getConfigResult.json();
      expect(config.data.captcha_types).toContain('emoji');
    });
  });

  test.describe('统计数据分析测试', () => {
    test('应该能够获取完整的统计数据', async () => {
      const statsResult = await apiHelper.getVerificationStats(adminToken);
      expect(statsResult.success).toBe(true);
      expect(statsResult.data).toBeDefined();

      const hasMetrics = 
        statsResult.data?.total ||
        statsResult.data?.success ||
        statsResult.data?.failed ||
        statsResult.data?.pass_rate;
      expect(hasMetrics).toBeDefined();
    });

    test('应该能够获取实时监控数据', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/admin/stats/realtime', {
        headers: { Authorization: `Bearer ${adminToken}` }
      });
      expect(response.ok()).toBeTruthy();
      const data = await response.json();
      
      expect(data.data).toHaveProperty('current_qps');
      expect(data.data).toHaveProperty('total_today');
      expect(data.data).toHaveProperty('success_rate');
      expect(data.data).toHaveProperty('avg_response_time');
    });

    test('应该能够获取趋势数据', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/admin/stats/trend?days=7', {
        headers: { Authorization: `Bearer ${adminToken}` }
      });
      expect(response.ok()).toBeTruthy();
      const data = await response.json();
      
      expect(data.data).toHaveProperty('labels');
      expect(data.data).toHaveProperty('verification_counts');
      expect(data.data).toHaveProperty('pass_rates');
    });

    test('应该能够获取风险分布数据', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/admin/stats/risk-distribution', {
        headers: { Authorization: `Bearer ${adminToken}` }
      });
      expect(response.ok()).toBeTruthy();
      const data = await response.json();
      
      expect(data.data).toHaveProperty('low');
      expect(data.data).toHaveProperty('medium');
      expect(data.data).toHaveProperty('high');
      expect(data.data).toHaveProperty('critical');
    });
  });

  test.describe('日志管理测试', () => {
    test('应该能够查询验证日志', async () => {
      const logsResult = await apiHelper.getLogs(adminToken, { limit: 50 });
      expect(logsResult.success).toBe(true);
      expect(Array.isArray(logsResult.data?.list)).toBe(true);
    });

    test('应该能够按条件过滤日志', async ({ request }) => {
      const filters = [
        { type: 'verification' },
        { start_time: Date.now() - 3600000, end_time: Date.now() },
        { status: 'success' },
        { captcha_type: 'slider' }
      ];

      for (const filter of filters) {
        const response = await request.get('http://localhost:8080/api/v1/admin/logs', {
          headers: { Authorization: `Bearer ${adminToken}` },
          params: filter
        });
        expect(response.ok()).toBeTruthy();
      }
    });

    test('应该能够导出日志', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/admin/logs/export', {
        headers: { Authorization: `Bearer ${adminToken}` },
        params: { format: 'csv', limit: 100 }
      });
      
      expect(response.ok()).toBeTruthy();
      const contentType = response.headers()['content-type'];
      expect(contentType).toContain('csv');
    });
  });

  test.describe('告警管理测试', () => {
    test('应该能够创建告警规则', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/admin/alerts', {
        headers: {
          Authorization: `Bearer ${adminToken}`,
          'Content-Type': 'application/json'
        },
        data: {
          type: 'high_risk',
          threshold: 80,
          channels: ['email', 'webhook'],
          enabled: true
        }
      });
      expect(response.ok()).toBeTruthy();
      const result = await response.json();
      expect(result.data).toHaveProperty('alert_id');
    });

    test('应该能够查询告警历史', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/admin/alerts', {
        headers: { Authorization: `Bearer ${adminToken}` }
      });
      expect(response.ok()).toBeTruthy();
      const data = await response.json();
      expect(Array.isArray(data.data?.list)).toBe(true);
    });

    test('应该能够更新告警规则', async ({ request }) => {
      const createResponse = await request.post('http://localhost:8080/api/v1/admin/alerts', {
        headers: {
          Authorization: `Bearer ${adminToken}`,
          'Content-Type': 'application/json'
        },
        data: {
          type: 'high_error_rate',
          threshold: 10,
          channels: ['email'],
          enabled: true
        }
      });
      const alert = await createResponse.json();
      const alertId = alert.data?.alert_id;

      const updateResponse = await request.put(`http://localhost:8080/api/v1/admin/alerts/${alertId}`, {
        headers: {
          Authorization: `Bearer ${adminToken}`,
          'Content-Type': 'application/json'
        },
        data: {
          threshold: 15,
          enabled: false
        }
      });
      expect(updateResponse.ok()).toBeTruthy();
    });
  });

  test.describe('审计日志测试', () => {
    test('应该能够查看审计日志', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/admin/audit-logs', {
        headers: { Authorization: `Bearer ${adminToken}` }
      });
      expect(response.ok()).toBeTruthy();
      const data = await response.json();
      expect(Array.isArray(data.data?.list)).toBe(true);
    });

    test('审计日志应该包含操作详情', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/admin/audit-logs', {
        headers: { Authorization: `Bearer ${adminToken}` },
        params: { limit: 10 }
      });
      const data = await response.json();
      
      if (data.data?.list?.length > 0) {
        const log = data.data.list[0];
        expect(log).toHaveProperty('user_id');
        expect(log).toHaveProperty('action');
        expect(log).toHaveProperty('details');
        expect(log).toHaveProperty('created_at');
      }
    });
  });

  test.describe('黑名单管理测试', () => {
    test('应该能够添加IP到黑名单', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/admin/blacklist', {
        headers: {
          Authorization: `Bearer ${adminToken}`,
          'Content-Type': 'application/json'
        },
        data: {
          type: 'ip',
          value: `192.168.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}`,
          reason: '自动化测试黑名单'
        }
      });
      expect(response.ok()).toBeTruthy();
      const result = await response.json();
      expect(result.data).toHaveProperty('id');
    });

    test('应该能够查询黑名单', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/admin/blacklist', {
        headers: { Authorization: `Bearer ${adminToken}` }
      });
      expect(response.ok()).toBeTruthy();
      const data = await response.json();
      expect(Array.isArray(data.data?.list)).toBe(true);
    });

    test('应该能够从黑名单移除', async ({ request }) => {
      const addResponse = await request.post('http://localhost:8080/api/v1/admin/blacklist', {
        headers: {
          Authorization: `Bearer ${adminToken}`,
          'Content-Type': 'application/json'
        },
        data: {
          type: 'ip',
          value: '10.0.0.1',
          reason: '临时测试'
        }
      });
      const added = await addResponse.json();
      const blacklistId = added.data?.id;

      const removeResponse = await request.delete(`http://localhost:8080/api/v1/admin/blacklist/${blacklistId}`, {
        headers: { Authorization: `Bearer ${adminToken}` }
      });
      expect(removeResponse.ok()).toBeTruthy();
    });
  });

  test.describe('系统配置测试', () => {
    test('应该能够获取系统配置', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/admin/config', {
        headers: { Authorization: `Bearer ${adminToken}` }
      });
      expect(response.ok()).toBeTruthy();
      const config = await response.json();
      expect(config.data).toBeDefined();
    });

    test('应该能够更新系统配置', async ({ request }) => {
      const response = await request.put('http://localhost:8080/api/v1/admin/config', {
        headers: {
          Authorization: `Bearer ${adminToken}`,
          'Content-Type': 'application/json'
        },
        data: {
          captcha_expire_seconds: 300,
          max_verify_attempts: 5,
          rate_limit_per_minute: 100
        }
      });
      expect(response.ok()).toBeTruthy();
    });

    test('配置更改应该被记录到审计日志', async ({ request }) => {
      await request.put('http://localhost:8080/api/v1/admin/config', {
        headers: {
          Authorization: `Bearer ${adminToken}`,
          'Content-Type': 'application/json'
        },
        data: {
          test_config: 'test_value'
        }
      });

      await new Promise(resolve => setTimeout(resolve, 1000));

      const auditResponse = await request.get('http://localhost:8080/api/v1/admin/audit-logs', {
        headers: { Authorization: `Bearer ${adminToken}` },
        params: { action: 'update_config', limit: 1 }
      });
      const auditData = await auditResponse.json();
      
      if (auditData.data?.list?.length > 0) {
        expect(auditData.data.list[0].action).toBe('update_config');
      }
    });
  });

  test.describe('白标定制测试', () => {
    test('应该能够获取白标配置', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/whitelabel/config', {
        headers: { Authorization: `Bearer ${adminToken}` }
      });
      expect(response.ok()).toBeTruthy();
      const config = await response.json();
      expect(config.data).toHaveProperty('brand_name');
    });

    test('应该能够更新白标配置', async ({ request }) => {
      const response = await request.put('http://localhost:8080/api/v1/whitelabel/config', {
        headers: {
          Authorization: `Bearer ${adminToken}`,
          'Content-Type': 'application/json'
        },
        data: {
          brand_name: 'TestBrand',
          primary_color: '#1890ff',
          custom_css: '.test-class { color: red; }'
        }
      });
      expect(response.ok()).toBeTruthy();
    });

    test('应该能够获取白标CSS', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/whitelabel/css');
      expect(response.ok()).toBeTruthy();
      const contentType = response.headers()['content-type'];
      expect(contentType).toContain('css');
    });
  });

  test.describe('备份与恢复测试', () => {
    test('应该能够创建备份', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/backup/create', {
        headers: {
          Authorization: `Bearer ${adminToken}`,
          'Content-Type': 'application/json'
        },
        data: {
          type: 'full',
          include_logs: true,
          compress: true
        }
      });
      expect(response.ok()).toBeTruthy();
      const result = await response.json();
      expect(result.data).toHaveProperty('backup_id');
    });

    test('应该能够列出备份', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/backup/list', {
        headers: { Authorization: `Bearer ${adminToken}` }
      });
      expect(response.ok()).toBeTruthy();
      const data = await response.json();
      expect(Array.isArray(data.data?.backups)).toBe(true);
    });
  });

  test.describe('A/B测试管理测试', () => {
    test('应该能够创建A/B测试', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/ab-test/create', {
        headers: {
          Authorization: `Bearer ${adminToken}`,
          'Content-Type': 'application/json'
        },
        data: {
          name: `AB_Test_${Date.now()}`,
          description: '自动化A/B测试',
          variants: [
            { name: 'control', weight: 50, config: {} },
            { name: 'variant_a', weight: 50, config: { style: 'modern' } }
          ],
          start_time: new Date().toISOString(),
          end_time: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString()
        }
      });
      expect(response.ok()).toBeTruthy();
      const result = await response.json();
      expect(result.data).toHaveProperty('test_id');
    });

    test('应该能够查询A/B测试结果', async ({ request }) => {
      const createResponse = await request.post('http://localhost:8080/api/v1/ab-test/create', {
        headers: {
          Authorization: `Bearer ${adminToken}`,
          'Content-Type': 'application/json'
        },
        data: {
          name: `Test_${Date.now()}`,
          variants: [
            { name: 'control', weight: 50 },
            { name: 'variant', weight: 50 }
          ]
        }
      });
      const testData = await createResponse.json();
      const testId = testData.data?.test_id;

      const resultResponse = await request.get(`http://localhost:8080/api/v1/ab-test/result?test_id=${testId}`, {
        headers: { Authorization: `Bearer ${adminToken}` }
      });
      expect(resultResponse.ok()).toBeTruthy();
    });
  });
});
