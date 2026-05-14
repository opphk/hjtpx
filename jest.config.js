module.exports = {
  testEnvironment: 'node',
  roots: ['<rootDir>/src', '<rootDir>'],
  testMatch: ['**/tests/**/*.test.js', '**/__tests__/**/*.test.js'],
  collectCoverageFrom: [
    'src/**/*.js',
    'src/**/*.jsx',
    '!src/**/*.test.js',
    'src/**/__tests__/**',
    '!src/**/node_modules/**'
  ],
  transform: {
    '^.+\\.js$': 'babel-jest'
  },
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
    '\\.(css|less|scss|sass)$': 'identity-obj-proxy',
    '^config/database/db$': '<rootDir>/tests/__mocks__/database.js',
    '^src/config/database/db$': '<rootDir>/tests/__mocks__/database.js',
    '^backend/config/database/db$': '<rootDir>/tests/__mocks__/database.js',
    '^src/backend/config/database/db$': '<rootDir>/tests/__mocks__/database.js',
    '^../../config/database/db$': '<rootDir>/tests/__mocks__/database.js',
    '^../config/database/db$': '<rootDir>/tests/__mocks__/database.js',
    '^../../../config/database/db$': '<rootDir>/tests/__mocks__/database.js'
  },
  setupFilesAfterEnv: ['<rootDir>/tests/setup.js'],
  clearMocks: true,
  coverageDirectory: 'coverage',
  coverageReporters: ['text', 'lcov', 'clover', 'html'],
  coverageThreshold: {
    global: {
      branches: 50,
      functions: 50,
      lines: 50,
      statements: 50
    },
    './src/backend/': {
      branches: 60,
      functions: 60,
      lines: 60,
      statements: 60
    },
    './src/services/': {
      branches: 70,
      functions: 70,
      lines: 70,
      statements: 70
    },
    './src/api/': {
      branches: 80,
      functions: 80,
      lines: 80,
      statements: 80
    }
  },
  coveragePathIgnorePatterns: [
    '/node_modules/',
    '/frontend/node_modules/',
    '/dist/',
    '/build/',
    '/coverage/'
  ],
  coverageProvider: 'v8',
  collectCoverage: process.env.COLLECT_COVERAGE === 'true',
  testPathIgnorePatterns: ['/node_modules/', '/frontend/node_modules/'],
  verbose: true,
  modulePathIgnorePatterns: ['<rootDir>/src/config/database/db.js'],
  testTimeout: 10000
};
