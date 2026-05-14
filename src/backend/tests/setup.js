process.env.NODE_ENV = 'test';
process.env.JWT_SECRET = 'test-secret-key-for-testing';

beforeAll(async () => {
  console.log('Test environment initialized');
});

afterAll(async () => {
  console.log('Test environment cleanup completed');
});
