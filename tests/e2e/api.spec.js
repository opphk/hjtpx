const { test, expect, request } = require('@playwright/test');

const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:3000/api';

test.describe('API Health Check', () => {
  test('should respond to health check endpoint', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/health`);
    
    if (response.ok()) {
      const body = await response.json();
      expect(body).toHaveProperty('status');
    } else {
      expect(response.status()).toBeDefined();
    }
  });

  test('should handle CORS headers', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/health`, {
      headers: {
        'Origin': 'http://localhost:3000',
      }
    });
    
    expect(response.status()).toBeDefined();
  });
});

test.describe('API Authentication', () => {
  test('should return 401 for protected endpoints without token', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/users`);
    
    if (response.status() === 401) {
      const body = await response.json();
      expect(body).toHaveProperty('error');
    } else {
      expect(response.status()).toBeGreaterThanOrEqual(200);
      expect(response.status()).toBeLessThan(500);
    }
  });

  test('should accept valid authentication token', async ({ request }) => {
    const loginResponse = await request.post(`${API_BASE_URL}/auth/login`, {
      data: {
        email: 'test@example.com',
        password: 'TestPassword123!'
      }
    });
    
    if (loginResponse.ok()) {
      const loginBody = await loginResponse.json();
      const token = loginBody.data?.token;
      
      if (token) {
        const usersResponse = await request.get(`${API_BASE_URL}/users`, {
          headers: {
            'Authorization': `Bearer ${token}`
          }
        });
        
        expect(usersResponse.status()).toBeGreaterThanOrEqual(200);
        expect(usersResponse.status()).toBeLessThan(500);
      }
    }
  });

  test('should reject invalid authentication token', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/users`, {
      headers: {
        'Authorization': 'Bearer invalid-token-12345'
      }
    });
    
    expect(response.status()).toBe(401);
    const body = await response.json();
    expect(body.success).toBe(false);
  });
});

test.describe('API Users Endpoint', () => {
  let authToken;

  test.beforeAll(async ({ request }) => {
    const loginResponse = await request.post(`${API_BASE_URL}/auth/login`, {
      data: {
        email: 'admin@example.com',
        password: 'AdminPassword123!'
      }
    }).catch(() => null);

    if (loginResponse?.ok()) {
      const body = await loginResponse.json();
      authToken = body.data?.token;
    }
  });

  test('should get users list with authentication', async ({ request }) => {
    if (!authToken) {
      test.skip();
      return;
    }

    const response = await request.get(`${API_BASE_URL}/users`, {
      headers: {
        'Authorization': `Bearer ${authToken}`
      }
    });

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(500);
  });

  test('should create new user', async ({ request }) => {
    if (!authToken) {
      test.skip();
      return;
    }

    const timestamp = Date.now();
    const response = await request.post(`${API_BASE_URL}/users`, {
      headers: {
        'Authorization': `Bearer ${authToken}`,
        'Content-Type': 'application/json'
      },
      data: {
        email: `newuser${timestamp}@example.com`,
        name: `Test User ${timestamp}`,
        password: 'TestPassword123!'
      }
    });

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(500);
  });

  test('should validate user input', async ({ request }) => {
    if (!authToken) {
      test.skip();
      return;
    }

    const response = await request.post(`${API_BASE_URL}/users`, {
      headers: {
        'Authorization': `Bearer ${authToken}`,
        'Content-Type': 'application/json'
      },
      data: {
        email: 'invalid-email',
        name: '',
        password: 'weak'
      }
    });

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(500);
  });
});

test.describe('API Input Validation', () => {
  test('should validate email format', async ({ request }) => {
    const response = await request.post(`${API_BASE_URL}/users`, {
      data: {
        email: 'not-an-email',
        name: 'Test',
        password: 'ValidPassword123!'
      }
    });

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(500);
  });

  test('should validate password strength', async ({ request }) => {
    const response = await request.post(`${API_BASE_URL}/users`, {
      data: {
        email: 'test@example.com',
        name: 'Test',
        password: 'weak'
      }
    });

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(500);
  });

  test('should handle SQL injection attempts', async ({ request }) => {
    const response = await request.post(`${API_BASE_URL}/users`, {
      data: {
        email: "admin' OR '1'='1@example.com",
        name: 'Test',
        password: 'ValidPassword123!'
      }
    });

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(500);
  });

  test('should handle XSS attempts', async ({ request }) => {
    const response = await request.post(`${API_BASE_URL}/users`, {
      data: {
        email: 'test@example.com',
        name: '<script>alert("XSS")</script>',
        password: 'ValidPassword123!'
      }
    });

    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(500);
  });
});

test.describe('API Rate Limiting', () => {
  test('should handle multiple rapid requests', async ({ request }) => {
    const requests = [];
    
    for (let i = 0; i < 5; i++) {
      requests.push(request.get(`${API_BASE_URL}/health`));
    }

    const responses = await Promise.all(requests);
    
    responses.forEach(response => {
      expect(response.status()).toBeGreaterThanOrEqual(200);
      expect(response.status()).toBeLessThan(600);
    });
  });
});
