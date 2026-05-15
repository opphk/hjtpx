#!/usr/bin/env node

const http = require('http');

function makeRequest(method, path, data = null) {
  return new Promise((resolve, reject) => {
    const options = {
      hostname: 'localhost',
      port: 3000,
      path: path,
      method: method,
      headers: {
        'Content-Type': 'application/json'
      }
    };

    const req = http.request(options, (res) => {
      let body = '';
      res.on('data', (chunk) => body += chunk);
      res.on('end', () => {
        try {
          resolve({
            status: res.statusCode,
            headers: res.headers,
            body: JSON.parse(body)
          });
        } catch (e) {
          resolve({
            status: res.statusCode,
            headers: res.headers,
            body: body
          });
        }
      });
    });

    req.on('error', reject);
    if (data) {
      req.write(JSON.stringify(data));
    }
    req.end();
  });
}

async function runTests() {
  const results = {
    timestamp: new Date().toISOString(),
    tests: []
  };

  console.log('Starting API Tests...\n');

  try {
    console.log('1. Testing Health Check...');
    const health = await makeRequest('GET', '/api/v1/health');
    results.tests.push({
      name: 'Health Check',
      method: 'GET',
      path: '/api/v1/health',
      status: health.status,
      passed: health.status === 200,
      response: health.body
    });
    console.log(`   Status: ${health.status}`);
    console.log(`   Response: ${JSON.stringify(health.body)}\n`);

    console.log('2. Testing User Registration...');
    const registerData = {
      name: 'Test User',
      email: `test_${Date.now()}@example.com`,
      password: 'Test@123456'
    };
    const register = await makeRequest('POST', '/api/v1/auth/register', registerData);
    results.tests.push({
      name: 'User Registration',
      method: 'POST',
      path: '/api/v1/auth/register',
      status: register.status,
      passed: register.status === 201 || register.status === 200,
      response: register.body
    });
    console.log(`   Status: ${register.status}`);
    console.log(`   Response: ${JSON.stringify(register.body)}\n`);

    if (register.body.token) {
      console.log('3. Testing User Login...');
      const loginData = {
        email: registerData.email,
        password: registerData.password
      };
      const login = await makeRequest('POST', '/api/v1/auth/login', loginData);
      results.tests.push({
        name: 'User Login',
        method: 'POST',
        path: '/api/v1/auth/login',
        status: login.status,
        passed: login.status === 200 && login.body.token,
        response: login.body
      });
      console.log(`   Status: ${login.status}`);
      console.log(`   Token: ${login.body.token ? 'Received' : 'Missing'}\n`);

      const token = login.body.token;

      console.log('4. Testing Get Current User...');
      const userReq = http.request({
        hostname: 'localhost',
        port: 3000,
        path: '/api/v1/auth/me',
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${token}`
        }
      }, (res) => {
        let body = '';
        res.on('data', (chunk) => body += chunk);
        res.on('end', () => {
          try {
            const userData = JSON.parse(body);
            console.log(`   Status: ${res.statusCode}`);
            console.log(`   User: ${JSON.stringify(userData)}\n`);
          } catch (e) {
            console.log(`   Status: ${res.statusCode}`);
            console.log(`   Response: ${body}\n`);
          }
        });
      });
      userReq.on('error', console.error);
      userReq.end();

      await new Promise(resolve => setTimeout(resolve, 500));

      console.log('5. Testing Users List (Admin)...');
      const usersReq = http.request({
        hostname: 'localhost',
        port: 3000,
        path: '/api/v1/users',
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${token}`
        }
      }, (res) => {
        let body = '';
        res.on('data', (chunk) => body += chunk);
        res.on('end', () => {
          try {
            const usersData = JSON.parse(body);
            console.log(`   Status: ${res.statusCode}`);
            console.log(`   Users: ${Array.isArray(usersData) ? usersData.length + ' users' : JSON.stringify(usersData)}\n`);
          } catch (e) {
            console.log(`   Status: ${res.statusCode}`);
            console.log(`   Response: ${body}\n`);
          }
        });
      });
      usersReq.on('error', console.error);
      usersReq.end();

      await new Promise(resolve => setTimeout(resolve, 500));
    }

    console.log('6. Testing Invalid Login...');
    const invalidLogin = await makeRequest('POST', '/api/v1/auth/login', {
      email: 'nonexistent@example.com',
      password: 'wrongpassword'
    });
    results.tests.push({
      name: 'Invalid Login',
      method: 'POST',
      path: '/api/v1/auth/login',
      status: invalidLogin.status,
      passed: invalidLogin.status === 401,
      response: invalidLogin.body
    });
    console.log(`   Status: ${invalidLogin.status}`);
    console.log(`   Passed: ${invalidLogin.status === 401}\n`);

    console.log('7. Testing Missing Auth Token...');
    const noAuth = await makeRequest('GET', '/api/v1/users');
    results.tests.push({
      name: 'Missing Auth Token',
      method: 'GET',
      path: '/api/v1/users',
      status: noAuth.status,
      passed: noAuth.status === 401 || noAuth.status === 403,
      response: noAuth.body
    });
    console.log(`   Status: ${noAuth.status}`);
    console.log(`   Passed: ${noAuth.status === 401 || noAuth.status === 403}\n`);

    console.log('8. Testing Not Found Route...');
    const notFound = await makeRequest('GET', '/api/v1/nonexistent');
    results.tests.push({
      name: 'Not Found Route',
      method: 'GET',
      path: '/api/v1/nonexistent',
      status: notFound.status,
      passed: notFound.status === 404,
      response: notFound.body
    });
    console.log(`   Status: ${notFound.status}`);
    console.log(`   Passed: ${notFound.status === 404}\n`);

  } catch (error) {
    console.error('Test Error:', error.message);
    results.error = error.message;
  }

  const passed = results.tests.filter(t => t.passed).length;
  const total = results.tests.length;
  console.log('='.repeat(50));
  console.log(`Test Results: ${passed}/${total} passed`);
  console.log('='.repeat(50));

  return results;
}

runTests().catch(console.error);
