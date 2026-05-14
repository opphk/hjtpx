# Testing Guide

## Overview

This document provides comprehensive guidelines for writing, running, and maintaining tests in the HJTPX project.

## Table of Contents

1. [Test Structure](#test-structure)
2. [Writing Tests](#writing-tests)
3. [Test Best Practices](#test-best-practices)
4. [Coverage Requirements](#coverage-requirements)
5. [Running Tests](#running-tests)
6. [Test Categories](#test-categories)

## Test Structure

```
src/backend/tests/
├── helpers/
│   ├── testHelpers.js          # Database and auth helpers
│   ├── integrationHelpers.js   # Integration test utilities
│   └── setup.js                # Jest setup configuration
├── services/
│   └── userService.test.js     # User service unit tests
├── middleware/
│   └── auth.test.js            # Auth middleware tests
├── routes/
│   └── users.test.js           # API route tests
└── auth/
    └── login.test.js           # Authentication tests
```

## Writing Tests

### Basic Test Structure

```javascript
describe('Feature Name', () => {
  beforeEach(() => {
    // Setup before each test
  });

  afterEach(() => {
    // Cleanup after each test
  });

  it('should do something specific', () => {
    // Test implementation
    expect(result).toBe(expectedValue);
  });
});
```

### Naming Conventions

- Use descriptive test names that explain the expected behavior
- Follow the pattern: `should [expected behavior] when [condition]`
- Group related tests using `describe` blocks

```javascript
describe('UserService', () => {
  describe('createUser', () => {
    it('should create user with hashed password', () => {
      // ...
    });

    it('should reject duplicate email', () => {
      // ...
    });
  });
});
```

## Test Best Practices

### 1. Use Mocks Appropriately

- Mock external dependencies (databases, APIs)
- Use `jest.mock()` for module-level mocks
- Use `jest.fn()` for function mocks

```javascript
jest.mock('../../../config/database/db', () => ({
  query: jest.fn()
}));
```

### 2. Test One Thing Per Test

- Each test should verify a single behavior
- Avoid testing multiple assertions that could fail independently
- Use multiple tests for complex scenarios

### 3. Keep Tests Independent

- Tests should not depend on each other
- Use `beforeEach` and `afterEach` for setup and cleanup
- Reset mocks between tests

```javascript
beforeEach(() => {
  jest.clearAllMocks();
});
```

### 4. Test Edge Cases

- Test empty arrays and objects
- Test boundary conditions
- Test error scenarios
- Test null and undefined values

### 5. Use Proper Assertions

- Use specific assertions that clearly express intent
- Avoid generic assertions that could hide bugs
- Use `toBe()` for primitives
- Use `toEqual()` for objects and arrays
- Use `toThrow()` for testing errors

## Coverage Requirements

### Minimum Thresholds

- **Branches**: 80%
- **Functions**: 80%
- **Lines**: 80%
- **Statements**: 80%

### Coverage Commands

```bash
# Run tests with coverage
npm run test:coverage

# Generate detailed coverage report
npm run test:coverage:report

# Check coverage thresholds (in CI)
npm run test:ci
```

### Coverage Reports

Coverage reports are generated in multiple formats:

- **HTML**: `coverage/lcov-report/index.html`
- **LCOV**: `coverage/lcov.info`
- **JSON**: `coverage/coverage-summary.json`
- **Cobertura XML**: `coverage/cobertura-coverage.xml`

## Running Tests

### Basic Commands

```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run specific test file
npm test -- users.test.js

# Run tests matching pattern
npm test -- --testNamePattern="should create"
```

### Test Categories

```bash
# Run unit tests only
npm run test:unit

# Run integration tests only
npm run test:integration

# Run tests with coverage
npm run test:coverage
```

### Continuous Integration

```bash
# Run tests in CI mode (optimized for CI environments)
npm run test:ci
```

## Test Categories

### 1. Unit Tests

Test individual functions and modules in isolation.

**Location**: `tests/unit/`

```javascript
describe('calculateTotal', () => {
  it('should sum all items', () => {
    const result = calculateTotal([1, 2, 3]);
    expect(result).toBe(6);
  });
});
```

### 2. Service Tests

Test business logic and data access layer.

**Location**: `tests/services/`

```javascript
describe('UserService', () => {
  it('should create user', async () => {
    const user = await userService.createUser({
      email: 'test@example.com',
      name: 'Test',
      password: 'password123'
    });
    expect(user.id).toBeDefined();
  });
});
```

### 3. Middleware Tests

Test Express middleware functions.

**Location**: `tests/middleware/`

```javascript
describe('auth middleware', () => {
  it('should reject invalid token', () => {
    const req = { headers: { authorization: 'Bearer invalid' } };
    const res = { status: jest.fn().mockReturnThis(), json: jest.fn() };
    auth(req, res, () => {});
    expect(res.status).toHaveBeenCalledWith(401);
  });
});
```

### 4. Route Tests

Test API endpoints with mocked services.

**Location**: `tests/routes/`

```javascript
describe('GET /api/users', () => {
  it('should return all users', async () => {
    const response = await request(app).get('/api/users');
    expect(response.status).toBe(200);
  });
});
```

### 5. Integration Tests

Test full application flow with real database.

**Location**: `tests/integration/`

```javascript
describe('User workflow', () => {
  beforeAll(async () => {
    await startTestServer();
  });

  afterAll(async () => {
    await stopTestServer();
  });

  it('should create and retrieve user', async () => {
    const createResponse = await sendRequest('POST', '/api/users', userData);
    const getResponse = await sendRequest('GET', `/api/users/${createResponse.data.id}`);
    expect(getResponse.data.name).toBe(userData.name);
  });
});
```

## Mocking Database

For service tests that interact with the database:

```javascript
jest.mock('../../../config/database/db', () => ({
  query: jest.fn()
}));

// In tests
pool.query.mockResolvedValue({ rows: [{ id: 1 }] });
pool.query.mockRejectedValue(new Error('Database error'));
```

## Mocking JWT

For testing authentication:

```javascript
const jwt = require('jsonwebtoken');

const token = jwt.sign(
  { userId: 1, email: 'test@example.com' },
  process.env.JWT_SECRET || 'test-secret-key',
  { expiresIn: '1h' }
);
```

## Common Patterns

### Testing Async Operations

```javascript
it('should handle async errors', async () => {
  userService.getUserById.mockRejectedValue(new Error('Not found'));
  await expect(userService.getUserById(999)).rejects.toThrow('Not found');
});
```

### Testing Express Routes

```javascript
const request = require('supertest');
const app = require('../../../src/index');

it('should respond to GET /', async () => {
  const response = await request(app).get('/');
  expect(response.status).toBe(200);
});
```

## Troubleshooting

### Tests Not Found

Ensure the test file matches Jest's `testMatch` pattern:

```javascript
// In jest.config.js
testMatch: ['**/*.test.js'],
```

### Database Connection Issues

Use mocks for database tests:

```javascript
jest.mock('../../../config/database/db');
```

### Coverage Thresholds Not Met

Run coverage report to identify uncovered areas:

```bash
npm run test:coverage:report
```

## Additional Resources

- [Jest Documentation](https://jestjs.io/docs/getting-started)
- [Testing Library](https://testing-library.com/docs/react-testing-library/intro/)
- [Supertest Documentation](https://github.com/visionmedia/supertest)
