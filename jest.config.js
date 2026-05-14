module.exports = {
  testEnvironment: 'jsdom',
  roots: ['<rootDir>/src'],
  testMatch: ['**/tests/**/*.test.js', '**/__tests__/**/*.test.js'],
  collectCoverageFrom: [
    'src/**/*.js',
    'src/**/*.jsx',
    '!src/**/*.test.js',
    '!src/**/__tests__/**',
    '!src/**/node_modules/**'
  ],
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
    '\\.(css|less|scss|sass)$': 'identity-obj-proxy'
  },
  setupFilesAfterEnv: ['<rootDir>/tests/setup.js'],
  clearMocks: true,
  coverageDirectory: 'coverage',
  coverageReporters: ['text', 'lcov', 'clover'],
  coverageThreshold: {
    global: {
      branches: 70,
      functions: 70,
      lines: 70,
      statements: 70
    }
  },
  testPathIgnorePatterns: ['/node_modules/', '/frontend/node_modules/'],
  verbose: true,
  modulePathIgnorePatterns: ['<rootDir>/src/config/database/db.js']
};
