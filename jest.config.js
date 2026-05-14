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
  coverageReporters: ['text', 'lcov', 'clover'],
  coverageThreshold: {
    global: {
      branches: 30,
      functions: 30,
      lines: 30,
      statements: 30
    }
  },
  testPathIgnorePatterns: ['/node_modules/', '/frontend/node_modules/'],
  verbose: true,
  modulePathIgnorePatterns: ['<rootDir>/src/config/database/db.js'],
  testTimeout: 10000
};
