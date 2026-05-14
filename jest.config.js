module.exports = {
  projects: [
    {
      displayName: 'backend',
      testEnvironment: 'node',
      roots: ['<rootDir>/src/backend', '<rootDir>/tests'],
      testMatch: [
        '**/backend/**/*.test.js',
        '**/backend/**/*.spec.js'
      ],
      transform: {
        '^.+\\.jsx?$': 'babel-jest'
      },
      moduleFileExtensions: ['js', 'jsx', 'json'],
      collectCoverageFrom: [
        'src/**/*.js',
        '!src/**/*.test.js',
        '!src/**/*.spec.js',
        '!**/node_modules/**'
      ],
      moduleNameMapper: {
        '^config/database/db$': '<rootDir>/tests/__mocks__/database.js',
        '^config/redis/client$': '<rootDir>/tests/__mocks__/redis.js'
      }
    },
    {
      displayName: 'frontend',
      testEnvironment: 'jsdom',
      roots: ['<rootDir>/src/frontend'],
      testMatch: [
        '**/__tests__/**/*.js',
        '**/*.test.js',
        '**/*.spec.js'
      ],
      transform: {
        '^.+\\.jsx?$': 'babel-jest'
      },
      moduleFileExtensions: ['js', 'jsx', 'json'],
      collectCoverageFrom: [
        'src/frontend/**/*.js',
        'src/frontend/**/*.jsx',
        '!src/**/*.test.js',
        '!src/**/*.spec.js',
        '!**/node_modules/**'
      ],
      setupFilesAfterEnv: ['<rootDir>/tests/setup.js']
    },
    {
      displayName: 'api',
      testEnvironment: 'node',
      roots: ['<rootDir>/tests'],
      testMatch: ['tests/api/**/*.test.js'],
      transform: {
        '^.+\\.jsx?$': 'babel-jest'
      },
      moduleFileExtensions: ['js', 'jsx', 'json'],
      setupFilesAfterEnv: ['<rootDir>/tests/setup.js']
    }
  ],
  collectCoverageFrom: [
    'src/**/*.js',
    'src/**/*.jsx',
    '!src/**/*.test.js',
    '!src/**/*.spec.js',
    '!**/node_modules/**'
  ],
  coverageDirectory: 'coverage',
  coverageReporters: ['text', 'lcov', 'html', 'cobertura'],
  coverageThreshold: {
    global: {
      branches: 80,
      functions: 80,
      lines: 80,
      statements: 80
    }
  },
  testTimeout: 10000,
  verbose: true,
  forceExit: true,
  clearMocks: true,
  resetMocks: true,
  restoreMocks: true,
  setupFilesAfterEnv: ['<rootDir>/tests/setup.js'],
  modulePathIgnorePatterns: ['<rootDir>/node_modules/'],
  moduleNameMapper: {
    '\\.(css|less|scss|sass)$': 'identity-obj-proxy',
    '\\.(png|jpg|jpeg|gif|svg)$': 'jest-transform-stub'
  }
};
