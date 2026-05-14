require('dotenv').config();

jest.setTimeout(30000);

beforeAll(() => {
  if (!process.env.JWT_SECRET) {
    process.env.JWT_SECRET = 'test-secret-key-for-testing';
  }
});

afterAll(() => {
  jest.clearAllMocks();
});
