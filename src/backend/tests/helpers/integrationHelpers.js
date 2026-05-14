const { spawn } = require('child_process');

let testServer = null;

async function startTestServer() {
  return new Promise((resolve, reject) => {
    const serverPath = process.cwd() + '/src/index.js';
    testServer = spawn('node', [serverPath], {
      env: {
        ...process.env,
        NODE_ENV: 'test',
        PORT: '3001'
      },
      stdio: ['pipe', 'pipe', 'pipe']
    });

    testServer.stdout.on('data', (data) => {
      console.log(`Server: ${data}`);
      if (data.toString().includes('running on port')) {
        setTimeout(() => resolve(testServer), 1000);
      }
    });

    testServer.stderr.on('data', (data) => {
      console.error(`Server error: ${data}`);
    });

    testServer.on('error', (error) => {
      console.error('Failed to start server:', error);
      reject(error);
    });

    setTimeout(() => {
      if (!testServer.killed) {
        resolve(testServer);
      }
    }, 3000);
  });
}

async function stopTestServer() {
  if (testServer) {
    testServer.kill('SIGTERM');
    await new Promise(resolve => setTimeout(resolve, 1000));
    testServer = null;
  }
}

async function sendRequest(method, path, body = null, token = null) {
  const baseUrl = process.env.TEST_BASE_URL || 'http://localhost:3001';
  const url = `${baseUrl}${path}`;
  
  const options = {
    method,
    headers: {
      'Content-Type': 'application/json'
    }
  };

  if (token) {
    options.headers['Authorization'] = `Bearer ${token}`;
  }

  if (body) {
    options.body = JSON.stringify(body);
  }

  try {
    const response = await fetch(url, options);
    const data = await response.json();
    return {
      status: response.status,
      data,
      headers: response.headers
    };
  } catch (error) {
    return {
      status: 0,
      error: error.message
    };
  }
}

async function verifyResponse(response, expectedStatus, expectedFields = []) {
  expect(response.status).toBe(expectedStatus);
  
  if (expectedFields.length > 0) {
    expectedFields.forEach(field => {
      expect(response.data).toHaveProperty(field);
    });
  }
}

module.exports = {
  startTestServer,
  stopTestServer,
  sendRequest,
  verifyResponse
};
