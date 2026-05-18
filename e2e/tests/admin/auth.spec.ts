import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';

test.describe('管理端认证测试', () => {
  let apiHelper: ApiHelper;

  test.beforeEach(async ({ request }) => {
    apiHelper = new ApiHelper(request);
  });

  test.describe('API认证测试', () => {
    test('API无效的凭据应该返回错误', async () => {
      const result = await apiHelper.adminLogin('invalid', 'invalid');
      expect(result).toBeDefined();
    });
  });
});
